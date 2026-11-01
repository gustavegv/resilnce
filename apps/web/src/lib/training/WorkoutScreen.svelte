<script lang="ts">
  import { onMount } from 'svelte';
  import { readWorkout } from './client';
  import type { Workout } from './types';

  let { workoutId }: { workoutId: string } = $props();
  let workout = $state.raw<Workout | null>(null);
  let loading = $state(true);
  let error = $state('');
  async function load() {
    loading = true;
    error = '';
    try {
      workout = (await readWorkout(workoutId)).workout;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to load the workout.';
    } finally {
      loading = false;
    }
  }
  onMount(() => {
    void load();
  });
</script>

<main>
  {#if loading}<p role="status">Loading workout…</p>
  {:else if error}<p role="alert">{error}</p>
    <button onclick={load}>Retry</button>
  {:else if workout}
    <span>Workout</span>
    <h2>{workout.sourceSession?.name ?? 'Workout'}</h2>
    <p role="status">{workout.status === 'active' ? 'Active workout · Saved' : 'Workout ended'}</p>
    {#each workout.exercises as attempt (attempt.id)}
      <section aria-label={`${attempt.exercise.name} ${attempt.currentVariant.name}`}>
        <h3>{attempt.position}. {attempt.exercise.name} · {attempt.currentVariant.name}</h3>
        <p>{attempt.state === 'completed' ? 'Completed' : 'Draft'}</p>
        {#each attempt.sets as set (set.id)}
          <div class="set">
            <h4>Set {set.position}</h4>
            <p>{set.targetReps} reps · {set.targetWeight} kg target</p>
            {#if set.state === 'performed'}<p>
                Performed: {set.recordedReps} reps · {set.recordedWeight} kg
              </p>
            {:else if set.state === 'skipped'}<p>Skipped · no performance recorded</p>
            {:else}<p>Pending · no performance recorded</p>{/if}
          </div>
        {/each}
      </section>
    {/each}
  {/if}
</main>

<style>
  main {
    width: 100%;
    padding: 4rem 1rem;
    box-sizing: border-box;
  }
  section {
    background: var(--surface-low);
    border-radius: 15px;
    padding: 1rem;
    margin: 1rem 0;
  }
  .set {
    padding: 0.75rem 0;
    border-top: 1px solid var(--border);
  }
</style>
