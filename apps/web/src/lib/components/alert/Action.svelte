<script lang="ts">
  import type { Snippet } from 'svelte';
  import { getAlertDialogContext } from './context';
  import AlertButton from './AlertButton.svelte';

  type Props = {
    children?: Snippet;
    onclick?: (event: MouseEvent) => void;
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    class?: string;
  };

  const dialog = getAlertDialogContext();

  let {
    children,
    onclick,
    type = 'button',
    disabled = false,
    class: className = '',
  }: Props = $props();

  function handleClick(event: MouseEvent) {
    onclick?.(event);

    if (!event.defaultPrevented) {
      dialog.close();
    }
  }
</script>

<AlertButton
  {type}
  {disabled}
  class={`border border-[var(--border)] bg-[var(--surface-middle)] text-[var(--color-text)] ${className}`.trim()}
  onclick={handleClick}
>
  {@render children?.()}
</AlertButton>
