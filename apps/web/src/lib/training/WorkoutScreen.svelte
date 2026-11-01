<script lang="ts">
  import { onMount } from 'svelte';
  import { beforeNavigate } from '$app/navigation';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import { readWorkout, recordSet, TrainingError } from './client';
  import type { RecordSetCommand, Workout, WorkoutExercise, WorkoutSet } from './types';

  let { workoutId }: { workoutId: string } = $props();
  const originalAccount = get(user)?.id ?? '';
  let accountChanged = $derived(
    ($user?.id ?? '') !== originalAccount ||
      ($user !== null && $user.capabilities.applicationModel !== 'exercise-centric')
  );
  let workout = $state.raw<Workout | null>(null);
  let loading = $state(true);
  let error = $state('');
  let candidates = $state<Record<string, string>>({});
  let pending = $state.raw<RecordSetCommand | null>(null);
  let saveState = $state<'saved' | 'saving' | 'failed'>('saved');
  let reviewed = $state.raw<Workout | null>(null);
  let conflict = $state(false);
  let navigationBlocked = $state(false);
  let hasInput = $derived(pending !== null || Object.keys(candidates).length > 0);
  let canReapply = $derived.by(() => {
    if (!pending || !reviewed || reviewed.status !== 'active') return false;
    const attempt = reviewed.exercises.find((a) => a.id === pending!.attemptId);
    return attempt?.state === 'draft' && attempt.sets.some((s) => s.id === pending!.setId);
  });

  beforeNavigate(({ cancel }) => {
    if (hasInput) {
      cancel();
      navigationBlocked = true;
    }
  });
  function beforeUnload(event: BeforeUnloadEvent) {
    if (hasInput) {
      event.preventDefault();
      event.returnValue = '';
    }
  }
  function sameAccount() {
    if ((get(user)?.id ?? '') !== originalAccount || accountChanged) {
      error =
        'Your account changed or expired. Return to the original account to resolve this input.';
      return false;
    }
    return true;
  }
  async function load() {
    if (!sameAccount()) return;
    loading = true;
    error = '';
    try {
      const result = await readWorkout(workoutId);
      if (sameAccount()) workout = result.workout;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to load the workout.';
    } finally {
      loading = false;
    }
  }
  async function reviewCurrent() {
    if (!sameAccount() || !pending) return;
    try {
      const result = await readWorkout(pending.workoutId);
      if (sameAccount()) reviewed = result.workout;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to review current state.';
    }
  }
  async function save() {
    if (!pending || saveState === 'saving' || !sameAccount()) return;
    if (pending.workoutId !== workoutId) {
      error = 'The workout changed. Return to the original workout before retrying this input.';
      return;
    }
    const intent = pending;
    saveState = 'saving';
    error = '';
    try {
      const result = await recordSet(intent);
      if (!sameAccount() || workoutId !== intent.workoutId) {
        saveState = 'failed';
        return;
      }
      workout = result.workout;
      delete candidates[intent.setId];
      pending = null;
      reviewed = null;
      conflict = false;
      saveState = 'saved';
      navigationBlocked = false;
    } catch (failure) {
      saveState = 'failed';
      error = failure instanceof Error ? failure.message : 'Unable to save this set.';
      conflict =
        failure instanceof TrainingError &&
        ['stale-revision', 'resource-finalized'].includes(failure.code);
      if (conflict) await reviewCurrent();
    }
  }
  function record(attempt: WorkoutExercise, set: WorkoutSet, state: WorkoutSet['state']) {
    if (
      pending ||
      !workout ||
      workout.status !== 'active' ||
      attempt.state !== 'draft' ||
      !sameAccount()
    )
      return;
    const base = {
      operationId: crypto.randomUUID(),
      workoutId: workout.id,
      attemptId: attempt.id,
      setId: set.id,
      expectedWorkoutRevision: workout.revision,
    };
    if (state === 'performed') {
      const candidate = candidates[set.id] ?? String(set.recordedReps ?? set.targetReps);
      if (!/^\d+$/.test(candidate) || Number(candidate) > 2147483647) {
        error = 'Enter whole reps from 0 to 2147483647.';
        return;
      }
      candidates[set.id] = candidate;
      pending = { ...base, state, recordedReps: Number(candidate) };
    } else pending = { ...base, state };
    void save();
  }
  function acceptServer() {
    if (!reviewed || !sameAccount()) return;
    workout = reviewed;
    clearInput();
  }
  function reapply() {
    if (!canReapply || !pending || !reviewed || !sameAccount()) return;
    pending = {
      ...pending,
      operationId: crypto.randomUUID(),
      expectedWorkoutRevision: reviewed.revision,
    };
    workout = reviewed;
    reviewed = null;
    conflict = false;
    void save();
  }
  function clearInput() {
    pending = null;
    candidates = {};
    reviewed = null;
    conflict = false;
    saveState = 'saved';
    error = '';
    navigationBlocked = false;
  }
  async function discard() {
    if (saveState === 'saving') return;
    // A lost response may already be committed; never label the old snapshot
    // current after discarding local input. Re-read before enabling recording.
    clearInput();
    workout = null;
    await load();
  }
  onMount(() => {
    void load();
  });
</script>

<svelte:window onbeforeunload={beforeUnload} />
<main>
  {#if accountChanged}<p role="alert">
      Your account changed or expired. Local input is retained. Return to the original account
      before retrying.
    </p>{/if}
  {#if loading}<p role="status">Loading workout…</p>
  {:else if workout}
    <span>Workout</span>
    <h2>{workout.sourceSession?.name ?? 'Workout'}</h2>
    <p role="status">
      {saveState === 'saving'
        ? 'Saving…'
        : pending || Object.keys(candidates).length
          ? 'Not saved'
          : workout.status === 'active'
            ? 'Active workout · Saved'
            : 'Workout ended · Saved'}
    </p>
    {#if error}<p role="alert">{error}</p>{/if}
    {#if saveState === 'failed' && pending && !conflict}<button
        onclick={save}
        disabled={accountChanged}>Retry</button
      >{/if}
    {#if conflict}
      <p role="alert">
        The workout changed. Your local input is retained. Review server state before choosing.
      </p>
      {#if reviewed}
        <section aria-label="Current server state">
          <h3>Current server revision {reviewed.revision} · {reviewed.status}</h3>
          {#each reviewed.exercises as attempt (attempt.id)}
            <h4>{attempt.exercise.name} · {attempt.currentVariant.name} · {attempt.state}</h4>
            {#each attempt.sets as set (set.id)}
              <p>
                Set {set.position}: {set.state}{set.state === 'performed'
                  ? ` · ${set.recordedReps} reps · ${set.recordedWeight} kg`
                  : ''}; target {set.targetReps} reps · {set.targetWeight} kg
              </p>
            {/each}
          {/each}
        </section>
        <button onclick={acceptServer} disabled={accountChanged}
          >Accept server state and discard local input</button
        >
        <button onclick={reapply} disabled={!canReapply || accountChanged}
          >Reapply local input to reviewed revision</button
        >
      {:else}<button onclick={reviewCurrent} disabled={accountChanged}>Review current state</button
        >{/if}
    {/if}
    {#if navigationBlocked}
      <p role="alert">
        Input is not saved. Stay and retry, or explicitly discard local input before leaving.
      </p>
      <button
        onclick={() => {
          navigationBlocked = false;
        }}>Stay</button
      >
    {/if}
    {#if hasInput && !conflict}<button onclick={discard} disabled={saveState === 'saving'}
        >Discard local input</button
      >{/if}
    <nav aria-label="Exercises">
      {#each workout.exercises as attempt (attempt.id)}
        <a href={`#attempt-${attempt.id}`}
          >{attempt.exercise.name} · {attempt.currentVariant.name}</a
        >
      {/each}
    </nav>
    {#each workout.exercises as attempt (attempt.id)}
      <section
        id={`attempt-${attempt.id}`}
        aria-label={`${attempt.exercise.name} ${attempt.currentVariant.name}`}
      >
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
            {#if workout.status === 'active' && attempt.state === 'draft'}
              <label for={`reps-${set.id}`}>Reps for set {set.position}</label>
              <input
                id={`reps-${set.id}`}
                type="number"
                min="0"
                max="2147483647"
                step="1"
                value={candidates[set.id] ?? set.recordedReps ?? set.targetReps}
                oninput={(event) => {
                  candidates[set.id] = event.currentTarget.value;
                }}
                disabled={pending !== null || accountChanged}
              />
              <button
                onclick={() => record(attempt, set, 'performed')}
                disabled={pending !== null || accountChanged}
                >{set.state === 'performed' ? 'Save reps correction' : 'Record set'}</button
              >
              <button
                onclick={() => record(attempt, set, 'skipped')}
                disabled={pending !== null || accountChanged}>Skip set</button
              >
              <button
                onclick={() => record(attempt, set, 'pending')}
                disabled={pending !== null || accountChanged || set.state === 'pending'}
                >Reset to pending</button
              >
            {/if}
          </div>
        {/each}
      </section>
    {/each}
  {:else if error}<p role="alert">{error}</p>
    <button onclick={load} disabled={accountChanged}>Retry</button>{/if}
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
