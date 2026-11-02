<script lang="ts">
  import Icon from '@iconify/svelte';
  import type { ExerciseInfo } from '../routes/tracker/dbFetches';
  import type { WorkoutExercise } from '$lib/training/types';
  import { CheckRepThresholdMet } from '../routes/tracker/saving';

  import { fade, scale } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';

  let {
    exercises = [],
    attempts = [],
  }: {
    exercises?: ExerciseInfo[];
    attempts?: WorkoutExercise[];
  } = $props();

  let entries = $derived(
    attempts.length
      ? attempts.map((attempt) => ({
          name: attempt.exercise.name,
          variantName: attempt.currentVariant.name,
          currentProgress: {
            repsPerSet: attempt.sets.map((set) =>
              set.state === 'performed' ? (set.recordedReps ?? 0) : 'Skipped'
            ),
            weightPerSet: [
              ...new Set(
                attempt.sets
                  .filter((set) => set.state === 'performed')
                  .map((set) => set.recordedWeight)
              ),
            ],
          },
          attempt,
        }))
      : exercises.map((exercise) => ({ ...exercise, variantName: '', attempt: undefined }))
  );

  function thumpIn(node: Element, { delay = 0, duration = 420 } = {}) {
    return {
      delay,
      duration,
      easing: cubicOut,
      css: (t: number) => {
        const scale =
          t < 0.65 ? 0.88 + (1.08 - 0.88) * (t / 0.65) : 1.08 - (1.08 - 1) * ((t - 0.65) / 0.35);

        return `
        opacity: ${t};
        transform: scale(${scale});
      `;
      },
    };
  }

  function checkPR(repsPerSet: (number | string)[], index: number): boolean {
    if (attempts.length) {
      const attempt = attempts[index];
      const prescribed = attempt.sets.filter((set) => set.kind === 'prescribed');
      return (
        attempt.state === 'completed' &&
        prescribed.length > 0 &&
        prescribed.every((set) => set.state === 'performed') &&
        CheckRepThresholdMet(
          prescribed.map((set) => set.recordedReps ?? 0),
          attempt.repThreshold
        )
      );
    }
    const repThreshold = exercises[index].rep_threshold ?? 12;

    return CheckRepThresholdMet(repsPerSet as number[], repThreshold);
  }

  function toTwoDigit(num: number): string {
    if (num < 10) {
      return '0' + num;
    }
    return String(num);
  }
</script>

{#each entries as blob, index (blob.attempt?.id ?? index)}
  <div class="blob-cont relative" in:thumpIn|global={{ delay: 150 + index * 150 }}>
    {#if checkPR(blob.currentProgress.repsPerSet, index)}
      <div class="absolute right-1" role="img" aria-label="Rep threshold met">
        <Icon icon="ri:medal-line" width="35" color="#444444" />
      </div>
    {/if}

    <div class="main-info">
      <div class="left-info">
        <p class="exercise-number">Exercise {toTwoDigit(index + 1)}</p>
        <p class="exercise-name capitalize">
          {blob.name}
          <span class="inline-arrow" class:pr={checkPR(blob.currentProgress.repsPerSet, index)}>
            <Icon icon="streamline:graph-arrow-increase-solid" color="#71d772" />
          </span>
        </p>
        {#if blob.variantName}<p class="variant-label">{blob.variantName} · Completed</p>{/if}
      </div>
      <p class="exercise-weight">
        {blob.currentProgress.weightPerSet.length > 1
          ? 'Mixed'
          : (blob.currentProgress.weightPerSet[0] ?? '—')} <span class="exercise-unit">kg</span>
      </p>
    </div>

    <div class="set-info">
      {#each blob.currentProgress.repsPerSet as rep, index (blob.attempt?.sets[index]?.id ?? index)}
        <div class="set-box">
          <p class="set-box-label">Set {index + 1}</p>
          <p class="set-box-value">{rep}</p>
        </div>
      {/each}
    </div>
    {#if blob.attempt}
      <details class="performance-details">
        <summary>Recorded performance and next targets</summary>
        {#each blob.attempt.sets as set (set.id)}
          <p>
            Set {set.position} · {set.state === 'performed'
              ? `Performed: ${set.recordedReps} reps · ${set.recordedWeight} kg`
              : 'Skipped · no performance recorded'}
          </p>
        {/each}
        <div aria-label="Current next targets">
          <h4>Current next targets</h4>
          {#each blob.attempt.currentVariant.targetSets as target (target.id)}
            <p>Set {target.position}: {target.targetReps} reps · {target.targetWeight} kg</p>
          {/each}
        </div>
      </details>
    {/if}
  </div>
{/each}

<style>
  .variant-label,
  .performance-details {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .performance-details {
    margin-top: 1rem;
    width: 100%;
    text-align: left;
  }
  .blob-cont {
    display: flex;
    flex-direction: column;
    box-sizing: border-box;
    padding: 0rem 2rem 1rem 2rem;
    background: var(--surface-low);

    justify-content: space-between;
    align-items: baseline;
    width: 100%;
    border-radius: 15px;
    margin: 5px;

    box-shadow: var(--shadow-dark);
  }

  .main-info {
    width: 100%;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    text-align: left;
    padding: 1rem 0;
  }

  .set-info {
    width: 100%;
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 0.5rem;
  }

  .set-box {
    padding: 0.5rem 1rem;
    border-radius: 0.5rem;
    background: var(--surface-middle);

    text-align: center;
  }

  .set-box-label {
    margin: 0 0 0.25rem;
    font-family: var(--font-label);
    font-size: 0.625rem;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .set-box-value {
    margin: 0;
    font-size: 1rem;
    font-weight: 700;
    color: var(--primary);
  }

  .exercise-number {
    margin: 0 0 0.25rem;
    font-family: var(--font-label);
    font-size: 0.625rem;
    letter-spacing: 0.16em;
    text-transform: uppercase;
    color: var(--primary);
  }

  .exercise-name {
    display: inline-flex;
    margin: 0;
    font-size: 1.25rem;
    font-weight: 700;
    letter-spacing: -0.02em;
    align-items: center;
  }

  .inline-arrow {
    margin-left: 10px;
    opacity: 0;
  }

  .inline-arrow.pr {
    opacity: 1;
  }

  .exercise-weight {
    margin: 0;
    text-align: right;
    font-size: 1.125rem;
    font-weight: 700;
    white-space: nowrap;
  }

  .exercise-unit {
    font-size: 0.75rem;
    font-weight: 400;
    color: var(--text-muted);
  }
</style>
