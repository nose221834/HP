import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { WebglAddon } from '@xterm/addon-webgl';
import { Terminal } from '@xterm/xterm';
import { createEffect, createSignal, onCleanup } from 'solid-js';
import { createStore } from 'solid-js/store';
import { z } from 'zod';

import {
  CommandSchema,
  INITIAL_DIR,
  MAX_HISTORY_SIZE,
  TERMINAL_OPTIONS,
  TerminalStateSchema,
  WS_RECONNECT_CONFIG,
  WS_URL,
  WebSocketMessageSchema,
} from '../types/terminal';

import type {
  TerminalReturn,
  TerminalState,
  WebSocketError,
  WebSocketMessage,
} from '../types/terminal';

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
 * WebSocket接続を管理する関数
 */
function createWebSocketManager(
  sessionId: string,
  term: Terminal,
  setStore: (fn: (state: TerminalState) => Partial<TerminalState>) => void,
  initialState: TerminalState
) {
  const [ws, setWs] = createSignal<WebSocket | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = createSignal(0);
  const [reconnectTimeout, setReconnectTimeout] = createSignal<number | null>(
    null
  );
  const [currentState, setCurrentState] = createSignal(initialState);

  const updateState = (update: Partial<TerminalState>) => {
    setCurrentState((prev) => ({ ...prev, ...update }));
    setStore(() => update);
  };

  const writePrompt = () => {
    if (!currentState().isReadyForInput) return;
    term.write('\r\n');
    term.write(`\x1b[32m${currentState().currentDir}\x1b[0m $ `);
  };

  const handleMessage = (data: WebSocketMessage) => {
    if (!data.message) return;

    if (typeof data.message === 'object' && data.message !== null) {
      if ('pwd' in data.message && typeof data.message.pwd === 'string') {
        updateState({ currentDir: data.message.pwd });
      }

      if ('error' in data.message && typeof data.message.error === 'string') {
        term.writeln(`\x1b[31m❌ エラー: ${data.message.error}\x1b[0m`);
        if (data.message.result?.trim()) {
          term.writeln(data.message.result);
        }
      } else if (data.message.result?.trim()) {
        term.writeln(data.message.result);
      }
    } else {
      term.writeln(String(data.message));
    }

    updateState({ isProcessingCommand: false });
    writePrompt();
  };

  const handleError = () => {
    updateState({
      isConnected: false,
      isSubscribed: false,
      isReadyForInput: false,
      isProcessingCommand: false,
    });
    attemptReconnect();
  };

  const attemptReconnect = () => {
    const timeout = reconnectTimeout();
    if (timeout) {
      window.clearTimeout(timeout);
    }

    if (reconnectAttempts() >= WS_RECONNECT_CONFIG.maxRetries) {
      term.writeln('\x1b[31m❌ 再接続の試行回数が上限に達しました\x1b[0m');
      return;
    }

    const delay = Math.min(
      WS_RECONNECT_CONFIG.initialDelay *
        Math.pow(WS_RECONNECT_CONFIG.backoffFactor, reconnectAttempts()),
      WS_RECONNECT_CONFIG.maxDelay
    );

    term.writeln(`\x1b[33m🔄 ${delay / 1000}秒後に再接続を試みます...\x1b[0m`);
    const newTimeout = window.setTimeout(() => {
      setReconnectAttempts((prev) => prev + 1);
      connect();
    }, delay);
    setReconnectTimeout(newTimeout);
  };

  const setupEventHandlers = (socket: WebSocket) => {
    socket.onopen = () => {
      term.writeln('✅ 接続しました');
      updateState({
        isConnected: true,
        isProcessingCommand: false,
      });
      setReconnectAttempts(0);

      socket.send(
        JSON.stringify({
          command: 'subscribe',
          identifier: JSON.stringify({
            channel: 'CommandChannel',
          }),
        })
      );
    };

    socket.onmessage = (event: MessageEvent) => {
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
          updateState({
            isSubscribed: true,
            isReadyForInput: true,
          });
          writePrompt();
          return;
        }

        if (data.message) {
          handleMessage(data);
        }
      } catch (error) {
        console.error('WebSocket message processing error:', error);
        handleError();
      }
    };

    socket.onclose = () => {
      term.writeln('🔌 接続が閉じられました');
      updateState({
        isConnected: false,
        isSubscribed: false,
        isReadyForInput: false,
        isProcessingCommand: false,
      });
      attemptReconnect();
    };

    socket.onerror = (event: Event) => {
      const error = event as WebSocketError;
      term.writeln(
        `\x1b[31m⚠️ エラー: ${error.message ?? '不明なエラーが発生しました'}\x1b[0m`
      );
      handleError();
    };
  };

  const connect = () => {
    if (ws()?.readyState === WebSocket.OPEN) return;

    const socket = new WebSocket(WS_URL);
    setWs(socket);
    setupEventHandlers(socket);
  };

  const sendCommand = (command: string): boolean => {
    try {
      const validatedCommand = CommandSchema.parse({
        command,
        session_id: sessionId,
      });

      const socket = ws();
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(
          JSON.stringify({
            command: 'message',
            identifier: JSON.stringify({
              channel: 'CommandChannel',
            }),
            data: JSON.stringify({
              action: 'execute_command',
              command: JSON.stringify(validatedCommand),
            }),
          })
        );
        return true;
      }
      return false;
    } catch (error) {
      if (error instanceof z.ZodError) {
        term.writeln(
          `\x1b[31m❌ バリデーションエラー: ${error.errors[0].message}\x1b[0m`
        );
      } else {
        term.writeln('\x1b[31m❌ コマンドの送信に失敗しました\x1b[0m');
      }
      return false;
    }
  };

  const disconnect = () => {
    const timeout = reconnectTimeout();
    if (timeout) {
      window.clearTimeout(timeout);
      setReconnectTimeout(null);
    }
    const socket = ws();
    if (socket) {
      socket.close();
      setWs(null);
    }
  };

  return {
    connect,
    disconnect,
    sendCommand,
  };
}

/**
 * ターミナルの初期化と状態管理を行う関数
 */
export function createTerm(container: HTMLDivElement): TerminalReturn {
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

  const wsManager = createWebSocketManager(sessionId, term, setStore, store);

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

    setStore((state) => ({
      isProcessingCommand: true,
      commandHistory: [...state.commandHistory, command].slice(
        -MAX_HISTORY_SIZE
      ),
      historyIndex: -1,
    }));

    if (!wsManager.sendCommand(command)) {
      setStore(() => ({ isProcessingCommand: false }));
      writePrompt();
    }
  };

  // WebSocket接続の開始
  wsManager.connect();

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
      wsManager.disconnect();
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
