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
  import { page } from '$app/state';
  import { toast } from 'svelte-sonner';
  import { getMe } from '../../me';

  let sessionName = $state('');
  let newExName = $state('');
  let newExSetCount: string = $state('');
  let newExWeight: string = $state('');
  let newExAutoIncAmount: number = $state(2.5);
  let newExRepThreshold: number = $state(12);

  let sessionExercisesList: SortableList;

  let exercises = $state<ExerciseDataPackaged[]>([]);
  let hasFlag = $derived(page.url.searchParams.get('quickload') === '1');
  const QUICKLOAD_STORAGE_KEY = 'workoutPayload';

  onMount(async () => {
    if (await !checkUserStatus()) {
      goHomeWithSonner('auth');
    }
    if (hasFlag) {
      const raw = sessionStorage.getItem(QUICKLOAD_STORAGE_KEY);
      exercises = raw ? JSON.parse(raw) : [];

      sessionExercisesList.fillFromArray(exercises);
      toast.success(`Quick filled session!`, { duration: 3000 });
    }
  });
  /*  
      ################
      HELPER FUNCTIONS
      ################ 
  */

  function removeQuickLoadFromStore() {
    sessionStorage.removeItem(QUICKLOAD_STORAGE_KEY); // ta bort efter användning
  }

  function goHomeWithSonner(value: string) {
    console.log('Going home...');
    goto(`${resolve('/')}?sonner=${value}`);
  }

  async function checkUserStatus(): Promise<boolean> {
    const value = get(user);
    if (value?.name || (await getMe())) {
      return true;
    } else {
      console.error('User not logged in.');
      return false;
    }
  }

  function checkIfInputFieldsFilled(safetyCheck: boolean): boolean {
    if (!newExName || !newExSetCount || !newExWeight) {
      if (!safetyCheck) {
        console.error(
          'One or multiple input fields not filled:',
          !newExName,
          !newExSetCount,
          !newExWeight
        );
        toast.error('One or multiple input fields not filled');
      }
      return false;
    }
    return true;
  }

  function checkInputLengthExceedsMax(safetyCheck: boolean): boolean {
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
      if (safetyCheck) {
        return false;
      }
      console.error('One or multiple input fields exceed max limits');
      toast.error('One or multiple input fields exceed max limits');
      return false;
    }
    return true;
  }

  function isInputFieldsFilledCorrectly(safetyCheck: boolean): boolean {
    if (!checkIfInputFieldsFilled(safetyCheck) || !checkInputLengthExceedsMax(safetyCheck)) {
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
      toast.error('No exercises added!');
      return false;
    }

    if (sessionName == '') {
      toast.error('No session name added!');
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
    if (!isInputFieldsFilledCorrectly(false)) {
      return;
    }

    const newEntry = packageExerciseEntry();
    sessionExercisesList.pushItemToList(newEntry);

    resetInputFields();
  }

  async function saveSession() {
    let addedExercisesList: ExerciseDataPackaged[] = sessionExercisesList.extractData();

    // if input field filled but not pushed, adds this
    if (isInputFieldsFilledCorrectly(true)) {
      addExercise();
    }

    if (!validateSessionData(addedExercisesList)) {
      return;
    }

    if (await !checkUserStatus()) {
      return;
    }

    const responseStatus: boolean = await CreateSession(sessionName, addedExercisesList);
    if (responseStatus) {
      removeQuickLoadFromStore();
      goHomeWithSonner('saved');
    } else {
      toast.error('Error saving, please try again.');
    }
  }

  function repThresholdChange(count: number) {
    newExRepThreshold = count;
  }

  function autoIncreaseChange(count: number) {
    newExAutoIncAmount = count;
  }

  let addBoxEl: HTMLElement | null = null;

  export function bop() {
    if (!addBoxEl) {
      return;
    }

    addBoxEl.classList.remove('bop');
    void addBoxEl.offsetWidth;
    addBoxEl.classList.add('bop');

    const onEnd = (ev: AnimationEvent) => {
      if (ev.animationName !== 'addBoxBop') return;
      addBoxEl?.classList.remove('bop');
      addBoxEl?.removeEventListener('animationend', onEnd);
    };

    addBoxEl.addEventListener('animationend', onEnd);
  }

  function fillInputFieldFromPackage(pack: ExerciseDataPackaged) {
    newExName = pack.name;
    newExSetCount = String(pack.sets);
    newExWeight = String(pack.weight);
    newExAutoIncAmount = pack.autoIncrease ?? 2.5;
    newExRepThreshold = pack.repThreshold ?? 12;
  }

  function useInputFieldForEditing(pack: ExerciseDataPackaged) {
    console.log('confirmed with', pack);
    if (isInputFieldsFilledCorrectly(true)) {
      toast.warning('Input fields filled but not yet added.');
      let co = confirm(
        'Input fields contain an exercise which has not yet been added. Proceed with edit and overwrite input?'
      );
      if (!co) return;
    }

    fillInputFieldFromPackage(pack);
    bop();
    toast.success('Exercise sent back to input field');
  }
</script>

<div class="container">
  <input maxlength="50" bind:value={sessionName} placeholder="Untitled session" class="title" />

  <div class="add-box shadow" bind:this={addBoxEl}>
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

  <SortableList bind:this={sessionExercisesList} editData={useInputFieldForEditing} />

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

    transform: translateZ(0);
    will-change: transform, filter, box-shadow;
  }

  :global(.add-box.bop) {
    animation: addBoxBop 750ms cubic-bezier(0.4, 0, 0.2, 1);
  }

  @keyframes addBoxBop {
    0% {
      transform: scale(1);
      filter: brightness(1);
      box-shadow: var(--shadow-dark);
    }

    15% {
      transform: scale(1.025);
      filter: brightness(1.12);
      box-shadow:
        var(--shadow-dark),
        0 0 0 3px rgba(109, 109, 109, 0.22),
        0 10px 28px rgba(0, 0, 0, 0.18);
    }

    100% {
      transform: scale(1);
      filter: brightness(1);
      box-shadow: var(--shadow-dark);
    }
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
