<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    show,
    onClose,
    children,
  }: {
    show: boolean;
    onClose: () => void;
    children?: Snippet;
  } = $props();

  function sas() {}
</script>

{#if show}
  <div class="backdrop" role="presentation" onclick={onClose}>
    <div class="modal">
      {@render children?.()}
      <button class="close" onclick={onClose} aria-label="Close modal" type="button"> x </button>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: absolute;
    inset: 0;
    background: rgba(255, 255, 255, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 999;
  }

  .modal {
    display: flex;
    flex-direction: column;
    background: var(--color-background);
    padding: 2rem;
    border-radius: 0.5rem;
    width: 70%;
    box-shadow: var(--shadow-dark);
    position: relative;
    justify-content: center;
    align-items: center;
  }

  .close {
    background: transparent;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    background: black;
  }

  p {
    color: black;
    z-index: 1099;
  }
</style>
