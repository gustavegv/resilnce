<script lang="ts">
  import { onMount } from 'svelte';
  import MobileLanding from './MobileLanding.svelte';
  import LandscapeLanding from './LandscapeLanding.svelte';
    import { fade, slide } from 'svelte/transition';

  let mounted = false;
  let isLandscape = false;

  onMount(() => {
    const mq = window.matchMedia('(orientation: landscape)');

    const update = () => {
      isLandscape = mq.matches;
      mounted = true;
    };

    update();
    mq.addEventListener('change', update);

    return () => mq.removeEventListener('change', update);
  });
</script>

{#if !mounted}
  <div style="height: 100dvh;" class=""></div>
{:else if !isLandscape}
<MobileLanding />
{:else}
<LandscapeLanding />
{/if}