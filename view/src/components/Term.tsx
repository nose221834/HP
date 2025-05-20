import { onCleanup, onMount } from 'solid-js';

import { createTerm } from '../hooks/useTerm';

import '@xterm/xterm/css/xterm.css';

export function Term() {
  let container!: HTMLDivElement;
  let disposeTerm: () => void;

  onMount(() => {
    const { term, dispose } = createTerm(container);
    disposeTerm = dispose;

    // 例: term.writeln('Hello from Solid + xterm.js');
  });

  onCleanup(() => {
    disposeTerm();
  });

  return (
    <div
      ref={(el) => (container = el!)}
      style={{ width: '100%', height: '400px' }}
    />
  );
}
