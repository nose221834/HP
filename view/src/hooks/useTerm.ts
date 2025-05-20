import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { Terminal } from '@xterm/xterm';

/**
 * 初期化済みの Terminal インスタンスと dispose 関数を返す。
 * @param container ターミナルを描画する HTMLDivElement
 * @param options 追加オプション（XTerm のオプション + addons）
 */
export function createTerm(container: HTMLDivElement): {
  term: Terminal;
  dispose: () => void;
  // shell: (data: string) => void;
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

  // dispose 用の関数
  const dispose = () => {
    try {
      term.dispose();
    } catch {
      // ignore
    }
  };

  // // 操作できるシェルを定義
  // const shell = (data: string) => {
  //   for (const char of data) {
  //     term.write(char);
  //     // 改行コードの処理
  //     // \n の場合は \r\n に変換
  //     // \r の場合は \r\n に変換
  //     if (char === '\r' || char === '\n') {
  //       term.write('\r\n');
  //     }
  //   }
  // };

  return { term, dispose /*, shell */ };
}
