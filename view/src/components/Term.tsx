import { FitAddon } from '@xterm/addon-fit';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { WebglAddon } from '@xterm/addon-webgl';
import { Terminal } from '@xterm/xterm';
import { onCleanup, onMount } from 'solid-js';

import type { ResizeMessage } from '../types/terminal';
import '@xterm/xterm/css/xterm.css';

const WS_URL = 'ws://localhost:9090/ws';

export function Term() {
  let containerRef!: HTMLDivElement;

  onMount(() => {
    const term = new Terminal({
      allowProposedApi: true,
      // フォント設定
      fontFamily: 'monospace',
      fontSize: 14,
      // カーソル設定
      cursorBlink: true,
      cursorStyle: 'block',
      // スクロールバック行数
      scrollback: 1000,
      // テーマ（黒背景・白文字）
      theme: {
        background: '#000000',
        foreground: '#ffffff',
        cursor: '#ffffff',
      },
    });

    // fitAddon: コンテナのサイズに合わせてターミナルをリサイズ
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    // unicode11Addon: Unicode 11のサポート（絵文字・特殊文字）
    const unicode11Addon = new Unicode11Addon();
    term.loadAddon(unicode11Addon);
    term.unicode.activeVersion = '11';

    // webLinksAddon: URL自動リンク化
    term.loadAddon(new WebLinksAddon());

    // ターミナルをDOMにマウント
    term.open(containerRef);

    // WebGL アドオン: 高速レンダリング
    // open()の後にロードする必要がある（DOMが必要なため）
    // WebGL非対応環境ではフォールバックして通常レンダリングで動く
    try {
      const webglAddon = new WebglAddon();
      webglAddon.onContextLoss(() => {
        // WebGLコンテキストが失われたときはアドオンを破棄
        webglAddon.dispose();
      });
      term.loadAddon(webglAddon);
    } catch {
      // WebGL非対応環境では無視してCanvasレンダラーで動作する
      console.warn('WebGL not available, falling back to canvas renderer');
    }

    // 初回フィット: DOMマウント直後にサイズを合わせる
    fitAddon.fit();

    const ws = new WebSocket(WS_URL);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      sendResize(ws, term.rows, term.cols);
    };

    ws.onmessage = (event: MessageEvent) => {
      if (event.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(event.data));
      }
    };

    ws.onclose = () => {
      term.writeln('\r\n\x1b[31m[接続が切断されました]\x1b[0m');
    };

    ws.onerror = () => {
      term.writeln('\r\n\x1b[31m[接続エラーが発生しました]\x1b[0m');
    };

    // ── 5. ユーザー入力をGoへ送信 ─────────────────────────────
    // xterm.jsのonDataはユーザーのキーストロークを文字列で渡す
    term.onData((data: string) => {
      if (ws.readyState !== WebSocket.OPEN) return;
      // 文字列 → Uint8Array → ArrayBuffer に変換してバイナリ送信
      const encoder = new TextEncoder();
      ws.send(encoder.encode(data));
    });

    // ── 6. ターミナルリサイズの処理 ──────────────────────────
    // ウィンドウのリサイズを検知してターミナルサイズを更新する
    const handleResize = () => {
      fitAddon.fit();
      // fitAddon.fit()後にterm.rows/colsが更新されるので送信
      if (ws.readyState === WebSocket.OPEN) {
        sendResize(ws, term.rows, term.cols);
      }
    };

    window.addEventListener('resize', handleResize);

    // ── 7. クリーンアップ ────────────────────────────────────
    // コンポーネントがアンマウントされたときにリソースを解放
    onCleanup(() => {
      window.removeEventListener('resize', handleResize);
      ws.close();
      term.dispose();
    });
  });

  return (
    // ターミナルを画面全体に表示する
    <div
      ref={containerRef}
      style={{
        width: '100%',
        height: '100%',
        // xterm.jsが内部でサイズを計算するために必要
        overflow: 'hidden',
      }}
    />
  );
}

// GoにリサイズメッセージをJSON（テキスト）で送信する
// バイナリ（キー入力）とテキスト（リサイズ）で種別を分けている
function sendResize(ws: WebSocket, rows: number, cols: number) {
  const msg: ResizeMessage = { type: 'resize', rows, cols };
  ws.send(JSON.stringify(msg));
}
