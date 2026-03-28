<script>
  let { children, class: className = '' } = $props();
</script>

<p class={`alert-description ${className}`.trim()}>
  {@render children?.()}
</p>

<style>
  .alert-description {
    margin: 0;
    color: var(--text-muted);
    line-height: 1.5;
  }
</style>
