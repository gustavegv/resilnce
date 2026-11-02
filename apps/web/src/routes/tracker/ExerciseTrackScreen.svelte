<script lang="ts">
  import SetBlock from '../../components/SetBlock.svelte';
  import ConfirmSelection from '../../components/ConfirmSelection.svelte';
  import Icon from '@iconify/svelte';
  import type { ExerciseInfo } from './dbFetches';
  import { fade } from 'svelte/transition';
  import RIRSlider from '../../components/RPESlider.svelte';
  import { onMount } from 'svelte';
  import type { WorkoutExercise, WorkoutSet } from '$lib/training/types';

  let {
    finished = false,
    exIndex = 0,
    onCount = () => {},
    onSubmit,
    exData,
    attempt,
    candidates = {},
    disabled = false,
    finishDisabled = false,
    onCandidate,
    onRecord,
  }: {
    finished?: boolean;
    exIndex?: number;
    onCount?: ({ id, count }: { id: number; count: number }) => void;
    onSubmit: () => void;
    exData?: ExerciseInfo;
    attempt?: WorkoutExercise;
    candidates?: Record<string, string>;
    disabled?: boolean;
    finishDisabled?: boolean;
    onCandidate?: (set: WorkoutSet, value: string) => void;
    onRecord?: (set: WorkoutSet, state: WorkoutSet['state'], count?: number) => void;
  } = $props();

  let infoShown = $state(false);
  let RIRValue = $state(5);

  let rpeMode = $state(false);

  onMount(() => {
    const value = document.cookie
      .split('; ')
      .find((row) => row.startsWith('rpe-mode='))
      ?.split('=')[1];

    rpeMode = value === 'true';
  });

  const errEx: ExerciseInfo = {
    name: 'err',
    currentProgress: {
      sets: 0,
      repsPerSet: [0],
      weightPerSet: [0],
      restSeconds: 0,
    },
  };

  let currrent = $derived(exData ?? errEx);
  let namev: string = $derived(attempt?.exercise.name ?? currrent.name);
  let weightv = $derived(
    attempt
      ? new Set(attempt.sets.map((set) => set.targetWeight)).size > 1
        ? 'Mixed'
        : (attempt.sets[0]?.targetWeight ?? '0')
      : currrent.currentProgress.weightPerSet[0]
  );
  let threshold: number = $derived(
    attempt?.currentVariant.repThreshold ?? currrent.rep_threshold ?? 12
  );
  let increase = $derived(attempt?.currentVariant.autoIncrement ?? currrent.auto_increase ?? 2.5);
  let repsv: number[] = $derived(currrent.currentProgress.repsPerSet);

  function showInfo() {
    infoShown = true;
    setTimeout(() => {
      infoShown = false;
    }, 4000);
  }

  function uniqueKey(set: number, excerID: number) {
    return `${excerID}-s${set}`;
  }
</script>

<header>
  <h1 class="exercise-title">
    {namev}
    <button class="info-trigger" aria-label="Exercise information" onclick={showInfo}
      ><Icon icon="material-symbols:info-outline-rounded" width={24} color="gray" /></button
    >
  </h1>
  {#if attempt}<p class="variant-label">
      {attempt.currentVariant.name} · {finished ? 'Completed' : 'Draft'}
    </p>{/if}
  <div class="info">
    <p class="exercise-subtitle">
      Target Weight: <strong>{weightv}{weightv === 'Mixed' ? '' : ' KG'}</strong>
    </p>
    <div class="additional-info" class:hidden={!infoShown}>
      <p class="exercise-info">
        Rep threshold at <strong>{threshold} reps</strong>
      </p>
      <p class="exercise-info">
        Auto increase with <strong>{increase} kg</strong>
      </p>
    </div>
  </div>
</header>

<cont class="cont">
  {#if attempt}
    {#each attempt.sets as set (set.id)}
      <div class="set w-[100%]" in:fade={{ delay: (set.position - 1) * 50 }}>
        <SetBlock
          id={set.position}
          {finished}
          reps={set.targetReps}
          workoutSet={set}
          candidate={candidates[set.id]}
          {disabled}
          onCandidate={(value) => onCandidate?.(set, value)}
          onRecord={(count) => onRecord?.(set, 'performed', count)}
          onSkip={() => onRecord?.(set, 'skipped')}
          onReset={() => onRecord?.(set, 'pending')}
        />
      </div>
    {/each}
    {#if !finished && attempt.sets.some((set) => set.kind === 'prescribed' && set.state === 'pending')}
      <p class="pending-help">
        Resolve every pending set as performed or skipped before finishing.
      </p>
    {/if}
    <ConfirmSelection {finished} onConfirm={onSubmit} disabled={finishDisabled} />
  {:else}
    {#each repsv as block, index (uniqueKey(index, exIndex))}
      <div class="w-[100%]" in:fade={{ delay: index * 50 }}>
        <SetBlock id={index + 1} {finished} reps={block} onCountChange={onCount} />
      </div>
    {/each}
    {#if rpeMode}<RIRSlider value={RIRValue} />{/if}
    <ConfirmSelection {finished} onConfirm={onSubmit} />
  {/if}
</cont>

<style>
  .info-trigger {
    background: transparent;
    padding: 0;
    display: flex;
  }
  .variant-label,
  .pending-help {
    font-size: 0.75rem;
    color: var(--text-muted);
    text-align: left;
  }
  .variant-label {
    margin: 0 0 0.5rem;
  }
  .pending-help {
    margin: 0.25rem 0 -1rem;
  }
  .cont {
    width: 100%;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    align-items: center;
    overflow-y: auto;
    overflow-x: hidden;
    touch-action: pan-y;
    gap: 0.5rem;
  }

  header {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: baseline;
    margin-bottom: 2rem;
  }

  .exercise-title {
    display: inline-flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    margin: 0 0 0.25rem;
    font-size: 2.25rem;
    line-height: 1.1;
    font-weight: 700;
    letter-spacing: -0.02em;
    text-align: left;
  }

  .exercise-subtitle {
    margin: 0;
    font-size: 0.875rem;
    letter-spacing: 0.03em;
    color: var(--text-muted);
    text-align: left;
    flex-shrink: 0;
  }

  .exercise-subtitle strong {
    color: var(--color-alt);
    font-weight: 700;
    white-space: nowrap;
  }

  .info {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    width: 100%;
    flex-wrap: wrap;
  }

  .additional-info {
    margin-top: var(--spacing-xs);
    display: flex;
    flex-direction: column;
    justify-content: end;
    overflow: hidden;
    height: 2.5rem;
    transition: all ease 400ms;
  }

  .additional-info.hidden {
    height: 0;
    width: 0;
    visibility: hidden;
  }

  .exercise-info {
    margin: 0;
    font-size: 0.8rem;
    letter-spacing: 0.03em;
    color: var(--text-muted);

    text-align: right;
  }

  .exercise-info strong {
    font-weight: 700;
  }
</style>
