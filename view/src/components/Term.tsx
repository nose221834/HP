import { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';

function Term() {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstance = useRef<Terminal | null>(null);

  useEffect(() => {
    // コンポーネントがマウントされた後にターミナルを初期化
    if (terminalRef.current) {
      // 既存のインスタンスがあれば破棄
      if (terminalInstance.current) {
        terminalInstance.current.dispose();
      }

      // 新しいターミナルインスタンスを作成
      const term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'monospace',
        theme: {
          background: '#1e1e1e',
          foreground: '#f8f8f8',
        },
      });

      term.open(terminalRef.current);
      term.write('Welcome to xterm.js!\r\n$ ');

      // ユーザー入力の基本的な処理
      term.onKey(({ key, domEvent }) => {
        const printable =
          !domEvent.altKey && !domEvent.ctrlKey && !domEvent.metaKey;

        if (domEvent.keyCode === 13) {
          // Enter キー
          term.write('\r\n$ ');
        } else if (domEvent.keyCode === 8) {
          // Backspace キー
          // カーソルが行の先頭（プロンプトの後）より右にある場合のみ削除
          if (term.buffer.active.cursorX > 2) {
            term.write('\b \b');
          }
        } else if (printable) {
          term.write(key);
        }
      });

      terminalInstance.current = term;
    }

    // クリーンアップ関数
    return () => {
      if (terminalInstance.current) {
        terminalInstance.current.dispose();
        terminalInstance.current = null;
      }
    };
  }, []);

  return (
    <div className="w-full h-screen flex flex-col bg-gray-900 p-2.5">
      <div
        ref={terminalRef}
        className="flex-1 overflow-hidden rounded-md p-1 shadow-lg"
      />
    </div>
  );
}

export default Term;
