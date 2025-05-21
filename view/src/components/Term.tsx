import { onCleanup, onMount } from 'solid-js';

import '@xterm/xterm/css/xterm.css';
import { createTerm } from '../hooks/createTerm';

export function Term() {
  let container!: HTMLDivElement;
  let disposeTerm: () => void;

  onMount(() => {
    const { dispose } = createTerm(container);
    disposeTerm = dispose;
  });

  onCleanup(() => {
    disposeTerm();
  });

  return (
    <div class="rounded bg-[#1e1e1e] p-4 shadow-md">
      <div ref={(el) => (container = el!)} class="h-[500px] w-full" />
    </div>
  );
}
