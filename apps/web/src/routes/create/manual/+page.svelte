<script lang="ts">
  import '../../../app.css';
  import Icon from '@iconify/svelte';
  import type { ExerciseInfo, ExInfoPackage } from '$lib/firebaseCreation';
  import { betterAdd } from '$lib/firebaseCreation';
  import InputField from '../../../components/InputField.svelte';
  import { goto } from '$app/navigation';
  import SetBlock from '../../../components/SetBlock.svelte';
  import SetAutoIncrease from '../../../components/SetAutoIncrease.svelte';
  import { onDestroy, onMount } from 'svelte';
  import { user } from '../../account/user';
  import { get } from 'svelte/store';
  import { base } from '$app/paths';
  import SortableList from '../SortableList.svelte';
  import { CreateSession } from '../dbWrite';

  type SessionInfo = {
    name: string;
    exercises: ExInfoPackage[];
  };

  const dummy1: ExInfoPackage = {
    name: 'Freelake',
    weight: 30,
    sets: 4,
  };
  const dummy2: ExInfoPackage = {
    name: 'Souvlaki',
    weight: 30,
    sets: 4,
  };

  let currentlyAdded: ExInfoPackage[] = $state([]);

  let seshName = $state('');
  let newName = $state('');
  let newSets: string = $state('');
  let newWeight: string = $state('');
  let newAutoInc: number = $state(2.5);
  let newRepThreshold: number = $state(12);

  onMount(async () => {
    const us = get(user);
    if (!us) {
      goto(`${base}/account`);
    }

    console.log('bru', us);
  });

  function addExercise() {
    if (!newName || !newSets || !newWeight) {
      console.log('No set to add:', !newName, !newSets, !newWeight);
      return;
    }

    //todo add auto inc here
    console.log('autothing added:', newAutoInc);

    // todo add serverside checking
    if (
      Number(newSets) > 20 ||
      Number(newWeight) > 9999 ||
      newName.length > 100 ||
      currentlyAdded.length > 100
    ) {
      alert('Max threshold met.');
      return;
    }

    const entry: ExInfoPackage = {
      name: newName,
      sets: Number(newSets),
      weight: Number(newWeight),
      autoIncrease: newAutoInc,
      repThreshold: newRepThreshold,
    };

    currentlyAdded = [...currentlyAdded, entry];

    console.log('NEW THANG');
    console.log(currentlyAdded);

    reorderableList.addToSortable(entry);

    newName = '';
    newSets = '';
    newWeight = '';
  }

  let reorderableList: SortableList;

  async function saveSession() {
    console.log('This is what we send pre');
    console.log(currentlyAdded);

    currentlyAdded = reorderableList.extractData();

    // adds inputed exercise in case you forgot
    addExercise();

    if (currentlyAdded.length == 0) {
      alert('No exercises added!');
      return;
    }

    if (seshName == '') {
      alert('No session name added!');
      return;
    }

    console.log('This is what we send post');
    // console.log(currentlyAdded);

    const username = get(user); // todo: är det här den gamla user logiken?
    if (username || true) {
      // todo: ta bort or true
      const okResponse = await CreateSession(seshName, currentlyAdded);
      if (okResponse) {
        alert('Session saved succesfully!');
        goto(`${base}/`); // todo goto deprecated?
      } else {
        alert('Error saving');
      }
      // betterAdd(username, seshName, currentlyAdded);
    } else {
      alert('Problem with log-in authentication');
    }
  }

  function removeItem(index: number) {
    currentlyAdded.splice(index, 1);
  }

  function repThresholdChange(count: number) {
    newRepThreshold = count;
  }

  function autoIncreaseChange(count: number) {
    newAutoInc = count;
  }

  function getNames(exs: ExInfoPackage[]): string[] {
    let newArray: string[] = exs.map((ex) => ex.name);
    return newArray;
  }
</script>

<div class="container">
  <input maxlength="50" bind:value={seshName} placeholder="Untitled session" class="title" />

  <div class="add-box shadow">
    <h3 class="w-full pb-2 text-xl font-semibold">Add an exercise</h3>
    <InputField label={'Exercise name'} bind:value={newName} type={'text'} />

    <InputField label={'Sets'} bind:value={newSets} type={'number'} />

    <InputField label={'Weight'} bind:value={newWeight} type={'number'} />

    <SetAutoIncrease
      title={'Auto-increase threshold'}
      unit={'reps'}
      weight={12}
      limit={2}
      interval={1}
      onCountChange={(count: number) => repThresholdChange(count)}
    />
    <SetAutoIncrease
      title={'Weight increase intervals'}
      weight={2.5}
      onCountChange={(count: number) => autoIncreaseChange(count)}
    />

    <button class="add buttonClass" onclick={addExercise}>Add to session</button>
  </div>

  <SortableList bind:this={reorderableList} items={currentlyAdded} />

  <button onclick={saveSession} class="finish buttonClass">Finish and save session</button>
</div>

<style>
  .add {
    background-color: var(--color-secondary);
    width: 60%;
    padding: 0.5rem;
    font-weight: 400;
    font-size: 16px;
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
    width: 80%;
    height: fit-content;
    background-color: var(--color-background);
    border-radius: 15px;
    align-items: center;

    box-shadow: var(--shadow-dark);
    margin-bottom: 1rem;
  }

  .finish {
    height: 5rem;
    box-shadow: var(--shadow-dark);
    background: var(--color-secondary);
    font-size: 20px;
    margin-bottom: 12rem;
    width: 80%;
  }
</style>
