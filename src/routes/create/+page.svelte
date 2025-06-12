<script lang="ts">
  import '../../app.css';
  import { addNewSession } from '$lib/firebaseDataHandler';

  import type { ExInfoPackage } from '$lib/firebaseDataHandler';
  import { betterAdd } from '$lib/firebaseCreation';

  type SessionInfo = {
    name: string;
    exercises: ExInfoPackage[];
  };

  const dummy1: ExInfoPackage = {
    name: 'lopos',
    weight: 30,
    sets: 4,
  };

  let currentlyAdded: ExInfoPackage[] = $state([dummy1]);

  let seshName = $state('');
  let newName = $state('');
  let newSets: string = $state('');
  let newWeight: string = $state('');

  function addExercise() {
    if (!newName || !newSets || !newWeight) return;

    const entry: ExInfoPackage = {
      name: newName,
      sets: Number(newSets),
      weight: Number(newWeight),
    };

    currentlyAdded = [...currentlyAdded, entry];

    newName = '';
    newSets = '';
    newWeight = '';
  }

  function saveSession() {
    // adds inputed exercise in case you forgot
    addExercise();

    const s: SessionInfo = {
      name: seshName,
      exercises: currentlyAdded,
    };

    betterAdd(seshName, currentlyAdded);
    //addNewSession(s);

    alert('session saved succesfully!');
    // todo: go back to main page
  }
</script>

<div class="container">
  <h1>Session name</h1>
  <input bind:value={seshName} placeholder="e.g. Leg day" />

  <div class="add-box">
    <h3>Add exercise</h3>
    <h4>Exercise name</h4>
    <input bind:value={newName} placeholder="e.g. Squat" />
    <h4>Number of sets</h4>
    <input type="number" bind:value={newSets} placeholder="e.g. 4" min="1" />
    <h4>Starting weigh</h4>
    <input type="number" bind:value={newWeight} placeholder="e.g. 60" min="0" />
    <button onclick={addExercise}>+</button>
  </div>

  <button onclick={saveSession}>Finish and save exercise</button>

  {#each currentlyAdded as blob, index}
    <div class="blob-cont">
      <div>
        <p>{index + 1}.</p>
        <h3>{blob.name}</h3>
      </div>
      <div class="blob-inner">
        <p>{blob.sets} sets</p>
        <p>{blob.weight} kg</p>
      </div>
    </div>
  {/each}
</div>

<style>
  .container {
    display: flex;
    flex-direction: column;
    align-items: center;
    overflow: scroll;
  }

  .add-box {
    display: flex;
    flex-direction: column;

    width: 70%;
    height: fit-content;
    background-color: var(--color-gray);
  }

  .blob-cont {
    display: flex;
    box-sizing: border-box;
    padding: 5px;
    flex-direction: row;
    background-color: antiquewhite;
    justify-content: space-between;
    align-items: center;
    width: 70%;
    margin: 5px;
  }

  .blob-inner {
    color: gray;
  }
</style>
