<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';
  import { getMe } from '../../me';
  import WorkoutScreen from '$lib/training/WorkoutScreen.svelte';
  let ready = $state(false);
  let accountGate = $state<'login' | 'unavailable' | 'allowed'>('login');
  onMount(async () => {
    if (!get(user)) await getMe();
    const account = get(user);
    accountGate = !account
      ? 'login'
      : account.capabilities.applicationModel === 'exercise-centric'
        ? 'allowed'
        : 'unavailable';
    ready = true;
  });
</script>

{#if !ready}<p role="status">Loading account…</p>
{:else if accountGate === 'login'}<p role="alert">Log in to resume your workout.</p>
{:else if accountGate === 'unavailable'}<p role="alert">
    Workouts are unavailable for this account.
  </p>
{:else if page.params.id}
  {#key page.params.id}<WorkoutScreen workoutId={page.params.id} />{/key}
{/if}
