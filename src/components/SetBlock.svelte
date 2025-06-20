<script lang="ts">
  import { onMount } from 'svelte';
  import '../app.css';
  import Icon from '@iconify/svelte';

  let {
    id,
    reps,
    finished,
    onCountChange,
  }: {
    id: number;
    reps: number;
    finished?: boolean;
    onCountChange?: (detail: { id: number; count: number }) => void;
  } = $props();

  let curCount = $state(reps);

  onMount(() => {
    onCountChange?.({ id, count: curCount });
  });

  function decrement() {
    curCount = Math.max(0, curCount - 1);
    checksRep();
    onCountChange?.({ id, count: curCount });
  }

  function increment() {
    curCount += 1;
    checksRep();
    onCountChange?.({ id, count: curCount });
  }

  function checksRep() {
    console.log('Checked rep');
    if (!checked) {
      checked = true;
    }
  }

  let checked: boolean = $state(false);
</script>

{#if !finished}
  <div class="counter-container">
    <div class="check {checked}"><Icon icon="gg:check-o" /></div>
    <div>Set {id}</div>
    <div class="controls">
      <button class="but buttonClass" onclick={decrement}>-</button>
      <button onclick={checksRep}>{curCount}</button>
      <button class="but buttonClass" onclick={increment}>+</button>
    </div>
  </div>
{:else}
  <div class="counter-container disabled">
    <div>Set {id}</div>
    <div class="controls">
      <button class="but disabled buttonClass" onclick={decrement}>-</button>
      <span class="mini">{curCount}</span>
      <button class="but disabled buttonClass" onclick={increment}>+</button>
    </div>
  </div>
{/if}

<style>
  .check {
    position: absolute;
    right: 0.5rem;
    transition: all 0.3s;
    opacity: 1;
  }

  .check.false {
    opacity: 0;
  }

  .counter-container {
    position: relative;
    text-align: center;
    margin: 0.5rem 0;
    background-color: var(--color-secondary);
    width: 80%;
    border-radius: 10px;
    padding: 0.5rem 0;
    box-shadow: var(--shadow-dark);
  }

  .controls {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    font-size: 2rem;
  }

  .but {
    background-color: var(--color-gray);
    color: var(--color-black);
    padding: 14px;
    width: 50px;
    border-radius: 50px;
  }

  .counter-container.mini {
    background-color: #111111;
    text-align: left;
    padding: 0.5rem 0.5rem;
    border-radius: 0;
    margin: 0;

    font-size: 16px;
    font-weight: 500;
    pointer-events: none;
    box-shadow: none;

    margin: auto;

    width: 100%;
    max-width: 280px;
  }

  .controls.mini {
    font-size: 16px;
  }

  .but.mini {
    text-align: center;
    padding: 0;
    border-radius: 10px;
    font-size: 16px;
    width: 40px;
  }

  .mini-count {
    font-size: 1rem;
  }

  button {
    padding: 0.5rem 1rem;
  }

  span {
    font-size: 2rem;
    width: 3rem;
    text-align: center;
  }

  .disabled {
    pointer-events: none;
    opacity: 0.5;
  }
</style>
