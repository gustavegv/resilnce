<script lang="ts">
  import { beforeNavigate, goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import { readSession, startWorkout, TrainingError } from './client';
  import type { Session, StartWorkoutCommand, Workout } from './types';

  let {
    session,
    disabled = false,
    autostart = false,
    onpending = (_pending: boolean) => {},
  }: {
    session: Session;
    disabled?: boolean;
    autostart?: boolean;
    onpending?: (pending: boolean) => void;
  } = $props();
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
      if (autostart)
        void goto(resolve('/workout/[id]', { id: workout.id }), { replaceState: true });
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
  onMount(() => {
    if (autostart) start();
  });
  async function review() {
    if (!canReview) return;
    try {
      reviewed = (await readSession(plan.id)).session;
      pending = null;
      onpending(false);
      saveState = 'idle';
      canReview = false;
      error = '';
      if (autostart) start();
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to review the plan.';
    }
  }
</script>

<svelte:window onbeforeunload={beforeUnload} />
{#if autostart && unavailable}
  <p role="alert">
    {!plan.placements.length
      ? 'Add an exercise to this session before starting a workout.'
      : 'This session contains an unavailable exercise or variant. Edit the session before starting.'}
  </p>
  <a class="buttonClass" href={resolve('/tracker')}>Return to tracker</a>
{:else if autostart && saveState !== 'failed'}
  <p role="status">Loading workout…</p>
{:else if workout}
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
  <a class="buttonClass continue" href={resolve('/workout/[id]', { id: workout.id })}
    >Continue workout</a
  >
{:else}
  {#if !autostart}
    <button class="buttonClass start" onclick={start} disabled={unavailable || pending !== null}
      >Start workout</button
    >
  {/if}
  {#if saveState === 'saving'}<p role="status">Saving…</p>{/if}
  {#if saveState === 'failed'}<p role="status">Not saved</p>
    <button class="buttonClass" onclick={save}>Retry</button>{/if}
  {#if error}<p role="alert">{error}</p>{/if}
  {#if canReview}<button class="buttonClass minimal" onclick={review}>Review current plan</button
    >{/if}
{/if}

<style>
  .start,
  .continue {
    width: 100%;
    margin-top: 1rem;
  }
  button:disabled {
    opacity: 0.5;
  }
</style>
