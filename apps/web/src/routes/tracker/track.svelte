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

  function flashLoadingScreen() {
    showOverlay = true;
    setTimeout(() => (showOverlay = false), 100);
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
    const savedProgress = packageUpdatedProgress();
    if (savedProgress == null) {
      console.error('Updated info packaging error');
      return;
    }

    SendUpdate(savedProgress, sesID);

    exercises[currentExerciseIndex].finished = true;
    flashLoadingScreen();

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

  function prevExercise() {
    if (currentExerciseIndex > 0) {
      currentExerciseIndex--;
    }
  }

  function skipExercise() {
    if (exercises.length > currentExerciseIndex + 1) {
      currentExerciseIndex++;
    }
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
</script>

{#if showOverlay}
  <div class="overlay" transition:fade={{ duration: 150 }}></div>
{/if}

<main class="app-container">
  {#if loading || error}
    <LoadingSkeleton />
    <h1 class="text-2xl">{error ?? ''}</h1>
  {:else if allFinished}
    <h1 class="pb-8 text-2xl font-bold">Session finished!</h1>
    <Card.Root class="box m-4 w-full p-4">
      <h2 class="py-2 text-lg font-semibold">Session overview</h2>
      <FinishBlob {exercises} />
    </Card.Root>
    <button class="buttonClass w-full" onclick={() => goto(resolve(`/`))}>Return to homepage</button
    >
  {:else if editMode}
    <ExerciseTrackScreen
      name={exName}
      weight={exWeight}
      reps={repArray}
      finished={true}
      exIndex={currentExerciseIndex}
      onCount={handleRepCountIncrementation}
      onSubmit={submitExercise}
      onCancel={exitEditMode}
      edit={true}
      {sesID}
    />
  {:else}
    <button onclick={() => quitSession()} class="abs buttonClass">
      <Icon icon="entypo:log-out" color="grey" width={20} />
    </button>
    <div class="movement-cont">
      <button class="movement-b mini buttonClass" onclick={() => prevExercise()}>Prev</button>
      <p class="text-muted-foreground">{currentExerciseIndex + 1}/{exercises.length}</p>
      <button class="movement-b mini buttonClass" onclick={() => skipExercise()}>Skip</button>
    </div>

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
    background: rgb(255, 255, 255);
    opacity: 0.2;
    pointer-events: none;
  }
  .movement-cont {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    width: 100%;
  }

  .movement-b {
    background-color: var(--color-secondary);
    color: var(--color-contrast);
    box-shadow: var(--shadow-dark);
  }
</style>
