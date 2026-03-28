<script lang="ts">
  import type { Snippet } from 'svelte';
  import { setAlertDialogContext } from './context';

  type Props = {
    open?: boolean;
    children?: Snippet;
  };

  let { open = $bindable(true), children }: Props = $props();

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
    background: rgb(0 0 0 / 45%);
    cursor: pointer;
    overflow-x: hidden;
  }

  .alert-layer {
    position: relative;
    z-index: 1;
    width: 100%;
    display: grid;
    place-items: center;
    overflow-x: hidden;
  }
</style>
