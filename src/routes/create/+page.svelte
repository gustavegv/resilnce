<script lang="ts">
  import '../../app.css';
  import Icon from '@iconify/svelte';
  import type { ExInfoPackage } from '$lib/firebaseCreation';
  import { betterAdd } from '$lib/firebaseCreation';
  import InputField from '../../components/InputField.svelte';
  import { goto } from '$app/navigation';
  import SetBlock from '../../components/SetBlock.svelte';
  import SetAutoIncrease from '../../components/SetAutoIncrease.svelte';
  import { onDestroy, onMount } from 'svelte';

  type SessionInfo = {
    name: string;
    exercises: ExInfoPackage[];
  };

  const dummy1: ExInfoPackage = {
    name: 'Freelake',
    weight: 30,
    sets: 4,
  };

  let currentlyAdded: ExInfoPackage[] = $state([]);

  let seshName = $state('');
  let newName = $state('');
  let newSets: string = $state('');
  let newWeight: string = $state('');
  let newAutoInc: number = $state(2.5);

  function addExercise() {
    if (!newName || !newSets || !newWeight) {
      console.log('No set to add:', !newName, !newSets, !newWeight);
      return;
    }

    //todo add auto inc here
    console.log('autothing added:', newAutoInc);

    const entry: ExInfoPackage = {
      name: newName,
      sets: Number(newSets),
      weight: Number(newWeight),
      autoIncrease: newAutoInc,
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
  }

  function removeItem(index: number) {
    currentlyAdded.splice(index, 1);
  }

  function autoIncreaseChange(count: number) {
    newAutoInc = count;
  }
</script>

<div class="container">
  <input bind:value={seshName} placeholder="Untitled session" class="title" />

  <div class="add-box">
    <h3>Add an exercise</h3>
    <InputField label={'Exercise name'} bind:value={newName} type={'text'} />

    <InputField label={'Sets'} bind:value={newSets} type={'number'} />

    <InputField label={'Weight'} bind:value={newWeight} type={'number'} />

    <SetAutoIncrease weight={2.5} onCountChange={(count: number) => autoIncreaseChange(count)} />

    <button class="add buttonClass" onclick={addExercise}>+</button>
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
      <div class="abs-icon" onclick={() => removeItem(index)}>
        <Icon icon={'typcn:delete'} color={'red'} width="40"></Icon>
      </div>
    </div>
  {/each}

  <button onclick={saveSession} class="finish buttonClass">Finish and save session</button>
</div>

<style>
  .add {
    background-color: var(--color-alt);
    width: 92%;
  }
  .abs-icon {
    position: absolute; /* Positioned relative to .container */
    top: -12px;
    right: -15px;
    z-index: 1000; /* To sit on top of other things */
  }

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
    background-color: var(--color-secondary);
    border-radius: 15px;

    box-shadow: var(--shadow-dark);
    margin-bottom: 1rem;
  }

  .blob-cont {
    position: relative;
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
    background: var(--color-secondary);
    font-size: 20px;
    margin-bottom: 12rem;
  }

  body {
    overflow: hidden;
  }
</style>
