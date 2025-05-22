import { createMemo, createSignal, onCleanup, onMount } from 'solid-js';

import '@xterm/xterm/css/xterm.css';
import { createTerm } from '../hooks/createTerm';

export function Term() {
  let container!: HTMLDivElement;
  let disposeTerm: () => void;
  const [store, setStore] = createSignal<
    ReturnType<typeof createTerm>['store'] | null
  >(null);

  const isConnected = createMemo(() => store()?.isConnected ?? false);
  const isSubscribed = createMemo(() => store()?.isSubscribed ?? false);
  const connectionStatus = createMemo(() => {
    if (!isConnected()) return '未接続';
    if (!isSubscribed()) return '接続中...';
    return '接続済み';
  });

  onMount(() => {
    const { dispose, store: termStore } = createTerm(container);
    disposeTerm = dispose;
    setStore(termStore);
  });

  onCleanup(() => {
    disposeTerm();
  });

  return (
    <div class="rounded bg-[#1e1e1e] p-4 shadow-md">
      <div class="mb-2 flex items-center gap-2">
        <div
          class={`h-2 w-2 rounded-full ${
            isConnected() ? 'bg-green-500' : 'bg-red-500'
          }`}
        />
        <span class="text-white">{connectionStatus()}</span>
      </div>
      <div ref={(el) => (container = el!)} class="h-[500px] w-full" />
    </div>
  );
}
