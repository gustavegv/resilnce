<script>
  import '../app.css';
  import { page } from '$app/state';
  import Logo from '../components/icons/Logo.svelte';

  import CustomHeader from '../components/CustomHeader.svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { Toaster } from '$lib/components/ui/sonner/index.js';

  let { children } = $props();

  function goHome(){
    goto('/')
  }
</script>

<Toaster />

<div class="abs">
  <CustomHeader size={3} />
</div>

<div class="head">
  <g class="compl">✖</g>

  <div
    class="icon"
    role="button"
    tabindex="0"
    aria-label="Account"
    onclick={() => goto(`${base}/`)}
    onkeydown={(e) => {
      if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') goto(`${base}/`);
    }}
  >
    <img src="/FriendsWhite.svg" alt="" draggable="false" />
  </div>
  <r class="compl">✖</r>
</div>




{@render children()}

<style>
    .icon {
    width: var(--size);
    height: var(--size);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 60px;   /* your desired size */
    height: 45px;
  }

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

</style>
