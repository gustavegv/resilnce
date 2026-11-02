<script lang="ts">
  import type { Snippet } from 'svelte';
  import { setAlertDialogContext } from './context';

  type Props = {
    open?: boolean;
    children?: Snippet;
    dismissible?: boolean;
  };

  let { open = $bindable(true), children, dismissible = true }: Props = $props();

  const dialog = {
    get open() {
      return open;
    },
    setOpen(value: boolean) {
      open = value;
    },
    close() {
      open = false;
    },
  };

  setAlertDialogContext(dialog);
</script>

{#if open}
  <div class="alert-root">
    <button
      type="button"
      class="alert-backdrop"
      aria-label="Close dialog"
      disabled={!dismissible}
      onclick={() => dialog.close()}
    ></button>

    <div class="alert-layer">
      {@render children?.()}
    </div>
  </div>
{/if}

<style>
  .alert-root {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: grid;
    place-items: center;
    padding: 1.5rem;
    overflow-x: hidden;
  }

  .alert-backdrop {
    position: absolute;
    inset: 0;
    border: 0;
    padding: 0;
    margin: 0;
    backdrop-filter: blur(4px);

    cursor: pointer;
    overflow-x: hidden;
  }

  .alert-layer {
    position: relative;
    z-index: 1;
    width: 100%;
    display: grid;
    place-items: center;
    align-items: start;
    max-height: calc(100dvh - 3rem);
    overflow-x: hidden;
    overflow-y: auto;
    overscroll-behavior: contain;
  }
</style>
