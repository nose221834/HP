import React, {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
} from 'react';
import { useXterm } from '../hooks/useXterm';
import type { ITerminalOptions } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';

// ターミナルの公開メソッドを定義するインターフェース
export interface TerminalRef {
  focus: () => void;
  write: (data: string) => void;
  clear: () => void;
  reset: () => void;
  // 必要に応じて他のメソッドも追加可能
}

export interface XtermTerminalProps extends ITerminalOptions {
  className?: string;
  // ReactNodeを削除し、ターミナルに書き込める型のみ許可
  initialContent?: string | number | boolean;
  onData?: (data: string) => void;
  onTitleChange?: (title: string) => void;
}

/**
 * XtermTerminal
 *
 * - ref 経由で focus() や write() を呼べるようにする
 * - initialContent は一度だけ書き込む
 * - useXterm 内のプロンプトはそのまま利用
 */
export const XtermTerminal = forwardRef<TerminalRef, XtermTerminalProps>(
  ({ className = '', initialContent, onData, onTitleChange, ...opts }, ref) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const { terminal } = useXterm(containerRef, opts);

    // ref 経由で端末操作を公開（必要なメソッドのみ公開）
    useImperativeHandle(
      ref,
      () => ({
        focus: () => terminal?.focus(),
        write: (data: string) => terminal?.write(data),
        clear: () => terminal?.clear(),
        reset: () => terminal?.reset(),
      }),
      [terminal]
    );

    // イベントハンドラ登録
    useEffect(() => {
      if (!terminal) return;
      const disposables: { dispose: () => void }[] = [];

      // simpleShell の onData に加え、外部 onData も呼びたい場合
      if (onData) {
        disposables.push(
          terminal.onData((data) => {
            onData(data);
          })
        );
      }

      if (onTitleChange) {
        disposables.push(
          terminal.onTitleChange((title) => {
            onTitleChange(title);
          })
        );
      }

      // 初回マウント時のみ initialContent を書き込む
      if (initialContent !== undefined) {
        // 安全に文字列変換
        terminal.write(String(initialContent));
      }

      return () => {
        // 全ハンドラを破棄
        disposables.forEach((d) => d.dispose());
      };
    }, [terminal, onData, onTitleChange, initialContent]);

    return (
      <div
        ref={containerRef}
        className={`
      xterm-container
      ${className}
      w-full
      h-full
      min-h-[200px]
      overflow-hidden
    `}
      />
    );
  }
);

XtermTerminal.displayName = 'XtermTerminal';
export default React.memo(XtermTerminal);
