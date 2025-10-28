<script lang="ts">
  import { Dialog as DialogPrimitive } from 'bits-ui';
  import { cn } from '$lib/utils.js';

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: DialogPrimitive.DescriptionProps = $props();
</script>

<DialogPrimitive.Description
  bind:ref
  data-slot="dialog-description"
  class={cn(
    // Added Tailwind classes for centering
    'text-muted-foreground flex flex-col items-center justify-center text-center text-sm',
    className
  )}
  {...restProps}
/>
