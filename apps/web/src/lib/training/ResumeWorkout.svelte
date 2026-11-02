<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { activeWorkout } from './client';
  import type { Workout } from './types';
  import ResumeSessionButton from '../../components/ResumeSessionButton.svelte';

  let { onloaded }: { onloaded?: (workout: Workout | null) => void } = $props();

  let workout = $state.raw<Workout | null>(null);
  let error = $state('');
  let loading = $state(true);
  async function load() {
    loading = true;
    error = '';
    try {
      workout = (await activeWorkout()).workout;
      onloaded?.(workout);
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

{#if loading}<p class="resume-status" role="status">Checking your active workout…</p>
{:else if error}<div class="resume-status">
    <p role="alert">{error}</p>
    <button class="buttonClass minimal" onclick={load}>Retry active workout</button>
  </div>
{:else if workout}
  <ResumeSessionButton
    href={resolve('/workout/[id]', { id: workout.id })}
    name={workout.sourceSession?.name ?? 'Workout'}
  />
{/if}

<style>
  .resume-status {
    width: 50%;
    font-size: 12px;
    color: var(--text-muted);
  }
</style>
