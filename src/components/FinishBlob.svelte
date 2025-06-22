<script lang="ts">
  import type { ExerciseInfo } from '$lib/firebaseCreation';
  import Icon from '@iconify/svelte';

  let {
    exercises,
  }: {
    exercises: ExerciseInfo[];
  } = $props();

  function checkPR(ar: number[]): boolean {
    let repLimit = 12;
    let count = 0;

    ar.forEach((nu) => {
      count += nu;
    });

    let avg = count / ar.length;
    return avg > repLimit;
  }
</script>

{#each exercises as blob, index}
  <div class="blob-cont relative">
    {#if checkPR(blob.currentProgress.repsPerSet)}
      <div class="absolute right-1">
        <Icon icon="ri:medal-line" width="35" color="gold" />
      </div>
    {/if}
    <div class="blob-info">
      <p class="lowkey">{index + 1}.</p>
      <h3 class="capitalize">{blob.name}</h3>
      <p>
        {blob.currentProgress.weightPerSet[0]} kg
        <span class="text-lg font-bold text-green-400"
          >{checkPR(blob.currentProgress.repsPerSet) ? '+' : ''}</span
        >
      </p>
    </div>
    <div class="blob-inner pr-3">
      {#each blob.currentProgress.repsPerSet as rep, index}
        <p>{rep} reps</p>
      {/each}
    </div>
  </div>
{/each}

<style>
  .blob-cont {
    display: flex;
    box-sizing: border-box;
    padding: 0 2rem;
    flex-direction: row;
    background-color: var(--color-background);
    justify-content: space-between;
    align-items: baseline;
    width: 100%;
    border-radius: 15px;
    margin: 5px;

    box-shadow: var(--shadow-dark);
  }

  .blob-info {
    display: flex;
    flex-direction: column;
    align-items: baseline;
    text-align: left;
    width: 70%;
    padding: 1rem 0;
  }

  .blob-inner {
    color: gray;
  }
</style>
