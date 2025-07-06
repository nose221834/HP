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
 * ターミナルエミュレータの実装
 *
 * このファイルは、ブラウザ上で動作するターミナルエミュレータの主要な機能を実装しています。
 * 主な機能:
 * - WebSocketを使用したサーバーとの通信
 * - ターミナルの状態管理
 * - コマンド履歴の管理
 * - キーボード入力の処理
 * - ターミナルの表示制御
 */

/**
 * ターミナルの初期状態を定義する関数
 *
 * @returns {TerminalState} 初期化されたターミナルの状態
 *
 * 状態には以下の情報が含まれます:
 * - isConnected: WebSocket接続の状態
 * - isSubscribed: ActionCableチャンネルの購読状態
 * - isReadyForInput: コマンド入力可能な状態かどうか
 * - currentDir: 現在のディレクトリパス
 * - username: 現在のユーザー名
 * - commandHistory: コマンド履歴の配列
 * - historyIndex: コマンド履歴の現在の位置
 * - isProcessingCommand: コマンド実行中かどうか
 */
function createInitialState(): TerminalState {
  return TerminalStateSchema.parse({
    isConnected: false,
    isSubscribed: false,
    isReadyForInput: false,
    currentDir: INITIAL_DIR,
    username: 'nonroot',
    commandHistory: [],
    historyIndex: -1,
    isProcessingCommand: false,
  });
}

/**
 * WebSocket接続を管理する関数
 */
function createWebSocketManager(
  sessionId: () => string,
  setSessionId: (id: string) => void,
  term: Terminal,
  setStore: (fn: (state: TerminalState) => Partial<TerminalState>) => void,
  initialState: TerminalState,
  _commandBuffer: () => string,
  setCommandBuffer: (fn: (prev: string) => string) => void,
  _cursorPosition: () => number,
  setCursorPosition: (fn: (prev: number) => number) => void
) {
  const [ws, setWs] = createSignal<WebSocket | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = createSignal(0);
  const [reconnectTimeout, setReconnectTimeout] = createSignal<number | null>(
    null
  );
  const [currentState, setCurrentState] = createSignal(initialState);

  /**
   * プロンプトを表示する関数
   *
   * 表示内容:
   * - ユーザー名（緑色）
   * - ホスト名
   * - 現在のディレクトリ（青色）
   * - プロンプト記号（$）
   *
   * 注意:
   * - ターミナルが準備完了していない場合は表示しない
   * - 表示後にコマンドバッファとカーソル位置をリセット
   */
  const writePrompt = () => {
    if (!currentState().isReadyForInput) return;
    const username = currentState().username || 'nonroot';
    const hostname = 'terminal';
    term.write(
      `\x1b[32m${username}@${hostname}\x1b[0m:\x1b[34m${currentState().currentDir}\x1b[0m $ `
    );
    setCommandBuffer(() => '');
    setCursorPosition(() => 0);
  };

  /**
   * 現在の行をクリアして新しいコマンドを表示する関数
   *
   * 処理内容:
   * 1. 現在の行をクリア
   * 2. プロンプトを再表示
   * 3. 新しいコマンドを表示
   * 4. コマンドバッファとカーソル位置を更新
   *
   * 注意:
   * - ターミナルが準備完了していない場合は何もしない
   */
  const clearAndWriteCommand = (command: string) => {
    if (!currentState().isReadyForInput) return;
    const username = currentState().username || 'nonroot';
    const hostname = 'terminal';
    term.write('\r');
    term.write(
      `\x1b[32m${username}@${hostname}\x1b[0m:\x1b[34m${currentState().currentDir}\x1b[0m $ `
    );
    term.write('\x1b[K');
    setCommandBuffer(() => command);
    setCursorPosition(() => command.length);
    term.write(command);
  };

  const updateState = (update: Partial<TerminalState>) => {
    setCurrentState((prev) => ({ ...prev, ...update }));
    setStore(() => update);
  };

  /**
   * WebSocketメッセージを処理する関数
   *
   * @param {WebSocketMessage} data - 受信したWebSocketメッセージ
   *
   * 処理内容:
   * 1. メッセージがオブジェクトの場合:
   *    - pwd: 現在のディレクトリを更新
   *    - username: ユーザー名を更新
   *    - error: エラーメッセージを表示
   *    - result: コマンド実行結果を表示
   *      - lsコマンドの場合は特別な表示処理を実施
   * 2. メッセージが文字列の場合:
   *    - そのままターミナルに表示
   */
  const handleMessage = (data: WebSocketMessage) => {
    if (!data.message) return;

    if (typeof data.message === 'object' && data.message !== null) {
      if ('pwd' in data.message && typeof data.message.pwd === 'string') {
        updateState({ currentDir: data.message.pwd });
      }

      if (
        'username' in data.message &&
        typeof data.message.username === 'string'
      ) {
        updateState({ username: data.message.username });
      }

      if ('error' in data.message && typeof data.message.error === 'string') {
        term.write(`\x1b[31m❌ エラー: ${data.message.error}\x1b[0m\r\n`);
        if (data.message.result?.trim()) {
          term.write(data.message.result + '\r\n');
        }
      } else if (data.message.result?.trim()) {
        // lsコマンドの出力を特別に処理
        const command = data.message.command ?? '';
        if (typeof command === 'string' && command.trim().startsWith('ls')) {
          const items = data.message.result.split('\n').filter(Boolean);

          // ls -l または ls -la の場合、詳細表示モードで処理
          if (command.includes('-l')) {
            // 各行を個別に表示
            for (const line of items) {
              term.write(line + '\r\n');
            }
          } else {
            // 通常のls表示（ターミナルの幅に合わせて表示を調整）
            const width = term.cols;
            const maxItemLength =
              Math.max(...items.map((item) => item.length)) + 1;
            const minItemWidth = 12;
            const effectiveItemWidth = Math.max(maxItemLength, minItemWidth);
            const itemsPerLine = Math.floor(width / effectiveItemWidth);

            for (let i = 0; i < items.length; i += itemsPerLine) {
              const line = items
                .slice(i, i + itemsPerLine)
                .map((item) => item.padEnd(effectiveItemWidth))
                .join('');
              term.write(line + '\r\n');
            }
          }
        } else {
          term.write(data.message.result + '\r\n');
        }
      }
    } else {
      term.write(String(data.message) + '\r\n');
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

  /**
   * WebSocket接続の再接続を試みる関数
   *
   * 実装の特徴:
   * - 指数バックオフ方式で再接続間隔を計算
   * - 最大試行回数に達した場合は再接続を中止
   * - ユーザーに再接続状態を通知
   */
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

  /**
   * WebSocketのイベントハンドラを設定する関数
   *
   * 処理するイベント:
   * - onopen: 接続確立時の処理
   * - onmessage: メッセージ受信時の処理
   *   - ping: 無視
   *   - welcome: ActionCable接続確立通知
   *   - confirm_subscription: チャンネル購読確認
   *   - message: 通常のメッセージ処理
   * - onclose: 接続切断時の処理
   * - onerror: エラー発生時の処理
   */
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

        if (data.type === 'session_id' && data.session_id) {
          setSessionId(data.session_id);
          term.writeln(`🔑 セッションID受信: ${data.session_id}`);
          return;
        }

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

  /**
   * WebSocket経由でコマンドを送信する関数
   *
   * 処理内容:
   * 1. コマンドのバリデーション
   * 2. WebSocket接続の状態確認
   * 3. コマンドの送信
   *
   * エラー処理:
   * - バリデーションエラー: エラーメッセージを表示
   * - 送信失敗: エラーメッセージを表示
   *
   * @returns {boolean} コマンドの送信が成功したかどうか
   */
  const sendCommand = (command: string): boolean => {
    const currentSessionId = sessionId();
    if (!currentSessionId) {
      term.writeln('\x1b[31m❌ セッションIDがまだ受信されていません\x1b[0m');
      return false;
    }

    try {
      const validatedCommand = CommandSchema.parse({
        command,
        session_id: currentSessionId,
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
              command: validatedCommand,
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

  /**
   * WebSocket接続を切断する関数
   *
   * 処理内容:
   * 1. 再接続タイマーのクリア
   * 2. WebSocket接続の切断
   * 3. WebSocketインスタンスのクリア
   */
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
    writePrompt,
    clearAndWriteCommand,
  };
}

/**
 * ターミナルの初期化と状態管理を行う関数
 *
 * @param {HTMLDivElement} container - ターミナルを表示するDOM要素
 * @returns {TerminalReturn} ターミナルの操作に必要な関数と状態
 *
 * 主な機能:
 * - ターミナルの初期化と設定
 * - 各種アドオンの適用（Fit, WebLinks, Search, Unicode11, WebGL）
 * - キーボード入力の処理
 * - コマンドの実行
 * - 状態管理
 * - クリーンアップ処理
 */
export function createTerm(container: HTMLDivElement): TerminalReturn {
  // 状態管理
  const [store, setStore] = createStore<TerminalState>(createInitialState());
  const [sessionId, setSessionId] = createSignal('');

  // コマンドバッファの管理
  const [commandBuffer, setCommandBuffer] = createSignal('');
  const [cursorPosition, setCursorPosition] = createSignal(0);

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
  term.writeln(`🔑 セッションID: サーバーから受信中...`);

  const wsManager = createWebSocketManager(
    sessionId,
    setSessionId,
    term,
    setStore,
    store,
    commandBuffer,
    setCommandBuffer,
    cursorPosition,
    setCursorPosition
  );

  // コマンドを実行する関数
  const executeCommand = (command: string) => {
    if (!command.trim()) {
      wsManager.writePrompt();
      return;
    }

    // コマンドバッファをクリア
    setCommandBuffer(() => '');
    setCursorPosition(() => 0);

    setStore((state) => ({
      isProcessingCommand: true,
      commandHistory: [...state.commandHistory, command].slice(
        -MAX_HISTORY_SIZE
      ),
      historyIndex: -1,
    }));

    if (!wsManager.sendCommand(command)) {
      setStore(() => ({ isProcessingCommand: false }));
      wsManager.writePrompt();
    }
  };

  // WebSocket接続の開始
  wsManager.connect();

  // キー入力の処理
  /**
   * キーボード入力ハンドラー
   *
   * 処理するキー入力:
   * - Tab: 無効化（デフォルトの補完機能を防止）
   * - Ctrl+C: コマンド中断
   * - Enter: コマンド実行
   * - Backspace: 文字削除
   * - 矢印キー:
   *   - 左右: カーソル移動
   *   - 上下: コマンド履歴の操作
   * - その他: 通常の文字入力
   *
   * 制限事項:
   * - コマンド実行中は入力を無視
   * - ターミナルが準備完了していない場合は入力を無視
   */
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
        wsManager.writePrompt();
        return;
      }

      // Enterキーの処理
      if (domEvent.code === 'Enter') {
        if (commandBuffer().trim()) {
          term.write('\r\n');
          executeCommand(commandBuffer());
        } else {
          term.write('\r\n');
          wsManager.writePrompt();
        }
        return;
      }

      // Backspaceキーの処理
      if (domEvent.code === 'Backspace') {
        if (cursorPosition() > 0) {
          setCommandBuffer((prev) => prev.slice(0, -1));
          setCursorPosition((prev) => prev - 1);
          term.write('\b \b');
        }
        return;
      }

      // 矢印キーの処理
      if (domEvent.code === 'ArrowLeft') {
        if (cursorPosition() > 0) {
          setCursorPosition((prev) => prev - 1);
          term.write('\x1b[D');
        }
        return;
      }

      if (domEvent.code === 'ArrowRight') {
        if (cursorPosition() < commandBuffer().length) {
          setCursorPosition((prev) => prev + 1);
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
          wsManager.clearAndWriteCommand(command);
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
          wsManager.clearAndWriteCommand(command);
        } else if (store.historyIndex === 0) {
          setStore('historyIndex', -1);
          wsManager.clearAndWriteCommand('');
        }
        return;
      }

      // 通常の文字入力
      if (printable) {
        setStore('historyIndex', -1);
        setCommandBuffer((prev) => prev + key);
        setCursorPosition((prev) => prev + 1);
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
