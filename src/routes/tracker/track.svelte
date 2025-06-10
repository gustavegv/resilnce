<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import SetBlock from '../../components/SetBlock.svelte';
  import ConfirmSelection from '../../components/ConfirmSelection.svelte';
  import { getOrderedExercises, saveRecordedLift } from '$lib/firebaseDataHandler';
  import type { Exercise } from '$lib/firebaseDataHandler';
  import '../../app.css';

  let exercises: Exercise[] = [];
  let exerciseData: { id: number; reps: number }[] = [];
  let exName = '';
  let exWeight = 0;
  let exTag = '';
  let currentExerciseIndex = 0;
  let loading = true;
  let error: string | null = null;
  let blockStates: Record<number, number> = {};
  let showOverlay = false;

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

  onMount(async () => {
    try {
      exercises = await getOrderedExercises();
      if (exercises.length > 0) {
        const cur = exercises[0];
        exerciseData = cur.sets;
        exName = cur.name;
        exWeight = cur.weight;
        exTag = cur.tag;
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  });

  $: if (!loading && exercises[currentExerciseIndex]) {
    blockStates = {};
    console.log("Index: ", currentExerciseIndex, "loaded")
    const cur = exercises[currentExerciseIndex];
    exerciseData = cur.sets;
    exName = cur.name;
    exWeight = cur.weight;
    exTag = cur.tag;


    console.log(exerciseData)
  }

  function uniqueKey(set: number, excerID: string){
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



    {#each exerciseData as block (uniqueKey(block.id,exName))}
      <SetBlock id={block.id} reps={block.reps} on:countChange={handleCountChange} />
    {/each}

    <ConfirmSelection on:confirm={handleSubmit} />

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