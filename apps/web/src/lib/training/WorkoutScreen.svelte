<script lang="ts">
  import { onMount } from 'svelte';
  import ExerciseTrackScreen from '../../routes/tracker/ExerciseTrackScreen.svelte';
  import ExerciseCompleteScreen from '../../components/ExerciseCompleteScreen.svelte';
  import InputField from '../../components/InputField.svelte';
  import Icon from '@iconify/svelte';
  import { beforeNavigate } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';
  import {
    readWorkout,
    recordSet,
    updateConfiguration,
    completeAttempt,
    quitWorkout,
    TrainingError,
  } from './client';
  import type {
    ConfigurationPatch,
    CompleteAttemptCommand,
    QuitWorkoutCommand,
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
  let currentExerciseIndex = $state(0);
  let currentAttempt = $derived(workout?.exercises[currentExerciseIndex]);
  function navigateExercise(index: number) {
    if (hasInput || !workout || index < 0 || index >= workout.exercises.length) return;
    currentExerciseIndex = index;
  }
  let error = $state('');
  let candidates = $state<Record<string, string>>({});
  let pending = $state.raw<
    | RecordSetCommand
    | UpdateConfigurationCommand
    | CompleteAttemptCommand
    | QuitWorkoutCommand
    | null
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
  let confirmingQuit = $state(false);
  let actionsOpen = $state(false);
  let quitPending = $derived(pending !== null && !('attemptId' in pending));
  let completionPending = $derived(
    pending !== null && 'attemptId' in pending && !('patch' in pending) && !('setId' in pending)
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
  let uncertainFinality = $derived((completionPending || quitPending) && !conflict);
  let navigationBlocked = $state(false);
  let hasInput = $derived(
    pending !== null || confirmingQuit || editorDirty || Object.keys(candidates).length > 0
  );
  let canReapply = $derived.by(() => {
    if (!pending || !reviewed || reviewed.status !== 'active') return false;
    if (!('attemptId' in pending)) return true;
    const attemptId = pending.attemptId;
    const attempt = reviewed.exercises.find((a) => a.id === attemptId);
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
          : 'attemptId' in intent
            ? completeAttempt(intent)
            : quitWorkout(intent));
      if (!sameAccount() || workoutId !== intent.workoutId) {
        saveState = 'failed';
        return;
      }
      workout = result.workout;
      if (
        'attemptId' in intent &&
        !('patch' in intent) &&
        !('setId' in intent) &&
        workout.status === 'active'
      ) {
        const next = workout.exercises.findIndex((attempt) => attempt.state === 'draft');
        if (next >= 0) currentExerciseIndex = next;
      }
      if ('patch' in intent) {
        editAttempt = null;
        applyWeight = false;
      } else if ('setId' in intent) delete candidates[intent.setId];
      rejectedConfiguration = false;
      pending = null;
      confirmingQuit = false;
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
      const rejectedFinality =
        !('patch' in intent) &&
        !('setId' in intent) &&
        failure instanceof TrainingError &&
        ['invalid-field', 'malformed-request', 'resource-unavailable', 'not-found'].includes(
          failure.code
        );
      conflict =
        failure instanceof TrainingError &&
        ['stale-revision', 'resource-finalized'].includes(failure.code);
      if (rejectedFinality) {
        pending = null;
        navigationBlocked = false;
      }
      if (conflict) await reviewCurrent();
    }
  }
  function confirmQuit() {
    if (
      !confirmingQuit ||
      pending ||
      editorDirty ||
      Object.keys(candidates).length > 0 ||
      !workout ||
      workout.status !== 'active' ||
      !sameAccount()
    )
      return;
    pending = {
      operationId: crypto.randomUUID(),
      workoutId: workout.id,
      expectedWorkoutRevision: workout.revision,
    };
    confirmingQuit = false;
    void save();
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
      confirmingQuit ||
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
    actionsOpen = false;
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
  function originalEditValue(key: keyof typeof editFields): string {
    if (!editAttempt) return '';
    const variant = editAttempt.currentVariant;
    if (key === 'exerciseName') return editAttempt.exercise.name;
    if (key === 'variantName') return variant.name;
    if (key === 'weight')
      return new Set(variant.targetSets.map((set) => set.targetWeight)).size === 1
        ? (variant.targetSets[0]?.targetWeight ?? '')
        : '';
    return String(variant[key]);
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
    if (!('attemptId' in pending)) {
      pending = {
        ...pending,
        operationId: crypto.randomUUID(),
        expectedWorkoutRevision: reviewed.revision,
      };
      workout = reviewed;
      reviewed = null;
      conflict = false;
      void save();
      return;
    }
    const attemptId = pending.attemptId;
    const attempt = reviewed.exercises.find((a) => a.id === attemptId)!;
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
    confirmingQuit = false;
    editAttempt = null;
    applyWeight = false;
    rejectedConfiguration = false;
    candidates = {};
    reviewed = null;
    conflict = false;
    saveState = 'saved';
    error = '';
    navigationBlocked = false;
  }
  onMount(() => {
    void load();
  });
</script>

<svelte:window onbeforeunload={beforeUnload} />
<main class="app-container" aria-label={workout?.sourceSession?.name ?? 'Workout'}>
  {#if accountChanged}<p role="alert">
      Your account changed or expired. Local input is retained. Return to the original account
      before retrying.
    </p>{/if}
  {#if loading}<p role="status">Loading workout…</p>
  {:else if workout}
    <div class="progress">
      <div class="progress-header">
        <span class="label">Session Progress</span><span class="label label--accent"
          >Exercise {workout.status === 'ended'
            ? workout.exercises.length
            : currentExerciseIndex + 1} of {workout.exercises.length}</span
        >
      </div>
      <div class="progress-track">
        <div
          class="progress-fill"
          style:width={`${workout.status === 'ended' ? 100 : (workout.exercises.filter((attempt) => attempt.state === 'completed').length / workout.exercises.length) * 100}%`}
        ></div>
      </div>
    </div>
    <p role="status" class="save-status">
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
          >Accept server state</button
        >
        <button onclick={reapply} disabled={!canReapply || accountChanged}
          >Reapply local input to reviewed revision</button
        >
      {:else}<button onclick={reviewCurrent} disabled={accountChanged}>Review current state</button
        >{/if}
    {/if}
    {#if navigationBlocked}
      <p role="alert">
        {uncertainFinality
          ? `${quitPending ? 'Quit' : 'Completion'} is unconfirmed. Stay and retry to resolve it before leaving or starting another workout.`
          : 'Input is not saved. Stay and resolve it before leaving.'}
      </p>
      <button
        onclick={() => {
          navigationBlocked = false;
        }}>Stay</button
      >
    {/if}
    {#if uncertainFinality && saveState === 'failed'}<p role="alert">
        {quitPending ? 'Quit' : 'Completion'} is unconfirmed. Retry the same {quitPending
          ? 'quit'
          : 'completion'} to resolve it before leaving.
      </p>{/if}
    {#if workout.status === 'ended'}
      <h1>Session finished!</h1>
      <h3 class="subtitle">Session overview</h3>
      <hr />
      <ExerciseCompleteScreen
        attempts={workout.exercises.filter((attempt) => attempt.state === 'completed')}
      />
      <a class="buttonClass home-button" href={resolve('/')}>Return to homepage</a>
    {:else}
      <details class="session-menu-anchor" bind:open={actionsOpen}>
        <summary aria-label="Session actions">...</summary>
        <div class="session-action-menu">
          <button
            onclick={() => currentAttempt && openEdit(currentAttempt)}
            disabled={hasInput || accountChanged || currentAttempt?.state !== 'draft'}
            >Edit exercise</button
          >
          <button
            onclick={() => {
              if (!hasInput && sameAccount()) {
                actionsOpen = false;
                confirmingQuit = true;
              }
            }}
            disabled={hasInput || accountChanged}>Quit workout</button
          >
        </div>
      </details>
      {#if confirmingQuit}
        <section class="quit-card" aria-label="Confirm quit workout">
          <p>
            Completed work and progression are kept. Unfinished attempts and their sets are
            discarded. Shared configuration edits are kept.
          </p>
          <button
            class="buttonClass"
            onclick={confirmQuit}
            disabled={pending !== null || accountChanged}>Confirm quit</button
          >
          <button onclick={() => (confirmingQuit = false)} disabled={pending !== null}
            >Keep training</button
          >
        </section>
      {/if}
      {#each currentAttempt ? [currentAttempt] : [] as attempt (attempt.id)}
        <div
          id={`attempt-${attempt.id}`}
          class="attempt"
          aria-label={`${attempt.exercise.name} ${attempt.currentVariant.name}`}
        >
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
                <InputField
                  label="Exercise name"
                  id="edit-exercise-name"
                  type="text"
                  bind:value={editFields.exerciseName}
                />
                <button
                  type="button"
                  class="edit-undo"
                  class:off={editFields.exerciseName === originalEditValue('exerciseName')}
                  aria-label="Undo exercise name"
                  onclick={() => {
                    editFields.exerciseName = originalEditValue('exerciseName');
                  }}><Icon icon="ic:round-undo" /></button
                >
                <InputField
                  label="Variant name"
                  id="edit-variant-name"
                  type="text"
                  bind:value={editFields.variantName}
                />
                <button
                  type="button"
                  class="edit-undo"
                  class:off={editFields.variantName === originalEditValue('variantName')}
                  aria-label="Undo variant name"
                  onclick={() => {
                    editFields.variantName = originalEditValue('variantName');
                  }}><Icon icon="ic:round-undo" /></button
                >
                <InputField
                  label="Sets"
                  id="edit-sets"
                  type="text"
                  inputmode="numeric"
                  bind:value={editFields.prescribedSets}
                />
                <button
                  type="button"
                  class="edit-undo"
                  class:off={editFields.prescribedSets === originalEditValue('prescribedSets')}
                  aria-label="Undo sets"
                  onclick={() => {
                    editFields.prescribedSets = originalEditValue('prescribedSets');
                  }}><Icon icon="ic:round-undo" /></button
                >
                {#if shrinking}<p role="alert">
                    Reducing sets removes excess draft sets, including recorded reps and weights.
                    Completed history is kept.
                  </p>{/if}
                <InputField
                  label="Weight (kg)"
                  id="edit-weight"
                  type="text"
                  inputmode="decimal"
                  bind:value={editFields.weight}
                  oninput={() => {
                    applyWeight = false;
                  }}
                />
                <button
                  type="button"
                  class="edit-undo"
                  class:off={editFields.weight === originalEditValue('weight')}
                  aria-label="Undo weight (kg)"
                  onclick={() => {
                    editFields.weight = originalEditValue('weight');
                    applyWeight = false;
                  }}><Icon icon="ic:round-undo" /></button
                >
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
                <InputField
                  label="Rep threshold"
                  id="edit-threshold"
                  type="text"
                  inputmode="numeric"
                  bind:value={editFields.repThreshold}
                />
                <button
                  type="button"
                  class="edit-undo"
                  class:off={editFields.repThreshold === originalEditValue('repThreshold')}
                  aria-label="Undo rep threshold"
                  onclick={() => {
                    editFields.repThreshold = originalEditValue('repThreshold');
                  }}><Icon icon="ic:round-undo" /></button
                >
                <InputField
                  label="Auto increase"
                  id="edit-increase"
                  type="text"
                  inputmode="decimal"
                  bind:value={editFields.autoIncrement}
                />
                <button
                  type="button"
                  class="edit-undo"
                  class:off={editFields.autoIncrement === originalEditValue('autoIncrement')}
                  aria-label="Undo auto increase"
                  onclick={() => {
                    editFields.autoIncrement = originalEditValue('autoIncrement');
                  }}><Icon icon="ic:round-undo" /></button
                >
                <InputField
                  label="Rest seconds"
                  id="edit-rest"
                  type="text"
                  inputmode="numeric"
                  bind:value={editFields.restSeconds}
                />
                <button
                  type="button"
                  class="edit-undo"
                  class:off={editFields.restSeconds === originalEditValue('restSeconds')}
                  aria-label="Undo rest seconds"
                  onclick={() => {
                    editFields.restSeconds = originalEditValue('restSeconds');
                  }}><Icon icon="ic:round-undo" /></button
                >
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
          <ExerciseTrackScreen
            {attempt}
            finished={attempt.state === 'completed'}
            {candidates}
            disabled={pending !== null || confirmingQuit || editAttempt !== null || accountChanged}
            finishDisabled={hasInput ||
              accountChanged ||
              attempt.sets.some((set) => set.kind === 'prescribed' && set.state === 'pending')}
            onSubmit={() => finish(attempt)}
            onCandidate={(set, value) => {
              candidates[set.id] = value;
            }}
            onRecord={(set, state, count) => {
              if (count !== undefined && !(set.id in candidates))
                candidates[set.id] = String(count);
              record(attempt, set, state);
            }}
          />
          {#if attempt.state === 'completed'}
            <div aria-label="Current next targets">
              <h4>Current next targets</h4>
              {#each attempt.currentVariant.targetSets as target (target.id)}
                <p>Set {target.position}: {target.targetReps} reps · {target.targetWeight} kg</p>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
      <nav class="movement-cont" aria-label="Exercises">
        <button
          class="movement-b"
          class:inactive={currentExerciseIndex === 0}
          disabled={hasInput || currentExerciseIndex === 0}
          onclick={() => navigateExercise(currentExerciseIndex - 1)}
          ><Icon icon="material-symbols:arrow-left-alt-rounded" /><span>Prev</span></button
        >
        <button
          class="movement-b long"
          class:inactive={currentAttempt?.state === 'draft'}
          disabled={hasInput || workout.exercises.every((attempt) => attempt.state === 'completed')}
          onclick={() =>
            navigateExercise(workout!.exercises.findIndex((attempt) => attempt.state === 'draft'))}
          >Go to current</button
        >
        <button
          class="movement-b"
          class:inactive={currentExerciseIndex === workout.exercises.length - 1}
          disabled={hasInput || currentExerciseIndex === workout.exercises.length - 1}
          onclick={() => navigateExercise(currentExerciseIndex + 1)}
          ><span>Skip</span><Icon icon="material-symbols:arrow-right-alt-rounded" /></button
        >
      </nav>
    {/if}
  {:else if error}<p role="alert">{error}</p>
    <button onclick={load} disabled={accountChanged}>Retry</button>{/if}
</main>

<style>
  main > button {
    background: var(--surface-middle);
    border: 1px solid var(--border);
    padding: 0.65rem 1rem;
    margin: 0.35rem 0;
    border-radius: 10px;
    font-size: 0.8rem;
  }
  main > [role='alert'] {
    width: 100%;
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 0.75rem;
    background: var(--surface-low);
    font-size: 0.8rem;
    text-align: left;
    margin-bottom: 0.5rem;
  }
  section[aria-label='Current server state'] {
    width: 100%;
    padding: 1rem;
    border-radius: 12px;
    background: var(--surface-low);
    font-size: 0.8rem;
    text-align: left;
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

  .attempt {
    width: 100%;
  }
  .save-status {
    font-size: 0.7rem;
    color: var(--text-muted);
    margin: -1.65rem 0 1.5rem;
    width: 100%;
    text-align: left;
  }
  .session-menu-anchor {
    position: absolute;
    top: 0.5rem;
    right: 1rem;
    z-index: 20;
  }
  .session-menu-anchor summary {
    list-style: none;
    cursor: pointer;
    padding: 1rem;
    font-size: 1.5rem;
    color: var(--text-muted);
  }
  .session-action-menu {
    position: absolute;
    right: 0;
    min-width: 10rem;
    background: var(--surface-low);
    border: 1px solid var(--border);
    padding: 0.5rem;
    border-radius: 12px;
    box-shadow: var(--shadow-dark);
  }
  .session-action-menu button {
    padding: 0.75rem;
    width: 100%;
    text-align: left;
    font-size: 0.8rem;
  }
  .quit-card {
    padding: 1rem;
    border-radius: 15px;
    background: var(--surface-low);
    font-size: 0.85rem;
    margin-bottom: 1rem;
  }
  .home-button {
    display: block;
    margin-top: 1rem;
    padding: 0.75rem;
    width: 100%;
  }
  hr {
    border: 1px solid var(--text-muted);
    margin: 10px 0;
    width: 100%;
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
  }
  .edit-fields > p {
    grid-column: 1 / -1;
  }
  .edit-undo {
    background: transparent;
  }
  .edit-undo.off {
    opacity: 0;
    pointer-events: none;
    visibility: hidden;
  }
  .movement-b.inactive {
    visibility: hidden;
  }
  .movement-b.long {
    width: 40%;
    font-size: 12px;
  }
  .movement-b span {
    font-size: 12px;
  }
</style>
