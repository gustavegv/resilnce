<script lang="ts">
  import Sortable from 'sortablejs';
  import type { DraftSessionItem } from '$lib/training/types';
  import Icon from '@iconify/svelte';
  let {
    editData,
    disabled = false,
  }: { editData?: (item: DraftSessionItem) => boolean | Promise<boolean>; disabled?: boolean } =
    $props();
  let exerciseList = $state<DraftSessionItem[]>([]);
  let threeDotsOpenFor = $state<string | null>(null);
  function attachSortable(element: HTMLElement) {
    const sortable = Sortable.create(element, {
      animation: 150,
      onMove: () => !disabled,
      onEnd: () => {
        const ids = sortable.toArray();
        sortable.sort(exerciseList.map((item) => item.draftId));
        const byId = new Map(exerciseList.map((item) => [item.draftId, item]));
        exerciseList = ids.map((id) => byId.get(id)!);
      },
    });
    return () => sortable.destroy();
  }
  export function extractData(): DraftSessionItem[] {
    return $state.snapshot(exerciseList);
  }
  export function pushItemToList(item: DraftSessionItem) {
    exerciseList.push(item);
  }
  export function fillFromArray(items: DraftSessionItem[]) {
    exerciseList = items;
  }
  function remove(id: string) {
    exerciseList = exerciseList.filter((item) => item.draftId !== id);
  }
  async function edit(item: DraftSessionItem) {
    if (await editData?.(item)) remove(item.draftId);
  }
  function exerciseName(item: DraftSessionItem) {
    return item.intent.kind === 'createExercise'
      ? item.intent.name
      : (item.exercise?.name ?? 'Exercise');
  }
  function variantName(item: DraftSessionItem) {
    return item.intent.kind === 'reuseVariant'
      ? item.reviewedVariant?.name
      : item.intent.kind === 'createVariant'
        ? item.intent.name
        : 'Standard';
  }
</script>

<ul {@attach attachSortable} class="component">
  {#each exerciseList as item (item.draftId)}
    <li class="blob-cont sortable-item" data-id={item.draftId}>
      <div class="name-cont">
        <h3>{exerciseName(item)}</h3>
        <span class="additional-info">{variantName(item)}</span>
      </div>
      <div class="blob-inner">
        {#if item.intent.kind === 'reuseVariant' && item.reviewedVariant}
          <p>{item.reviewedVariant.prescribedSets} sets · Shared targets</p>
          <p>
            {item.reviewedVariant.targetSets
              .map((set) => `${set.targetReps} reps × ${set.targetWeight} kg`)
              .join(' / ')}
          </p>
        {:else if item.intent.kind !== 'reuseVariant'}
          <p>{item.intent.configuration.prescribedSets} sets</p>
          <i class="additional-info">{item.intent.configuration.repThreshold ?? 12} rep max.</i>
          <p>{item.intent.configuration.initialWeight} kg</p>
          <i class="additional-info">{item.intent.configuration.autoIncrement ?? '2.50'} kg inc.</i>
        {/if}
      </div>
      <button
        class="abs-icon"
        aria-label={`Options for ${exerciseName(item)}`}
        onclick={() => (threeDotsOpenFor = threeDotsOpenFor === item.draftId ? null : item.draftId)}
      >
        <Icon icon="mingcute:more-3-line" color="grey" width="30" />
      </button>
      <div class={['abs-selection-box', { open: threeDotsOpenFor === item.draftId }]}>
        <button aria-label={`Edit ${exerciseName(item)}`} onclick={() => edit(item)}
          ><Icon icon="material-symbols:edit-outline" color="white" width="24" /></button
        >
        <button aria-label={`Remove ${exerciseName(item)}`} onclick={() => remove(item.draftId)}
          ><Icon icon="material-symbols:delete-outline" color="red" width="24" /></button
        >
      </div>
    </li>
  {/each}
</ul>
<p class="text-sm text-gray-400 italic">(Drag exercises to reorder them)</p>

<style>
  .component {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }

  .abs-icon {
    position: absolute; /* Positioned relative to .container */
    padding: 4px;
    top: 5px;
    right: 5px;
    z-index: 1000; /* To sit on top of other things */
  }

  .abs-selection-box.open {
    opacity: 1;
    transform: scale(1);
    pointer-events: auto;
  }

  .abs-selection-box {
    display: flex;
    flex-direction: column;
    position: absolute;
    justify-content: space-around;
    top: 1px;
    right: 1px;
    background-color: var(--color-background);
    border-radius: 15px;
    height: 98%;
    padding: 5px 10px;
    z-index: 1001;

    opacity: 0;
    transform: scale(0.98);
    pointer-events: none;
    transition:
      opacity 180ms ease,
      transform 220ms ease;
  }

  .blob-cont {
    position: relative;
    display: flex;
    box-sizing: border-box;
    padding: 1rem 2rem;
    flex-direction: row;
    background-color: var(--surface-low);
    justify-content: space-between;
    width: 80%;
    border-radius: 15px;
    margin: 5px;

    box-shadow: var(--shadow-dark);
  }

  .name-cont {
    width: 70%;
    margin-right: 1rem;
  }

  .blob-inner {
    font-size: 12px;
    color: var(--color-secondary);
    width: 40%;
  }
  .additional-info {
    font-size: 12px;
    color: gray;
  }
</style>
