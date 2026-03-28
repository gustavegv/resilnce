<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = {
    title: string;
    children?: Snippet;
    class?: string;
  };

  let { title, children, class: className = '' }: Props = $props();

  const titleId = `alert-title-${crypto.randomUUID()}`;
</script>

<div
  class={`alert-content ${className}`.trim()}
  role="alertdialog"
  aria-modal="true"
  aria-labelledby={titleId}
  tabindex="-1"
>
  <h2 id={titleId} class="alert-title">{title}</h2>

  <div class="alert-body">
    {@render children?.()}
  </div>
</div>

<style>
  .alert-content {
    position: relative;
    width: min(100%, 32rem);
    border-radius: 1rem;
    background: var(--color-background);
    color: var(--color-text);
    box-shadow: var(--shadow-dark);
    padding: 1.25rem;
  }

  .alert-title {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 700;
    line-height: 1.2;
  }

  .alert-body {
    margin-top: 0.75rem;
  }
</style>
