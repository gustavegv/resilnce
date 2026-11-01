<script lang="ts">
  import { beforeNavigate } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import { readSession, startWorkout, TrainingError } from './client';
  import type { Session, StartWorkoutCommand, Workout } from './types';

  let {
    session,
    disabled = false,
    onpending = (_pending: boolean) => {},
  }: { session: Session; disabled?: boolean; onpending?: (pending: boolean) => void } = $props();
  let reviewed = $state.raw<Session | null>(null);
  let plan = $derived(reviewed ?? session);
  let pending = $state.raw<StartWorkoutCommand | null>(null);
  let workout = $state.raw<Workout | null>(null);
  let saveState = $state<'idle' | 'saving' | 'failed' | 'saved'>('idle');
  let error = $state('');
  let canReview = $state(false);
  let commandAccount = '';
  let unavailable = $derived(
    disabled ||
      !plan.placements.length ||
      plan.placements.some((p) => p.exercise.archived || p.variant.archived)
  );
  beforeNavigate(({ cancel }) => {
    if (pending) {
      cancel();
      error = 'Resolve this start with Retry before leaving.';
    }
  });
  function beforeUnload(event: BeforeUnloadEvent) {
    if (pending) {
      event.preventDefault();
      event.returnValue = '';
    }
  }
  async function save() {
    if (!pending) return;
    if ((get(user)?.id ?? '') !== commandAccount) {
      error = 'Your account changed. Return to the original account to resolve this start.';
      return;
    }
    saveState = 'saving';
    error = '';
    canReview = false;
    try {
      workout = (await startWorkout(pending)).workout;
      pending = null;
      onpending(false);
      saveState = 'saved';
    } catch (failure) {
      saveState = 'failed';
      error = failure instanceof Error ? failure.message : 'Unable to start the workout.';
      canReview = failure instanceof TrainingError && [400, 404, 409, 422].includes(failure.status);
    }
  }
  function start() {
    if (pending || unavailable) return;
    pending = {
      operationId: crypto.randomUUID(),
      sessionId: plan.id,
      expectedSessionRevision: plan.revision,
    };
    commandAccount = get(user)?.id ?? '';
    onpending(true);
    void save();
  }
  async function review() {
    if (!canReview) return;
    try {
      reviewed = (await readSession(plan.id)).session;
      pending = null;
      onpending(false);
      saveState = 'idle';
      canReview = false;
      error = '';
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to review the plan.';
    }
  }
</script>

<svelte:window onbeforeunload={beforeUnload} />
{#if workout}
  <p role="status">Saved</p>
  {#if workout.sourceSession?.id !== plan.id}<p>
      A workout is already active: {workout.sourceSession?.name ?? 'Workout'}. Continue it before
      starting {plan.name}.
    </p>
  {:else}<p>
      {workout.sourceSession?.name ?? plan.name} · {workout.status === 'active'
        ? 'Workout ready'
        : 'Workout ended'}
    </p>{/if}
  <a href={resolve('/workout/[id]', { id: workout.id })}>Continue workout</a>
{:else}
  <button class="start" onclick={start} disabled={unavailable || pending !== null}
    >Start workout</button
  >
  {#if saveState === 'saving'}<p role="status">Saving…</p>{/if}
  {#if saveState === 'failed'}<p role="status">Not saved</p>
    <button onclick={save}>Retry</button>{/if}
  {#if error}<p role="alert">{error}</p>{/if}
  {#if canReview}<button onclick={review}>Review current plan</button>{/if}
{/if}

<style>
  .start {
    background: var(--color-secondary);
    padding: 0.75rem 1rem;
    border-radius: 12px;
  }
  button:disabled {
    opacity: 0.5;
  }
</style>
