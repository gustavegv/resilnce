<script lang="ts">
  import { onMount, tick } from 'svelte';
  import Icon from '@iconify/svelte';
  import InputField from '../../components/InputField.svelte';
  import SetAutoIncrease from '../../components/SetAutoIncrease.svelte';
  import ExerciseAutocomplete from './ExerciseAutocomplete.svelte';
  import { searchExercises, readExercise, readCatalog } from './client';
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
    onremove,
  }: {
    suggestion?: ExerciseSuggestion;
    onresolve: (item: DraftSessionItem) => boolean;
    disabled?: boolean;
    onremove?: () => void;
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
  let catalog = $state.raw<string[]>([]);
  let expanded = $state(false);
  const configurationID = $props.id();

  async function showVariantForm() {
    if (disabled || !selected) return;
    mode = 'createVariant';
    error = '';
    expanded = false;
    await tick();
    document.getElementById(`${configurationID}-variant-name`)?.focus();
  }

  async function showVariantPicker() {
    if (disabled) return;
    mode = 'unresolved';
    error = '';
    expanded = false;
    await tick();
    document.getElementById(`${configurationID}-create-variant`)?.focus();
  }

  function targetSummary(variant: Variant): string {
    const [first] = variant.targetSets;
    if (!first) return `${variant.prescribedSets} sets`;
    const uniform = variant.targetSets.every(
      (set) =>
        set.targetReps === first.targetReps &&
        Number(set.targetWeight) === Number(first.targetWeight)
    );
    const targets = uniform ? [first] : variant.targetSets;
    return `${variant.prescribedSets} sets · ${targets.map((set) => `${set.targetReps} reps × ${set.targetWeight} kg`).join(' / ')}`;
  }

  onMount(() => {
    let mounted = true;
    void readCatalog()
      .then((response) => {
        if (mounted) catalog = response.exerciseNames ?? [];
      })
      .catch(() => {
        // Name suggestions are optional; personal-library search remains available.
      });
    if (suggestion) {
      name = suggestion.name;
      muscleGroupCodes = suggestion.muscleGroupCodes ?? [];
      applyConfiguration(suggestion.configuration);
      expanded = true;
      void search();
    }
    return () => {
      mounted = false;
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
  async function search(composing = false) {
    const request = ++requestNumber;
    selected = null;
    mode = 'unresolved';
    error = '';
    results = [];
    if (composing) {
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
    mode = 'unresolved';
    error = '';
    expanded = false;
  }
  function create(exerciseName: string) {
    requestNumber++;
    searching = false;
    name = exerciseName;
    selected = null;
    mode = 'createExercise';
    expanded = false;
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
    expanded = false;
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
      expanded = false;
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
        error = 'Review the current settings, then select the variant again.';
      }
      return true;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to review this exercise.';
      return false;
    }
  }
</script>

{#snippet removeSuggestionButton()}
  <button
    type="button"
    class="remove-suggestion"
    aria-label="Remove this suggestion"
    title="Remove this suggestion"
    {disabled}
    onclick={onremove}><Icon icon="material-symbols:delete-outline-rounded" width={22} /></button
  >
{/snippet}

<fieldset {disabled}>
  <div class="exercise-search">
    <div class="exercise-search-field">
      <ExerciseAutocomplete
        bind:value={name}
        bind:expanded
        personal={results}
        {catalog}
        {searching}
        {disabled}
        onquery={(composing) => {
          void search(composing);
        }}
        onselect={(selection) => {
          if (typeof selection === 'string') create(selection);
          else choose(selection);
        }}
      />
    </div>
    {#if onremove && !selected && mode === 'unresolved'}{@render removeSuggestionButton()}{/if}
  </div>
  {#if searching}<p role="status">Searching your library…</p>{/if}
  {#if selected && mode !== 'createVariant'}
    <section class="variant-picker" aria-label={`Variants for ${selected.exercise.name}`}>
      <div class="variant-heading">
        <div>
          <span class="variant-eyebrow">Choose a variant</span>
          <p>{selected.exercise.name}</p>
        </div>
        {#if onremove && mode === 'unresolved'}{@render removeSuggestionButton()}{/if}
      </div>
      {#if suggestion}
        <div class="suggestion-comparison">
          <p class="comparison-title">
            Suggested: {suggestion.configuration.prescribedSets} sets · {suggestion.configuration
              .initialWeight} kg
          </p>
          <p>
            {suggestion.configuration.repThreshold ?? 12} reps · {suggestion.configuration
              .autoIncrement ?? '2.50'} kg increase · {`${suggestion.configuration.restSeconds ?? 0}s rest`}
          </p>
        </div>
      {/if}
      {#each selected.variants as variant (variant.id)}
        <button
          type="button"
          class="variant-choice"
          aria-label={`Reuse ${variant.name} unchanged`}
          onclick={() => reuse(variant)}
        >
          <span class="variant-title"
            ><strong>{variant.name}</strong><span class="variant-arrow" aria-hidden="true">→</span
            ></span
          >
          <span class="variant-targets">{targetSummary(variant)}</span>
          <span class="variant-settings"
            >Shared settings · {variant.repThreshold} rep threshold · {variant.autoIncrement} kg increase
            · {variant.restSeconds}s rest</span
          >
        </button>
      {/each}
      <button
        type="button"
        id={`${configurationID}-create-variant`}
        class="variant-create"
        aria-label="Create new variant"
        onclick={showVariantForm}
      >
        <span class="variant-plus" aria-hidden="true">+</span>
        <span
          ><strong>Create new variant</strong><span class="variant-create-copy"
            >Separate targets and progression</span
          ></span
        >
      </button>
    </section>
  {/if}
  {#if mode !== 'unresolved'}
    <div class="configuration" id={`${configurationID}-configuration`}>
      {#if mode === 'createVariant'}
        <button type="button" class="back-to-variants" {disabled} onclick={showVariantPicker}
          ><span aria-hidden="true">←</span> Back to variants</button
        >
      {/if}
      <p>
        {mode === 'createExercise'
          ? 'New exercise · Standard'
          : `New variant under ${selected?.exercise.name}`}
      </p>
      {#if mode === 'createExercise' && muscleGroupCodes.length}
        <p>New exercise muscle groups: {muscleGroupCodes.join(', ')}</p>
      {/if}
      {#if mode === 'createVariant'}<InputField
          label="Variant name"
          type="text"
          aria-label="Variant name"
          id={`${configurationID}-variant-name`}
          maxlength={100}
          bind:value={variantName}
        />{/if}
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
      <div class={['suggestion-actions', { 'with-remove': onremove !== undefined }]}>
        <button class="buttonClass minimal add-to-session" onclick={add}>Add to session</button>
        {#if onremove}{@render removeSuggestionButton()}{/if}
      </div>
    </div>
  {/if}
  {#if error}<p role="alert">{error}</p>
    <button
      class="buttonClass minimal"
      onclick={() => {
        expanded = true;
        void search();
      }}>Retry search</button
    >{/if}
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
  .variant-picker {
    display: flex;
    flex-direction: column;
    gap: 0.65rem;
    width: 100%;
  }
  .variant-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.5rem 0 0.15rem;
  }
  .exercise-search {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    width: 100%;
  }
  .exercise-search-field {
    flex: 1;
    min-width: 0;
    max-width: 280px;
  }
  .suggestion-actions.with-remove {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    max-width: 280px;
  }
  .suggestion-actions.with-remove .add-to-session {
    flex: 1;
    min-width: 0;
    margin: 0;
    height: 44px;
    min-height: 44px;
    padding: 0.65rem 1rem;
    font-size: 14px;
    line-height: 1.2;
  }
  .remove-suggestion {
    display: grid;
    place-items: center;
    flex: 0 0 44px;
    width: 44px;
    height: 44px;
    margin: 0;
    padding: 0;
    border: 1px solid var(--border);
    border-radius: var(--border-middle);
    background: transparent;
    box-shadow: none;
    color: var(--text-muted);
  }
  .remove-suggestion:hover {
    color: var(--destructive);
    border-color: var(--destructive);
  }
  .variant-eyebrow {
    color: var(--text-muted);
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .variant-heading p {
    margin: 0.15rem 0 0;
    font-size: 14px;
    font-weight: 600;
  }
  .suggestion-comparison {
    padding: 0.65rem 0.75rem;
    border-left: 2px solid var(--border);
    color: var(--text-muted);
    background: color-mix(in srgb, var(--surface-middle), transparent 70%);
    border-radius: 0 6px 6px 0;
  }
  .suggestion-comparison p {
    margin: 0;
    font-size: 11px;
    line-height: 1.5;
  }
  .suggestion-comparison .comparison-title {
    font-weight: 600;
  }
  .variant-choice,
  .variant-create {
    width: 100%;
    margin: 0;
    padding: 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--border-middle);
    box-shadow: none;
    text-align: left;
    color: var(--text);
  }
  .variant-choice {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    background: var(--surface-middle);
  }
  .variant-title {
    display: flex;
    width: 100%;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
  }
  .variant-title strong {
    font-size: 15px;
    font-weight: 600;
  }
  .variant-arrow {
    color: var(--text-muted);
    font-size: 17px;
    font-weight: 400;
  }
  .variant-targets {
    font-size: 12px;
    line-height: 1.5;
    font-weight: 400;
  }
  .variant-settings {
    color: var(--text-muted);
    font-size: 10px;
    line-height: 1.5;
    font-weight: 400;
  }
  .variant-choice:hover {
    border-color: var(--text-muted);
  }
  .variant-choice:focus-visible,
  .variant-create:focus-visible {
    outline: 2px solid var(--color-secondary);
    outline-offset: 2px;
  }
  .variant-create {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    background: transparent;
    border-style: dashed;
  }
  .variant-plus {
    color: var(--text-muted);
    font-size: 26px;
    font-weight: 300;
    line-height: 1;
  }
  .variant-create strong {
    font-size: 13px;
    font-weight: 500;
  }
  .variant-create-copy {
    display: block;
    color: var(--text-muted);
    font-size: 10px;
    line-height: 1.5;
    font-weight: 400;
  }
  .variant-create:hover {
    border-color: var(--text-muted);
  }
  .back-to-variants {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    align-self: flex-start;
    padding: 0.5rem 0;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .back-to-variants:hover {
    color: var(--text);
  }
  .back-to-variants:focus-visible {
    outline: 2px solid var(--color-secondary);
    outline-offset: 3px;
    border-radius: 4px;
  }
  .configuration {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
  }
  [role='alert'] {
    color: #ffb3b3;
  }
</style>
