<script lang="ts">
  import '../../../app.css';
  import type { ExerciseDataPackaged } from '../dbWrite';
  import InputField from '../../../components/InputField.svelte';
  import { goto } from '$app/navigation';
  import SetAutoIncrease from '../../../components/SetAutoIncrease.svelte';
  import { onMount } from 'svelte';
  import { user } from '$lib/stores/appState';

  import { get } from 'svelte/store';
  import { base, resolve } from '$app/paths';
  import SortableList from '../SortableList.svelte';
  import { CreateSession } from '../dbWrite';

  let sessionName = $state('');
  let newExName = $state('');
  let newExSetCount: string = $state('');
  let newExWeight: string = $state('');
  let newExAutoIncAmount: number = $state(2.5);
  let newExRepThreshold: number = $state(12);

  let sessionExercisesList: SortableList;

  onMount(async () => {
    const value = get(user);
    if (!value?.name) {
      goto(resolve(`/`));
    }
  });
  /*  
      ################
      HELPER FUNCTIONS
      ################ 
  */
  function checkIfInputFieldsFilled(): boolean {
    if (!newExName || !newExSetCount || !newExWeight) {
      console.error(
        'One or multiple input fields not filled:',
        !newExName,
        !newExSetCount,
        !newExWeight
      );
      return false;
    }
    return true;
  }

  function checkInputLengthExceedsMax(): boolean {
    // todo add a serverside check too
    const maxSetsAllowed = 20;
    const maxWeightAllowed = 9999;
    const maxExerciseNameAllowed = 100;
    const maxExercisesAllowed = 100;

    if (
      Number(newExSetCount) > maxSetsAllowed ||
      Number(newExWeight) > maxWeightAllowed ||
      newExName.length > maxExerciseNameAllowed ||
      sessionExercisesList.extractData().length > maxExercisesAllowed
    ) {
      console.error('One or multiple input fields exceed max limits');
      return false;
    }
    return true;
  }

  function vaidateInputFields(): boolean {
    if (!checkIfInputFieldsFilled() || !checkInputLengthExceedsMax()) {
      return false;
    }

    return true;
  }

  function packageExerciseEntry(): ExerciseDataPackaged {
    const entry: ExerciseDataPackaged = {
      name: newExName,
      sets: Number(newExSetCount),
      weight: Number(newExWeight),
      autoIncrease: newExAutoIncAmount,
      repThreshold: newExRepThreshold,
    };
    return entry;
  }

  function resetInputFields() {
    newExName = '';
    newExSetCount = '';
    newExWeight = '';
  }

  function validateSessionData(addedExercisesList: ExerciseDataPackaged[]): boolean {
    if (addedExercisesList.length == 0) {
      alert('No exercises added!');
      return false;
    }

    if (sessionName == '') {
      alert('No session name added!');
      return false;
    }
    return true;
  }

  function clientSideAuthorizationCheck(): boolean {
    const userStatus = get(user);
    if (!userStatus?.name) {
      console.error('Not logged in. Please update authorization store.');
      return false;
    }
    return true;
  }

  /*  
      ################
      MAIN   FUNCTIONS  
      ################ 
  */
  function addExercise() {
    if (!vaidateInputFields()) {
      alert('Input(s) empty or exceeds max limit.');
      return;
    }

    const newEntry = packageExerciseEntry();
    sessionExercisesList.pushItemToList(newEntry);

    resetInputFields();
  }

  async function saveSession() {
    let addedExercisesList: ExerciseDataPackaged[] = sessionExercisesList.extractData();

    // if input field filled but not pushed, adds this
    if (vaidateInputFields()) {
      addExercise();
    }

    if (!validateSessionData(addedExercisesList)) {
      return;
    }

    if (!clientSideAuthorizationCheck()) {
      return;
    }

    const responseStatus: boolean = await CreateSession(sessionName, addedExercisesList);
    if (responseStatus) {
      alert('Session saved succesfully!'); // add sonner after redirect instead of alert
      goto(resolve(`/`));
    } else {
      alert('Error saving');
    }
  }

  function repThresholdChange(count: number) {
    newExRepThreshold = count;
  }

  function autoIncreaseChange(count: number) {
    newExAutoIncAmount = count;
  }
</script>

<div class="container">
  <input maxlength="50" bind:value={sessionName} placeholder="Untitled session" class="title" />

  <div class="add-box shadow">
    <h3 class="w-full pb-2 text-xl font-semibold">Add an exercise</h3>
    <InputField label={'Exercise name'} bind:value={newExName} type={'text'} />

    <InputField label={'Sets'} bind:value={newExSetCount} type={'number'} />

    <InputField label={'Weight'} bind:value={newExWeight} type={'number'} />

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

  <SortableList bind:this={sessionExercisesList} />

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
