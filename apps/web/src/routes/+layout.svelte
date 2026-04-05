<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';

  import CustomHeader from '../components/CustomHeader.svelte';
  import { goto } from '$app/navigation';
  import { Toaster } from '$lib/components/ui/sonner/index.js';
  import Icon from '@iconify/svelte';

  import { asset, resolve } from '$app/paths';
    import { onMount } from 'svelte';
    import LandscapeLanding from './LandscapeLanding.svelte';
    import { isMobileNotWebapp, setHelpOpen, toggleHelpOpen } from './webapp.svelte';

  let home = $derived(page.url.pathname === resolve('/'));
  let showHelp = $state(false)
  let { children } = $props();

  let isLandscape = $state(false);

  onMount(() => {
    const mq = window.matchMedia('(orientation: landscape)');

    const update = () => {
      isLandscape = mq.matches;
    };
    
    showHelp = isMobileNotWebapp()
    setHelpOpen(showHelp)

    update();
    mq.addEventListener('change', update);

    return () => mq.removeEventListener('change', update);
  });

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
{#if isLandscape}

{:else if home}
  <div class="abs">
    <CustomHeader size={3} />
  </div>

  <div class="head">
    <g 
      class="function-button" 
      class:hide={!showHelp}
      onclick={() => toggleHelpOpen()}
      onkeydown={(e: any) => {
        if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') toggleHelpOpen();
      }}
      aria-label="Help with webapp"
      tabindex="0"
      role="button"
    >
      <Icon icon="material-symbols:help-outline" color="gray"/>
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
        if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') go('account');
      }}
    >
      <Icon icon="material-symbols:account-circle" color="grey" />
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
    pointer-events: none;
    height: 0px;
    width: 24px;
  }
</style>
