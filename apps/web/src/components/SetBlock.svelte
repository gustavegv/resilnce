<script lang="ts">
  import { onMount } from 'svelte';
  import '../app.css';
  import Icon from '@iconify/svelte';
  import type { WorkoutSet } from '$lib/training/types';

  let {
    id,
    reps,
    finished,
    onCountChange,
    workoutSet,
    candidate,
    disabled = false,
    onCandidate,
    onRecord,
    onSkip,
    onReset,
  }: {
    id: number;
    reps: number;
    finished?: boolean;
    onCountChange?: (detail: { id: number; count: number }) => void;
    workoutSet?: WorkoutSet;
    candidate?: string;
    disabled?: boolean;
    onCandidate?: (value: string) => void;
    onRecord?: (count: number) => void;
    onSkip?: () => void;
    onReset?: () => void;
  } = $props();

  let curCount = $state(0);
  let checked = $state(false);
  let weightPopped = $state(false);
  let shownCount = $derived(
    workoutSet
      ? candidate !== undefined && /^\d+$/.test(candidate)
        ? Number(candidate)
        : (workoutSet.recordedReps ?? reps)
      : curCount
  );
  let completed = $derived(workoutSet ? workoutSet.state === 'performed' : checked);
  onMount(() => {
    if (!workoutSet) {
      curCount = reps;
      onCountChange?.({ id, count: reps });
    }
  });
  function record(count = shownCount) {
    if (disabled || finished) return;
    weightPopped = true;
    setTimeout(() => {
      weightPopped = false;
    }, 150);
    if (workoutSet) {
      onCandidate?.(String(count));
      onRecord?.(count);
    } else {
      curCount = count;
      checked = true;
      onCountChange?.({ id, count });
    }
  }
</script>

<div class="counter-container set--{completed ? 'active' : 'inactive'} finished-{finished}">
  <div class="check {completed}"><Icon icon="gg:check-o" /></div>
  <div class="set-card-left" class:complete={completed}>
    <div class="set-count">{id}</div>
    <div class="set-description">
      <p class="set-status">
        {completed ? 'Completed' : workoutSet?.state === 'skipped' ? 'Skipped' : 'Set goal'}
      </p>
      <p class="set-goal">{reps} <span> reps</span></p>
      {#if workoutSet}<p class="set-weight">{workoutSet.targetWeight} kg target</p>{/if}
    </div>
  </div>
  <div class="controls">
    <button
      class="stepper"
      aria-label={`Decrease reps for set ${id}`}
      disabled={disabled || finished}
      onclick={() => record(Math.max(0, shownCount - 1))}><Icon icon="ic:sharp-minus" /></button
    >
    <button
      class="stepper-count"
      class:stepper-scale={weightPopped}
      aria-label={`Record set ${id}: ${shownCount} reps`}
      disabled={disabled || finished}
      onclick={() => record()}>{shownCount}</button
    >
    <button
      class="stepper"
      aria-label={`Increase reps for set ${id}`}
      disabled={disabled || finished}
      onclick={() => record(shownCount + 1)}><Icon icon="ic:sharp-plus" /></button
    >
  </div>
</div>
{#if workoutSet}
  <details class="set-options">
    <summary
      >Set {id} · {workoutSet.state === 'performed'
        ? `${workoutSet.recordedReps} reps recorded`
        : workoutSet.state === 'skipped'
          ? 'Skipped'
          : 'Pending'} · options</summary
    >
    {#if workoutSet.state === 'performed'}<p>
        Performed: {workoutSet.recordedReps} reps · {workoutSet.recordedWeight} kg
      </p>
    {:else if workoutSet.state === 'skipped'}<p>Skipped · no performance recorded</p>
    {:else}<p>Pending · no performance recorded</p>{/if}
    {#if !finished}
      <button onclick={onSkip} {disabled}>Skip set</button>
      <button onclick={onReset} disabled={disabled || workoutSet.state === 'pending'}
        >Reset to pending</button
      >
    {/if}
  </details>
{/if}

<style>
  .set-weight,
  .set-options {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .set-weight {
    margin: 0.25rem 0 0;
  }
  .set-options {
    width: 100%;
    text-align: left;
    margin: 0.35rem 0 0.75rem;
  }
  .set-options button {
    margin: 0.35rem;
    border-radius: 8px;
    background: var(--surface-middle);
  }
  .check {
    position: absolute;
    right: 2.5rem;
    transition: all 0.3s;
    opacity: 0;
  }

  .check.false {
    opacity: 0;
  }

  .counter-container {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: var(--spacing-md);
    background: var(--surface-low);
    width: 100%;
    border-radius: 10px;
    box-shadow: var(--shadow-dark);
  }

  .finished-true {
    pointer-events: none;
    opacity: 0.5;
  }

  .set-count {
    background: var(--surface-middle);
    padding: 1rem;
    width: 2.5rem;
    height: 2.5rem;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    transition: all ease 300ms;
  }

  .set--active .set-count {
    background: #3c513b;
    border: 1px solid rgba(133, 255, 103, 0.441);
    color: rgb(117, 255, 117);
  }

  .finished-true .set-count {
    background: #2b322b;
    border: 1px solid rgba(133, 255, 103, 0.441);
    color: rgb(160, 255, 160);
  }

  .set-card-left {
    display: flex;
    align-items: center;
    gap: 1rem;
    min-width: 0;
    color: var(--color-text);
  }

  .set-card-left.complete {
    color: #838782;
  }

  .set-description {
    display: flex;
    flex-direction: column;
    justify-content: baseline;
  }

  .set-status {
    margin: 0 0 0.125rem;
    font-size: 0.625rem;
    letter-spacing: -0.01em;
    text-transform: uppercase;
    text-align: left;
    color: inherit;
  }

  .set-goal {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 700;
    white-space: nowrap;
    text-align: left;
  }

  .set-goal span {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-muted);
  }

  .complete .set-goal span {
    font-size: 0.875rem;
    font-weight: 500;
    color: inherit;
  }

  .stepper {
    height: 2.5rem;
    width: 2.5rem;

    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--surface-middle);
    color: var(--color-text);
    transition:
      scale 80ms cubic-bezier(0.175, 0.885, 0.32, 1.275),
      filter ease 100ms;
    filter: brightness(1);
  }

  .stepper:active {
    filter: brightness(0.9);
    transform: scale(0.9);
  }

  .stepper-count {
    transition: scale ease-out 250ms;
    width: 3rem;
    text-align: center;
    font-size: 1.25rem;
    font-weight: 700;
    scale: 1;
  }

  .stepper-scale {
    scale: 1.1;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex: 0 0 auto;
  }

  .counter-container.mini {
    background-color: #111111;
    text-align: left;
    padding: 0.5rem 0.5rem;
    border-radius: 0;
    margin: 0;

    font-size: 16px;
    font-weight: 500;
    pointer-events: none;
    box-shadow: none;
    margin: auto;

    width: 100%;
    max-width: 280px;
  }

  .controls.mini {
    font-size: 16px;
  }

  .mini-count {
    font-size: 1rem;
  }

  button {
    padding: 0.5rem 1rem;
  }

  span {
    font-size: 2rem;
    width: 3rem;
    text-align: center;
  }

  .disabled {
    pointer-events: none;
    opacity: 0.5;
  }
</style>
