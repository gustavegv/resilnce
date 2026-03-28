<script lang="ts">
  import '../../app.css';
  import { fly, fade } from 'svelte/transition';

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
    if (!confirm('Log out?')) return;

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

  let weightInput: string = $state('');
  let repsInput: string = $state('');

  function roundMax(value: number) {
    return Math.round(value * 10) / 10;
  }

  function calculateOneRM(weight: number, reps: number) {
    return weight * Math.pow(reps, 0.1);
  }

  function calculateRepMax(oneRM: number, reps: number) {
    return oneRM / Math.pow(reps, 0.1);
  }

  function getMaxes() {
    const weight = Number(weightInput);
    const reps = Number(repsInput);

    if (!weight || !reps) return [];

    const oneRM = calculateOneRM(weight, reps);

    return [
      roundMax(oneRM),
      roundMax(calculateRepMax(oneRM, 2)),
      roundMax(calculateRepMax(oneRM, 3)),
      roundMax(calculateRepMax(oneRM, 5)),
      roundMax(calculateRepMax(oneRM, 8)),
      roundMax(calculateRepMax(oneRM, 10)),
    ];
  }

  let maxes: number[] = $derived.by(() => getMaxes());
  const rmLabels = ['1RM', '2RM', '3RM', '5RM', '8RM', '10RM'];
</script>

<main class="main">
  <h1 class="mb-4 w-80 text-3xl leading-none font-bold tracking-tight text-white">Account</h1>
  <Card.Root class="border-background m-2 w-sm px-2">
    <Card.Header class="border-neutral-700">
      {#if username}
        <Card.Title>
          Logged in as <strong>{username}</strong>
        </Card.Title>
      {:else}
        <Card.Title>Login to your account</Card.Title>
        <Card.Description>Sign up / in with any of the providers below</Card.Description>
      {/if}
    </Card.Header>

    {#if username}
      <Button
        variant="destructive"
        class="text-lg&_svg:not([class*='size-'])]:size-8 m-8 mt-4 py-5"
        onclick={logOut}
      >
        Log out <Icon icon="mdi:user-arrow-right-outline" width={45} />
      </Button>
    {:else}
      <Button variant="outline" class="m-8 mt-4 py-5 text-lg" onclick={loginRequest}>
        <Icon icon="ri:google-fill" />
      </Button>
    {/if}
  </Card.Root>

  {#if username}
    <Card.Root class="border-background mb-2 w-sm px-2">
      <Card.Header class="border-neutral-700">
        <Card.Title>Settings</Card.Title>
      </Card.Header>

      <Card.Content class="space-y-4">
        <div class="flex items-center space-x-3">
          <Checkbox id="cb1" />
          <label for="cb1" class="text-neutral-300 select-none"> Coming soon </label>
        </div>
      </Card.Content>
    </Card.Root>

    <Card.Root class="border-background w-sm px-2">
      <Card.Header class="border-neutral-700">
        <Card.Title>Calculate 1RM</Card.Title>
      </Card.Header>

      <Card.Content class="space-y-4">
        <div class="flex items-center space-x-3">
          <Input
            class="basis-[65%]"
            bind:value={weightInput}
            oninput={() =>
              (weightInput = weightInput.replace(/[^0-9.]/g, '').replace(/(\..*)\./g, '$1'))}
            maxlength={4}
            id="weight"
            type="text"
            inputmode="decimal"
            placeholder="Weight"
            required
          />
          <Input
            class="basis-[35%]"
            bind:value={repsInput}
            oninput={() => (repsInput = repsInput.replace(/\D/g, ''))}
            maxlength={2}
            id="reps"
            type="text"
            inputmode="numeric"
            placeholder="Reps"
            required
          />
        </div>

        {#if maxes.length}
          <div
            class="rm-results rounded-md border border-neutral-700 p-3"
            in:fade={{ duration: 380 }}
            out:fade={{ duration: 100 }}
          >
            {#each maxes as max, i}
              <div class="rm-row">
                <span class="rm-label">{rmLabels[i]}</span>
                <span class="rm-value rm-value-primary">{maxes[i]}</span>
              </div>
            {/each}
          </div>
        {/if}
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
    overflow-y: auto;
  }

  :global(.mar) {
    margin-top: 1rem;
  }

  .rm-results {
    display: grid;
    gap: 0.25rem;
  }

  .rm-results {
    display: grid;
    gap: 0.5rem;
  }

  .rm-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    border-radius: 0.5rem;
    background: var(--surface-middle);
    padding: 0.65rem 0.85rem;
  }

  .rm-label {
    color: rgb(212 212 212);
    font-size: 0.95rem;
  }

  .rm-value {
    font-weight: 700;
  }

  .rm-value-primary {
    font-size: 1.15rem;
  }
</style>
