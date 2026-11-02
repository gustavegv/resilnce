<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import { getMe } from '../../routes/me';
  import { readSession } from './client';
  import type { Session } from './types';
  import WorkoutStart from './WorkoutStart.svelte';

  let { sessionId }: { sessionId: string } = $props();
  let session = $state.raw<Session | null>(null);
  let loading = $state(true);
  let error = $state('');

  async function load() {
    loading = true;
    error = '';
    try {
      if (!get(user)) await getMe();
      const account = get(user);
      if (!account) throw new Error('Log in to start your workout.');
      if (account.capabilities.applicationModel !== 'exercise-centric')
        throw new Error('Workouts are unavailable for this account.');
      const result = await readSession(sessionId);
      if (get(user)?.id !== account.id)
        throw new Error('Your account changed. Return to the tracker to start your workout.');
      session = result.session;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to load your workout.';
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<main class="app-container">
  {#if loading}
    <p role="status">Loading workout…</p>
  {:else if error}
    <p role="alert">{error}</p>
    <button class="buttonClass" onclick={load}>Retry</button>
  {:else if session}
    <WorkoutStart {session} autostart />
  {/if}
</main>

<style>
  .app-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 4rem 2rem;
    max-width: 30rem;
    min-height: 100vh;
    margin: 0 auto;
    background: var(--gradient-prim);
  }
</style>
