<script>
  import { page } from '$app/state';
  import Logo from '../components/icons/Logo.svelte';
  import CustomHeader from '../components/CustomHeader.svelte';
  let { children } = $props();

  // reactive flags for directory
  // e.g. “/blog”, “/shop”, etc.
  let isBlog = $derived(page.url.pathname.startsWith('/tracker/'));
  let isCreate = $derived(page.url.pathname.startsWith('/create'));

  console.log('yalalallalal', isCreate);
</script>

{#if !isBlog && !isCreate}
  <div class="abs">
    <CustomHeader />
  </div>

  <div class="head">
    <p class="compl">a</p>
    <Logo />
    <p class="compl">☰</p>
  </div>
{:else if isCreate}
  <div class="abs">
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
    <g class="compl seen">✖</g>

    <Logo size={3} />
    <r class="compl seen">☰</r>
  </div>
{/if}

{@render children()}

<style>
  .compl {
    opacity: 0;
    margin: 0;
    width: 4rem;
    align-items: center;
    text-align: center;
    font-size: 25px;
    padding: 0;
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

  .nav {
    margin: 0 10px;
    color: rgb(255, 255, 255);
    font-size: 30px;
  }

  a {
    color: inherit;
    text-decoration: none;
  }

  h1 {
    color: var(--color-white);
    margin: 0 10px;
    -webkit-text-stroke: 0.6px rgb(255, 255, 255);
  }
</style>
