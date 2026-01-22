<script lang="ts">
  import { onMount } from 'svelte';
  import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
  import Icon from '@iconify/svelte';

  const directory: string = PUBLIC_BACKEND_BASE_URL + '/oauth/google/callback';
  let queryString: string;

  onMount(() => {
    queryString = window.location.search;
    const params = new URLSearchParams(queryString);

    authorizationRequest();
    console.log(queryString, window.location);
  });

  function authorizationRequest() {
    console.log('Starting Auth...');
    let callbackAddress = directory + queryString;
    console.log(callbackAddress);

    // start the flow
    window.location.href = callbackAddress;
  }
</script>

<div class="mt-16 grid grid-cols-1 justify-items-center p-8">
  <h1>Authenticating, redirecting...</h1>
  <Icon icon="svg-spinners:pulse" width={60} class="mt-8" />
</div>
