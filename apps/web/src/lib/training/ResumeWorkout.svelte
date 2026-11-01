<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { activeWorkout } from './client';
  import type { Workout } from './types';

  let workout = $state.raw<Workout | null>(null);
  let error = $state('');
  let loading = $state(true);
  async function load() {
    loading = true;
    error = '';
    try {
      workout = (await activeWorkout()).workout;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to load your active workout.';
    } finally {
      loading = false;
    }
  }
  onMount(() => {
    void load();
  });
</script>

{#if loading}<p role="status">Checking your active workout…</p>
{:else if error}<p role="alert">{error}</p>
  <button onclick={load}>Retry active workout</button>
{:else if workout}
  <a class="resume" href={resolve('/workout/[id]', { id: workout.id })}>
    Continue workout · {workout.sourceSession?.name ?? 'Workout'}
  </a>
{/if}

<style>
  .resume {
    display: block;
    padding: 1rem;
    border-radius: 16px;
    background: var(--surface-middle);
  }
</style>
