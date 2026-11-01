<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';
  import { getMe } from '../../me';
  import WorkoutScreen from '$lib/training/WorkoutScreen.svelte';
  let ready = $state(false);
  onMount(async () => {
    if (!get(user)) await getMe();
    ready = true;
  });
</script>

{#if !ready}<p role="status">Loading account…</p>
{:else if !$user}<p role="alert">Log in to resume your workout.</p>
{:else if $user.capabilities.applicationModel !== 'exercise-centric'}<p role="alert">
    Workouts are unavailable for this account.
  </p>
{:else if page.params.id}
  {#key page.params.id}<WorkoutScreen workoutId={page.params.id} />{/key}
{/if}
