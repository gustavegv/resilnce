<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  import '../../app.css';
  import { goto } from '$app/navigation';
  import Icon from '@iconify/svelte';
  import InputField from '../../components/InputField.svelte';
  import ExerciseTrackScreen from './ExerciseTrackScreen.svelte';
  import LoadingSkeleton from './LoadingSkeleton.svelte';
  import * as Alert from '$lib/components/alert/index.js';

  import ExerciseCompleteScreen from '../../components/ExerciseCompleteScreen.svelte';
  import { resolve } from '$app/paths';
  import { cubicOut } from 'svelte/easing';
  import { toast } from 'svelte-sonner';

  import {
    CompleteSession,
    GetSessionExercises,
    SendEdit,
    SendUpdate,
    SetActiveSession,
    type ExerciseInfo,
  } from './dbFetches';
  import {
    applyExerciseEdit,
    buildExerciseEdit,
    createEditFieldsFromExercise,
    createEmptyEditFields,
    hasEditChanges,
    validateExerciseEdit,
  } from './edit';
  import { areAllExercisesFinished, buildProgressUpdate } from './saving';

  // Exercise ID som blir tilldelad när man callar komponenten:
  //   Exempel:  <Track sesID="push" />
  let { sesID }: { sesID: string } = $props();

  // State variabler
  let exercises: ExerciseInfo[] = $state([]);

  let currentExerciseIndex = $state(0);
  let loading: boolean = $state(true);

  let error: string | null = $state(null);
  let allFinished: boolean = $state(false);

  let editMode = $state(false);
  let editFields = $state(createEmptyEditFields());
  let editSubmitting = $state(false);
  // Derived variabler

  type ComparableField = 'name' | 'sets' | 'weight' | 'reps' | 'auto';

  const comparableFieldMap = {
    name: 'name',
    sets: 'sets',
    weight: 'weight',
    reps: 'repThreshold',
    auto: 'autoIncrease',
  } as const;

  const currentExercise = $derived(exercises[currentExerciseIndex]);
  const prevExists = $derived(currentExerciseIndex > 0);
  const nextExists = $derived(currentExerciseIndex < exercises.length - 1);
  const nextUnfinishedIndex = $derived(exercises.findIndex((ex) => !ex.finished));

  const finished: boolean = $derived(currentExercise?.finished ?? false);

  onMount(() => {
    void initializeSession();
  });

  async function initializeSession() {
    loading = true;
    error = null;

    const sessionId = Number(sesID);
    if (!Number.isInteger(sessionId) || sessionId <= 0) {
      error = 'Invalid session id.';
      loading = false;
      return;
    }

    try {
      exercises = await GetSessionExercises(sessionId);
      await SetActiveSession(String(sessionId));
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load session.';
      console.error(err);
    } finally {
      loading = false;
    }
  }
  //Functions

  function handleRepCountIncrementation({ id, count }: { id: number; count: number }) {
    if (currentExercise) {
      currentExercise.currentProgress.repsPerSet[id - 1] = count;
    }
  }

  function goToExercise(index: number) {
    if (index < 0 || index >= exercises.length) return;
    currentExerciseIndex = index;
  }

  function loadNextExercise() {
    goToExercise(currentExerciseIndex + 1);
  }

  function prevExercise() {
    goToExercise(currentExerciseIndex - 1);
  }

  function skipExercise() {
    loadNextExercise();
  }

  let submitting = $state(false);

  async function submitExercise() {
    if (submitting || !currentExercise) return;

    const update = buildProgressUpdate(currentExercise);
    if (!update) {
      error = 'Could not prepare exercise update.';
      return;
    }

    submitting = true;
    error = null;

    try {
      const success = await SendUpdate(update, sesID);
      if (!success) {
        throw new Error('Failed to save progress.');
      }

      exercises[currentExerciseIndex].finished = true;

      if (areAllExercisesFinished(exercises)) {
        allFinished = true;
        await CompleteSession(sesID);
        return;
      }

      loadNextExercise();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to save progress.';
      console.error(err);
    } finally {
      submitting = false;
    }
  }

  async function quitSession() {
    await CompleteSession(sesID);
    goto(resolve('/'));
  }

  function getProgressRatio(): number {
    if (allFinished) {
      return 100;
    }
    const ratio: number = currentExerciseIndex / exercises.length;
    return 100 * ratio;
  }

  function getProgressFraction(): string {
    if (allFinished) {
      return exercises.length + ' of ' + exercises.length;
    }

    return currentExerciseIndex + 1 + ' of ' + exercises.length;
  }

  let showActionMenu: boolean = $state(false);

  function growDown(node: Element, { duration = 180 } = {}) {
    return {
      duration,
      easing: cubicOut,
      css: (t: number) => `
      opacity: ${t};
      transform: scaleY(${0.72 + 0.28 * t}) scaleX(${0.94 + 0.06 * t});
    `,
    };
  }

  function toggleActionMenu() {
    showActionMenu = !showActionMenu;
  }

  function closeActionMenu() {
    showActionMenu = false;
  }

  function handleWindowKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && showActionMenu) {
      closeActionMenu();
    }
  }

  let alertVisible = $state(false);

  async function handleQuitAction() {
    closeActionMenu();
    alertVisible = true;
  }

  function handleEditAction() {
    closeActionMenu();
    resetEditFields();
    editMode = true;
  }

  async function finishSessionNow() {
    closeActionMenu();

    if (confirm('Finish the session now?\nYou will go to the session summary.')) {
    }
  }

  function getNextUnfinishedIndex(): number {
    return exercises.findIndex((ex) => !ex.finished);
  }

  function checkNextUnfinished(): boolean {
    return nextUnfinishedIndex !== -1 && exercises[nextUnfinishedIndex].id === currentExercise?.id;
  }

  function gotoNextUnfinished() {
    const index = getNextUnfinishedIndex();
    if (index !== -1) {
      currentExerciseIndex = index;
    }
  }

  function resetEditFields() {
    editFields = createEditFieldsFromExercise(currentExercise);
  }

  function compareFields(field: ComparableField): boolean {
    const editField = comparableFieldMap[field];
    const currentValues = createEditFieldsFromExercise(currentExercise);
    return editFields[editField] == currentValues[editField];
  }

  function revertEditField(field: ComparableField) {
    const editField = comparableFieldMap[field];
    const currentValues = createEditFieldsFromExercise(currentExercise);
    editFields[editField] = currentValues[editField];
  }

  function cancelEditMode() {
    resetEditFields();
    editMode = false;
  }

  async function submitExerciseEdit() {
    if (editSubmitting) {
      return;
    }

    const edit = buildExerciseEdit(currentExercise, editFields);
    if (!edit) {
      toast.error('Could not prepare exercise edit.');
      return;
    }

    if (!hasEditChanges(edit)) {
      toast.error('Enter at least one field to edit.');
      return;
    }

    const validationError = validateExerciseEdit(edit);
    if (validationError) {
      toast.error(validationError);
      return;
    }

    editSubmitting = true;
    error = null;

    try {
      const success = await SendEdit(edit, sesID);
      if (!success) {
        throw new Error('Failed to save edit.');
      }

      applyExerciseEdit(currentExercise, edit);
      cancelEditMode();
      toast.success('Exercise updated.');
    } catch (err) {
      let errorMsg = err instanceof Error ? err.message : 'Failed to save edit.';
      console.error(err);
      toast.error(errorMsg);
    } finally {
      editSubmitting = false;
    }
  }
</script>

<Alert.Root bind:open={alertVisible}>
  <Alert.Content title="Quit session?" class="border-border border">
    <Alert.Description>All completed exercise are already saved.</Alert.Description>

    <div class="mt-5 flex justify-evenly gap-3">
      <Alert.Cancel>No, take me back</Alert.Cancel>
      <Alert.Action class="bg-destructive text-white" onclick={() => quitSession()}>
        Quit session.
      </Alert.Action>
    </div>
  </Alert.Content>
</Alert.Root>

<svelte:window onkeydown={handleWindowKeydown} />

<main class="app-container">
  {#if loading || error}
    {#if error}
      <div class="mt-4 rounded-xl border border-red-500/20 bg-red-500/10 p-4">
        <div class="flex items-start justify-between gap-3">
          <div
            class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-red-500/15 text-red-200"
          >
            <Icon icon="mdi:alert-circle" />
          </div>

          <div class="mt-1">
            <p class="text-left text-sm font-semibold text-red-50">Something went wrong</p>
            <p class="mt-1 text-left text-sm leading-6 text-red-100/75">
              {error}. Please try again.
            </p>
          </div>
        </div>
      </div>
    {/if}
    <LoadingSkeleton />
  {:else}
    <div class="progress">
      <div class="progress-header">
        <span class="label">Session Progress</span>
        <span class="label label--accent">Exercise {getProgressFraction()}</span>
      </div>

      <div class="progress-track">
        <div class="progress-fill" style="width: {getProgressRatio()}%"></div>
      </div>
    </div>

    {#if allFinished}
      <h1 class:is-celebrating={allFinished}>Session finished!</h1>
      <h3 class="subtitle">Session overview</h3>
      <hr />
      <ExerciseCompleteScreen {exercises} />
      <button class="buttonClass mt-2 w-full" onclick={() => goto(resolve(`/`))}
        >Return to homepage</button
      >
    {:else}
      <div class="session-menu-anchor z-1">
        <button
          type="button"
          onclick={toggleActionMenu}
          class="action-trigger"
          aria-haspopup="menu"
          aria-expanded={showActionMenu}
          aria-controls="session-action-menu"
        >
          <Icon icon="material-symbols:more-horiz" color="grey" width={24} />
        </button>

        {#if showActionMenu}
          <div class="session-action-menu-shell">
            <div
              id="session-action-menu"
              class="session-action-menu"
              aria-label="Session actions"
              transition:growDown
            >
              <button
                type="button"
                class="session-action"
                aria-label="Quit session"
                onclick={handleQuitAction}
              >
                <Icon icon="material-symbols:logout-rounded" width={20} />
              </button>

              <button
                type="button"
                class="session-action"
                aria-label="Edit"
                onclick={handleEditAction}
              >
                <Icon icon="material-symbols:edit-outline-rounded" width={20} />
              </button>

              <button
                type="button"
                class="session-action"
                aria-label="Finish session"
                onclick={finishSessionNow}
              >
                <Icon icon="material-symbols:check-circle-outline-rounded" width={20} />
              </button>
            </div>
          </div>
        {/if}
      </div>

      {#if editMode && currentExercise}
        <section class="edit-card">
          <div class="edit-copy">
            <p class="edit-label">
              <Icon icon="material-symbols:edit-outline-rounded" width={20} />
              Edit exercise
            </p>
            <p class="edit-help">Update any field and save to apply only the changed values.</p>
          </div>

          <div class="edit-current-values">

          </div>

          <div class="edit-fields">
            <InputField label={'Exercise name'} bind:value={editFields.name} type={'text'} />
            <button
              type="button"
              class="edit-undo"
              class:off={compareFields('name')}
              onclick={() => revertEditField('name')}
            >
              <Icon icon="ic:round-undo" />
            </button>

            <InputField label={'Sets'} bind:value={editFields.sets} type={'number'} />
            <button
              type="button"
              class="edit-undo"
              class:off={compareFields('sets')}
              onclick={() => revertEditField('sets')}
            >
              <Icon icon="ic:round-undo" />
            </button>

            <InputField label={'Weight'} bind:value={editFields.weight} type={'number'} />
            <button
              type="button"
              class="edit-undo"
              class:off={compareFields('weight')}
              onclick={() => revertEditField('weight')}
            >
              <Icon icon="ic:round-undo" />
            </button>

            <InputField
              label={'Rep threshold'}
              bind:value={editFields.repThreshold}
              type={'number'}
            />
            <button
              type="button"
              class="edit-undo"
              class:off={compareFields('reps')}
              onclick={() => revertEditField('reps')}
            >
              <Icon icon="ic:round-undo" />
            </button>

            <InputField
              label={'Auto increase'}
              bind:value={editFields.autoIncrease}
              type={'number'}
            />
            <button
              type="button"
              class="edit-undo"
              class:off={compareFields('auto')}
              onclick={() => revertEditField('auto')}
            >
              <Icon icon="ic:round-undo" />
            </button>
          </div>

          <div class="edit-actions">
            <button type="button" class="movement-b edit-secondary" onclick={cancelEditMode}>
              Cancel
            </button>
            <button type="button" class="movement-b edit-primary" onclick={submitExerciseEdit}>
              {editSubmitting ? 'Saving...' : 'Save edit'}
            </button>
          </div>
        </section>
      {/if}

      <ExerciseTrackScreen
        {finished}
        exIndex={currentExerciseIndex}
        onCount={handleRepCountIncrementation}
        onSubmit={submitExercise}
        exData={currentExercise}
      />

      <div class="movement-cont">
        <button class="movement-b" class:inactive={!prevExists} onclick={() => prevExercise()}>
          <Icon icon="material-symbols:arrow-left-alt-rounded" />
          <span>Prev</span>
        </button>
        <button
          class="movement-b long"
          class:inactive={checkNextUnfinished()}
          onclick={() => gotoNextUnfinished()}
        >
          <span>Go to current</span>
        </button>
        <button class="movement-b" class:inactive={!nextExists} onclick={() => skipExercise()}>
          <span>Skip</span>
          <Icon icon="material-symbols:arrow-right-alt-rounded" />
        </button>
      </div>
    {/if}
  {/if}

  {#if showActionMenu}
    <button
      type="button"
      class="menu-backdrop"
      aria-label="Close session actions"
      onclick={closeActionMenu}
      transition:fade={{ duration: 140 }}
    ></button>
  {/if}
</main>

<style>
  hr {
    border: 1px solid var(--text-muted);
    margin: 10px 0;
    width: 100%;
  }

  .is-celebrating {
    animation: session-finished-celebration 900ms ease both;
  }

  @keyframes session-finished-celebration {
    0% {
      transform: scale(0.92);
    }

    22% {
      transform: scale(1.05);
    }

    38% {
      transform: scale(1.06);
    }

    58% {
      transform: scale(1.09);
    }

    78% {
      transform: scale(0.99);
    }

    100% {
      transform: scale(1);
    }
  }

  .subtitle {
    margin-top: 2rem;
    font-size: 1.125rem;
    letter-spacing: 0.03em;
    color: var(--color-gray);
    text-align: left;
    width: 100%;
  }

  .progress {
    margin-bottom: 2rem;
    width: 100%;
  }

  .progress-header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    margin-bottom: 0.5rem;
  }

  .label {
    font-weight: 400;
    font-size: 0.75rem;
  }

  .label--accent {
    font-weight: 600;
    color: var(--color-alt);
  }

  .progress-track {
    width: 100%;
    height: 2px;
    background: var(--border);
    overflow: hidden;
  }

  .progress-fill {
    transition: all ease-in-out 280ms;
    height: 100%;
    background: var(--color-alt);
  }

  .app-container {
    display: flex;
    flex-direction: column;
    padding: 4rem 2rem;
    max-width: 30rem;
    margin: 0 auto;
    text-align: center;
    align-items: center;
    background: var(--gradient-prim);
    overflow-y: auto;
    min-height: 100vh;
    overflow-x: hidden;
  }

  .movement-cont {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    width: 100%;
    margin-top: 2rem;
  }

  .edit-label {
    display: inline-flex;
    align-items: center;
    gap: 1rem;
    font-size: 0.75rem;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .edit-card {
    width: 100%;
    margin-bottom: 1.5rem;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--card);
    text-align: left;
  }

  .edit-copy {
    margin-bottom: 0.75rem;
  }

  .edit-help {
    margin-top: 0.35rem;
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .edit-current-values {
    margin-bottom: 1rem;
    color: var(--text-muted);
    font-size: 0.95rem;
  }

  .edit-current-values p {
    margin: 0.2rem 0;
  }

  .edit-fields {
    display: grid;
    grid-template-columns: 4fr 1fr;
    gap: 0.75rem;
  }

  .edit-undo {
    background-color: transparent;
  }

  .edit-undo.off {
    opacity: 0;
    pointer-events: none;
    visibility: hidden;
  }

  .edit-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    margin-top: 1rem;
    font-size: 12px;
  }

  .edit-secondary,
  .edit-primary {
    width: auto;
    min-width: 7rem;
    background-color: var(--surface-middle);
  }

  .movement-b {
    display: flex;

    flex-direction: row;
    justify-content: space-evenly;
    align-items: center;
    width: 5rem;
    height: 2rem;
    border-radius: 20px;
    border: none;
    outline: none;
    box-shadow: var(--shadow-dark);
    transition-property: opacity;
    transition-timing-function: ease;
    transition-duration: 700ms;
    transition-delay: 150ms;
  }

  .movement-b.inactive {
    filter: brightness(0.5);
    transition: opacity ease 200ms;
    opacity: 0;
  }

  .movement-b.long {
    width: 40%;
  }

  .movement-b span {
    font-size: 12px;
    color: var(--color-text);
  }

  .menu-backdrop {
    position: fixed;
    inset: 0;
    z-index: 1;
    border: none;
    padding: 0;
    margin: 0;
    background: linear-gradient(180deg, var(--color-background), transparent);
    mask: linear-gradient(black, transparent);
    backdrop-filter: blur(4px);

    cursor: default;
  }

  .session-menu-anchor {
    position: absolute;
    right: 0;
    top: 0;
    transform: translate(1rem, -1.3rem);
    display: flex;
    flex-direction: column;
    align-items: center;
    z-index: 20;
  }

  .action-trigger {
    width: fit-content;
    height: fit-content;
    background-color: rgba(240, 248, 255, 0);
    box-shadow: none;
    z-index: 21;

    padding: 1.5rem 1.5rem;
    outline: none;
  }

  .session-action-menu-shell {
    position: absolute;
    top: calc(100% + 0.6rem);
    left: 50%;
    transform: translateX(-50%);
    z-index: 22;
  }

  .session-action-menu {
    transform-origin: top center;
    padding: var(--spacing-xs);
    display: flex;
    flex-direction: column;
    border-radius: 20px 0 0 20px;
    background: var(--color-background);
    border: 1px solid rgba(255, 255, 255, 0.15);
    backdrop-filter: blur(14px);
    box-shadow:
      0 18px 40px rgba(15, 23, 42, 0.18),
      var(--shadow-dark);
  }

  .session-action {
    display: grid;
    place-items: center;
    width: 3rem;
    height: 3.45rem;
    border: none;
    border-radius: 999px;
    background: transparent;
    color: var(--color-text);
    transition:
      transform 140ms ease,
      background-color 140ms ease,
      box-shadow 140ms ease;
  }

  .session-action:active {
    transform: scale(0.96);
  }

  .session-action:focus-visible {
    outline: 2px solid var(--color-alt);
    outline-offset: 2px;
  }
</style>
