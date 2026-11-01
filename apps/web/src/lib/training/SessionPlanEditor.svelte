<script lang="ts">
  import { beforeNavigate } from '$app/navigation';
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
  }: {
    session: Session;
    onsaved: (session: Session) => void;
    ondeleted: (id: string) => void;
    oncancel: () => void;
    onbusy?: (busy: boolean) => void;
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
  let resolver: DraftItemResolver;
  let pending = $state.raw<
    | { kind: 'update'; command: UpdateSessionCommand }
    | { kind: 'delete'; command: DeleteSessionCommand }
    | null
  >(null);
  let saveState = $state<'idle' | 'saving' | 'failed' | 'saved'>('idle');
  let error = $state('');
  let canEdit = $state(false);
  let confirming = $state(false);
  let commandAccount = '';
  const label = (entry: Entry) =>
    entry.placement?.exercise.name ??
    entry.draft?.exercise?.name ??
    (entry.draft?.intent.kind === 'createExercise' ? entry.draft.intent.name : 'Exercise');
  const variantID = (entry: Entry) =>
    entry.placement?.variantId ??
    (entry.draft?.intent.kind === 'reuseVariant' ? entry.draft.intent.variantId : undefined);

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
      if (resolver.hasInput() && !resolver.add()) return;
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
    confirming = false;
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
<section aria-label="Edit saved session">
  <h3>Edit session</h3>
  <fieldset disabled={pending !== null}>
    <label>Session name <input aria-label="Session name" maxlength="100" bind:value={name} /></label
    >
    <p>
      Plan changes apply to the next workout. Removing a placement keeps its exercise, targets and
      history.
    </p>
    {#if !entries.length}<p>This plan is empty. Add a variant before starting a workout.</p>{/if}
    {#each entries as entry, index (entry.id)}
      <article>
        <h4>
          {index + 1}. {label(entry)} · {entry.placement?.variant.name ??
            entry.draft?.reviewedVariant?.name ??
            (entry.draft?.intent.kind === 'createVariant' ? entry.draft.intent.name : 'Standard')}
        </h4>
        <button
          aria-label={`Move ${label(entry)} up`}
          disabled={index === 0}
          onclick={() => move(index, -1)}>Move up</button
        >
        <button
          aria-label={`Move ${label(entry)} down`}
          disabled={index === entries.length - 1}
          onclick={() => move(index, 1)}>Move down</button
        >
        <button
          aria-label={`Remove ${label(entry)}`}
          onclick={() => (entries = entries.filter((item) => item.id !== entry.id))}>Remove</button
        >
      </article>
    {/each}
    <h4>Add a placement</h4>
    <DraftItemResolver bind:this={resolver} onresolve={append} disabled={pending !== null} />
  </fieldset>
  {#if saveState !== 'idle'}<p role="status">
      {saveState === 'saving' ? 'Saving' : saveState === 'saved' ? 'Saved' : 'Not saved'}
    </p>{/if}
  {#if error}<p role="alert">{error}</p>{/if}
  {#if saveState === 'failed'}
    <button onclick={send}>Retry</button>
    {#if canEdit}<button onclick={release}>Edit draft</button><button onclick={reload}
        >Discard draft and reload current plan</button
      >{/if}
  {:else}<button disabled={pending !== null} onclick={save}>Save session</button>{/if}
  <button disabled={pending !== null} onclick={oncancel}>Cancel</button>
  {#if confirming}
    <p>Delete this session plan? Your exercises and completed history will stay.</p>
    <button disabled={pending !== null} onclick={confirmDelete}>Confirm delete</button>
    <button disabled={pending !== null} onclick={() => (confirming = false)}>Keep session</button>
  {:else}<button disabled={pending !== null} onclick={() => (confirming = true)}
      >Delete session</button
    >{/if}
</section>

<style>
  section {
    width: 100%;
  }
  fieldset {
    border: 0;
    padding: 0;
    min-width: 0;
  }
  input {
    display: block;
    width: 100%;
    box-sizing: border-box;
  }
  article {
    padding: 1rem;
    margin: 0.5rem 0;
    border-radius: 15px;
    background: var(--surface-low);
  }
  button {
    margin: 0.25rem;
  }
</style>
