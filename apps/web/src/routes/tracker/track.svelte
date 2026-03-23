<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  import '../../app.css';
  import { goto } from '$app/navigation';
  import Icon from '@iconify/svelte';
  import ExerciseTrackScreen from './ExerciseTrackScreen.svelte';
  import LoadingSkeleton from './LoadingSkeleton.svelte';
  import * as Card from '$lib/components/ui/card/';
  import FinishBlob from '../../components/FinishBlob.svelte';
  import { resolve } from '$app/paths';

  import {
    CompleteSession,
    GetFinishedExercises,
    GetSessionExercises,
    SendUpdate,
    SetActiveSession,
    type ExerciseInfo,
  } from './dbFetches';

  // Exercise ID som blir tilldelad när man callar komponenten:
  //   Exempel:  <Track sesID="push" />
  let { sesID }: { sesID: string } = $props();

  // State variabler
  let exercises: ExerciseInfo[] = $state([]);

  let currentExerciseIndex = $state(0);
  let loading: boolean = $state(true);

  let showOverlay: boolean = $state(false);
  let error: string | null = $state(null);
  let allFinished: boolean = $state(false);

  let editMode: boolean = $state(false);

  // Derived variabler
  const currentExercise = $derived(
    !loading && exercises[currentExerciseIndex] ? exercises[currentExerciseIndex] : undefined
  );

  const exName: string = $derived(currentExercise?.name ?? '');
  const exID: number = $derived(currentExercise?.id ?? -1);
  const repArray: number[] = $derived(currentExercise?.currentProgress.repsPerSet ?? []);
  const exWeight: number = $derived(currentExercise?.currentProgress.weightPerSet[0] ?? 0);
  const finished: boolean = $derived(currentExercise?.finished ?? false);

  onMount(async () => {
    try {
      exercises = await GetSessionExercises(parseInt(sesID));
      await SetActiveSession(sesID);
    } catch (e) {
      error = (e as Error).message;
      console.error(error);
    } finally {
      loading = false;
    }
  });

  $effect(() => {
    if (currentExercise) {
      console.log('Exercise index', currentExerciseIndex, 'rendered');
    }
  });

  //Functions

  function handleRepCountIncrementation({ id, count }: { id: number; count: number }) {
    if (currentExercise) {
      currentExercise.currentProgress.repsPerSet[id - 1] = count;
    }
  }

  function flashLoadingScreen(speed: number) {
    showOverlay = true;
    setTimeout(() => (showOverlay = false), speed);
  }

  function loadNextExercise() {
    if (currentExerciseIndex + 1 < exercises.length) {
      currentExerciseIndex += 1;
    } else {
      console.log('Workout finished?');
    }
  }

  function checkRepThresholdMet(r: number[], threshold: number): boolean {
    if (r.length === 0) return false;
    const t = r.reduce((acc, c) => acc + c, 0) / r.length;
    return t > threshold ? true : false;
  }

  function packageUpdatedProgress() {
    if (currentExercise) {
      const finalReps = currentExercise.currentProgress.repsPerSet;
      let finalWeight: number = exWeight;
      const threshold = currentExercise.repThreshold ?? 12;

      var packagedReps: number[] = finalReps.slice(); // kopiera array

      if (checkRepThresholdMet(finalReps, threshold)) {
        const baseRep = Math.floor(threshold / 1.7);
        packagedReps.fill(baseRep, 0, finalReps.length);
        finalWeight += currentExercise.autoIncrease ?? 2.5;
      }

      var packagedWeight: number[] = new Array(packagedReps.length);
      packagedWeight.fill(finalWeight, 0, packagedReps.length);

      const updateInfo = {
        reps: packagedReps,
        weights: packagedWeight,
        id: String(currentExercise.id ?? -1),
      };
      return updateInfo;
    }
    return null;
  }

  async function submitExercise() {
    monitorOutOfBoundsMovement();
    const savedProgress = packageUpdatedProgress();
    if (savedProgress == null) {
      console.error('Updated info packaging error');
      return;
    }

    SendUpdate(savedProgress, sesID);

    exercises[currentExerciseIndex].finished = true;
    flashLoadingScreen(100);

    if (checkAllFinished()) {
      allFinished = true;
      await CompleteSession(sesID);

      console.log('Workout finished, active set to false');
    } else {
      setTimeout(loadNextExercise, 100);
    }
  }

  function checkAllFinished(): boolean {
    let finished: boolean = true;

    exercises.forEach((ex) => {
      if (ex.finished == undefined || ex.finished == false) {
        finished = false;
      }
    });
    return finished;
  }

  let prevExists = $state(false);
  let nextExists = $state(true);

  function monitorOutOfBoundsMovement() {
    if (currentExerciseIndex - 1 < 0) {
      prevExists = false;
    } else {
      prevExists = true;
    }
    if (currentExerciseIndex + 1 >= exercises.length) {
      nextExists = false;
    } else {
      nextExists = true;
    }
  }

  2 / 2;
  1 + 1 / 2;

  function prevExercise() {
    if (currentExerciseIndex > 0) {
      currentExerciseIndex--;
    }
    monitorOutOfBoundsMovement();
  }

  function skipExercise() {
    if (exercises.length > currentExerciseIndex + 1) {
      currentExerciseIndex++;
    }
    monitorOutOfBoundsMovement();
  }

  async function quitSession() {
    if (
      confirm('Are you sure you want to quit the session?\n(All confirmed sets are already saved)')
    ) {
      await CompleteSession(sesID);
      goto(resolve('/'));
    } else {
      console.log('Quit adverted.');
    }
  }

  function exitEditMode() {
    editMode = false;
  }

  function enterEditMode() {
    editMode = true;
  }

  function getProgressRatio(): number {
    if (allFinished) {
      return 100;
    }
    const ratio: number = currentExerciseIndex / exercises.length;
    return 100 * ratio;
  }

  function getProgressFraction(): string {
    if (allFinished) {
      return exercises.length + ' of ' + exercises.length;
    }

    return currentExerciseIndex + 1 + ' of ' + exercises.length;
  }
</script>

{#if showOverlay}
  <div class="overlay" transition:fade={{ duration: 150 }}></div>
{/if}

<main class="app-container">
  {#if loading || error}
    <LoadingSkeleton />
    <h1 class="text-2xl">{error ?? ''}</h1>
  {:else}
    <div class="progress">
      <div class="progress-header">
        <span class="label">Session Progress</span>
        <span class="label label--accent">Exercise {getProgressFraction()}</span>
      </div>

      <div class="progress-track">
        <div class="progress-fill" style="width: {getProgressRatio()}%"></div>
      </div>
    </div>

    {#if allFinished}
      <h1 class="pb-8 text-2xl font-bold">Session finished!</h1>
      <Card.Root class="box m-4 w-full p-4">
        <h2 class="py-2 text-lg font-semibold">Session overview</h2>
        <FinishBlob {exercises} />
      </Card.Root>
      <button class="buttonClass w-full" onclick={() => goto(resolve(`/`))}
        >Return to homepage</button
      >
    {:else}
      <button onclick={() => quitSession()} class="abs buttonClass">
        <Icon icon="entypo:log-out" color="grey" width={20} />
      </button>

      <ExerciseTrackScreen
        name={exName}
        weight={exWeight}
        reps={repArray}
        {finished}
        exIndex={currentExerciseIndex}
        onCount={handleRepCountIncrementation}
        onSubmit={submitExercise}
        {sesID}
      />

      <div class="movement-cont">
        <button class="movement-b" class:inactive={!prevExists} onclick={() => prevExercise()}>
          <Icon icon="material-symbols:arrow-left-alt-rounded" />
          <span>Prev</span>
        </button>
        <button class="movement-b" class:inactive={!nextExists} onclick={() => skipExercise()}>
          <span>Skip</span>
          <Icon icon="material-symbols:arrow-right-alt-rounded" />
        </button>
      </div>
    {/if}
  {/if}
</main>

<button class="floating-edit buttonClass {editMode}" onclick={enterEditMode}>
  <Icon icon={'material-symbols:edit-outline-rounded'} height={40} />
</button>

<style>
  .but {
    width: 80%;
    background: var(--color-alt);
  }

  .progress {
    margin-bottom: 2rem;
    width: 100%;
  }

  .progress-header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    margin-bottom: 0.5rem;
  }

  .label {
    font-family: var(--font-label);
    font-size: 0.75rem;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .label--accent {
    font-weight: 600;
    color: var(--color-alt);
  }

  .progress-track {
    width: 100%;
    height: 2px;
    background: var(--border);
    overflow: hidden;
  }

  .progress-fill {
    transition: all ease-in-out 280ms;
    height: 100%;
    background: var(--color-alt);
  }

  .floating-edit {
    position: absolute;
    bottom: 5rem;
    right: 1rem;
    width: fit-content;
    border-radius: 100px;
    background-color: var(--color-secondary);

    opacity: 0;
    touch-action: none;
  }

  .floating-edit.true {
    opacity: 0;
  }

  .abs {
    position: absolute;
    width: fit-content;
    height: fit-content;
    background-color: rgba(240, 248, 255, 0);
    box-shadow: none;
    right: -1.1rem;
    top: -1.3rem;
    z-index: 6;
  }
  .container {
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    background: var(--color-background);
    overflow-x: hidden;
  }

  .box {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;

    width: 80%;
    box-sizing: border-box;

    border-radius: 15px;
    background-color: var(--color-secondary);
    padding: 2rem 1rem;
    padding-top: 0;
    text-align: left;
    box-shadow: var(--shadow-dark);
  }

  .lowkey {
    color: grey;
  }

  .app-container {
    display: flex;
    flex-direction: column;
    padding: 4rem 2rem;
    max-width: 30rem;
    margin: 0 auto;
    text-align: center;
    align-items: center;
    background: var(--gradient-prim);

    min-height: 100vh;
  }
  .app-container.edit-mode {
    background: var(--gradient-edit);
  }

  .overlay {
    position: fixed;
    inset: 0;
    background: #fff;
    opacity: 0.2;
    pointer-events: none;
  }
  .movement-cont {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    width: 100%;
    margin-top: 2rem;
  }

  .movement-b {
    display: flex;

    flex-direction: row;
    justify-content: space-evenly;
    align-items: center;
    width: 5rem;
    height: 2rem;
    border-radius: 20px;
    border: none;
    outline: none;
    box-shadow: var(--shadow-dark);
  }

  .movement-b.inactive {
    filter: brightness(0.5);
    opacity: 0;
  }

  .movement-b span {
    font-size: 12px;
    color: var(--color-contrast);
  }
</style>
