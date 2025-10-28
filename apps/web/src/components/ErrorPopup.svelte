<script lang="ts">
  import { onDestroy, tick } from 'svelte';

  let { message = '' } = $props();

  let visible = $state(false);
  let timeoutId: ReturnType<typeof setTimeout>;

  let displayTime = 3000;

  let curclass = $state('popup');

  $effect(() => {
    // Only show when there's a message
    if (!message) return;

    // Reset visibility to retrigger animation
    visible = false;

    tick().then(() => {
      visible = true;
      curclass = 'popup an';

      // Clear previous timer, set new one
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => {
        curclass = 'popup';
      }, displayTime);
    });
  });

  onDestroy(() => clearTimeout(timeoutId));
</script>

{#if visible}
  <div class={curclass}>
    {message}
  </div>
{/if}

<style>
  .popup {
    position: fixed;
    bottom: 2rem;
    right: 2rem;
    background: #ff4d4f;
    color: white;
    padding: 1rem 1.5rem;
    border-radius: 0.5rem;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.15);
    transform: translateY(100px);
    transition: all 200ms;
    z-index: 9999;
  }

  .popup.an {
    transform: translateY(0px);
  }
</style>
