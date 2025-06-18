<script lang="ts">
  import '../app.css';

  let {
    onConfirm,
    finished,
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
  <button class="confirm-button diabled" onclick={handleClick}> Set saved! ✓</button>
{/if}

<style>
  .confirm-button {
    display: block;
    width: 100%;
    padding: 1rem;
    margin-top: 2rem;
    background-color: var(--color-secondary);
    font-size: 1.25rem;
    border: none;
    border-radius: 0.5rem;
    cursor: pointer;
  }

  .confirm-button.diabled {
    background-color: #95a38c;
    color: rgb(255, 255, 255);
    pointer-events: none;
  }
</style>
