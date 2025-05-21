import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { WebglAddon } from '@xterm/addon-webgl';
import { Terminal } from '@xterm/xterm';
import { createEffect, onCleanup } from 'solid-js';
import { createStore } from 'solid-js/store';

import {
  INITIAL_DIR,
  MAX_HISTORY_SIZE,
  TERMINAL_OPTIONS,
  TerminalStateSchema,
  WS_URL,
  WebSocketMessageSchema,
} from '../types/terminal';

import type { TerminalState } from '../types/terminal';

// 型定義
interface WebSocketError extends Event {
  message?: string;
}

/**
 * ターミナルの状態を初期化
 */
function createInitialState(): TerminalState {
  return TerminalStateSchema.parse({
    isConnected: false,
    isSubscribed: false,
    isReadyForInput: false,
    currentDir: INITIAL_DIR,
    commandHistory: [],
    historyIndex: -1,
    isProcessingCommand: false,
  });
}

/**
 * ターミナルの初期化と状態管理を行う関数
 */
export function createTerm(container: HTMLDivElement) {
  // 状態管理
  const [store, setStore] = createStore<TerminalState>(createInitialState());
  const sessionId = crypto.randomUUID();

  // ターミナルの初期化
  const term = new Terminal(TERMINAL_OPTIONS);

  // アドオンの適用
  const fit = new FitAddon();
  term.loadAddon(fit);
  term.loadAddon(new WebLinksAddon());
  term.loadAddon(new SearchAddon());
  term.loadAddon(new Unicode11Addon());

  // WebGLアドオンの適用
  let webglAddon: WebglAddon | undefined;
  try {
    webglAddon = new WebglAddon();
    term.loadAddon(webglAddon);
    webglAddon.onContextLoss(() => {
      if (webglAddon) {
        webglAddon.dispose();
        webglAddon = undefined;
      }
      console.warn('WebGL context was lost, falling back to canvas renderer');
    });
  } catch (e) {
    console.warn(
      'WebGL addon could not be loaded, falling back to canvas renderer:',
      e
    );
  }

  // 描画 & フィット
  term.open(container);
  fit.fit();

  // 初期メッセージの表示
  term.writeln(`🔌 接続先: ${WS_URL}`);
  term.writeln(`🔑 セッションID: ${sessionId}`);

  // WebSocket接続の設定
  const ws = new WebSocket(WS_URL);

  // 現在のコマンドバッファを管理
  let commandBuffer = '';
  let cursorPosition = 0;

  // プロンプトを表示する関数
  const writePrompt = () => {
    if (!store.isReadyForInput) return;
    term.write('\r\n');
    term.write(`\x1b[32m${store.currentDir}\x1b[0m $ `);
    commandBuffer = '';
    cursorPosition = 0;
  };

  // 現在の行をクリアして新しいコマンドを表示する関数
  const clearAndWriteCommand = (command: string) => {
    if (!store.isReadyForInput) return;
    term.write('\r');
    term.write(`\x1b[32m${store.currentDir}\x1b[0m $ `);
    term.write('\x1b[K');
    commandBuffer = command;
    cursorPosition = command.length;
    term.write(command);
  };

  // コマンドを実行する関数
  const executeCommand = (command: string) => {
    if (!command.trim()) {
      writePrompt();
      return;
    }

    setStore({
      isProcessingCommand: true,
      commandHistory: [...store.commandHistory, command].slice(
        -MAX_HISTORY_SIZE
      ),
      historyIndex: -1,
    });

    ws.send(
      JSON.stringify({
        command: 'message',
        identifier: JSON.stringify({
          channel: 'CommandChannel',
        }),
        data: JSON.stringify({
          action: 'execute_command',
          command: JSON.stringify({
            command,
            session_id: sessionId,
          }),
        }),
      })
    );
  };

  // WebSocketイベントハンドラ
  ws.onopen = () => {
    term.writeln('✅ 接続しました');
    setStore('isConnected', true);

    ws.send(
      JSON.stringify({
        command: 'subscribe',
        identifier: JSON.stringify({
          channel: 'CommandChannel',
        }),
      })
    );
  };

  ws.onmessage = (event: MessageEvent) => {
    try {
      const data = WebSocketMessageSchema.parse(
        JSON.parse(event.data as string)
      );

      if (data.type === 'ping') return;

      if (data.type === 'welcome') {
        term.writeln('✅ ActionCable接続が確立されました');
        return;
      }

      if (data.type === 'confirm_subscription') {
        term.writeln('✅ チャンネルにサブスクライブしました');
        setStore({
          isSubscribed: true,
          isReadyForInput: true,
        });
        writePrompt();
        return;
      }

      if (data.message) {
        // メッセージがオブジェクトの場合
        if (typeof data.message === 'object') {
          if (data.message.pwd) {
            setStore('currentDir', data.message.pwd);
          }

          if (data.message.error) {
            term.writeln(`\x1b[31m❌ エラー: ${data.message.error}\x1b[0m`);
            if (data.message.result?.trim()) {
              term.writeln(data.message.result);
            }
          } else if (data.message.result?.trim()) {
            term.writeln(data.message.result);
          }
        }
        // メッセージが文字列または数値の場合
        else {
          term.writeln(String(data.message));
        }

        if (store.isProcessingCommand) {
          setStore('isProcessingCommand', false);
          writePrompt();
        }
      }
    } catch (error) {
      console.error('WebSocket message processing error:', error);
      if (store.isProcessingCommand) {
        setStore('isProcessingCommand', false);
        if (store.isReadyForInput) {
          writePrompt();
        }
      }
    }
  };

  ws.onclose = () => {
    term.writeln('🔌 接続が閉じられました');
    setStore({
      isConnected: false,
      isSubscribed: false,
      isReadyForInput: false,
      isProcessingCommand: false,
    });
  };

  ws.onerror = (event: Event) => {
    const error = event as WebSocketError;
    term.writeln(
      `\x1b[31m⚠️ エラー: ${error.message ?? '不明なエラーが発生しました'}\x1b[0m`
    );
    setStore({
      isConnected: false,
      isSubscribed: false,
      isReadyForInput: false,
      isProcessingCommand: false,
    });
  };

  // キー入力の処理
  createEffect(() => {
    const handler = ({
      key,
      domEvent,
    }: {
      key: string;
      domEvent: KeyboardEvent;
    }) => {
      if (!store.isReadyForInput || store.isProcessingCommand) return;

      // タブキーを無効化
      if (domEvent.code === 'Tab') {
        domEvent.preventDefault();
        return;
      }

      const printable =
        !domEvent.altKey && !domEvent.ctrlKey && !domEvent.metaKey;

      // Ctrl+Cの処理
      if (domEvent.ctrlKey && domEvent.code === 'KeyC') {
        term.write('^C\r\n');
        setStore('historyIndex', -1);
        writePrompt();
        return;
      }

      // Enterキーの処理
      if (domEvent.code === 'Enter') {
        term.write('\r\n');
        if (commandBuffer.trim()) {
          executeCommand(commandBuffer);
        } else {
          writePrompt();
        }
        return;
      }

      // Backspaceキーの処理
      if (domEvent.code === 'Backspace') {
        if (cursorPosition > 0) {
          commandBuffer = commandBuffer.slice(0, -1);
          cursorPosition--;
          term.write('\b \b');
        }
        return;
      }

      // 矢印キーの処理
      if (domEvent.code === 'ArrowLeft') {
        if (cursorPosition > 0) {
          cursorPosition--;
          term.write('\x1b[D');
        }
        return;
      }

      if (domEvent.code === 'ArrowRight') {
        if (cursorPosition < commandBuffer.length) {
          cursorPosition++;
          term.write('\x1b[C');
        }
        return;
      }

      if (domEvent.code === 'ArrowUp') {
        if (store.historyIndex < store.commandHistory.length - 1) {
          setStore('historyIndex', store.historyIndex + 1);
          const command =
            store.commandHistory[
              store.commandHistory.length - 1 - store.historyIndex
            ];
          clearAndWriteCommand(command);
        }
        return;
      }

      if (domEvent.code === 'ArrowDown') {
        if (store.historyIndex > 0) {
          setStore('historyIndex', store.historyIndex - 1);
          const command =
            store.commandHistory[
              store.commandHistory.length - 1 - store.historyIndex
            ];
          clearAndWriteCommand(command);
        } else if (store.historyIndex === 0) {
          setStore('historyIndex', -1);
          clearAndWriteCommand('');
        }
        return;
      }

      // 通常の文字入力
      if (printable) {
        setStore('historyIndex', -1);
        commandBuffer += key;
        cursorPosition++;
        term.write(key);
      }
    };

    term.onKey(handler);
    onCleanup(() => {
      // キーハンドラーはterm.dispose()で自動的にクリーンアップされる
    });
  });

  // クリーンアップ関数
  const dispose = () => {
    try {
      if (webglAddon) {
        webglAddon.dispose();
      }
      ws.close();
      term.dispose();
    } catch {
      // ignore
    }
  };

  onCleanup(dispose);

  return {
    term,
    store,
    dispose,
    executeCommand,
  };
}
