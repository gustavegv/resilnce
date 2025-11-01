<script lang="ts">
  import { onMount } from 'svelte';
  import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';

  let sending: boolean = $state(false);

  const directory: string = PUBLIC_BACKEND_BASE_URL + '/oauth/google/callback';
  let queryString: string;

  onMount(() => {
    queryString = window.location.search;
    const params = new URLSearchParams(queryString);

    console.log(queryString, window.location);
  });

  function authorizationRequest() {
    sending = true;
    console.log('Starting Auth...');
    let callbackAddress = directory + queryString;
    console.log(callbackAddress);

    // start the flow
    window.location.href = callbackAddress;
  }
</script>

<div class="mt-16 grid grid-cols-1 justify-items-center p-8">
  {#if sending}
    <h1>Waiting for Authorization response...</h1>
  {:else}
    <h1>Not yet sent auth</h1>
  {/if}
  <button class="buttonClass" onclick={() => authorizationRequest()}>Click to send Auth</button>
</div>
