<script lang="ts">
  import Icon from '@iconify/svelte';
  import { useId } from 'bits-ui';
  import { onMount } from 'svelte';
  import { fade, blur, slide, draw } from 'svelte/transition';
  import { user } from '$lib/stores/appState';
  import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
  import { get } from 'svelte/store';

  var loggedIn: boolean = $state(false);

  const baseURL: string = PUBLIC_BACKEND_BASE_URL;

  function address(dir: string): string {
    return baseURL + dir;
  }

  onMount(async () => {
    const uN = get(user);

    if (uN == null) {
      console.log('Fetching info');
      getMe();
    } else {
      console.log('Info already stored');
      loggedIn = true;
    }
  });

  async function getMe() {
    const res = await fetch(address('/api/me'), {
      method: 'GET',
      credentials: 'include',
    });
    if (res.ok) {
      const data = await res.json();
      console.log(data);
      user.set(data);
      loggedIn = true;
    } else {
      console.error('Log in first');
      // todo: Sonner? Set state to be logged out so you only see the login button?
      user.set(null);
    }
  }

  async function logOut() {
    const res = await fetch(address('/api/logout'), {
      method: 'POST',
      credentials: 'include',
    });
    if (res.ok) {
      loggedIn = false;
      user.set(null);

      console.log('Successfully logged out.');
    } else {
      console.error('Log out failed');
    }
  }

  function loginRequest() {
    window.location.href = address('/login/google');
  }
</script>

<div
  class="justify-items-space mt-16 grid grid-cols-1 space-y-4 p-8 transition-all duration-300 ease-in-out"
>
  <div class="flex items-stretch space-x-4">
    <button
      onclick={() => getMe()}
      class="group flex h-full items-center justify-center rounded-xl
      border border-white/20 bg-white/10 p-6 backdrop-blur-md transition duration-150 hover:bg-white/20"
    >
      <Icon
        icon="majesticons:scan-user-line"
        class="transition-transform duration-150 group-active:scale-90"
        width={45}
      />
    </button>

    <div
      class="flex h-full w-full min-w-[200px] flex-col items-start justify-center
      rounded-xl border border-white/20 bg-white/10 p-6 backdrop-blur-md"
    >
      {#if loggedIn}
        <h1 transition:blur={{ duration: 700 }} class="m-0 p-0 text-xl leading-[1em] font-semibold">
          {$user?.name ?? 'Guest'}
        </h1>
        <p transition:blur={{ duration: 700, delay: 150 }} class="text-md m-0 p-0 font-light">
          Signed in!
        </p>
      {:else}
        <i in:slide={{ delay: 850 }} class="text-md font-regular text-neutral-400"
          >{$user?.name ?? 'Guest'}</i
        >
      {/if}
    </div>
  </div>
  {#if $user}
    <button
      in:slide={{ delay: 0 }}
      onclick={() => logOut()}
      class="group flex h-full items-center justify-center rounded-xl
      border border-white/20 bg-white/10 p-6 backdrop-blur-md transition duration-150 hover:bg-white/20"
    >
      <Icon icon="mdi:user-arrow-right-outline" width={45} />
    </button>
  {/if}
</div>

<div
  class="mt-16 grid grid-cols-1 justify-items-center p-8 transition-all duration-300 ease-in-out"
>
  <h1 class="text-xl font-semibold">Sign in with Google</h1>
  <button
    onclick={() => loginRequest()}
    class="mt-4 flex w-46 items-center justify-center rounded-xl border border-white/20
  bg-white/10 px-5 py-2.5 backdrop-blur-md"
  >
    <Icon icon="ri:google-fill" class="h-5 w-5" />
  </button>
</div>
