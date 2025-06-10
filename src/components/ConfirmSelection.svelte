<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import '../app.css';


  const dispatch = createEventDispatcher();

  function handleClick() {
    dispatch('confirm');
  }
</script>

<button class="confirm-button" on:click={handleClick}>
  Confirm
</button>

<style>
.confirm-button {
  display: block;
  width: 100%;
  padding: 1rem;
  margin-top: 2rem;
  background-color: var(--color-secondary);
  color: rgb(0, 0, 0);
  font-size: 1.25rem;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
}

.confirm-button:hover{

}
</style>