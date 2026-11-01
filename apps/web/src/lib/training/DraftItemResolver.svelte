<script lang="ts">
  import { onMount } from 'svelte';
  import InputField from '../../components/InputField.svelte';
  import SetAutoIncrease from '../../components/SetAutoIncrease.svelte';
  import { searchExercises, readExercise } from './client';
  import type {
    DraftSessionItem,
    ExerciseSelection,
    ExerciseSuggestion,
    InitialConfiguration,
    Variant,
  } from './types';

  let {
    suggestion,
    onresolve,
    disabled = false,
  }: {
    suggestion?: ExerciseSuggestion;
    onresolve: (item: DraftSessionItem) => boolean;
    disabled?: boolean;
  } = $props();
  let name = $state('');
  let count = $state('');
  let weight = $state('');
  let threshold = $state(12);
  let increment = $state(2.5);
  let rest = 0;
  let muscleGroupCodes = $state<string[]>([]);
  let results = $state.raw<ExerciseSelection[]>([]);
  let selected = $state.raw<ExerciseSelection | null>(null);
  let mode = $state<'unresolved' | 'createExercise' | 'createVariant'>('unresolved');
  let variantName = $state('');
  let error = $state('');
  let searching = $state(false);
  let requestNumber = 0;

  onMount(() => {
    if (suggestion) {
      name = suggestion.name;
      muscleGroupCodes = suggestion.muscleGroupCodes ?? [];
      applyConfiguration(suggestion.configuration);
      void search();
    }
    return () => {
      requestNumber++;
    };
  });
  function applyConfiguration(configuration: InitialConfiguration) {
    count = String(configuration.prescribedSets);
    weight = configuration.initialWeight;
    threshold = configuration.repThreshold ?? 12;
    increment = Number(configuration.autoIncrement ?? '2.50');
    rest = configuration.restSeconds ?? 0;
  }
  async function search() {
    const request = ++requestNumber;
    selected = null;
    mode = 'unresolved';
    error = '';
    results = [];
    if (!name.trim()) {
      searching = false;
      return;
    }
    searching = true;
    try {
      const response = await searchExercises(name);
      if (request === requestNumber) results = response.exercises;
    } catch (failure) {
      if (request === requestNumber)
        error =
          failure instanceof Error
            ? failure.message
            : 'Unable to search your library. Retry search.';
    } finally {
      if (request === requestNumber) searching = false;
    }
  }
  function choose(exercise: ExerciseSelection) {
    requestNumber++;
    searching = false;
    selected = exercise;
    name = exercise.exercise.name;
    results = [];
    mode = 'unresolved';
    error = '';
  }
  function reset() {
    requestNumber++;
    name = '';
    count = '';
    weight = '';
    threshold = 12;
    increment = 2.5;
    rest = 0;
    muscleGroupCodes = [];
    results = [];
    selected = null;
    mode = 'unresolved';
    variantName = '';
    error = '';
    searching = false;
  }
  function publish(item: DraftSessionItem) {
    if (!onresolve(item)) return false;
    reset();
    return true;
  }
  function reuse(variant: Variant) {
    if (!selected || disabled) return;
    publish({
      draftId: crypto.randomUUID(),
      exercise: selected.exercise,
      reviewedVariant: variant,
      intent: {
        kind: 'reuseVariant',
        exerciseId: selected.exercise.id,
        variantId: variant.id,
        expectedVariantRevision: variant.revision,
      },
    });
  }
  export function hasInput() {
    return Boolean(name || count || weight || selected);
  }
  export function add(): boolean {
    if (disabled) return false;
    if (mode === 'unresolved') {
      error = 'Choose a variant to reuse or explicitly create an exercise or named variant.';
      return false;
    }
    const sets = Number(count);
    if (
      !name.trim() ||
      [...name.trim()].length > 100 ||
      !Number.isInteger(sets) ||
      sets < 1 ||
      sets > 20 ||
      !/^[0-9]+(?:\.[0-9]{1,2})?$/.test(weight) ||
      Number(weight) > 9999 ||
      !Number.isInteger(threshold) ||
      threshold < 1 ||
      threshold > 99 ||
      increment < 0.25 ||
      increment > 99
    ) {
      error =
        'Use a name, 1–20 sets, a weight from 0–9999, a threshold from 1–99 and an increment from 0.25–99.';
      return false;
    }
    const configuration: InitialConfiguration = {
      prescribedSets: sets,
      initialWeight: weight,
      repThreshold: threshold,
      autoIncrement: String(increment),
      restSeconds: rest,
    };
    if (mode === 'createVariant') {
      if (!selected || !variantName.trim() || [...variantName.trim()].length > 100) {
        error = 'Name the new variant (1–100 characters).';
        return false;
      }
      return publish({
        draftId: crypto.randomUUID(),
        exercise: selected.exercise,
        intent: {
          kind: 'createVariant',
          exerciseId: selected.exercise.id,
          expectedExerciseRevision: selected.exercise.revision,
          name: variantName.trim(),
          configuration,
        },
      });
    }
    return publish({
      draftId: crypto.randomUUID(),
      intent: { kind: 'createExercise', name: name.trim(), muscleGroupCodes, configuration },
    });
  }
  export async function edit(item: DraftSessionItem): Promise<boolean> {
    if (hasInput()) {
      error = 'Add the current exercise before editing another one.';
      return false;
    }
    if (item.intent.kind === 'createExercise') {
      name = item.intent.name;
      muscleGroupCodes = item.intent.muscleGroupCodes;
      applyConfiguration(item.intent.configuration);
      mode = 'createExercise';
      return true;
    }
    try {
      const current = await readExercise(item.intent.exerciseId);
      choose(current);
      if (item.intent.kind === 'createVariant') {
        mode = 'createVariant';
        variantName = item.intent.name;
        applyConfiguration(item.intent.configuration);
      } else {
        error = 'Review the current settings, then choose Reuse unchanged again.';
      }
      return true;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to review this exercise.';
      return false;
    }
  }
</script>

<fieldset {disabled}>
  <label class="exercise-search"
    >Exercise name
    <input
      aria-label="Exercise name"
      type="text"
      maxlength="100"
      bind:value={name}
      oninput={search}
      autocomplete="off"
    />
  </label>
  {#if searching}<p role="status">Searching your library…</p>{/if}
  {#if results.length}
    <ul aria-label="Personal exercises">
      {#each results as result (result.exercise.id)}
        <li>
          <button class="buttonClass minimal" onclick={() => choose(result)}
            >{result.exercise.name}</button
          >
        </li>
      {/each}
    </ul>
  {/if}
  {#if name.trim() && !selected}
    <button
      class="buttonClass minimal"
      onclick={() => {
        mode = 'createExercise';
        error = '';
      }}>Create new exercise “{name.trim()}” with Standard</button
    >
  {/if}
  {#if selected}
    <p>{selected.exercise.name}</p>
    {#if suggestion}<p>
        Suggested: {suggestion.configuration.prescribedSets} sets · {suggestion.configuration
          .initialWeight} kg · {suggestion.configuration.repThreshold ?? 12} reps · {suggestion
          .configuration.autoIncrement ?? '2.50'} kg increase · {suggestion.configuration
          .restSeconds ?? 0}s rest
      </p>{/if}
    {#each selected.variants as variant (variant.id)}
      <div class="variant">
        <strong>{variant.name}</strong>
        <p>
          Current: {variant.prescribedSets} sets · {variant.repThreshold} rep threshold · {variant.autoIncrement}
          kg increase · {variant.restSeconds}s rest
        </p>
        <p>
          {variant.targetSets
            .map((set) => `${set.targetReps} reps × ${set.targetWeight} kg`)
            .join(' / ')}
        </p>
        <button class="buttonClass minimal" onclick={() => reuse(variant)}
          >Reuse {variant.name} unchanged</button
        >
      </div>
    {/each}
    <button
      class="buttonClass minimal"
      onclick={() => {
        mode = 'createVariant';
        error = '';
      }}>Create named variant</button
    >
  {/if}
  {#if mode !== 'unresolved'}
    <p>
      {mode === 'createExercise'
        ? 'New exercise · Standard'
        : `New variant under ${selected?.exercise.name}`}
    </p>
    {#if mode === 'createExercise' && muscleGroupCodes.length}
      <p>New exercise muscle groups: {muscleGroupCodes.join(', ')}</p>
    {/if}
    {#if mode === 'createVariant'}<label
        >Variant name<input
          aria-label="Variant name"
          maxlength="100"
          bind:value={variantName}
        /></label
      >{/if}
    <InputField label="Sets" bind:value={count} type="number" />
    <InputField label="Weight" bind:value={weight} type="text" inputmode="decimal" />
    <SetAutoIncrease
      title="Auto-increase threshold"
      unit="reps"
      weight={threshold}
      limit={1}
      interval={1}
      onCountChange={(value) => (threshold = value)}
    />
    <SetAutoIncrease
      title="Weight increase intervals"
      weight={increment}
      onCountChange={(value) => (increment = value)}
    />
    <button class="buttonClass minimal" onclick={add}>Add to session</button>
  {/if}
  {#if error}<p role="alert">{error}</p>
    <button class="buttonClass minimal" onclick={search}>Retry search</button>{/if}
</fieldset>

<style>
  fieldset {
    border: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
    gap: 0.5rem;
    padding: 0;
  }
  .exercise-search {
    display: flex;
    flex-direction: column;
    width: 100%;
    max-width: 280px;
    gap: 0.25rem;
  }
  input {
    background: var(--surface-middle);
    border-radius: 3px;
    padding: 1rem;
    width: 100%;
    box-sizing: border-box;
  }
  .variant {
    width: 100%;
    padding: 0.75rem;
    background: var(--surface-middle);
    border-radius: var(--border-middle);
  }
  [role='alert'] {
    color: #ffb3b3;
  }
</style>
