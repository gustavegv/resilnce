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
  import type { Exercise } from '$lib/firebaseDataHandler';
  import type { ExerciseInfo } from '$lib/firebaseCreation';

  import '../../app.css';
  import { goto } from '$app/navigation';

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

  // Derived variabler
  const currentExercise = $derived(
    !loading && exercises[currentExerciseIndex] ? exercises[currentExerciseIndex] : undefined,
  );

  const exName: string = $derived(currentExercise?.name ?? '');
  const exID: string = $derived(currentExercise?.id ?? '');
  const repArray: number[] = $derived(currentExercise?.currentProgress.repsPerSet ?? []);
  const exWeight: number = $derived(currentExercise?.currentProgress.weightPerSet[0] ?? 0);
  const finished: boolean = $derived(currentExercise?.finished ?? false);

  onMount(async () => {
    try {
      exercises = await getOrderedExercises('user1', sesID);
      console.log('SesID:', sesID);
      console.log('Fetched exercises:', exercises);
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  });

  $effect(() => {
    if (currentExercise) {
      console.log('Index:', currentExerciseIndex, 'loaded');
      $inspect(repArray);
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
      const updatedReps = await saveRecordedLift(finalReps, exWeight, exID);
    }

    exercises[currentExerciseIndex].finished = true;

    flashLoadingScreen();

    if (checkAllFinished()) {
      allFinished = true;
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

  function uniqueKey(set: number, excerID: number) {
    return `${excerID}-s${set}`;
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
</script>

{#if loading}
  <p>Loading exercises...</p>
{:else if error}
  <p>Guen error: {error}</p>
{:else if allFinished}
  <div class="container">
    <h1>Session finished!</h1>
    <div class="box">
      <h2>Session overview</h2>
      {#each exercises as blob, index}
        <div class="blob-cont">
          <div class="blob-info">
            <p class="lowkey">{index + 1}.</p>
            <h3>{blob.name}</h3>
            <p>{blob.currentProgress.weightPerSet[0]} kg</p>
          </div>
          <div class="blob-inner">
            {#each blob.currentProgress.repsPerSet as rep, index}
              <p>{rep} reps</p>
            {/each}
          </div>
        </div>
      {/each}
    </div>
    <button onclick={() => goto('/')}>Return to homepage</button>
  </div>
{:else}
  <main class="app-container">
    <div class="movement-cont">
      <button class="movement-b mini" onclick={() => prevExercise()}>Prev</button>
      <p>{currentExerciseIndex + 1}/{exercises.length}</p>
      <button class="movement-b mini" onclick={() => skipExercise()}>Skip</button>
    </div>
    <header>
      <h1>{exName}</h1>
      <h2>{exWeight} kg</h2>
    </header>

    {#each repArray as block, index (uniqueKey(index, currentExerciseIndex))}
      <SetBlock id={index + 1} {finished} reps={block} onCountChange={handleCountChange} />
    {/each}

    <ConfirmSelection {finished} onConfirm={handleSubmit} />

    {#if showOverlay}
      <div class="overlay" transition:fade={{ duration: 150 }}></div>
    {/if}
  </main>
{/if}

<style>
  .container {
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
  }

  .box {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;

    width: 80%;
    box-sizing: border-box;

    border-radius: 15px;
    background-color: #2b2b2b;
    padding: 20px;
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
  header h1 {
    font-size: 2.5rem;
    margin-bottom: 0.5rem;
    margin-top: 0.5rem;
  }
  header h2 {
    font-size: 1.5rem;
    color: var(--color-gray);
    font-weight: 400;
    margin-top: 0;
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
    background-color: var(--color-gray);
    color: black;
  }
</style>
