<script lang="ts">
  import '../app.css';
  import Icon from '@iconify/svelte';

  let {
    weight,
    onCountChange,
    title,
    limit = 0.25,
    interval = 0.5,
    unit = 'kg',
  }: {
    weight: number;
    onCountChange?: (count: number) => void;
    title: string;
    limit?: number;
    interval?: number;
    unit?: string;
  } = $props();

  let curCount = $derived(weight);
  const maxLimit = 99;
  let limitReached = $derived(curCount <= limit);
  const precsisionThreshold = 1.5;

  function decrement() {
    const decrementInterval =
      unit !== 'reps' && curCount <= precsisionThreshold ? interval / 2 : interval;
    const newCount = Math.max(limit, curCount - decrementInterval);

    if (newCount < limit) {
      console.log('LIMIT REACHED');
      return;
    }

    curCount = newCount;
    onCountChange?.(curCount);
  }

  function increment() {
    const step = unit !== 'reps' && curCount < 1.5 ? interval / 2 : interval;
    if (curCount + step > maxLimit) {
      return;
    }
    curCount += step;

    onCountChange?.(curCount);
  }
</script>

<div class="counter-container mini">
  <div class="ml-1">{title}</div>
  <div class="controls mini">
    <button class="but mini" onclick={decrement}>
      <Icon icon="gg:remove" height="25" color={limitReached ? 'grey' : '#fff'} />
    </button>
    <gf class="mini-count">{curCount} {unit}</gf>
    <button class="but mini" onclick={increment}>
      <Icon icon="gg:add" height="25" />
    </button>
  </div>
</div>

<style>
  .counter-container {
    display: flex;
    justify-content: space-between;
    background-color: var(--surface-low);
    text-align: left;
    padding: 1rem 0.5rem;
    border-radius: 0;
    margin: 0;

    font-size: 16px;
    font-weight: 400;
    pointer-events: none;
    box-shadow: none;
    margin: auto;
    width: 100%;
    max-width: 280px;
    opacity: 0.9;
  }

  .controls {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.7rem;
    font-size: 1rem;
  }

  .but {
    background-color: #11111100;
    text-align: center;
    padding: 0;
    box-shadow: none;
    pointer-events: all;
    width: fit-content;
    margin: 0;
  }

  .mini-count {
    width: 3rem;
    text-align: center;
    font-size: 0.9rem;
    white-space: pre;
    line-height: 1.2em;
  }
</style>
