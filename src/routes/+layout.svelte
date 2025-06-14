<script>
  import { page } from '$app/state';
  let { children } = $props();

  // reactive flags for directory
  // e.g. “/blog”, “/shop”, etc.
  let isBlog = $derived(page.url.pathname.startsWith('/tracker/'));
</script>

{#if !isBlog}
  <div class="head">
    <a href="/"><h1>resilnce</h1></a>

    <a class="nav">&#9776;</a>
  </div>
{:else}
  <div class="head low">
    <div class="corner">
      <a href="/"><h1>resilnce</h1></a>
    </div>
  </div>
{/if}

{@render children()}

<style>
  .head {
    position: fixed;
    width: 100%;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    background-color: var(--color-background);
    box-shadow: rgba(0, 0, 0, 0.35) 0px 5px 15px;
    z-index: 2;
    overflow: visible;
  }

  .head.low {
    height: 1rem;
  }

  .corner {
    background-color: var(--color-background);
    box-shadow: rgba(0, 0, 0, 0.35) 0px 5px 15px;

    margin-top: 2rem;
    padding: 5px 0;
    border-radius: 0 0 10px 0;
  }

  .invis {
    height: 4rem;
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
    margin: 10px 10px;
    -webkit-text-stroke: 0.6px rgb(255, 255, 255);
  }
</style>
