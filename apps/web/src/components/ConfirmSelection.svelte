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
  <button class="confirm-button buttonClass" onclick={handleClick}> Confirm </button>
{:else}
  <button class="confirm-button diabled buttonClass" onclick={handleClick}> Set saved! ✓</button>
{/if}

<style>
  .confirm-button {
    display: block;
    width: 100%;
    padding: 0.75rem;
    margin-top: 2rem;
    background-color: var(--color-secondary);
    font-size: 18px;
    border: none;
    border-radius: 10px;
    cursor: pointer;
  }

  .confirm-button.diabled {
    color: rgb(255, 255, 255);
    pointer-events: none;
    background-color: var(--color-secondary-muted);
  }
</style>
