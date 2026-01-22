<script lang="ts">
  import '../../app.css';

  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Checkbox } from '$lib/components/ui/checkbox';

  import * as Card from '$lib/components/ui/card';

  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';

  import Icon from '@iconify/svelte';
  import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
  import { onMount } from 'svelte';

  let username = $state('');
  let cooldown = false;

  async function handleSubmit(event: Event) {
    cooldown = true;
    alert('Login error. (Wrong username/password perhaps?)');
    setTimeout(() => {
      cooldown = false;
    }, 3000);
  }

  onMount(async () => {
    const value = get(user);

    if (value?.name || (await getMe())) {
      username = value?.name ?? get(user)?.name ?? 'error';
    } else {
      username = '';
    }
  });

  function backendAdress(dir: string): string {
    const baseURL: string = PUBLIC_BACKEND_BASE_URL;
    return baseURL + dir;
  }

  async function getMe(): Promise<boolean> {
    const res = await fetch(backendAdress('/api/me'), {
      method: 'GET',
      credentials: 'include',
    });
    if (res.ok) {
      const data = await res.json();
      console.log(data);
      user.set(data);
      return true;
    } else {
      user.set(null);
      console.error('User not logged in.');
      return false;
    }
  }

  function loginRequest() {
    window.location.href = backendAdress('/login/google');
  }

  async function logOut() {
    const res = await fetch(backendAdress('/api/logout'), {
      method: 'POST',
      credentials: 'include',
    });
    if (res.ok) {
      user.set(null);
      username = '';
      console.log('Successfully logged out.');
    } else {
      console.error('Log out failed');
    }
  }
</script>

<main class="main">
  <h1 class="mb-4 w-80 text-3xl leading-none font-bold tracking-tight text-white">Account</h1>
  {#if username}
    <Card.Root class="m-2 w-xs px-2">
      <Card.Header class="border-b border-neutral-700">
        <Card.Title>
          Logged in as <strong>{username}</strong>
        </Card.Title>
      </Card.Header>

      <Card.Footer class="flex justify-start">
        <Button
          variant="destructive"
          class="w-20 py-5 text-lg backdrop-blur-md [&_svg:not([class*='size-'])]:size-8"
          onclick={logOut}
        >
          <Icon icon="mdi:user-arrow-right-outline" width={45} />
        </Button>
      </Card.Footer>
    </Card.Root>
  {:else}
    <Card.Root class="m-2 w-xs px-2">
      <Card.Header>
        <Card.Title>Login to your account</Card.Title>
        <Card.Description>Sign up / in with any of the providers below</Card.Description>
      </Card.Header>

      <Button
        variant="outline"
        class="m-8 mt-4 py-5 text-lg backdrop-blur-md [&_svg:not([class*='size-'])]:size-5"
        onclick={loginRequest}
      >
        <Icon icon="ri:google-fill" />
      </Button>
    </Card.Root>
  {/if}

  {#if username}
    <Card.Root class="w-xs px-2">
      <Card.Header class="border-b border-neutral-700">
        <Card.Title>Settings</Card.Title>
      </Card.Header>

      <Card.Content class="space-y-4">
        <div class="flex items-center space-x-3">
          <Checkbox id="cb1" />
          <label for="cb1" class="text-neutral-300 select-none"> Checkbox 1 </label>
        </div>

        <div class="flex items-center space-x-3">
          <Input maxlength={2} id="rep" type="number" placeholder="LeBron" class="" required />
        </div>

        <div class="flex items-center space-x-3">
          <Checkbox id="cb2" />
          <label for="cb2" class="text-neutral-300 select-none"> Checkbox 2 </label>
        </div>
      </Card.Content>
    </Card.Root>
  {/if}
</main>

<style>
  .main {
    margin-top: 4rem;
    display: flex;
    justify-content: baseline;
    align-items: center;
    height: 100vh;
    flex-direction: column;
  }

  :global(.mar) {
    margin-top: 1rem;
  }
</style>
