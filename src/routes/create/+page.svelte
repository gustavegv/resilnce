<script lang="ts">
  import '../../app.css';

  import type { ExInfoPackage } from '$lib/firebaseDataHandler';
  import { betterAdd } from '$lib/firebaseCreation';
  import InputField from '../../components/InputField.svelte';
  import { goto } from '$app/navigation';

  type SessionInfo = {
    name: string;
    exercises: ExInfoPackage[];
  };

  const dummy1: ExInfoPackage = {
    name: 'lopos',
    weight: 30,
    sets: 4,
  };

  let currentlyAdded: ExInfoPackage[] = $state([]);

  let seshName = $state('');
  let newName = $state('');
  let newSets: string = $state('');
  let newWeight: string = $state('');

  function addExercise() {
    if (!newName || !newSets || !newWeight) {
      console.log('No set to add:', !newName, !newSets, !newWeight);
      return;
    }

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
    console.log(s);
    betterAdd(seshName, currentlyAdded);
    //addNewSession(s);

    alert('session saved succesfully!');
    goto('/');
    // todo: go back to main page
  }
</script>

<div class="container">
  <input bind:value={seshName} placeholder="Untitled session" class="title" />

  <div class="add-box">
    <h3>Add an exercise</h3>
    <InputField label={'Exercise name'} bind:value={newName} type={'text'} />

    <InputField label={'Sets'} bind:value={newSets} type={'number'} />

    <InputField label={'Weight'} bind:value={newWeight} type={'number'} />
    <button onclick={addExercise}>+</button>
  </div>

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

  <button onclick={saveSession} class="finish">Finish and save session</button>
</div>

<style>
  .container {
    display: flex;
    flex-direction: column;
    align-items: center;
    overflow: scroll;
    background: var(--gradient-prim);
    height: 100%;
    box-sizing: border-box;
    padding: 5rem 0;
  }

  .title {
    font-size: 2em; /* Like an h1 */
    font-weight: bold;
    border: none;
    outline: none;
    background: transparent;
    width: 80%;
    font-family: inherit;
    color: white;
    margin: 1rem 0;
  }

  .add-box {
    display: flex;
    flex-direction: column;
    padding: 1rem;
    width: 70%;
    height: fit-content;
    background-color: var(--color-background);
    border-radius: 15px;

    box-shadow: var(--shadow-dark);
  }

  .blob-cont {
    display: flex;
    box-sizing: border-box;
    padding: 0 2rem;
    flex-direction: row;
    background-color: var(--color-secondary);
    justify-content: space-between;
    align-items: center;
    width: 80%;
    border-radius: 15px;
    margin: 5px;

    box-shadow: var(--shadow-dark);
  }

  .blob-inner {
    color: gray;
  }

  .finish {
    -webkit-text-stroke: 0.7px rgb(255, 255, 255);
    height: 5rem;
    box-shadow: var(--shadow-dark);
    background: var(--gradient-alt);
    font-size: 20px;
    margin-bottom: 12rem;
  }

  body {
    overflow: hidden;
  }
</style>
