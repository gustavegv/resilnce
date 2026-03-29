<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';

  import CustomHeader from '../components/CustomHeader.svelte';
  import { goto } from '$app/navigation';
  import { Toaster } from '$lib/components/ui/sonner/index.js';
  import Icon from '@iconify/svelte';

  import { asset, resolve } from '$app/paths';

  let home = $derived(page.url.pathname === resolve('/'));

  let { children } = $props();

  function go(destination: string) {
    switch (destination) {
      case 'home':
        goto(resolve('/'));
        break;
      case 'account':
        goto(resolve('/account'));
        break;
      default:
        const currentPath = page.url.pathname;
        const newPath = currentPath.substring(0, currentPath.lastIndexOf('/'));
        console.log(currentPath);
        goto(newPath || resolve('/'));
        break;
    }
  }
</script>

<Toaster />

{#if home}
  <div class="abs">
    <CustomHeader size={3} />
  </div>

  <div class="head">
    <g class="function-button hide">
      <Icon icon="mdi:user" />
    </g>

    <div class="icon">
      <img src={asset('/FriendsWhite.svg')} alt="" draggable="false" />
    </div>
    <r
      class="function-button"
      role="button"
      tabindex="0"
      aria-label="Account"
      onclick={() => go('account')}
      onkeydown={(e: any) => {
        if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') go('back');
      }}
    >
      <Icon icon="mdi:user" color="grey" />
    </r>
  </div>
{:else}
  <div class="head half">
    <g
      class="function-button"
      role="button"
      tabindex="0"
      aria-label="Back"
      onclick={() => go('back')}
      onkeydown={(e) => {
        if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') go('back');
      }}
    >
      <Icon icon="material-symbols:arrow-back-rounded" color="grey"></Icon>
    </g>

    <div
      class="icon"
      role="button"
      tabindex="0"
      aria-label="Account"
      onclick={() => go('home')}
      onkeydown={(e) => {
        if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') go('home');
      }}
    >
      <img src={asset('/FriendsWhite.svg')} alt="" draggable="false" />
    </div>
  </div>
{/if}

{@render children()}

<style>
  :root {
    --logo-size: 60px;
  }

  .icon {
    width: var(--size);
    height: var(--size);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--logo-size);
    height: 45px;
    z-index: 5;
  }

  .function-button {
    opacity: 1;
    margin: 0 0.5rem;
    align-items: center;
    text-align: center;
    font-size: 25px;
    padding: 0;
    height: 3rem;
    z-index: 5;
  }

  .abs {
    position: fixed;
    display: flex;
    flex-direction: column;
    justify-content: baseline;
    top: -20px;
    left: 0px;
    width: 100%;
    z-index: 1;
  }

  .head {
    position: fixed;
    width: 100%;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    z-index: 1;
    overflow: visible;
  }

  .head.half {
    width: calc(50vw + var(--logo-size) / 2);
  }

  .hide {
    opacity: 0;
    height: 0px;
    width: 24px;
  }
</style>
