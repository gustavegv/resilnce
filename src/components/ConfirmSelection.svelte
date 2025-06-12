<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import '../app.css';


  let { 
    onConfirm, 
    finished
  }: { 
    onConfirm?: () => void;
    finished?: boolean;
  } = $props();


  function handleClick() {
    if (onConfirm) onConfirm();
  }
</script>


{#if !finished}
<button class="confirm-button" onclick={handleClick}> Confirm </button>
{:else}
<button class="confirm-button diabled" onclick={handleClick}> Set saved! </button>
{/if}




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
    box-shadow: rgba(0, 0, 0, 0.16) 0px 1px 4px;
  }

  .confirm-button.diabled {
    background-color: var(--color-gray);
    color: rgb(62, 162, 28);
    font-weight: 600;
    pointer-events: none;
  }
</style>
