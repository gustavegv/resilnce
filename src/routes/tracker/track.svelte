<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import SetBlock from '../../components/SetBlock.svelte';
  import ConfirmSelection from '../../components/ConfirmSelection.svelte';
  import { getOrderedExercises, saveRecordedLift } from '$lib/firebaseDataHandler';
  import type { Exercise } from '$lib/firebaseDataHandler';
  import '../../app.css';

// Exercise ID som blir tilldelad när man callar komponenten:
//   Exempel:  <Track exID="push" />
  let { exID }: { exID: string } = $props();

  type RecNum = Record<number, number>
  type ExData = { id: number; reps: number }

  
  
  // State variabler
  let exercises: Exercise[] = $state([]);
  let currentExerciseIndex = $state(0);
  let loading: boolean =    $state(true);
  let blockStates: RecNum = $state({});
  let showOverlay: boolean = $state(false);
  
  let error: string | null = $state(null);



  // Derived variabler
    const currentExercise = $derived(
    !loading && exercises[currentExerciseIndex]
      ? exercises[currentExerciseIndex]
      : undefined
  );

    const exName: string    = $derived(currentExercise?.name   ?? '');
    const exWeight: number  = $derived(currentExercise?.weight ?? 0);
    const exTag: string     = $derived(currentExercise?.tag    ?? '');
    const exerciseData: ExData[] = $derived(currentExercise?.sets ?? []);


  onMount(async () => {
    try {
      exercises = await getOrderedExercises();
      console.log("Fetched exercises:", exercises)

    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  });

  $effect(() => {
    if (currentExercise) {
      blockStates = {};
      console.log('Index:', currentExerciseIndex, 'loaded');
      $inspect(exerciseData)
    }
  });



  //Functions

function handleCountChange(event: CustomEvent<{ id: number; count: number }>) {
    const { id, count } = event.detail;
    blockStates = { ...blockStates, [id]: count };
  }

  function flashLoadingScreen() {
    showOverlay = true;
    setTimeout(() => (showOverlay = false), 100);
  }

  function loadNextExercise() {
    currentExerciseIndex += 1;
  }

  async function handleSubmit() {
    await saveRecordedLift(blockStates, exWeight, exTag);
    console.log("Block state saved:", blockStates)
    flashLoadingScreen();
    setTimeout(loadNextExercise, 100);
  }

  function uniqueKey(set: number, excerID: number){
    return `${excerID}-s${set}`
  }

</script>

{#if loading}
  <p>Loading exercises...</p>
{:else if error}
  <p>Error: {error}</p>
{:else}
  <main class="app-container">
    <header>
      <h1>{exName}</h1>
      <h2>{exWeight} kg</h2>
    </header>


    {#each exerciseData as block (uniqueKey(block.id, currentExerciseIndex))}
      <SetBlock id={block.id} reps={block.reps} on:countChange={handleCountChange} />
    {/each}

    <ConfirmSelection onConfirm={handleSubmit} />

    {#if showOverlay}
      <div class="overlay" transition:fade={{ duration: 100 }}></div>
    {/if}
  </main>
{/if}

<style>
.app-container {
  padding: 2rem;
  max-width: 600px;
  margin: 0 auto;
  text-align: center;
}
header h1 {
  font-size: 2.5rem;
  margin-bottom: 0.5rem;
}
header h2 {
  font-size: 1.5rem;
  color: var(--color-secondary);
}
.overlay {
  position: fixed;
  inset: 0;
  background: black;
  pointer-events: none;
}
</style>