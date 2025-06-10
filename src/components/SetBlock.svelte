<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import '../app.css';

  export let id: number;
  export let reps: number;
  const dispatch = createEventDispatcher<{ countChange: { id: number; count: number } }>();
  let curCount = reps;

  onMount(() => {
    dispatch('countChange', { id, count: curCount });
  });

  function decrement() {
    curCount = Math.max(0, curCount - 1);
    dispatch('countChange', { id, count: curCount });
  }

  function increment() {
    curCount += 1;
    dispatch('countChange', { id, count: curCount });
  }
</script>

<div class="counter-container">
  <div>Set {id}</div>
  <div class="controls">
    <button class="but" on:click={decrement}>-</button>
    <span>{curCount}</span>
    <button class="but" on:click={increment}>+</button>
  </div>
</div>

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
</style>