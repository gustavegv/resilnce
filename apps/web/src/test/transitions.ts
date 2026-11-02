// jsdom has no Web Animations API. Complete animations immediately while keeping
// Svelte's transition lifecycle (including custom transitions and cleanup) intact.
Object.defineProperty(Element.prototype, 'animate', {
  configurable: true,
  writable: true,
  value: () => {
    let cancelled = false;
    const animation = {
      currentTime: 0,
      playState: 'finished',
      effect: null,
      onfinish: null as ((event: Event) => void) | null,
      cancel: () => {
        cancelled = true;
      },
    };
    queueMicrotask(() => {
      if (!cancelled) animation.onfinish?.(new Event('finish'));
    });
    return animation;
  },
});
