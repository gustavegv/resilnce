<script lang="ts">
  import { onMount } from 'svelte';
  import Sortable from 'sortablejs';
  import type { ExerciseDataPackaged } from './dbWrite';
  import Icon from '@iconify/svelte';
  import { toast, Toaster } from 'svelte-sonner';
  import ExerciseTrackScreen from '../tracker/ExerciseTrackScreen.svelte';
  import { error } from '@sveltejs/kit';

  type Props = {
    editData?: (payload: ExerciseDataPackaged) => void;
  };

  let { editData }: Props = $props();

  function sendEditData(pack: ExerciseDataPackaged) {
    editData?.(pack);
  }

  let exerciseList: ExerciseDataPackaged[] = $state([]);
  let els: HTMLElement;

  onMount(() => {
    const sortable = Sortable.create(els, {
      animation: 150,

      onEnd: () => {},
    });
  });

  function packageElement(child: HTMLElement): ExerciseDataPackaged {
    const exName = child.querySelector('#info-name');
    const exWeight = child.querySelector('#info-weight');
    const exSets = child.querySelector('#info-sets');
    const exAuto = child.querySelector('#info-auto');
    const exThres = child.querySelector('#info-thres');

    if (!exName || !exWeight || !exSets) {
      console.error('SortableList info not found.');
      throw new Error('SortableList info not found.');
    }

    let packaged: ExerciseDataPackaged = {
      name: exName.textContent ?? 'Error',
      weight: Number(exWeight.textContent) ?? -1,
      sets: Number(exSets.textContent) ?? -1,
      autoIncrease: Number(exAuto?.textContent) ?? 2.5,
      repThreshold: Number(exThres?.textContent) ?? 12,
    };
    return packaged;
  }

  function gatherSessionListFromDOM(container: HTMLElement): ExerciseDataPackaged[] {
    const collectedInfo: ExerciseDataPackaged[] = [];

    for (const child of Array.from(container.children) as HTMLElement[]) {
      const packagedChild = packageElement(child);
      collectedInfo.push(packagedChild);
    }

    return collectedInfo;
  }

  export function extractData() {
    return gatherSessionListFromDOM(els);
  }

  export function pushItemToList(newEx: ExerciseDataPackaged) {
    exerciseList.push(newEx);
  }

  export function fillFromArray(data: ExerciseDataPackaged[]) {
    data.forEach((exercise) => {
      pushItemToList(exercise);
    });
  }

  function removeFromStack(name: string, popup: boolean) {
    const index = exerciseList.findIndex((item) => item.name === name);
    if (index !== -1) {
      exerciseList.splice(index, 1);
      if (popup) toast.success('Exercise removed!');
    }
  }

  function editFromStack(name: string) {
    const child = document.getElementById(name) as HTMLElement;
    if (!child) throw new Error(`No element with id="${name}"`);
    const pack = packageElement(child);
    sendEditData(pack);

    removeFromStack(name, false);
  }

  let threeDotsOpenFor: string | null = $state(null);

  function openSelectionMenu(name: string) {
    threeDotsOpenFor = threeDotsOpenFor === name ? null : name;
    setTimeout(() => {
      threeDotsOpenFor = null;
    }, 3000);
  }
</script>

<ul bind:this={els} class="component">
  {#each exerciseList as blob, i (blob)}
    <li class="blob-cont sortable-item" id={blob.name}>
      <div class="name-cont">
        <h3 id="info-name">{blob.name}</h3>
      </div>
      <div class="blob-inner">
        <p>
          <span id="info-sets">{blob.sets}</span> sets
        </p>
        <i class="additional-info"> <span id="info-thres">{blob.repThreshold}</span> rep max.</i>
        <p>
          <span id="info-weight">{blob.weight}</span> kg
        </p>
        <i class="additional-info"> <span id="info-auto">{blob.autoIncrease}</span> kg inc.</i>
      </div>
      <button class="abs-icon" onclick={() => openSelectionMenu(blob.name)}>
        <Icon icon={'mingcute:more-3-line'} color={'grey'} width="30" />
      </button>
      <div
        class="abs-selection-box"
        class:open={threeDotsOpenFor === blob.name}
        id={blob.name + '-selection'}
      >
        <button onclick={() => editFromStack(blob.name)}>
          <Icon icon={'material-symbols:edit-outline'} color={'white'} width="24" />
        </button>
        <button onclick={() => removeFromStack(blob.name, true)}>
          <Icon icon={'material-symbols:delete-outline'} color={'red'} width="24" />
        </button>
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
    background-color: var(--color-background);
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
