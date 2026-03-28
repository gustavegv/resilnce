<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = {
    children?: Snippet;
    onclick?: (event: MouseEvent) => void;
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    class?: string;
  };

  let {
    children,
    onclick,
    type = 'button',
    disabled = false,
    class: className = '',
  }: Props = $props();
</script>

<button {type} {disabled} class={`alert-button ${className}`.trim()} {onclick}>
  {@render children?.()}
</button>

<style>
  .alert-button {
    border-radius: 0.625rem;
    padding: 0.4rem 0.85rem;
    font: inherit;
    font-weight: 600;
    font-size: 14px;
    width: 100%;
    cursor: pointer;
    transition:
      background-color 0.15s ease,
      border-color 0.15s ease,
      color 0.15s ease,
      opacity 0.15s ease;
  }

  .alert-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
