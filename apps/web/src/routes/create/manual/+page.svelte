<script lang="ts">
  import '../../../app.css';
  import DraftItemResolver from '$lib/training/DraftItemResolver.svelte';
  import SortableList from '../SortableList.svelte';
  import { beforeNavigate, goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import { getMe } from '../../me';
  import { createSession, readCatalog, TrainingError } from '$lib/training/client';
  import type {
    CreateSessionCommand,
    DraftSessionItem,
    ExerciseSuggestion,
  } from '$lib/training/types';
  import { readGeneratedDraft } from '$lib/training/generatedDraft';

  let sessionName = $state(page.url.searchParams.get('title') ?? '');
  let resolver = $state<DraftItemResolver>();
  let sessionExercisesList: SortableList;
  let error = $state('');
  let reviewing = $state(false);
  let saveState = $state<'idle' | 'saving' | 'failed' | 'saved'>('idle');
  let pending = $state.raw<CreateSessionCommand | null>(null);
  let commandAccount = '';
  let canEdit = $state(false);
  const quickloadKey = 'workoutPayload';
  let generated = $state(false);
  let invalidDraft = $state(false);
  let suggestions = $state.raw<{ id: string; suggestion: ExerciseSuggestion }[]>([]);
  let reviewIndex = $state(0);
  let currentSuggestion = $derived(suggestions[reviewIndex]);
  let loadingDraft = $state(false);

  onMount(async () => {
    if (!get(user) && !(await getMe())) {
      // eslint-disable-next-line svelte/no-navigation-without-resolve -- The root is resolved before appending the notification query.
      await goto(`${resolve('/')}?sonner=auth`);
      return;
    }
    if (page.url.searchParams.get('quickload') === '1') {
      generated = true;
      await reviewSuggestions(sessionStorage.getItem(quickloadKey) ?? '');
    }
  });
  async function reviewSuggestions(rawDraft: string) {
    loadingDraft = true;
    try {
      let draft = readGeneratedDraft(rawDraft);
      if (draft.some((suggestion) => suggestion.muscleGroupCodes?.length)) {
        const catalog = await readCatalog();
        draft = readGeneratedDraft(
          rawDraft,
          new Set(catalog.muscleGroups.map((group) => group.code))
        );
      }
      suggestions = draft.map((suggestion) => ({ id: crypto.randomUUID(), suggestion }));
      reviewIndex = 0;
      sessionExercisesList.fillFromArray([]);
      invalidDraft = false;
      error = '';
    } catch {
      invalidDraft = true;
      error = '';
    } finally {
      loadingDraft = false;
    }
  }
  function resolveSuggestion(item: DraftSessionItem): boolean {
    if (!appendDraft(item)) return false;
    reviewIndex++;
    return true;
  }
  async function cancelGenerated() {
    if (pending) return;
    sessionStorage.removeItem(quickloadKey);
    await goto(resolve('/create'));
  }
  async function startManually() {
    if (pending) return;
    sessionStorage.removeItem(quickloadKey);
    generated = false;
    invalidDraft = false;
    suggestions = [];
    reviewIndex = 0;
    error = '';
    sessionExercisesList.fillFromArray([]);
    await goto(resolve('/create/manual'));
  }
  function removeSuggestion() {
    if (pending || !currentSuggestion) return;
    reviewIndex++;
    error = '';
  }
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

  function appendDraft(item: DraftSessionItem): boolean {
    const current = sessionExercisesList.extractData();
    if (current.length >= 100) {
      error = 'A session can contain at most 100 exercises.';
      return false;
    }
    const intent = item.intent;
    if (
      intent.kind === 'reuseVariant' &&
      current.some(
        (entry) =>
          entry.intent.kind === 'reuseVariant' && entry.intent.variantId === intent.variantId
      )
    ) {
      error = 'This variant is already in this session. Choose a different variant.';
      return false;
    }
    if (
      intent.kind !== 'reuseVariant' &&
      current.some(
        (entry) =>
          entry.intent.kind === intent.kind &&
          entry.intent.name.trim().toLowerCase() === intent.name.trim().toLowerCase() &&
          (intent.kind === 'createExercise' ||
            (entry.intent.kind === 'createVariant' &&
              entry.intent.exerciseId === intent.exerciseId))
      )
    ) {
      error = 'This creation name is already in this session. Choose a different name.';
      return false;
    }
    sessionExercisesList.pushItemToList(item);
    error = '';
    return true;
  }
  async function editDraft(item: DraftSessionItem): Promise<boolean> {
    if (!resolver) {
      error = 'Finish reviewing the suggestions before editing a reviewed exercise.';
      return false;
    }
    reviewing = true;
    try {
      return await resolver.edit(item);
    } finally {
      reviewing = false;
    }
  }
  async function saveSession() {
    if (saveState === 'saving' || reviewing || loadingDraft) return;
    if (!pending) {
      if (invalidDraft || currentSuggestion) {
        error =
          'Review every suggestion before saving. Each needs an explicit reuse or creation choice.';
        return;
      }
      if (resolver?.hasInput() && !resolver.add()) return;
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
  <fieldset disabled={pending !== null || reviewing || loadingDraft}>
    <input
      aria-label="Session name"
      maxlength="100"
      bind:value={sessionName}
      placeholder="Untitled session"
      class="title"
    />
    <div class="add-box shadow">
      {#if generated}
        <div class="review-header">
          <h3>
            {invalidDraft
              ? 'Unable to load session'
              : currentSuggestion
                ? `Review suggestion ${reviewIndex + 1} of ${suggestions.length}`
                : loadingDraft
                  ? 'Generated session'
                  : 'Add an exercise'}
          </h3>
          <button
            type="button"
            class="cancel-generated"
            aria-label="Cancel generated session"
            onclick={cancelGenerated}>Cancel</button
          >
        </div>
      {/if}
      {#if loadingDraft}
        <p role="status">Checking the suggested session…</p>
      {:else if invalidDraft}
        <div class="invalid-generated">
          <p role="alert">This generated session could not be loaded.</p>
          <p>Return to creation to try again, or start a session manually.</p>
          <button type="button" class="buttonClass" onclick={cancelGenerated}>Back to create</button
          >
          <button type="button" class="start-manually" onclick={startManually}
            >Start manually</button
          >
        </div>
      {:else if currentSuggestion}
        <p class="review-copy">
          Choose shared current settings or explicitly create separate settings.
        </p>
        {#each [currentSuggestion] as row (row.id)}
          <DraftItemResolver
            suggestion={row.suggestion}
            onresolve={resolveSuggestion}
            onremove={removeSuggestion}
            disabled={pending !== null || reviewing}
          />
        {/each}
      {:else}
        {#if !generated}<h3 class="w-full pb-2 text-xl font-semibold">Add an exercise</h3>{/if}
        <DraftItemResolver
          bind:this={resolver}
          onresolve={appendDraft}
          disabled={pending !== null || reviewing}
        />
      {/if}
    </div>
    <div class={['session-exercises', { 'invalid-list': invalidDraft }]}>
      <SortableList
        bind:this={sessionExercisesList}
        editData={editDraft}
        disabled={pending !== null || reviewing}
      />
    </div>
  </fieldset>
  {#if saveState !== 'idle'}<p role="status">
      {saveState === 'saving' ? 'Saving' : saveState === 'saved' ? 'Saved' : 'Not saved'}
    </p>{/if}
  {#if error}<p role="alert">{error}</p>{/if}
  {#if saveState === 'failed'}
    <button onclick={saveSession} class="buttonClass">Retry</button>
    {#if canEdit}<button onclick={resumeEditing} class="buttonClass">Edit draft</button>{/if}
  {:else if !invalidDraft}
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
  .review-header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.75rem;
    width: 100%;
    margin-bottom: 0.5rem;
  }
  .review-header h3 {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    line-height: 1.4;
    text-wrap: balance;
  }
  .cancel-generated,
  .start-manually {
    margin: 0;
    padding: 0.35rem 0;
    border: 0;
    background: transparent;
    box-shadow: none;
    color: var(--text-muted);
    font-size: 12px;
    font-weight: 400;
  }
  .cancel-generated:hover,
  .start-manually:hover {
    color: var(--text);
  }
  .review-copy {
    margin: 0 0 0.75rem;
    color: var(--text-muted);
    font-size: 12px;
    line-height: 1.5;
  }
  .invalid-generated {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
  }
  .invalid-generated p {
    margin: 0;
    font-size: 13px;
    line-height: 1.5;
  }
  .invalid-generated [role='alert'] {
    width: 100%;
    color: var(--text);
    font-weight: 500;
  }
  .invalid-generated .buttonClass {
    width: 100%;
    margin-top: 0.5rem;
  }
  .invalid-list {
    display: none;
  }
  .session-exercises {
    width: 100%;
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
    position: relative;
    z-index: 2;
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
