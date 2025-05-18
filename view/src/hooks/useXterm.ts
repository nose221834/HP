import { useLayoutEffect, useRef } from 'react';
import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import { WebLinksAddon } from '@xterm/addon-web-links';
import type { ITerminalAddon, ITerminalOptions } from '@xterm/xterm';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';

interface UseXtermOptions extends ITerminalOptions {
  addons?: ITerminalAddon[];
}

export function useXterm(
  containerRef: React.RefObject<HTMLDivElement | null>,
  options: UseXtermOptions = {}
) {
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);

  // 行バッファと履歴バッファをrefで保持
  const lineBufferRef = useRef<string[]>([]);
  const historyRef = useRef<string[]>([]);
  const historyIndexRef = useRef<number>(0);
  const shellListenerRef = useRef<ReturnType<Terminal['onData']> | null>(null);

  const optsJSON = JSON.stringify(options);

  useLayoutEffect(() => {
    if (!containerRef.current) return;

    // 既存インスタンス破棄
    termRef.current?.dispose();

    const term = new Terminal({
      convertEol: true,
      cursorBlink: true,
      allowProposedApi: true,
      ...options,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.loadAddon(new WebLinksAddon());
    term.loadAddon(new SearchAddon());
    term.loadAddon(new Unicode11Addon());
    (options.addons || []).forEach((a) => term.loadAddon(a));

    // コンテナスタイル
    Object.assign(containerRef.current.style, {
      width: '100%',
      height: '100%',
      overflow: 'hidden',
    });

    term.open(containerRef.current);
    // サイズ合わせ
    setTimeout(() => fit.fit(), 0);
    // リサイズ監視
    const ro = new ResizeObserver(() => fit.fit());
    ro.observe(containerRef.current);

    termRef.current = term;
    fitRef.current = fit;

    // --- simpleShell 定義 ---
    const simpleShell = (data: string) => {
      for (let i = 0; i < data.length; ++i) {
        const c = data[i];
        // Enter
        if (c === '\r') {
          term.write('\r\n');
          const cmd = lineBufferRef.current.join('');
          if (cmd) {
            historyRef.current.push(cmd);
            historyIndexRef.current = historyRef.current.length;
            // ここでバックエンド実行など async 処理があれば
            // await exec(cmd);
          }
          lineBufferRef.current.length = 0;
          term.write('SimpleShell> ');
        }
        // Backspace
        else if (c === '\x7f' || c === '\b') {
          if (lineBufferRef.current.length) {
            lineBufferRef.current.pop();
            term.write('\b \b');
          }
        }
        // Arrow keys（履歴参照）
        else if (data.slice(i, i + 3) === '\x1b[A') {
          // ↑
          if (historyRef.current.length && historyIndexRef.current > 0) {
            historyIndexRef.current--;
          }
          const entry = historyRef.current[historyIndexRef.current] || '';
          lineBufferRef.current = Array.from(entry);
          term.write('\x1b[2K\rSimpleShell> ' + entry);
          i += 2;
        } else if (data.slice(i, i + 3) === '\x1b[B') {
          // ↓
          if (historyIndexRef.current < historyRef.current.length) {
            historyIndexRef.current++;
          }
          const entry = historyRef.current[historyIndexRef.current] || '';
          lineBufferRef.current = Array.from(entry);
          term.write('\x1b[2K\rSimpleShell> ' + entry);
          i += 2;
        }
        // それ以外の通常文字
        else {
          lineBufferRef.current.push(c);
          term.write(c);
        }
      }
    };

    // shell listener を登録
    shellListenerRef.current = term.onData(simpleShell);
    // プロンプト表示
    term.write('SimpleShell> ');

    // window リサイズでも fit
    const onWinResize = () => fit.fit();
    window.addEventListener('resize', onWinResize);

    return () => {
      // ハンドラ破棄
      shellListenerRef.current?.dispose();
      ro.disconnect();
      window.removeEventListener('resize', onWinResize);
      term.dispose();
    };
  }, [containerRef, optsJSON, options.addons, options]);

  return { terminal: termRef.current, fitAddon: fitRef.current };
}
