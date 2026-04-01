<script lang="ts">
  import InputField from '../../components/InputField.svelte';
  import * as Alert from '$lib/components/alert/index.js';

  import { GetSessionExercises, type ExerciseInfo } from './dbFetches';
  import Icon from '@iconify/svelte';

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
  let exercisesLoading = $state(false);
  let exercisesError: string | null = $state(null);
  let oldSessionName: string = $state('');
  let loadedSessionID = $state(-1);
  let adding = $state(false)

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

  function handleDelete(exercise: ExerciseInfo){
    // todo
    confirm(`Delete ${exercise.name}?`)
  }

  function handleAdd(){
    // todo
  }

  function checkChangesMade():boolean{
    return oldSessionName == newSessionName
  }

</script>

<Alert.Root bind:open>
  <Alert.Content title="Edit session" class="border-border border">
    <Alert.Description>
      What do you want to change with <strong>{sessionName}</strong>?
    </Alert.Description>

    <div class="mt-5 flex flex-col gap-3">
      <div class="flex gap-4">
        <input
          bind:this={titleInput}
          bind:value={newSessionName}
          type={'text'}
          class="clean-input"
          class:unedited={oldSessionName == newSessionName}
        />
        {#if oldSessionName != newSessionName}
        <button
          type="button"
          class="title-input-icon"
          onclick={() => {
            newSessionName = oldSessionName;
          }}
        >
          <Icon icon="material-symbols:undo-rounded" width={24}/>
        </button> 
        {:else}
        <button
          type="button"
          class="title-input-icon off"
          onclick={() => focusTitle()}
        >
          <Icon icon="material-symbols:edit-outline-rounded" width={24}/>
        </button>
        {/if}
      </div>

      <section class="exercise-overview">
        <p class="exercise-overview-title">Exercises in this session</p>

        {#if exercisesLoading}
          <p class="exercise-overview-copy">Loading exercises...</p>
        {:else if exercisesError}
          <p class="exercise-overview-copy exercise-overview-copy--error">{exercisesError}</p>
        {:else if exercises.length}
          <div class="exercise-list">
            {#each exercises as exercise, index}
              <div class="exercise-row">
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
              <button class="flex justify-center" onclick={() => handleDelete(exercise)}>
                <Icon icon="material-symbols:delete-outline-rounded" width={30} color="gray" />
              </button>
            {/each}
          </div>
          {#if !adding}
          <button class="flex justify-center" onclick={() => adding = true}>
            <Icon icon="material-symbols:add-circle-outline-rounded" width={30} color="grey" />
          </button>
          {:else}
            <p class="exercise-overview-title">Add an exercise</p>
            
            <div class="exercise-row flex-col">
                <div>
                    <input class="adding-input" type="text" placeholder="Exercise name">
                    <input class="adding-input" type="number" placeholder="Weight">
                    <input class="adding-input" type="number" placeholder="Sets">
                </div>
                <div class="flex justify-evenly w-full">
                    <button class="adding-button cancel" onclick={() => adding = false}>Cancel</button>
                    <button class="adding-button" onclick={() => handleAdd()}>Add</button>
                </div>

            </div>
          {/if}
        {:else}
          <p class="exercise-overview-copy">No exercises found for this session.</p>
        {/if}
      </section>

      <div class="mt-5 flex justify-end gap-3">
        <Alert.Cancel>Cancel</Alert.Cancel>
        <div class="w-full" class:inactive={checkChangesMade()}>
            <Alert.Action class="bg-accent" onclick={() => onConfirm()}>Confirm edits</Alert.Action>
        </div>
      </div>
    </div>
  </Alert.Content>
</Alert.Root>

<style>
  .exercise-overview {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 1rem;
    border-radius: 0.75rem;
    background: var(--surface-low);
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
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem 0.9rem;
    border-radius: 0.75rem;
    background: var(--surface-middle);
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
    opacity: 80%
  }

  .adding-button.cancel {
    background-color: var(--destructive);
  }

  .inactive {
    opacity: 0.3;
    pointer-events: none;
  }

</style>
