<script lang="ts">
  import { Slider } from 'bits-ui';
  import { cn } from '$lib/utils.js';

  type TickItem = {
    index: number;
    value: number;
  };

  type ThumbItem = {
    index: number;
  };

  type Props = {
    value: number;
    min?: number;
    max?: number;
    step?: number;
    class?: string;
    disabled?: boolean;
  };

  let {
    value = $bindable(0),
    min = 0,
    max = 10,
    step = 1,
    class: className,
    disabled = false,
  }: Props = $props();

  const colorBands = [
    { from: 0, to: 3.5, class: 'bg-green-500' },
    { from: 3.15, to: 7, class: 'bg-yellow-400' },
    { from: 6.85, to: 9, class: 'bg-orange-500' },
    { from: 8.7, to: 10, class: 'bg-red-500' },
  ];

  function handleValueChange(nextValue: number) {
    value = nextValue;
  }

  function getThumbColor(value: number) {
    if (value <= 2) return 'bg-green-500 border-green-600';
    if (value <= 7) return 'bg-yellow-400 border-yellow-500';
    if (value <= 9) return 'bg-orange-500 border-orange-600';

    return 'bg-red-500 border-red-600';
  }

  function getPercent(input: number) {
    return ((input - min) / (max - min)) * 100;
  }

  function getBandStyle(from: number, to: number) {
    const left = getPercent(from);
    const width = getPercent(to) - left;

    return `left: ${left}%; width: ${width}%;`;
  }
</script>

<div class="counter-container">
  <div class="mb-4 w-full space-y-2">
    <div>
      <h3 class="text-foreground w-full text-left text-sm leading-tight font-bold tracking-tight">
        Effort estimate
      </h3>

      <p class="text-muted-foreground mt-1 w-full text-left text-xs leading-snug">
        How hard was the effort?
      </p>
    </div>

    <div class="bg-border/80 h-px w-full"></div>
  </div>
  <div class="w-full">
    <Slider.Root
      type="single"
      {value}
      {min}
      {max}
      {step}
      {disabled}
      onValueChange={handleValueChange}
      class="relative flex w-full touch-none items-center py-5 select-none"
      trackPadding={0}
    >
      {#snippet children({
        tickItems,
        thumbItems,
      }: {
        tickItems: TickItem[];
        thumbItems: ThumbItem[];
      })}
        <span
          class="bg-muted relative h-3 w-full grow cursor-pointer overflow-hidden rounded-full shadow-inner"
        >
          {#each colorBands as band}
            <span
              class={cn('absolute top-0 h-full', band.class)}
              style={getBandStyle(band.from, band.to)}
            ></span>
          {/each}
        </span>

        {#each thumbItems as { index } (index)}
          <Slider.Thumb
            {index}
            class={cn(
              'z-20 block size-6 cursor-pointer rounded-full border-2 shadow-lg transition-all',
              'ring-background/80',
              'focus-visible:ring-ring focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-hidden',
              'disabled:pointer-events-none disabled:opacity-50 data-active:scale-95',
              getThumbColor(value)
            )}
          />
        {/each}

        {#each tickItems as { index, value } (index)}
          <Slider.Tick {index} class="bg-background/75 z-10 h-5 w-px" />

          <Slider.TickLabel
            {index}
            class="text-muted-foreground data-bounded:text-foreground text-xs leading-none font-medium"
            position="bottom"
          >
            {value}
          </Slider.TickLabel>
        {/each}
      {/snippet}
    </Slider.Root>
  </div>
</div>

<style>
  .counter-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: space-between;
    align-items: center;
    padding: var(--spacing-md);
    background: var(--surface-low);
    width: 100%;
    border-radius: 10px;
    box-shadow: var(--shadow-dark);
    padding-bottom: 2rem;
  }
</style>
