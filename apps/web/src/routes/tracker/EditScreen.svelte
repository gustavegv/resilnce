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

  $effect(() => {
    if (!open || sessionToEditID <= 0) {
        exercises = [];
      exercisesLoading = false;
      exercisesError = null;
      return;
    }

    void init()

    });


    async function init(){
            console.log("init")

        try {
            exercises = await GetSessionExercises(sessionToEditID);
            console.log(exercises)
            } catch (err) {
            exercisesError = err instanceof Error ? err.message : 'Failed to load session.';
            console.error(err);
            } finally {
            exercisesLoading = false;
        }
  
    }
</script>

<Alert.Root bind:open>
  <Alert.Content title="Edit session" class="border-border border">
    <Alert.Description>
      What do you want to change with <strong>{sessionName}</strong>?
    </Alert.Description>

    <div class="mt-5 flex flex-col gap-3">
      <input bind:value={newSessionName} 
      type={'text'} class="clean-input"
      class:unedited={true}
      />

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
              <button class="flex justify-center">
                <Icon icon="material-symbols:delete-outline-rounded" width={30} color="gray"/>
              </button>
            {/each}
          </div>
          <button class="flex justify-center">
                <Icon icon="material-symbols:add-circle-outline-rounded" width={30}/>
              </button>
        {:else}
          <p class="exercise-overview-copy">No exercises found for this session.</p>
        {/if}
      </section>

      <div class="mt-5 flex justify-end gap-3">
        <Alert.Cancel>Cancel</Alert.Cancel>
        <Alert.Action class="bg-accent" onclick={() => onConfirm()}>Confirm edits</Alert.Action>
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
</style>
