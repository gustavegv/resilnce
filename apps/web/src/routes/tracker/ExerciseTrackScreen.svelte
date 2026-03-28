<script lang="ts">
  import SetBlock from '../../components/SetBlock.svelte';
  import ConfirmSelection from '../../components/ConfirmSelection.svelte';
  import Icon from '@iconify/svelte';
  import type { ExerciseInfo } from './dbFetches';
  import { fade } from 'svelte/transition';

  let {
    finished,
    exIndex,
    onCount,
    onSubmit,
    exData,
  }: {
    finished: boolean;
    exIndex: number;
    onCount: ({ id, count }: { id: number; count: number }) => void;
    onSubmit: () => void;
    exData: ExerciseInfo | undefined;
  } = $props();

  let infoShown = $state(false);
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
  let namev: string = $derived(currrent.name);
  let weightv: number = $derived(currrent.currentProgress.weightPerSet[0]);
  let threshold: number = $derived(currrent.rep_threshold ?? 12);
  let increase: number = $derived(currrent.auto_increase ?? 2.5);
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
    <Icon icon="material-symbols:info-outline-rounded" width={24} color="gray" onclick={showInfo} />
  </h1>
  <div class="info">
    <p class="exercise-subtitle">
      Target Weight: <strong>{weightv} KG</strong>
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
  {#each repsv as block, index (uniqueKey(index, exIndex))}
    <div class="w-[100%]" in:fade={{ delay: index * 50 }}>
      <SetBlock id={index + 1} {finished} reps={block} onCountChange={onCount} />
    </div>
  {/each}

  <ConfirmSelection {finished} onConfirm={onSubmit} />
</cont>

<style>
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
  }

  .exercise-subtitle strong {
    color: var(--color-alt);
    font-weight: 700;
  }

  .info {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    width: 100%;
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
