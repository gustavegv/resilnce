<script lang="ts">
  import Icon from '@iconify/svelte';
  import type { ExerciseInfo } from '../routes/tracker/dbFetches';
  import { fade, scale } from 'svelte/transition';

  let {
    exercises,
  }: {
    exercises: ExerciseInfo[];
  } = $props();

  function checkPR(setList: number[], index: number): boolean {
    const repThreshold = exercises[index].repThreshold ?? 12;

    let totalReps = 0;

    setList.forEach((set) => {
      totalReps += set;
    });

    let avgRepsPerSet = totalReps / setList.length;
    return avgRepsPerSet > repThreshold;
  }

  function toTwoDigit(num: number): string {
    if (num < 10) {
      return '0' + num;
    }
    return String(num);
  }
</script>

{#each exercises as blob, index}
  <div class="blob-cont relative" in:fade|global={{ delay: 150 + index * 150 }}>
    {#if checkPR(blob.currentProgress.repsPerSet, index)}
      <div class="absolute right-1">
        <Icon icon="ri:medal-line" width="35" color={'444444'} />
      </div>
    {/if}

    <div class="main-info">
      <div class="left-info">
        <p class="exercise-number">Exercise {toTwoDigit(index + 1)}</p>
        <p class="exercise-name capitalize">
          {blob.name}
          <span class="inline-arrow" class:pr={checkPR(blob.currentProgress.repsPerSet, index)}>
            <Icon icon="streamline:graph-arrow-increase-solid" color={'#71d772'} />
          </span>
        </p>
      </div>
      <p class="exercise-weight">
        {blob.currentProgress.weightPerSet[0]} <span class="exercise-unit">kg</span>
      </p>
    </div>

    <div class="set-info">
      {#each blob.currentProgress.repsPerSet as rep, index}
        <div class="set-box">
          <p class="set-box-label">Set {index + 1}</p>
          <p class="set-box-value">{rep}</p>
        </div>
      {/each}
    </div>
  </div>
{/each}

<style>
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
    background: var(--color-surface-middle);

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
