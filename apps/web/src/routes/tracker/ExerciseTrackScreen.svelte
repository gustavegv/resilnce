<script lang="ts">
  import SetBlock from '../../components/SetBlock.svelte';
  import ConfirmSelection from '../../components/ConfirmSelection.svelte';
  import { get } from 'svelte/store';
  import { user } from '$lib/stores/appState';

  interface EditData {
    user: string;
    sesID: string;
    exID: string;
    oldName: string;
    newName?: string;
    newW?: number;
    addedSets: number;
  }

  let {
    name = $bindable(),
    weight = $bindable(),
    reps,
    finished,
    exIndex,
    onCount,
    onSubmit,
    edit,
    sesID,
    onCancel,
    exID,
  }: {
    name: string;
    weight: number;
    reps: number[];
    finished: boolean;
    exIndex: number;
    onCount: ({ id, count }: { id: number; count: number }) => void;
    onSubmit: () => void;
    edit?: boolean;
    sesID: string;
    onCancel?: () => void;
    exID?: string;
  } = $props();

  function uniqueKey(set: number, excerID: number) {
    return `${excerID}-s${set}`;
  }
</script>

<header>
  <h1 class="exercise-title">{name}</h1>
  <p class="exercise-subtitle">
    Target Weight: <strong>{weight} KG</strong>
  </p>
</header>

<cont class="cont">
  {#each reps as block, index (uniqueKey(index, exIndex))}
    <SetBlock id={index + 1} {finished} reps={block} onCountChange={onCount} />
  {/each}

  <ConfirmSelection {finished} onConfirm={onSubmit} />
</cont>

<style>
  .cont {
    width: 100%;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    align-items: center;
    overflow-y: auto;
    overflow-x: hidden;
    touch-action: pan-y;
    gap: 0.5rem;
  }

  header {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: baseline;
    margin-bottom: 2rem;
  }

  .exercise-title {
    margin: 0 0 0.25rem;
    font-size: 2.25rem;
    line-height: 1.1;
    font-weight: 700;
    letter-spacing: -0.02em;
    text-align: left;
  }

  .exercise-subtitle {
    margin: 0;
    font-size: 0.875rem;
    letter-spacing: 0.03em;
    color: var(--text-muted);
  }

  .exercise-subtitle strong {
    color: var(--color-alt);
    font-weight: 700;
  }
</style>
