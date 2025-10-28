<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';
  import Logo from '../components/icons/Logo.svelte';

  import CustomHeader from '../components/CustomHeader.svelte';
  import { goto } from '$app/navigation';
  import { Toaster } from '$lib/components/ui/sonner/index.js';
  import Icon from '@iconify/svelte';

  import { asset, resolve } from '$app/paths';

  let home = $derived(page.url.pathname === resolve('/'));

  let { children } = $props();

  function go(destination: string) {
    if (destination == 'home') {
      goto(resolve('/'));
    } else {
      const currentPath = page.url.pathname;
      const newPath = currentPath.substring(0, currentPath.lastIndexOf('/'));
      console.log(currentPath);

      goto(newPath || resolve('/'));
    }
  }
</script>

<Toaster />

{#if home}
  <div class="abs">
    <CustomHeader size={3} />
  </div>

  <div class="head">
    <g class="function-button"></g>

    <div class="icon">
      <img src={asset('/FriendsWhite.svg')} alt="" draggable="false" />
    </div>
    <r class="function-button"></r>
  </div>
{:else}
  <div class="abs">
    <CustomHeader size={3} />
  </div>

  <div class="head">
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
      <Icon icon="entypo:back" color="grey"></Icon>
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
    <r class="function-button"></r>
  </div>
{/if}

{@render children()}

<style>
  .icon {
    width: var(--size);
    height: var(--size);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 60px; /* your desired size */
    height: 45px;
  }

  .function-button {
    opacity: 1;
    margin: 0 0.5rem;
    width: 3rem;
    align-items: center;
    text-align: center;
    font-size: 25px;
    padding: 0;
    height: 3rem;
  }

  .abs {
    position: fixed;
    display: flex;
    flex-direction: column;
    justify-content: baseline;
    top: -20px;
    left: 0px;
    width: 100%;
    z-index: 3;
  }

  .head {
    position: fixed;
    width: 100%;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    z-index: 5;
    overflow: visible;
  }
</style>
