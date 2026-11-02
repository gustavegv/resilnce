<script lang="ts">
  import { beforeNavigate } from '$app/navigation';
  import EditScreen from '../../routes/tracker/EditScreen.svelte';
  import Icon from '@iconify/svelte';
  import * as Alert from '$lib/components/alert/index.js';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import DraftItemResolver from './DraftItemResolver.svelte';
  import { updateSession, deleteSession, readSession, TrainingError } from './client';
  import type {
    Session,
    DraftSessionItem,
    Placement,
    UpdateSessionCommand,
    DeleteSessionCommand,
  } from './types';

  let {
    session,
    onsaved,
    ondeleted,
    oncancel,
    onbusy = () => {},
    confirmDeletion = false,
  }: {
    session: Session;
    onsaved: (session: Session) => void;
    ondeleted: (id: string) => void;
    oncancel: () => void;
    onbusy?: (busy: boolean) => void;
    confirmDeletion?: boolean;
  } = $props();
  type Entry = { id: string; placement?: Placement; draft?: DraftSessionItem };
  // A keyed editor owns one deliberate draft until Save, Cancel or Discard/reload.
  // svelte-ignore state_referenced_locally
  let name = $state(session.name);
  // svelte-ignore state_referenced_locally
  let revision = session.revision;
  // svelte-ignore state_referenced_locally
  let entries = $state.raw<Entry[]>(
    session.placements.map((placement) => ({ id: placement.id, placement }))
  );
  let resolver = $state<DraftItemResolver>();
  let pending = $state.raw<
    | { kind: 'update'; command: UpdateSessionCommand }
    | { kind: 'delete'; command: DeleteSessionCommand }
    | null
  >(null);
  let saveState = $state<'idle' | 'saving' | 'failed' | 'saved'>('idle');
  let error = $state('');
  let canEdit = $state(false);
  // svelte-ignore state_referenced_locally
  let confirming = $state(confirmDeletion);
  let open = $state(true);
  let adding = $state(false);
  let titleInput = $state<HTMLInputElement>();
  let commandAccount = '';
  let deleting = $derived(confirming || pending?.kind === 'delete');
  const label = (entry: Entry) =>
    entry.placement?.exercise.name ??
    entry.draft?.exercise?.name ??
    (entry.draft?.intent.kind === 'createExercise' ? entry.draft.intent.name : 'Exercise');
  const variantID = (entry: Entry) =>
    entry.placement?.variantId ??
    (entry.draft?.intent.kind === 'reuseVariant' ? entry.draft.intent.variantId : undefined);

  function configurationSummary(entry: Entry): string {
    const variant = entry.placement?.variant ?? entry.draft?.reviewedVariant;
    if (variant) {
      const weights = variant.targetSets.map((target) => Number(target.targetWeight));
      const minimum = Math.min(...weights);
      const maximum = Math.max(...weights);
      const weight = !weights.length
        ? ''
        : minimum === maximum
          ? `${minimum} kg`
          : `${minimum}–${maximum} kg`;
      return `${variant.prescribedSets} sets${weight ? ` · ${weight}` : ''}`;
    }
    const intent = entry.draft?.intent;
    return intent && intent.kind !== 'reuseVariant'
      ? `${intent.configuration.prescribedSets} sets · ${intent.configuration.initialWeight} kg`
      : '';
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
  function append(draft: DraftSessionItem): boolean {
    if (entries.length >= 100) {
      error = 'A session can contain at most 100 placements.';
      return false;
    }
    if (
      draft.intent.kind === 'reuseVariant' &&
      entries.some(
        (entry) =>
          variantID(entry) ===
          (draft.intent.kind === 'reuseVariant' ? draft.intent.variantId : undefined)
      )
    ) {
      error = 'This variant is already in this session. Choose a different variant.';
      return false;
    }
    entries = [...entries, { id: draft.draftId, draft }];
    error = '';
    return true;
  }
  function move(index: number, delta: number) {
    const next = [...entries];
    [next[index], next[index + delta]] = [next[index + delta], next[index]];
    entries = next;
  }
  function release() {
    pending = null;
    onbusy(false);
    saveState = 'idle';
    canEdit = false;
    error = '';
  }
  async function save() {
    if (saveState === 'saving') return;
    if (!pending) {
      if (resolver?.hasInput() && !resolver.add()) return;
      if (!name.trim() || [...name.trim()].length > 100) {
        error = 'Use a session name of 1–100 characters.';
        return;
      }
      pending = {
        kind: 'update',
        command: {
          operationId: crypto.randomUUID(),
          expectedRevision: revision,
          name: name.trim(),
          items: structuredClone(
            entries.map((entry) =>
              entry.placement
                ? { kind: 'retainPlacement' as const, placementId: entry.placement.id }
                : entry.draft!.intent
            )
          ),
        },
      };
      commandAccount = get(user)?.id ?? '';
      onbusy(true);
    }
    await send();
  }
  async function confirmDelete() {
    if (pending) return;
    pending = {
      kind: 'delete',
      command: { operationId: crypto.randomUUID(), expectedRevision: revision },
    };
    commandAccount = get(user)?.id ?? '';
    onbusy(true);
    await send();
  }
  async function send() {
    if (!pending || saveState === 'saving') return;
    if (commandAccount !== (get(user)?.id ?? '')) {
      error = 'The account changed. Return to the original account before retrying.';
      saveState = 'failed';
      return;
    }
    saveState = 'saving';
    error = '';
    canEdit = false;
    try {
      if (pending.kind === 'update') {
        const result = await updateSession(session.id, pending.command);
        pending = null;
        onbusy(false);
        saveState = 'saved';
        onsaved(result.session);
      } else {
        await deleteSession(session.id, pending.command);
        pending = null;
        onbusy(false);
        saveState = 'saved';
        ondeleted(session.id);
      }
    } catch (failure) {
      saveState = 'failed';
      error = failure instanceof Error ? failure.message : 'Unable to save session.';
      canEdit = failure instanceof TrainingError && failure.status >= 400 && failure.status < 500;
    }
  }
  async function reload() {
    if (!canEdit) return;
    try {
      const current = (await readSession(session.id)).session;
      name = current.name;
      revision = current.revision;
      entries = current.placements.map((placement) => ({ id: placement.id, placement }));
      release();
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to reload session.';
    }
  }
</script>

<svelte:window onbeforeunload={beforeUnload} />
<EditScreen
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) oncancel();
    }
  }
  dismissible={pending === null}
  title={deleting ? 'Are you absolutely sure?' : 'Edit session'}
>
  <section aria-label="Edit saved session">
    {#if deleting}
      <Alert.Description
        >Delete <strong>{session.name}</strong>? This removes the saved session. Your reusable
        exercises, variants and completed history will stay.</Alert.Description
      >
      <div class="edit-actions">
        <Alert.Cancel disabled={pending !== null}>No, take me back</Alert.Cancel>
        <Alert.Action
          class="bg-destructive text-white"
          disabled={pending !== null}
          onclick={(event) => {
            event.preventDefault();
            void confirmDelete();
          }}>Confirm delete</Alert.Action
        >
      </div>
    {:else}
      <Alert.Description>
        No changes are made until you press confirm. What do you want to change with <strong
          >{session.name}</strong
        >?
      </Alert.Description>
      <fieldset disabled={pending !== null}>
        <div class="title-row">
          <input
            bind:this={titleInput}
            class={['clean-input', { unedited: name.trim() === session.name }]}
            aria-label="Session name"
            maxlength="100"
            bind:value={name}
          />
          <button
            type="button"
            class="title-input-icon"
            aria-label={name.trim() === session.name ? 'Edit session title' : 'Undo title edit'}
            onclick={() => {
              if (name.trim() === session.name) titleInput?.focus();
              else name = session.name;
            }}
          >
            <Icon
              icon={name.trim() === session.name
                ? 'material-symbols:edit-outline-rounded'
                : 'material-symbols:undo-rounded'}
              width={24}
            />
          </button>
        </div>
        <section class="exercise-overview">
          <p class="exercise-overview-title">Exercises in this session</p>
          <p class="exercise-overview-copy">
            Plan changes apply to the next workout. Removing a placement keeps its exercise, targets
            and history.
          </p>
          {#if !entries.length}<p class="exercise-overview-copy">
              This plan is empty. Add a variant before starting a workout.
            </p>{/if}
          <div class="exercise-list">
            {#each entries as entry, index (entry.id)}
              <div class={['exercise-row', { 'new-exercise': entry.draft }]}>
                <div>
                  <p class="exercise-order">Exercise {index + 1}</p>
                  <p class="exercise-name">
                    {label(entry)} · {entry.placement?.variant.name ??
                      entry.draft?.reviewedVariant?.name ??
                      (entry.draft?.intent.kind === 'createVariant'
                        ? entry.draft.intent.name
                        : 'Standard')}
                  </p>
                  <p class="exercise-meta">{configurationSummary(entry)}</p>
                </div>
                <div class="placement-actions">
                  <button
                    class="title-input-icon"
                    aria-label={`Move ${label(entry)} up`}
                    disabled={index === 0}
                    onclick={() => move(index, -1)}
                    ><Icon icon="material-symbols:arrow-upward-rounded" width={18} /></button
                  >
                  <button
                    class="title-input-icon"
                    aria-label={`Move ${label(entry)} down`}
                    disabled={index === entries.length - 1}
                    onclick={() => move(index, 1)}
                    ><Icon icon="material-symbols:arrow-downward-rounded" width={18} /></button
                  >
                </div>
              </div>
              <button
                class="title-input-icon"
                aria-label={`Remove ${label(entry)}`}
                onclick={() => (entries = entries.filter((item) => item.id !== entry.id))}
                ><Icon
                  icon="material-symbols:delete-outline-rounded"
                  width={30}
                  color="gray"
                /></button
              >
            {/each}
          </div>
          {#if !adding}
            <button
              class="title-input-icon add-placement"
              aria-label="Add an exercise"
              onclick={() => (adding = true)}
              ><Icon
                icon="material-symbols:add-circle-outline-rounded"
                width={30}
                color="grey"
              /></button
            >
          {:else}
            <div class="add-exercise">
              <p class="exercise-overview-title">Add an exercise</p>
              <DraftItemResolver
                bind:this={resolver}
                onresolve={append}
                disabled={pending !== null}
              />
            </div>
          {/if}
        </section>
      </fieldset>
    {/if}
    {#if saveState !== 'idle'}<p role="status">
        {saveState === 'saving' ? 'Saving' : saveState === 'saved' ? 'Saved' : 'Not saved'}
      </p>{/if}
    {#if error}<p role="alert">{error}</p>{/if}
    {#if saveState === 'failed'}
      <button class="buttonClass" onclick={send}>Retry</button>
      {#if canEdit}<button class="buttonClass minimal" onclick={release}>Edit draft</button><button
          class="buttonClass minimal"
          onclick={reload}>Discard draft and reload current plan</button
        >{/if}
    {/if}
    {#if !deleting}
      <div class="edit-actions">
        <Alert.Cancel disabled={pending !== null}>Cancel</Alert.Cancel>
        <Alert.Action
          class="bg-accent"
          disabled={pending !== null}
          onclick={(event) => {
            event.preventDefault();
            void save();
          }}>{saveState === 'saving' ? 'Saving...' : 'Confirm edits'}</Alert.Action
        >
      </div>
      <button class="delete-session" disabled={pending !== null} onclick={() => (confirming = true)}
        >Delete session</button
      >
    {/if}
  </section>
</EditScreen>

<style>
  section {
    width: 100%;
  }
  fieldset {
    border: 0;
    padding: 0;
    min-width: 0;
  }
  .title-row {
    display: flex;
    gap: 1rem;
  }
  .clean-input {
    font-size: 1.5rem;
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
  .title-input-icon {
    background-color: transparent;
    padding: 0;
    box-shadow: none;
    transition:
      opacity 0.18s ease,
      transform 0.18s ease;
  }
  .title-input-icon:disabled {
    opacity: 0.3;
  }
  .exercise-overview {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 1rem;
    border-radius: 0.75rem;
    background: var(--card);
  }
  .exercise-overview-title {
    margin: 0;
    font-size: 0.8rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
  }
  .exercise-overview-copy {
    margin: 0;
    font-size: 0.85rem;
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
  }
  .exercise-row.new-exercise {
    border-color: rgba(96, 255, 96, 0.243);
    background-color: color-mix(in srgb, var(--surface-middle), rgb(96, 255, 96) 5%);
  }
  .exercise-order {
    margin: 0;
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
  }
  .exercise-name {
    margin: 0;
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text);
  }
  .exercise-meta {
    margin: 0;
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  .placement-actions {
    display: flex;
    gap: 0.4rem;
  }
  .add-placement {
    display: flex;
    justify-content: center;
  }
  .add-exercise {
    display: grid;
    gap: 0.75rem;
  }
  .edit-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    margin-top: 1rem;
  }
  .delete-session {
    display: block;
    margin: 1rem auto 0;
    background: transparent;
    color: var(--text-muted);
    font-size: 0.85rem;
    box-shadow: none;
  }
</style>
