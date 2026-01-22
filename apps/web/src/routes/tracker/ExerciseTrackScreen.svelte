<script lang="ts">
  import SetBlock from '../../components/SetBlock.svelte';
  import ConfirmSelection from '../../components/ConfirmSelection.svelte';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';

  interface EditData {
    user: string;
    sesID: string;
    exID: string;
    oldName: string;
    newName?: string;
    newW?: number;
    addedSets: number;
  }

  async function editExercise(data: EditData) {
    // WIP implementation
  }

  let {
    name = $bindable(),
    weight = $bindable(),
    reps,
    finished,
    exIndex,
    onCount,
    onSubmit,
    edit,
    sesID,
    onCancel,
    exID,
  }: {
    name: string;
    weight: number;
    reps: number[];
    finished: boolean;
    exIndex: number;
    onCount: ({ id, count }: { id: number; count: number }) => void;
    onSubmit: () => void;
    edit?: boolean;
    sesID: string;
    onCancel?: () => void;
    exID?: string;
  } = $props();

  function uniqueKey(set: number, excerID: number) {
    return `${excerID}-s${set}`;
  }

  // todo: Finishe edit-logic
  async function onFinishedEditing() {
    const userID = get(user);
    if (!userID?.name) {
      alert('No user signed in');
      return;
    }
    let data: EditData = {
      user: userID.name,
      sesID: sesID,
      exID: exID ?? '',

      oldName: name,
      newName: newName ?? undefined,
      newW: newWeight ?? undefined,
      addedSets: addedSetsCount,
    };

    console.log('Done. Added:', data);

    await editExercise(data);

    if (onCancel) {
      onCancel();
    }
  }

  function addSet() {
    addedSetsCount++;
    reps = [...reps, 7];
  }

  let newName: string = $state('');
  let newWeight: number = $state(-1);
  let addedSetsCount: number = $state(0);
</script>

{#if edit}
  <div class="edit-mother">
    <div class="edit-pill">
      <smp
        onkeydown={(e: KeyboardEvent) => {
          if (e.key === 'Enter' || e.key === ' ') {
          }
        }}
        role="button"
        tabindex="0"
        onclick={onCancel}>Cancel</smp
      >

      <h1>Edit mode</h1>
      <smp
        onkeydown={(e: KeyboardEvent) => {
          if (e.key === 'Enter' || e.key === ' ') {
          }
        }}
        role="button"
        tabindex="0"
        onclick={onFinishedEditing}>Done</smp
      >
    </div>
  </div>

  <header>
    <input bind:value={newName} placeholder={name + ' (tap to edit)'} class="title" />
    <input
      bind:value={newWeight}
      placeholder={weight + ' kg (tap to edit)'}
      class="title"
      type={'number'}
    />
  </header>
{:else}
  <header>
    <h1 class="text-3xl font-bold">{name}</h1>
    <h2 class="text-2xl">{weight} kg</h2>
  </header>
{/if}

<cont class="cont">
  {#each reps as block, index (uniqueKey(index, exIndex))}
    <SetBlock id={index + 1} {finished} reps={block} onCountChange={onCount} />
  {/each}

  {#if edit}
    <button class="buttonClass" onclick={addSet}><h1>Add set</h1></button>
  {:else}
    <ConfirmSelection {finished} onConfirm={onSubmit} />
  {/if}
</cont>

<style>
  .cont {
    width: 100%;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    align-items: center;
    overflow-y: auto;
    overflow-x: hidden;
    touch-action: pan-y;
  }

  .title {
    font-size: 2em;
    font-weight: bold;
    border: none;
    outline: none;
    background: transparent;
    width: 80%;
    font-family: inherit;
    color: var(--color-secondary);
    margin: 1rem 0;
    text-align: center;
  }

  .edit-mother {
    position: absolute;
    top: 0;
    background-color: blue;
    width: 100%;
    z-index: 100;
    display: flex;

    justify-content: center;
  }

  .edit-pill {
    background-color: purple;
    width: 90%;
    border-radius: 100rem;
    display: flex;
    flex-direction: row;
    justify-content: space-evenly;
    align-items: center;
  }

  smp {
    background-color: red;
    padding: 0.5rem 1rem;
    border-radius: 100rem;
  }
</style>
