<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import '../app.css';

  let { 
    id, 
    reps, 
    finished,
    onCountChange 
  } = $props<{
    id: number;
    reps: number;
    finished?: boolean;
    onCountChange?: (detail: { id: number; count: number }) => void;
  }>();

  let curCount = $state(reps);


  onMount(() => {
    onCountChange?.({ id, count: curCount });
  });

  function decrement() {
    curCount = Math.max(0, curCount - 1);
    onCountChange?.({ id, count: curCount });
  }

  function increment() {
    curCount += 1;
    onCountChange?.({ id, count: curCount });
  }
</script>


{#if !finished}
<div class="counter-container">
  <div>Set {id}</div>
  <div class="controls">
    <button class="but" onclick={decrement}>-</button>
    <span>{curCount}</span>
    <button class="but" onclick={increment}>+</button>
  </div>
</div>
{:else}
<div class="counter-container">
  <div>Set {id}</div>
  <div class="controls">
    <button class="but disabled" onclick={decrement}>-</button>
    <span>{curCount}</span>
    <button class="but disabled" onclick={increment}>+</button>
  </div>
</div>
{/if}


<style>
  .counter-container {
    text-align: center;
    margin: 1rem 0;
  }

  .controls {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
  }

  button {
    padding: 0.5rem 1rem;
  }

  span {
    font-size: 2rem;
    width: 3rem;
    text-align: center;
  }

  .but {
    background-color: var(--color-gray);
    color: var(--color-black);
    padding: 16px;
    width: 50px;
    border-radius: 50px;
  }

  .disabled {
    pointer-events: none;
    opacity: 0.2;
  }
</style>
