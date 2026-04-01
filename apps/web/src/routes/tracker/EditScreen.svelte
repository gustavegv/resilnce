<script lang="ts">
  import * as Alert from '$lib/components/alert/index.js';
  import { fade, slide } from 'svelte/transition';

  import {
    AddSessionExercises,
    EditSessionName,
    GetSessionExercises,
    RemoveSessionExercises,
    type ExerciseInfo,
  } from './dbFetches';
  import Icon from '@iconify/svelte';
  import { toast } from 'svelte-sonner';

  let {
    open = $bindable(false),
    newSessionName = $bindable(''),
    sessionToEditID = -1,
    sessionName = '',
    onConfirm = async () => {},
  }: {
    open?: boolean;
    newSessionName?: string;
    sessionToEditID?: number;
    sessionName?: string;
    onConfirm?: () => void | Promise<void>;
  } = $props();

  let exercises: ExerciseInfo[] = $state([]);
  let deletedExercises: ExerciseInfo[] = $state([]);
  let addedExercises: ExerciseInfo[] = $state([]);

  let exercisesLoading = $state(false);
  let exercisesError: string | null = $state(null);
  let oldSessionName: string = $state('');
  let loadedSessionID = $state(-1);
  let adding = $state(false);
  let savingEdit = $state(false);
  let addedExerciseName = $state('');
  let addedExerciseWeight = $state('');
  let addedExerciseSets = $state('');

  $effect(() => {
    if (!open || sessionToEditID <= 0) {
      exercises = [];
      exercisesLoading = false;
      exercisesError = null;
      loadedSessionID = -1;
      return;
    }

    if (loadedSessionID === sessionToEditID) {
      return;
    }

    loadedSessionID = sessionToEditID;
    oldSessionName = sessionName;
    newSessionName = sessionName;
    exercisesLoading = true;
    exercisesError = null;

    void init(sessionToEditID);
  });

  async function init(id: number) {
    try {
      exercises = await GetSessionExercises(id);
    } catch (err) {
      exercisesError = err instanceof Error ? err.message : 'Failed to load session.';
      console.error(err);
    } finally {
      exercisesLoading = false;
    }
  }

  let titleInput: HTMLInputElement;

  function focusTitle() {
    titleInput.focus();
  }

  function sameExercise(a: ExerciseInfo, b: ExerciseInfo): boolean {
    if (a.id != null && b.id != null) {
      return a.id === b.id;
    }

    return (
      a.name === b.name &&
      a.currentProgress.sets === b.currentProgress.sets &&
      a.currentProgress.weightPerSet[0] === b.currentProgress.weightPerSet[0]
    );
  }

  function handleDelete(exercise: ExerciseInfo) {
    if (!confirm(`Delete ${exercise.name}?`)) {
      return;
    }

    const addedIndex = addedExercises.findIndex((candidate) => sameExercise(candidate, exercise));

    if (addedIndex !== -1) {
      addedExercises.splice(addedIndex, 1);
    } else if (!deletedExercises.some((candidate) => sameExercise(candidate, exercise))) {
      deletedExercises.push(exercise);
    }

    const exerciseIndex = exercises.findIndex((candidate) => sameExercise(candidate, exercise));
    if (exerciseIndex !== -1) {
      exercises.splice(exerciseIndex, 1);
    }

    toast.success(`${exercise.name} removed.`);
  }

  function handleAdd() {
    const name = addedExerciseName.trim();
    const weight = Number(addedExerciseWeight);
    const sets = Number(addedExerciseSets);

    if (!name) {
      toast.error('Exercise name is required.');
      return;
    }

    if (!Number.isFinite(weight) || weight < 0) {
      toast.error('Weight must be a valid number.');
      return;
    }

    if (!Number.isInteger(sets) || sets <= 0) {
      toast.error('Sets must be a whole number greater than 0.');
      return;
    }

    // fixed constants
    const reps = 7;
    const repth = 12;
    const auto = 2.5;

    const rps = new Array(sets).fill(reps);
    const wps = new Array(sets).fill(weight);
    const cp = {
      sets: sets,
      repsPerSet: rps,
      weightPerSet: wps,
      restSeconds: 0,
    };
    const newExercise: ExerciseInfo = {
      name: name,
      currentProgress: cp,
      auto_increase: auto,
      rep_threshold: repth,
      finished: true, // only used to mark a session as newly added in this case
    };

    addedExercises.push(newExercise);
    // add to exercises to render on the screen
    exercises.push(newExercise);
    toast.success(`${name} added!`);
    addedExerciseName = '';
    addedExerciseWeight = '';
    addedExerciseSets = '';
    adding = false;
  }

  function checkChangesMade(): boolean {
    return (
      oldSessionName == newSessionName.trim() &&
      addedExercises.length == 0 &&
      deletedExercises.length == 0
    );
  }

  function clearEditScreen() {
    addedExercises = [];
    deletedExercises = [];
    exercises = [];
    adding = false;
    addedExerciseName = '';
    addedExerciseWeight = '';
    addedExerciseSets = '';
    exercisesLoading = false;
    exercisesError = null;
    loadedSessionID = -1;
  }

  function handleCancelEdit() {
    clearEditScreen();
    toast.info('Edit cancelled, no changes were made.');
  }

  async function handleConfirm(event: MouseEvent) {
    event.preventDefault();

    if (savingEdit || checkChangesMade()) {
      return;
    }

    savingEdit = true;

    try {
      if (!(await confirmEdit(addedExercises, deletedExercises))) {
        return;
      }

      await onConfirm();
      open = false;
      clearEditScreen();
    } finally {
      savingEdit = false;
    }
  }

  async function confirmEdit(
    exsToAdd: ExerciseInfo[],
    exsToRemove: ExerciseInfo[]
  ): Promise<boolean> {
    const trimmedSessionName = newSessionName.trim();

    if (!trimmedSessionName) {
      toast.error('New title cannot be empty');
      return false;
    }

    if (sessionToEditID == -1) {
      toast.error('Session to edit not found');
      return false;
    }

    if (exsToAdd.length && !(await AddSessionExercises(sessionToEditID, exsToAdd))) {
      toast.error('Could not add exercises. Try again later.', {
        duration: 5000,
        style: 'background: red;',
      });
      return false;
    }

    if (exsToRemove.length && !(await RemoveSessionExercises(sessionToEditID, exsToRemove))) {
      toast.error('Could not remove exercises. Try again later.', {
        duration: 5000,
        style: 'background: red;',
      });
      return false;
    }

    if (trimmedSessionName != oldSessionName) {
      if (!(await EditSessionName(sessionToEditID, trimmedSessionName))) {
        toast.error('Could not edit session name. Try again later.', {
          duration: 5000,
          style: 'background: red;',
        });
        return false;
      }
    }

    const completedChanges: string[] = [];
    if (trimmedSessionName != oldSessionName) {
      completedChanges.push(`renamed to ${trimmedSessionName}`);
    }
    if (exsToAdd.length) {
      completedChanges.push(`added ${exsToAdd.length} exercise${exsToAdd.length === 1 ? '' : 's'}`);
    }
    if (exsToRemove.length) {
      completedChanges.push(
        `removed ${exsToRemove.length} exercise${exsToRemove.length === 1 ? '' : 's'}`
      );
    }

    toast.success(
      completedChanges.length
        ? `Session updated: ${completedChanges.join(', ')}.`
        : 'No edits made.'
    );

    return true;
  }
</script>

<Alert.Root bind:open>
  <div in:fade={{ duration: 200 }}>
    <Alert.Content title="Edit session" class="border-border border">
      <Alert.Description>
        No changes are made until you press confirm. What do you want to change with <strong
          >{sessionName}</strong
        >?
      </Alert.Description>

      <div class="mt-5 flex flex-col gap-3">
        <div class="flex gap-4">
          <input
            bind:this={titleInput}
            bind:value={newSessionName}
            type={'text'}
            class="clean-input"
            class:unedited={oldSessionName == newSessionName.trim()}
          />
          {#if oldSessionName != newSessionName.trim()}
            <button
              type="button"
              class="title-input-icon"
              onclick={() => {
                newSessionName = oldSessionName;
              }}
            >
              <Icon icon="material-symbols:undo-rounded" width={24} />
            </button>
          {:else}
            <button type="button" class="title-input-icon off" onclick={() => focusTitle()}>
              <Icon icon="material-symbols:edit-outline-rounded" width={24} />
            </button>
          {/if}
        </div>

        <section class="exercise-overview">
          <p class="exercise-overview-title">Exercises in this session</p>

          {#if exercisesLoading}
            <p class="exercise-overview-copy">Loading exercises...</p>
          {:else if exercisesError}
            <p class="exercise-overview-copy exercise-overview-copy--error">{exercisesError}</p>
          {:else}
            <div class="exercise-list">
              {#each exercises as exercise, index}
                <div
                  class="exercise-row"
                  class:new-exercise={exercise.finished}
                  in:slide={{ duration: 700, axis: 'y', delay: 350 }}
                  out:slide={{ duration: 440, axis: 'y' }}
                >
                  <div>
                    <p class="exercise-order">Exercise {index + 1}</p>
                    <p class="exercise-name">{exercise.name}</p>
                  </div>
                  <p class="exercise-meta">
                    {exercise.currentProgress.sets} sets · {exercise.currentProgress
                      .weightPerSet[0] ?? 0}
                    kg
                  </p>
                </div>
                <button
                  class="flex justify-center"
                  in:fade={{ duration: 300, delay: 650 }}
                  out:fade={{ duration: 140 }}
                  onclick={() => handleDelete(exercise)}
                >
                  <Icon icon="material-symbols:delete-outline-rounded" width={30} color="gray" />
                </button>
              {/each}
            </div>
            {#if !exercises.length}
              <p
                class="exercise-overview-copy text-italic w-full text-center"
                in:fade={{ duration: 480, delay: 280 }}
                out:fade={{ duration: 140 }}
              >
                No exercises in this session.
              </p>
            {/if}
            {#if !adding}
              <button
                class="flex justify-center"
                in:fade={{ duration: 280, delay: 180 }}
                onclick={() => (adding = true)}
              >
                <Icon icon="material-symbols:add-circle-outline-rounded" width={30} color="grey" />
              </button>
            {:else}
              <div in:slide={{ duration: 280, axis: 'y' }} out:slide={{ duration: 280, axis: 'y' }}>
                <p class="exercise-overview-title">Add an exercise</p>

                <div class="exercise-row mt-2 flex-col">
                  <div>
                    <input
                      class="adding-input"
                      type="text"
                      placeholder="Exercise name"
                      bind:value={addedExerciseName}
                    />
                    <input
                      class="adding-input"
                      type="number"
                      placeholder="Weight"
                      min="0"
                      step="0.5"
                      bind:value={addedExerciseWeight}
                    />
                    <input
                      class="adding-input"
                      type="number"
                      placeholder="Sets"
                      min="1"
                      step="1"
                      bind:value={addedExerciseSets}
                    />
                  </div>

                  <div class="flex w-full justify-evenly">
                    <button class="adding-button cancel" onclick={() => (adding = false)}
                      >Cancel</button
                    >
                    <button class="adding-button" onclick={() => handleAdd()}>Add</button>
                  </div>
                </div>
              </div>
            {/if}
          {/if}
        </section>
        <div class="grid w-full grid-cols-[max-content_max-content] gap-2 gap-y-2">
          {#if addedExercises.length}
            <p
              class="rounded-md bg-green-300/10 px-2 text-sm text-green-300"
              in:fade={{ duration: 250 }}
            >
              +{addedExercises.length} Exercises added
            </p>
          {/if}
          {#if deletedExercises.length}
            <p
              class="rounded-md bg-red-400/10 px-2 text-sm text-red-400"
              in:fade={{ duration: 250 }}
            >
              -{deletedExercises.length} Exercises removed
            </p>
          {/if}
          {#if newSessionName.trim() != oldSessionName}
            <p
              class="w-fit rounded-md bg-neutral-300/10 px-2 text-sm text-neutral-300"
              in:fade={{ duration: 250 }}
            >
              Title edited to {newSessionName}
            </p>
          {/if}
          {#if !addedExercises.length && !deletedExercises.length && newSessionName.trim() == oldSessionName}
            <p
              class="w-fit rounded-md bg-neutral-300/10 px-2 text-sm text-neutral-300"
              in:fade={{ duration: 250 }}
            >
              No edits made
            </p>
          {/if}
        </div>

        <div class="mt-4 flex justify-end gap-3">
          <Alert.Cancel onclick={() => handleCancelEdit()}>Cancel</Alert.Cancel>
          <div class="w-full" class:inactive={checkChangesMade() || savingEdit}>
            <Alert.Action
              class="bg-accent"
              disabled={checkChangesMade() || savingEdit}
              onclick={handleConfirm}
            >
              {savingEdit ? 'Saving...' : 'Confirm edits'}
            </Alert.Action>
          </div>
        </div>
      </div>
    </Alert.Content>
  </div>
</Alert.Root>

<style>
  .exercise-overview {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 1rem;
    border-radius: 0.75rem;
    background: var(--card);
  }

  .clean-input {
    font-size: 1.5rem; /* Like an h1 */
    font-weight: bold;
    border: none;
    outline: none;
    background: transparent;
    width: 80%;
    font-family: inherit;
    margin: 1rem 0;
    border-radius: 10px;
    transition:
      color 0.18s ease,
      opacity 0.18s ease;
  }

  .clean-input.unedited {
    color: gray;
  }

  .exercise-overview-title {
    margin: 0;
    font-size: 0.8rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .exercise-list {
    display: grid;
    grid-template-columns: 6fr 1fr;
    align-items: center;
    gap: 0.5rem;
    max-height: 14rem;
    overflow-y: auto;
  }

  .exercise-row {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem 0.9rem;
    border-radius: 0.75rem;
    background: var(--surface-middle);
    border: 1px solid var(--border);
    transition:
      border-color 0.18s ease,
      background-color 0.18s ease,
      box-shadow 0.18s ease;
  }

  .exercise-row.new-exercise {
    border: 1px solid rgba(96, 255, 96, 0.243);
    background-color: color-mix(in srgb, var(--surface-middle), rgb(96, 255, 96) 5%);
  }

  .exercise-order,
  .exercise-name,
  .exercise-meta,
  .exercise-overview-copy {
    margin: 0;
  }

  .exercise-order {
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .exercise-name {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text);
  }

  .exercise-meta,
  .exercise-overview-copy {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .exercise-meta {
    text-align: right;
    white-space: nowrap;
  }

  .exercise-overview-copy--error {
    color: var(--destructive);
  }

  .title-input-icon {
    background-color: transparent;
    transition:
      opacity 0.18s ease,
      transform 0.18s ease;
  }

  .adding-input {
    background: var(--surface-middle);
    border: 1px solid var(--border);
    color: var(--text);
    padding: 10px 14px;
    border-radius: 8px;
    font-size: 14px;
    width: 220px;
    outline: none;
    transition: all 0.2s ease;
    margin-bottom: 0.5rem;
    width: 100%;
  }

  .adding-input::placeholder {
    color: var(--placeholder);
  }

  .adding-input:focus {
    border-color: var(--focus);
    box-shadow: 0 0 0 2px rgba(91, 156, 255, 0.15);
  }

  .adding-button {
    background-color: var(--card);
    padding: 0.2rem 0;
    font-size: 14px;
    width: 7rem;
    border-radius: 6px;
    opacity: 80%;
    transition:
      opacity 0.18s ease,
      transform 0.18s ease,
      background-color 0.18s ease;
  }

  .adding-button.cancel {
    background-color: var(--destructive);
  }

  .inactive {
    opacity: 0.3;
    pointer-events: none;
  }
</style>
