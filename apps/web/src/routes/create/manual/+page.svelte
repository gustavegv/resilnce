<script lang="ts">
  import '../../../app.css';
  import InputField from '../../../components/InputField.svelte';
  import SetAutoIncrease from '../../../components/SetAutoIncrease.svelte';
  import SortableList from '../SortableList.svelte';
  import { beforeNavigate, goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import { getMe } from '../../me';
  import { createSession, TrainingError } from '$lib/training/client';
  import type { CreateSessionCommand, DraftSessionItem } from '$lib/training/types';
  import type { ExerciseDataPackaged } from '../dbWrite';
  import Icon from '@iconify/svelte';

  let sessionName = $state(page.url.searchParams.get('title') ?? '');
  let newExName = $state('');
  let newExSetCount = $state('');
  let newExWeight = $state('');
  let newExAutoIncAmount = $state(2.5);
  let newExRepThreshold = $state(12);
  let sessionExercisesList: SortableList;
  let error = $state('');
  let saveState = $state<'idle' | 'saving' | 'failed' | 'saved'>('idle');
  let pending = $state.raw<CreateSessionCommand | null>(null);
  let commandAccount = '';
  let canEdit = $state(false);
  const quickloadKey = 'workoutPayload';

  onMount(async () => {
    if (!get(user) && !(await getMe())) {
      // eslint-disable-next-line svelte/no-navigation-without-resolve -- The root is resolved before appending the notification query.
      await goto(`${resolve('/')}?sonner=auth`);
      return;
    }
    if (page.url.searchParams.get('quickload') === '1') {
      try {
        const draft: ExerciseDataPackaged[] = JSON.parse(
          sessionStorage.getItem(quickloadKey) ?? '[]'
        );
        sessionExercisesList.fillFromArray(
          draft.map((pack) => ({
            draftId: crypto.randomUUID(),
            intent: {
              kind: 'createExercise',
              name: pack.name,
              muscleGroupCodes: [],
              configuration: {
                prescribedSets: pack.sets,
                initialWeight: String(pack.weight),
                repThreshold: pack.repThreshold ?? 12,
                autoIncrement: String(pack.autoIncrease ?? 2.5),
                restSeconds: 0,
              },
            },
          }))
        );
      } catch {
        error = 'Unable to read the draft. Add exercises manually.';
      }
    }
  });
  beforeNavigate(({ cancel }) => {
    if (pending) {
      cancel();
      error = 'Resolve this save with Retry before leaving.';
    }
  });
  function beforeUnload(event: BeforeUnloadEvent) {
    if (pending) {
      event.preventDefault();
      event.returnValue = '';
    }
  }

  function addExercise(): boolean {
    const count = Number(newExSetCount);
    if (
      !newExName.trim() ||
      [...newExName.trim()].length > 100 ||
      !Number.isInteger(count) ||
      count < 1 ||
      count > 20 ||
      !/^[0-9]+(?:\.[0-9]{1,2})?$/.test(newExWeight) ||
      Number(newExWeight) > 9999
    ) {
      error =
        'Use an exercise name, 1–20 sets, and a weight from 0–9999 with at most two decimal places.';
      return false;
    }
    if (sessionExercisesList.extractData().length >= 100) {
      error = 'A session can contain at most 100 exercises.';
      return false;
    }
    sessionExercisesList.pushItemToList({
      draftId: crypto.randomUUID(),
      intent: {
        kind: 'createExercise',
        name: newExName.trim(),
        muscleGroupCodes: [],
        configuration: {
          prescribedSets: count,
          initialWeight: newExWeight,
          repThreshold: newExRepThreshold,
          autoIncrement: String(newExAutoIncAmount),
          restSeconds: 0,
        },
      },
    });
    newExName = '';
    newExSetCount = '';
    newExWeight = '';
    error = '';
    return true;
  }
  function editDraft(item: DraftSessionItem): boolean {
    if (newExName || newExSetCount || newExWeight) {
      error = 'Add the current exercise before editing another one.';
      return false;
    }
    const intent = item.intent;
    newExName = intent.name;
    newExSetCount = String(intent.configuration.prescribedSets);
    newExWeight = intent.configuration.initialWeight;
    newExAutoIncAmount = Number(intent.configuration.autoIncrement ?? '2.50');
    newExRepThreshold = intent.configuration.repThreshold ?? 12;
    return true;
  }
  async function saveSession() {
    if (saveState === 'saving') return;
    if (!pending) {
      if ((newExName || newExSetCount || newExWeight) && !addExercise()) return;
      const items = sessionExercisesList.extractData().map((item) => item.intent);
      if (!sessionName.trim() || [...sessionName.trim()].length > 100 || !items.length) {
        error = 'Add a session name and at least one exercise.';
        return;
      }
      commandAccount = get(user)?.id ?? '';
      pending = {
        operationId: crypto.randomUUID(),
        name: sessionName.trim(),
        items: structuredClone(items),
      };
    }
    if (!commandAccount || commandAccount !== get(user)?.id) {
      error = 'The account changed. Return to the original account before retrying this save.';
      return;
    }
    saveState = 'saving';
    error = '';
    canEdit = false;
    try {
      await createSession(pending);
      pending = null;
      saveState = 'saved';
      sessionStorage.removeItem(quickloadKey);
      // eslint-disable-next-line svelte/no-navigation-without-resolve -- The root is resolved before appending the notification query.
      await goto(`${resolve('/')}?sonner=saved`);
    } catch (failure) {
      saveState = 'failed';
      error = failure instanceof Error ? failure.message : 'Unable to save the session.';
      canEdit = failure instanceof TrainingError && failure.status >= 400 && failure.status < 500;
    }
  }
  function resumeEditing() {
    pending = null;
    saveState = 'idle';
    error = '';
    canEdit = false;
  }
</script>

<svelte:window onbeforeunload={beforeUnload} />
<div class="container">
  <fieldset disabled={pending !== null}>
    <input
      aria-label="Session name"
      maxlength="100"
      bind:value={sessionName}
      placeholder="Untitled session"
      class="title"
    />
    <div class="add-box shadow">
      <h3 class="w-full pb-2 text-xl font-semibold">Add an exercise</h3>
      <InputField label="Exercise name" bind:value={newExName} type="text" />
      <InputField label="Sets" bind:value={newExSetCount} type="number" />
      <InputField label="Weight" bind:value={newExWeight} type="text" inputmode="decimal" />
      <SetAutoIncrease
        title="Auto-increase threshold"
        unit="reps"
        weight={newExRepThreshold}
        limit={1}
        interval={1}
        onCountChange={(count) => (newExRepThreshold = count)}
      />
      <SetAutoIncrease
        title="Weight increase intervals"
        weight={newExAutoIncAmount}
        onCountChange={(count) => (newExAutoIncAmount = count)}
      />
      <button class="add buttonClass minimal" onclick={addExercise}
        >Add to session <Icon icon="material-symbols:add" /></button
      >
    </div>
    <SortableList
      bind:this={sessionExercisesList}
      editData={editDraft}
      disabled={pending !== null}
    />
  </fieldset>
  {#if saveState !== 'idle'}<p role="status">
      {saveState === 'saving' ? 'Saving' : saveState === 'saved' ? 'Saved' : 'Not saved'}
    </p>{/if}
  {#if error}<p role="alert">{error}</p>{/if}
  {#if saveState === 'failed'}
    <button onclick={saveSession} class="buttonClass">Retry</button>
    {#if canEdit}<button onclick={resumeEditing} class="buttonClass">Edit draft</button>{/if}
  {:else}
    <button onclick={saveSession} disabled={saveState === 'saving'} class="finish buttonClass"
      >Finish and save session</button
    >
  {/if}
</div>

<style>
  fieldset {
    display: contents;
  }
  [role='alert'] {
    width: 80%;
    color: #ffb3b3;
  }
  .add {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    background-color: var(--color-secondary);
    width: 60%;
    padding: var(--spacing-xs);
    border-radius: var(--border-middle);
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
    margin: 1rem 0;
  }

  .add-box {
    display: flex;
    flex-direction: column;
    padding: var(--spacing-md);
    width: 80%;
    height: fit-content;
    background-color: var(--surface-low);
    border-radius: var(--border-outer);
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
    margin-top: 1rem;
    margin-bottom: 12rem;
    width: 80%;
  }
</style>
