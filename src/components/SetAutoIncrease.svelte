<script lang="ts">
  import { onMount } from 'svelte';
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

  let curCount = $state(weight);
  const maxLimit = 99;
  let limitReached = $state(false);
  const precsisionThreshold = 1.5;

  onMount(() => {
    onCountChange?.(curCount);

    if (unit.length > 3) {
      unit = '\n' + unit;
    }
  });

  function decrement() {
    const decrementInterval = curCount <= precsisionThreshold ? interval / 2 : interval;
    const newCount = Math.max(0.25, curCount - decrementInterval);

    if (newCount < limit) {
      console.log('LIMIT REACHED');
      return;
    }

    limitReached = newCount - decrementInterval < limit;
    if (limitReached) {
      console.log('Greyed out');
    }
    console.log(newCount - decrementInterval);

    curCount = newCount;
    onCountChange?.(curCount);
  }

  function increment() {
    if (curCount + 1 > maxLimit) {
      return;
    }
    limitReached = false;

    if (curCount < 1.5) {
      curCount += interval / 2;
    } else {
      curCount += interval;
    }

    onCountChange?.(curCount);
  }
</script>

<div class="counter-container mini">
  <div class="ml-1">{title}</div>
  <div class="controls mini">
    <button class="but mini buttonClass" onclick={decrement}>
      <Icon icon={'gg:remove'} height={'25'} color={limitReached ? 'grey' : '#fff'} />
    </button>
    <gf class="mini-count">{curCount} {unit}</gf>
    <button class="but mini buttonClass" onclick={increment}>
      <Icon icon={'gg:add'} height={'25'} />
    </button>
  </div>
</div>

<style>
  .counter-container {
    display: flex;
    justify-content: space-between;
    background-color: var(--color-background);
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
