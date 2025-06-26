<script>
  import '../app.css';
  import { page } from '$app/state';
  import Logo from '../components/icons/Logo.svelte';
  import CustomHeader from '../components/CustomHeader.svelte';
  import quitSession from './tracker/track.svelte';
  import { addUserByForm } from '$lib/firebaseCreation';
  import Icon from '@iconify/svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';

  let { children } = $props();
  let isBlog = $derived(page.url.pathname.startsWith('/tracker/'));
  let isCreate = $derived(page.url.pathname.startsWith('/create'));
</script>

{#if !isBlog && !isCreate}
  <div class="abs">
    <CustomHeader />
  </div>

  <div class="head">
    <p class="compl">a</p>
    <Logo />
    <button class="compl seen" onclick={() => goto(`${base}/account`)}>
      <Icon icon={'si:user-fill'} fill={'#fff'} height={28}></Icon>
    </button>
  </div>
{:else if isCreate}
  <div class="abs small">
    <CustomHeader size={3} />
  </div>

  <div class="head view">
    <g class="compl">✖</g>

    <Logo size={3} />
    <r class="compl seen">☰</r>
  </div>
{:else}
  <div class="abs">
    <CustomHeader size={3} />
  </div>

  <div class="head">
    <g class="compl">✖</g>

    <Logo size={3} />
    <r class="compl">✖</r>
  </div>
{/if}

{@render children()}

<style>
  .compl {
    opacity: 0;
    margin: 0;
    width: 3rem;
    align-items: center;
    text-align: center;
    font-size: 25px;
    padding: 0;
    height: 3rem;
  }

  .compl.seen {
    opacity: 1;
  }

  .abs {
    position: absolute;
    display: flex;
    flex-direction: column;
    justify-content: baseline;
    top: -20px;
    left: 0px;
    width: 100%;
  }

  .head {
    position: fixed;
    width: 100%;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    z-index: 2;
    overflow: visible;
  }

  .head.view {
    background-color: var(--color-background);
  }
</style>
