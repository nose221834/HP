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
    <div class="terminal-container">
      <div
        ref={(el) => (container = el!)}
        style={{ width: '100%', height: '500px' }}
      />
      <style>{`
        .terminal-container {
          padding: 1rem;
          background-color: #1e1e1e;
          border-radius: 4px;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
        }
        .terminal-container .xterm {
          padding: 0.5rem;
        }
      `}</style>
    </div>
  );
}
