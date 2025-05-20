import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { Terminal } from '@xterm/xterm';

interface WebSocketMessage {
  type?: string;
  message?: {
    result?: string;
    error?: string;
    pwd?: string;
  };
  identifier?: string;
}

interface WebSocketError extends Event {
  message?: string;
}

/**
 * 初期化済みの Terminal インスタンスと dispose 関数を返す。
 * @param container ターミナルを描画する HTMLDivElement
 */
export function createTerm(container: HTMLDivElement): {
  term: Terminal;
  dispose: () => void;
  executeCommand: (command: string) => void;
} {
  // Terminal の初期化
  const term = new Terminal({
    convertEol: true,
    cursorBlink: true,
    allowProposedApi: true,
  });

  // FitAddon の適用
  const fit = new FitAddon();
  term.loadAddon(fit);

  // その他の標準アドオン
  term.loadAddon(new WebLinksAddon());
  term.loadAddon(new SearchAddon());
  term.loadAddon(new Unicode11Addon());

  // 描画 & フィット
  term.open(container);
  fit.fit();

  // WebSocket接続の設定
  const wsHost =
    window.location.hostname === 'localhost' ? 'localhost' : '192.168.97.1';
  const ws = new WebSocket(`ws://${wsHost}:8000/api/v1/cable`);
  const sessionId = crypto.randomUUID();

  // 現在のディレクトリを保持
  let currentDir = '/home/nonroot';
  let commandBuffer = '';
  let cursorPosition = 0;
  let isProcessingCommand = false;
  let isInitialConnection = true;

  // プロンプトを表示する関数
  const writePrompt = () => {
    term.write('\r\n');
    term.write(`\x1b[32m${currentDir}\x1b[0m $ `);
    cursorPosition = 0;
    commandBuffer = '';
  };

  // 初期メッセージの表示
  term.writeln(`🔌 接続先: ws://${wsHost}:8000/api/v1/cable`);
  term.writeln(`🔑 セッションID: ${sessionId}`);

  // キー入力の処理
  term.onKey(({ key, domEvent }) => {
    if (isProcessingCommand) return;

    const printable =
      !domEvent.altKey && !domEvent.ctrlKey && !domEvent.metaKey;

    if (domEvent.keyCode === 13) {
      // Enter
      if (commandBuffer.trim()) {
        term.write('\r\n');
        executeCommand(commandBuffer);
      } else {
        writePrompt();
      }
    } else if (domEvent.keyCode === 8) {
      // Backspace
      if (cursorPosition > 0) {
        commandBuffer = commandBuffer.slice(0, -1);
        cursorPosition--;
        term.write('\b \b');
      }
    } else if (domEvent.keyCode === 37) {
      // Left arrow
      if (cursorPosition > 0) {
        cursorPosition--;
        term.write('\x1b[D');
      }
    } else if (domEvent.keyCode === 39) {
      // Right arrow
      if (cursorPosition < commandBuffer.length) {
        cursorPosition++;
        term.write('\x1b[C');
      }
    } else if (printable) {
      commandBuffer += key;
      cursorPosition++;
      term.write(key);
    }
  });

  ws.onopen = () => {
    term.writeln('✅ 接続しました');
    // ActionCableの接続確立メッセージを送信
    ws.send(
      JSON.stringify({
        command: 'subscribe',
        identifier: JSON.stringify({
          channel: 'CommandChannel',
        }),
      })
    );
    writePrompt();
  };

  ws.onmessage = (event: MessageEvent) => {
    try {
      const data = JSON.parse(event.data as string) as WebSocketMessage;

      // pingメッセージは完全に無視
      if (data.type === 'ping') {
        return;
      }

      // 初期接続時のメッセージは最初の1回だけ表示
      if (isInitialConnection) {
        if (data.type === 'welcome') {
          term.writeln('✅ ActionCable接続が確立されました');
          return;
        }
        if (data.type === 'confirm_subscription') {
          term.writeln('✅ チャンネルにサブスクライブしました');
          isInitialConnection = false;
          return;
        }
      }

      // コマンド実行結果の処理
      if (data.message) {
        // ディレクトリ変更の処理
        if (data.message.pwd) {
          currentDir = data.message.pwd;
        }

        // エラーメッセージの処理
        if (data.message.error) {
          term.writeln(`\x1b[31m❌ エラー: ${data.message.error}\x1b[0m`);
          if (data.message.result?.trim()) {
            term.writeln(data.message.result);
          }
        }
        // 通常のコマンド結果の処理
        else if (data.message.result?.trim()) {
          term.writeln(data.message.result);
        }

        // コマンド実行が完了したら必ずプロンプトを表示
        if (isProcessingCommand) {
          isProcessingCommand = false;
          writePrompt();
        }
      }
    } catch (error) {
      console.error('WebSocket message processing error:', error);
      // エラー時もプロンプトを表示して操作可能な状態を維持
      if (isProcessingCommand) {
        isProcessingCommand = false;
        writePrompt();
      }
    }
  };

  ws.onclose = () => {
    term.writeln('🔌 接続が閉じられました');
    isProcessingCommand = false;
    writePrompt();
  };

  ws.onerror = (event: Event) => {
    const error = event as WebSocketError;
    term.writeln(
      `\x1b[31m⚠️ エラー: ${error.message ?? '不明なエラーが発生しました'}\x1b[0m`
    );
    isProcessingCommand = false;
    writePrompt();
  };

  // コマンド実行関数
  const executeCommand = (command: string) => {
    if (!command.trim()) {
      writePrompt();
      return;
    }

    const message = {
      command,
      session_id: sessionId,
    };

    isProcessingCommand = true;

    ws.send(
      JSON.stringify({
        command: 'message',
        identifier: JSON.stringify({
          channel: 'CommandChannel',
        }),
        data: JSON.stringify({
          action: 'execute_command',
          command: JSON.stringify(message),
        }),
      })
    );
  };

  // dispose 用の関数
  const dispose = () => {
    try {
      ws.close();
      term.dispose();
    } catch {
      // ignore
    }
  };

  return { term, dispose, executeCommand };
}
