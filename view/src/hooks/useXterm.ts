import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { Terminal } from '@xterm/xterm';
import { onMount } from 'solid-js';

import type { ITerminalAddon } from '@xterm/xterm';

export interface ITerminalOptions {
  addons?: ITerminalAddon[];
}

export const createTerm = (
  options: ITerminalOptions,
  terminalref: HTMLDivElement | null
) => {
  // ターミナルの初期化処理をここに記述
  onMount(() => {
    // terminalrefがnullでないことを確認
    if (!terminalref) {
      return;
    }
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
    (options.addons ?? []).forEach((a) => term.loadAddon(a));
  });

  return;
};
