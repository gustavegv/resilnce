<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    show,
    onAccept,
    onDecline,
    children,
  }: {
    show: boolean;
    onAccept: () => void;
    children?: Snippet;
    onDecline?: () => void;
  } = $props();

  function sas() {}
</script>

{#if show}
  {#if onDecline != undefined}
    <div class="backdrop" role="presentation">
      <div class="modal">
        {@render children?.()}
        <button class="close buttonClass" onclick={onAccept} aria-label="Accept" type="button"> Yes </button>
        <button class="close buttonClass" onclick={onDecline} aria-label="Decline" type="button"> No </button>
      </div>
    </div>
  {:else}
    <div class="backdrop" role="presentation" onclick={onAccept}>
      <div class="modal">
        {@render children?.()}
        <button class="close buttonClass" onclick={onAccept} aria-label="Close modal" type="button"> x </button>
      </div>
    </div>
  {/if}
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
