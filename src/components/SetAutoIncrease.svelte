<script lang="ts">
  import { onMount } from 'svelte';
  import '../app.css';
  import Icon from '@iconify/svelte';

  let {
    weight,
    onCountChange,
  }: {
    weight: number;
    onCountChange?: (count: number) => void;
  } = $props();

  let curCount = $state(weight);

  onMount(() => {
    onCountChange?.(curCount);
  });

  function decrement() {
    console.log('mountneifehifei');
    if (curCount <= 1.5) {
      curCount = Math.max(0.25, curCount - 0.25);
      onCountChange?.(curCount);
    } else {
      curCount = Math.max(0.25, curCount - 0.5);
      onCountChange?.(curCount);
    }
  }

  function increment() {
    console.log('mountneifehifei');
    if (curCount < 1.5) {
      curCount += 0.25;
    } else {
      curCount += 0.5;
    }

    onCountChange?.(curCount);
  }
</script>

<div class="counter-container mini">
  <div>Increase</div>
  <div class="controls mini">
    <button class="but mini" onclick={decrement}>
      <Icon icon={'gg:remove'} height={'30'} />
    </button>
    <gf class="mini-count">{curCount}</gf>
    <button class="but mini" onclick={increment}>
      <Icon icon={'gg:add'} height={'30'} />
    </button>
  </div>
</div>

<style>
  .counter-container {
    background-color: var(--color-secondary);
    text-align: left;
    padding: 1rem 0.5rem;
    border-radius: 0;
    margin: 0;

    font-size: 12px;
    font-weight: 500;
    pointer-events: none;
    box-shadow: none;
    margin: auto;
    width: 87%;
    max-width: 280px;
  }

  .controls {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
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
    width: 2rem;
    text-align: center;
  }
</style>
