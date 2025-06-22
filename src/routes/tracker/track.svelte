<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import SetBlock from '../../components/SetBlock.svelte';
  import ConfirmSelection from '../../components/ConfirmSelection.svelte';
  import {
    getAllSessionMeta,
    getOrderedExercises,
    saveRecordedLift,
  } from '$lib/firebaseDataHandler';
  import { setActivityStatus, loadFinishedExercises } from '$lib/firebaseCreation';
  import type { Exercise } from '$lib/firebaseDataHandler';
  import type { ExerciseInfo } from '$lib/firebaseCreation';

  import '../../app.css';
  import { goto } from '$app/navigation';
  import Icon from '@iconify/svelte';
  import ExerciseTrackScreen from './ExerciseTrackScreen.svelte';
  import LoadingSkeleton from './LoadingSkeleton.svelte';
  import { get } from 'svelte/store';
  import { user } from '../account/user';

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
  const exID: string = $derived(currentExercise?.id ?? '');
  const repArray: number[] = $derived(currentExercise?.currentProgress.repsPerSet ?? []);
  const exWeight: number = $derived(currentExercise?.currentProgress.weightPerSet[0] ?? 0);
  const finished: boolean = $derived(currentExercise?.finished ?? false);

  const userID = $derived(get(user));

  onMount(async () => {
    if (!userID) {
      goto('/account');
      return;
    }
    try {
      exercises = await getOrderedExercises(userID, sesID);
      console.log('\n\nSesID:', sesID);

      await loadUnfinishedSession();
    } catch (e) {
      error = (e as Error).message;
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

  function handleCountChange({ id, count }: { id: number; count: number }) {
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

  async function handleSubmit() {
    if (currentExercise) {
      const finalReps = currentExercise.currentProgress.repsPerSet;
      await saveRecordedLift(
        userID ?? 'error',
        sesID,
        finalReps,
        exWeight,
        currentExercise.id ?? 'error'
      );
    }

    exercises[currentExerciseIndex].finished = true;

    flashLoadingScreen();

    if (checkAllFinished()) {
      allFinished = true;
      await setActivityStatus(userID ?? 'error', sesID, false, []);
      console.log('Workout finished, active set to false');
    } else {
      setUnfinishedSession();
      setTimeout(loadNextExercise, 100);
    }
  }

  async function setUnfinishedSession() {
    const fin = getFinished();
    console.log('Finished so far:', fin);
    await setActivityStatus(userID ?? 'error', sesID, true, fin);
  }

  async function loadUnfinishedSession() {
    const info = await loadFinishedExercises(userID ?? 'error', sesID);
    let arrFin: number[] = [];
    if (info.unfinished) {
      const fins = info.finishedIDXS;

      fins.forEach((fin) => {
        exercises[fin].finished = true;
        console.log(fin, 'is finished already!');
      });
      arrFin = fins;
    }

    await setActivityStatus(userID ?? 'error', sesID, true, arrFin);
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

  function getFinished(): number[] {
    let finished: number[] = [];

    exercises.forEach((ex, index) => {
      if (ex.finished == true) {
        finished = [...finished, index];
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

  function quitSession() {
    if (confirm('Are you sure you want to quit the session?')) {
      setActivityStatus(userID ?? 'error', sesID, false);
      goto('/');
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

{#if loading || error}
  {#if loading}
    <main class="app-container">
      <LoadingSkeleton />
    </main>
  {:else if error}
    <p>Guen error: {error}</p>
  {/if}
{:else if allFinished}
  <div class="container">
    <h1 class="pb-8 text-2xl font-bold">Session finished!</h1>
    <div class="box">
      <h2 class="py-2 text-lg font-semibold">Session overview</h2>
      {#each exercises as blob, index}
        <div class="blob-cont relative">
          <div class="blob-info">
            <p class="lowkey">{index + 1}.</p>
            <h3 class="capitalize">{blob.name}</h3>
            <p>{blob.currentProgress.weightPerSet[0]} kg</p>
          </div>
          <div class="blob-inner pr-3">
            {#each blob.currentProgress.repsPerSet as rep, index}
              <p>{rep} reps</p>
            {/each}
          </div>
          {#if checkPR(blob.currentProgress.repsPerSet)}
            <div class="absolute right-1">
              <Icon icon="ri:medal-line" width="35" color="gold" />
            </div>
          {/if}
        </div>
      {/each}
    </div>
    <button class="buttonClass but" onclick={() => goto('/')}>Return to homepage</button>
  </div>
{:else if editMode}
  <main class="app-container edit-mode">
    <ExerciseTrackScreen
      name={exName}
      weight={exWeight}
      reps={repArray}
      finished={true}
      {exID}
      exIndex={currentExerciseIndex}
      onCount={handleCountChange}
      onSubmit={handleSubmit}
      onCancel={exitEditMode}
      edit={true}
      {sesID}
    />
  </main>
{:else}
  <button onclick={() => quitSession()} class="abs buttonClass">Quit</button>
  <main class="app-container">
    <div class="movement-cont">
      <button class="movement-b mini buttonClass" onclick={() => prevExercise()}>Prev</button>
      <p>{currentExerciseIndex + 1}/{exercises.length}</p>
      <button class="movement-b mini buttonClass" onclick={() => skipExercise()}>Skip</button>
    </div>

    <ExerciseTrackScreen
      name={exName}
      weight={exWeight}
      reps={repArray}
      {finished}
      exIndex={currentExerciseIndex}
      onCount={handleCountChange}
      onSubmit={handleSubmit}
      {sesID}
    />
  </main>
{/if}

<button class="floating-edit buttonClass {editMode}" onclick={enterEditMode}>
  <Icon icon={'material-symbols:edit-outline-rounded'} height={40} />
</button>

{#if showOverlay}
  <div class="overlay" transition:fade={{ duration: 150 }}></div>
{/if}

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
    font-weight: 900;
    width: fit-content;
    height: fit-content;
    background-color: rgba(240, 248, 255, 0);
    box-shadow: none;
    right: -0rem;
    top: -1rem;
    z-index: 4;
  }
  .container {
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    background: var(--color-background);
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

  h3 {
    margin: 0;
  }

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
