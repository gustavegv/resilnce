<script lang="ts">
  import { onMount } from 'svelte';
  import { beforeNavigate } from '$app/navigation';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import {
    readWorkout,
    recordSet,
    updateConfiguration,
    completeAttempt,
    TrainingError,
  } from './client';
  import type {
    ConfigurationPatch,
    CompleteAttemptCommand,
    UpdateConfigurationCommand,
    RecordSetCommand,
    Workout,
    WorkoutExercise,
    WorkoutSet,
  } from './types';

  let { workoutId }: { workoutId: string } = $props();
  const originalAccount = get(user)?.id ?? '';
  let accountChanged = $derived(
    ($user?.id ?? '') !== originalAccount ||
      ($user !== null && $user.capabilities.applicationModel !== 'exercise-centric')
  );
  let workout = $state.raw<Workout | null>(null);
  let loading = $state(true);
  let error = $state('');
  let candidates = $state<Record<string, string>>({});
  let pending = $state.raw<
    RecordSetCommand | UpdateConfigurationCommand | CompleteAttemptCommand | null
  >(null);
  let editAttempt = $state.raw<WorkoutExercise | null>(null);
  let editFields = $state({
    exerciseName: '',
    variantName: '',
    prescribedSets: '',
    repThreshold: '',
    autoIncrement: '',
    restSeconds: '',
    weight: '',
  });
  let applyWeight = $state(false);
  let rejectedConfiguration = $state(false);
  let rejectedCompletion = $state(false);
  let completionPending = $derived(
    pending !== null && !('patch' in pending) && !('setId' in pending)
  );
  let mixedWeights = $derived(
    editAttempt !== null &&
      new Set(editAttempt.currentVariant.targetSets.map((s) => s.targetWeight)).size > 1
  );
  let shrinking = $derived(
    editAttempt !== null &&
      Number(editFields.prescribedSets) < editAttempt.currentVariant.prescribedSets
  );
  let editorDirty = $derived(editAttempt !== null);

  let saveState = $state<'saved' | 'saving' | 'failed'>('saved');
  let reviewed = $state.raw<Workout | null>(null);
  let conflict = $state(false);
  let uncertainCompletion = $derived(completionPending && !conflict && !rejectedCompletion);
  let navigationBlocked = $state(false);
  let hasInput = $derived(pending !== null || editorDirty || Object.keys(candidates).length > 0);
  let canReapply = $derived.by(() => {
    if (!pending || !reviewed || reviewed.status !== 'active') return false;
    const attempt = reviewed.exercises.find((a) => a.id === pending!.attemptId);
    if (attempt?.state !== 'draft') return false;
    const intent = pending;
    return 'patch' in intent
      ? attempt.exercise.id === intent.exerciseId &&
          attempt.variantId === intent.variantId &&
          !attempt.exercise.archived &&
          !attempt.currentVariant.archived
      : 'setId' in intent
        ? attempt.sets.some((s) => s.id === intent.setId)
        : attempt.exercise.id === intent.exerciseId &&
          attempt.variantId === intent.variantId &&
          !attempt.exercise.archived &&
          !attempt.currentVariant.archived &&
          attempt.sets.filter((s) => s.kind === 'prescribed').every((s) => s.state !== 'pending');
  });

  beforeNavigate(({ cancel }) => {
    if (hasInput) {
      cancel();
      navigationBlocked = true;
    }
  });
  function beforeUnload(event: BeforeUnloadEvent) {
    if (hasInput) {
      event.preventDefault();
      event.returnValue = '';
    }
  }
  function sameAccount() {
    if ((get(user)?.id ?? '') !== originalAccount || accountChanged) {
      error =
        'Your account changed or expired. Return to the original account to resolve this input.';
      return false;
    }
    return true;
  }
  async function load() {
    if (!sameAccount()) return;
    loading = true;
    error = '';
    try {
      const result = await readWorkout(workoutId);
      if (sameAccount()) workout = result.workout;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to load the workout.';
    } finally {
      loading = false;
    }
  }
  async function reviewCurrent() {
    if (!sameAccount() || !pending) return;
    try {
      const result = await readWorkout(pending.workoutId);
      if (sameAccount()) reviewed = result.workout;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to review current state.';
    }
  }
  async function save() {
    if (!pending || saveState === 'saving' || !sameAccount()) return;
    if (pending.workoutId !== workoutId) {
      error = 'The workout changed. Return to the original workout before retrying this input.';
      return;
    }
    const intent = pending;
    saveState = 'saving';
    error = '';
    try {
      const result = await ('patch' in intent
        ? updateConfiguration(intent)
        : 'setId' in intent
          ? recordSet(intent)
          : completeAttempt(intent));
      if (!sameAccount() || workoutId !== intent.workoutId) {
        saveState = 'failed';
        return;
      }
      workout = result.workout;
      if ('patch' in intent) {
        editAttempt = null;
        applyWeight = false;
      } else if ('setId' in intent) delete candidates[intent.setId];
      rejectedConfiguration = false;
      rejectedCompletion = false;
      pending = null;
      reviewed = null;
      conflict = false;
      saveState = 'saved';
      navigationBlocked = false;
    } catch (failure) {
      saveState = 'failed';
      error = failure instanceof Error ? failure.message : 'Unable to save this input.';
      rejectedConfiguration =
        'patch' in intent &&
        failure instanceof TrainingError &&
        [
          'name-conflict',
          'invalid-field',
          'malformed-request',
          'resource-unavailable',
          'not-found',
        ].includes(failure.code);
      rejectedCompletion =
        !('patch' in intent) &&
        !('setId' in intent) &&
        failure instanceof TrainingError &&
        ['invalid-field', 'malformed-request', 'resource-unavailable', 'not-found'].includes(
          failure.code
        );
      conflict =
        failure instanceof TrainingError &&
        ['stale-revision', 'resource-finalized'].includes(failure.code);
      if (conflict) await reviewCurrent();
    }
  }
  function finish(attempt: WorkoutExercise) {
    if (
      hasInput ||
      !workout ||
      workout.status !== 'active' ||
      attempt.state !== 'draft' ||
      attempt.sets.some((s) => s.kind === 'prescribed' && s.state === 'pending') ||
      !sameAccount()
    )
      return;
    pending = {
      operationId: crypto.randomUUID(),
      workoutId: workout.id,
      attemptId: attempt.id,
      exerciseId: attempt.exercise.id,
      variantId: attempt.variantId,
      expectedWorkoutRevision: workout.revision,
      expectedVariantRevision: attempt.currentVariant.revision,
    };
    void save();
  }
  function record(attempt: WorkoutExercise, set: WorkoutSet, state: WorkoutSet['state']) {
    if (
      pending ||
      editAttempt ||
      !workout ||
      workout.status !== 'active' ||
      attempt.state !== 'draft' ||
      !sameAccount()
    )
      return;
    const base = {
      operationId: crypto.randomUUID(),
      workoutId: workout.id,
      attemptId: attempt.id,
      setId: set.id,
      expectedWorkoutRevision: workout.revision,
    };
    if (state === 'performed') {
      const candidate = candidates[set.id] ?? String(set.recordedReps ?? set.targetReps);
      if (!/^\d+$/.test(candidate) || Number(candidate) > 2147483647) {
        error = 'Enter whole reps from 0 to 2147483647.';
        return;
      }
      candidates[set.id] = candidate;
      pending = { ...base, state, recordedReps: Number(candidate) };
    } else pending = { ...base, state };
    void save();
  }
  function openEdit(attempt: WorkoutExercise) {
    if (hasInput || !sameAccount()) return;
    const v = attempt.currentVariant;
    editAttempt = attempt;
    editFields = {
      exerciseName: attempt.exercise.name,
      variantName: v.name,
      prescribedSets: String(v.prescribedSets),
      repThreshold: String(v.repThreshold),
      autoIncrement: v.autoIncrement,
      restSeconds: String(v.restSeconds),
      weight:
        new Set(v.targetSets.map((s) => s.targetWeight)).size === 1
          ? v.targetSets[0].targetWeight
          : '',
    };
    applyWeight = false;
  }
  function submitConfiguration() {
    if (pending || !editAttempt || !workout || !sameAccount()) return;
    const a = editAttempt,
      v = a.currentVariant;
    const patch: ConfigurationPatch = {};
    for (const [key, current] of [
      ['exerciseName', a.exercise.name],
      ['variantName', v.name],
    ] as const) {
      const value = editFields[key].trim();
      if (!value || [...value].length > 100) {
        error = 'Use names of 1–100 characters.';
        return;
      }
      if (value !== current) patch[key] = value;
    }
    for (const [key, current, min, max, label] of [
      ['prescribedSets', v.prescribedSets, 1, 20, 'sets'],
      ['repThreshold', v.repThreshold, 1, 99, 'rep threshold'],
      ['restSeconds', v.restSeconds, 0, 2147483647, 'rest seconds'],
    ] as const) {
      const raw = editFields[key];
      if (!/^\d+$/.test(raw) || Number(raw) < min || Number(raw) > max) {
        error = `Enter whole ${label} from ${min} to ${max}.`;
        return;
      }
      if (Number(raw) !== current) patch[key] = Number(raw);
    }
    const decimal = (value: string, min: number, max: number) =>
      /^\d+(?:\.\d{1,2})?$/.test(value) && Number(value) >= min && Number(value) <= max;
    if (!decimal(editFields.autoIncrement, 0.25, 99)) {
      error = 'Enter an auto increase from 0.25 to 99 with at most two decimals.';
      return;
    }
    if (Number(editFields.autoIncrement) !== Number(v.autoIncrement))
      patch.autoIncrement = editFields.autoIncrement;
    if (applyWeight) {
      if (!decimal(editFields.weight, 0, 9999)) {
        error = 'Enter a weight from 0 to 9999 with at most two decimals.';
        return;
      }
      patch.applyWeightToAllSets = editFields.weight;
    }
    if (Object.keys(patch).length === 0) {
      error = 'Enter at least one field to edit, or cancel.';
      return;
    }
    pending = {
      operationId: crypto.randomUUID(),
      workoutId: workout.id,
      attemptId: a.id,
      exerciseId: a.exercise.id,
      variantId: a.variantId,
      expectedWorkoutRevision: workout.revision,
      expectedExerciseRevision: a.exercise.revision,
      expectedVariantRevision: v.revision,
      patch,
    };
    void save();
  }
  function acceptServer() {
    if (!reviewed || !sameAccount()) return;
    workout = reviewed;
    clearInput();
  }
  function reapply() {
    if (!canReapply || !pending || !reviewed || !sameAccount()) return;
    const attempt = reviewed.exercises.find((a) => a.id === pending!.attemptId)!;
    pending =
      'patch' in pending
        ? {
            ...pending,
            operationId: crypto.randomUUID(),
            expectedWorkoutRevision: reviewed.revision,
            expectedExerciseRevision: attempt.exercise.revision,
            expectedVariantRevision: attempt.currentVariant.revision,
          }
        : 'setId' in pending
          ? {
              ...pending,
              operationId: crypto.randomUUID(),
              expectedWorkoutRevision: reviewed.revision,
            }
          : {
              ...pending,
              operationId: crypto.randomUUID(),
              expectedWorkoutRevision: reviewed.revision,
              expectedVariantRevision: attempt.currentVariant.revision,
            };
    workout = reviewed;
    reviewed = null;
    conflict = false;
    void save();
  }
  function clearInput() {
    pending = null;
    editAttempt = null;
    applyWeight = false;
    rejectedConfiguration = false;
    rejectedCompletion = false;
    candidates = {};
    reviewed = null;
    conflict = false;
    saveState = 'saved';
    error = '';
    navigationBlocked = false;
  }
  async function discard() {
    if (saveState === 'saving' || uncertainCompletion) return;
    // A lost response may already be committed; never label the old snapshot
    // current after discarding local input. Re-read before enabling recording.
    clearInput();
    workout = null;
    await load();
  }
  onMount(() => {
    void load();
  });
</script>

<svelte:window onbeforeunload={beforeUnload} />
<main>
  {#if accountChanged}<p role="alert">
      Your account changed or expired. Local input is retained. Return to the original account
      before retrying.
    </p>{/if}
  {#if loading}<p role="status">Loading workout…</p>
  {:else if workout}
    <span>Workout</span>
    <h2>{workout.sourceSession?.name ?? 'Workout'}</h2>
    <p role="status">
      {saveState === 'saving'
        ? 'Saving…'
        : hasInput
          ? 'Not saved'
          : workout.status === 'active'
            ? 'Active workout · Saved'
            : 'Workout ended · Saved'}
    </p>
    {#if error}<p role="alert">{error}</p>{/if}
    {#if saveState === 'failed' && pending && !conflict}<button
        onclick={save}
        disabled={accountChanged}>Retry</button
      >{/if}
    {#if rejectedConfiguration && pending}<button
        onclick={() => {
          pending = null;
          rejectedConfiguration = false;
          saveState = 'saved';
        }}
        disabled={accountChanged}>Edit rejected configuration</button
      >{/if}
    {#if conflict}
      <p role="alert">
        The workout changed. Your local input is retained. Review server state before choosing.
      </p>
      {#if reviewed}
        <section aria-label="Current server state">
          <h3>Current server revision {reviewed.revision} · {reviewed.status}</h3>
          {#each reviewed.exercises as attempt (attempt.id)}
            <h4>{attempt.exercise.name} · {attempt.currentVariant.name} · {attempt.state}</h4>
            <p>
              Exercise revision {attempt.exercise.revision}; variant revision {attempt
                .currentVariant.revision}; {attempt.currentVariant.prescribedSets} sets; threshold {attempt
                .currentVariant.repThreshold}; increase {attempt.currentVariant.autoIncrement} kg; rest
              {attempt.currentVariant.restSeconds}s
            </p>
            {#each attempt.currentVariant.targetSets as target (target.id)}<p>
                Current target {target.position}: {target.targetReps} reps · {target.targetWeight} kg
              </p>{/each}
            {#each attempt.sets as set (set.id)}
              <p>
                Set {set.position}: {set.state}{set.state === 'performed'
                  ? ` · ${set.recordedReps} reps · ${set.recordedWeight} kg`
                  : ''}; target {set.targetReps} reps · {set.targetWeight} kg
              </p>
            {/each}
          {/each}
        </section>
        <button onclick={acceptServer} disabled={accountChanged}
          >Accept server state and discard local input</button
        >
        <button onclick={reapply} disabled={!canReapply || accountChanged}
          >Reapply local input to reviewed revision</button
        >
      {:else}<button onclick={reviewCurrent} disabled={accountChanged}>Review current state</button
        >{/if}
    {/if}
    {#if navigationBlocked}
      <p role="alert">
        {uncertainCompletion
          ? 'Completion is unconfirmed. Stay and retry to resolve it before leaving or starting another workout.'
          : 'Input is not saved. Stay and retry, or explicitly discard local input before leaving.'}
      </p>
      <button
        onclick={() => {
          navigationBlocked = false;
        }}>Stay</button
      >
    {/if}
    {#if uncertainCompletion && saveState === 'failed'}<p role="alert">
        Completion is unconfirmed. Retry the same completion to resolve it before leaving.
      </p>{/if}
    {#if hasInput && !conflict && !uncertainCompletion}<button
        onclick={discard}
        disabled={saveState === 'saving'}>Discard local input</button
      >{/if}
    <nav aria-label="Exercises">
      {#each workout.exercises as attempt (attempt.id)}
        <a href={`#attempt-${attempt.id}`}
          >{attempt.exercise.name} · {attempt.currentVariant.name}</a
        >
      {/each}
    </nav>
    {#each workout.exercises as attempt (attempt.id)}
      <section
        id={`attempt-${attempt.id}`}
        aria-label={`${attempt.exercise.name} ${attempt.currentVariant.name}`}
      >
        <h3>{attempt.position}. {attempt.exercise.name} · {attempt.currentVariant.name}</h3>
        <p>{attempt.state === 'completed' ? 'Completed' : 'Draft'}</p>
        {#if workout.status === 'active' && attempt.state === 'draft'}
          <button onclick={() => openEdit(attempt)} disabled={hasInput || accountChanged}
            >Edit exercise</button
          >
        {/if}
        {#if editAttempt?.id === attempt.id}
          <section class="edit-card" aria-label="Edit shared configuration">
            <div class="edit-copy">
              <p class="edit-label">Editing exercise</p>
              <p class="edit-help">
                Changes affect every session using this variant. For independent settings, create
                another variant in session editing.
              </p>
              <p class="edit-help">Update any field and save to apply only the changed values.</p>
            </div>
            <div class="edit-current-values">
              {#if mixedWeights}<p>
                  Mixed target weights · retained sets keep their own weights.
                </p>{/if}
              {#each editAttempt.currentVariant.targetSets as target (target.id)}<p>
                  Set {target.position}: {target.targetReps} reps · {target.targetWeight} kg
                </p>{/each}
            </div>
            <fieldset disabled={pending !== null || accountChanged} class="edit-fields">
              <label for="edit-exercise-name">Exercise name</label><input
                id="edit-exercise-name"
                bind:value={editFields.exerciseName}
              />
              <label for="edit-variant-name">Variant name</label><input
                id="edit-variant-name"
                bind:value={editFields.variantName}
              />
              <label for="edit-sets">Sets</label><input
                id="edit-sets"
                inputmode="numeric"
                bind:value={editFields.prescribedSets}
              />
              {#if shrinking}<p role="alert">
                  Reducing sets removes excess draft sets, including recorded reps and weights.
                  Completed history is kept.
                </p>{/if}
              <label for="edit-weight">Weight (kg)</label><input
                id="edit-weight"
                inputmode="decimal"
                placeholder={mixedWeights ? 'Mixed' : ''}
                bind:value={editFields.weight}
                oninput={() => {
                  applyWeight = false;
                }}
              />
              <button
                type="button"
                onclick={() => {
                  applyWeight = true;
                }}>Apply weight to all sets</button
              >
              <p>
                {applyWeight
                  ? 'This save will apply the entered weight to every prescribed set. Recorded weights stay unchanged.'
                  : 'Only Apply weight to all sets changes existing target weights.'}
              </p>
              <label for="edit-threshold">Rep threshold</label><input
                id="edit-threshold"
                inputmode="numeric"
                bind:value={editFields.repThreshold}
              />
              <label for="edit-increase">Auto increase</label><input
                id="edit-increase"
                inputmode="decimal"
                bind:value={editFields.autoIncrement}
              />
              <label for="edit-rest">Rest seconds</label><input
                id="edit-rest"
                inputmode="numeric"
                bind:value={editFields.restSeconds}
              />
            </fieldset>
            <div class="edit-actions">
              <button
                class="movement-b edit-secondary"
                onclick={() => {
                  editAttempt = null;
                  applyWeight = false;
                  error = '';
                }}
                disabled={pending !== null || accountChanged}>Cancel</button
              >
              <button
                class="movement-b edit-primary"
                onclick={submitConfiguration}
                disabled={pending !== null || accountChanged}>Save edit</button
              >
            </div>
          </section>
        {/if}
        {#each attempt.sets as set (set.id)}
          <div class="set">
            <h4>Set {set.position}</h4>
            <p>{set.targetReps} reps · {set.targetWeight} kg target</p>
            {#if set.state === 'performed'}<p>
                Performed: {set.recordedReps} reps · {set.recordedWeight} kg
              </p>
            {:else if set.state === 'skipped'}<p>Skipped · no performance recorded</p>
            {:else}<p>Pending · no performance recorded</p>{/if}
            {#if workout.status === 'active' && attempt.state === 'draft'}
              <label for={`reps-${set.id}`}>Reps for set {set.position}</label>
              <input
                id={`reps-${set.id}`}
                type="number"
                min="0"
                max="2147483647"
                step="1"
                value={candidates[set.id] ?? set.recordedReps ?? set.targetReps}
                oninput={(event) => {
                  candidates[set.id] = event.currentTarget.value;
                }}
                disabled={pending !== null || editAttempt !== null || accountChanged}
              />
              <button
                onclick={() => record(attempt, set, 'performed')}
                disabled={pending !== null || editAttempt !== null || accountChanged}
                >{set.state === 'performed' ? 'Save reps correction' : 'Record set'}</button
              >
              <button
                onclick={() => record(attempt, set, 'skipped')}
                disabled={pending !== null || editAttempt !== null || accountChanged}
                >Skip set</button
              >
              <button
                onclick={() => record(attempt, set, 'pending')}
                disabled={pending !== null ||
                  editAttempt !== null ||
                  accountChanged ||
                  set.state === 'pending'}>Reset to pending</button
              >
            {/if}
          </div>
        {/each}
        {#if workout.status === 'active' && attempt.state === 'draft'}
          {#if attempt.sets.some((s) => s.kind === 'prescribed' && s.state === 'pending')}
            <p>Resolve every pending set as performed or skipped before finishing.</p>
          {/if}
          <button
            onclick={() => finish(attempt)}
            disabled={hasInput ||
              accountChanged ||
              attempt.sets.some((s) => s.kind === 'prescribed' && s.state === 'pending')}
            >Finish</button
          >
        {:else if attempt.state === 'completed'}
          <div aria-label="Current next targets">
            <h4>Current next targets</h4>
            {#each attempt.currentVariant.targetSets as target (target.id)}
              <p>Set {target.position}: {target.targetReps} reps · {target.targetWeight} kg</p>
            {/each}
          </div>
        {/if}
      </section>
    {/each}
  {:else if error}<p role="alert">{error}</p>
    <button onclick={load} disabled={accountChanged}>Retry</button>{/if}
</main>

<style>
  main {
    width: 100%;
    padding: 4rem 1rem;
    box-sizing: border-box;
  }
  section {
    background: var(--surface-low);
    border-radius: 15px;
    padding: 1rem;
    margin: 1rem 0;
  }
  .set {
    padding: 0.75rem 0;
    border-top: 1px solid var(--border);
  }
  .edit-label {
    display: inline-flex;
    align-items: center;
    gap: 1rem;
    font-weight: 600;
    font-size: 1.5rem;
    color: var(--text);
  }

  .edit-card {
    width: 100%;
    margin-bottom: 1.5rem;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--surface-low);
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
    box-shadow: var(--shadow-liquid);
    transition-property: opacity;
    transition-timing-function: ease;
    transition-duration: 700ms;
    transition-delay: 150ms;
  }

  fieldset.edit-fields {
    border: 0;
    padding: 0;
    grid-template-columns: 1fr;
  }
</style>
