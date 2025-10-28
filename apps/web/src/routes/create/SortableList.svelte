<script lang="ts">
  import { onMount } from 'svelte';
  import Sortable from 'sortablejs';
  import type { SortableEvent } from 'sortablejs';
  import type { ExInfoPackage } from '$lib/firebaseCreation';
  import Icon from '@iconify/svelte';
  import { error } from '@sveltejs/kit';

  let {
    onConfirm,
    items,
  }: {
    onConfirm?: () => void;
    items: ExInfoPackage[];
  } = $props();

  let items2x = items;
  let els: HTMLElement;

  onMount(() => {
    const sortable = Sortable.create(els, {
      animation: 150,

      onEnd: () => {},
    });
  });

  function getListTexts(container: HTMLElement): ExInfoPackage[] {
    const collectedInfo: ExInfoPackage[] = [];

    for (const child of Array.from(container.children) as HTMLElement[]) {
      const exName = child.querySelector('#info-name');
      const exWeight = child.querySelector('#info-weight');
      const exSets = child.querySelector('#info-sets');
      const exAuto = child.querySelector('#info-auto');
      const exThres = child.querySelector('#info-thres');

      if (!exName || !exWeight || !exSets) {
        console.error('SortableList info not found.');
        return [];
      }

      let packaged: ExInfoPackage = {
        name: exName.textContent ?? 'Error',
        weight: Number(exWeight.textContent) ?? -1,
        sets: Number(exSets.textContent) ?? -1,
        autoIncrease: Number(exAuto?.textContent) ?? 2.5,
        repThreshold: Number(exThres?.textContent) ?? 12,
      };

      collectedInfo.push(packaged);
    }

    return collectedInfo;
  }

  export function extractData() {
    return getListTexts(els);
  }

  export function addToSortable(newEx: ExInfoPackage) {
    items2x.push(newEx);
  }

  function removeFromStack(name: string) {
    const index = items2x.findIndex((item) => item.name === name);
    if (index !== -1) items2x.splice(index, 1);
  }
</script>

<ul bind:this={els} class="component">
  {#each items2x as blob, i (blob)}
    <li class="blob-cont sortable-item">
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
      <div class="abs-icon">
        <Icon
          icon={'typcn:delete'}
          color={'red'}
          width="40"
          onclick={() => removeFromStack(blob.name)}
        />
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
    top: -12px;
    right: -15px;
    z-index: 1000; /* To sit on top of other things */
  }

  .blob-cont {
    position: relative;
    display: flex;
    box-sizing: border-box;
    padding: 1rem 2rem;
    flex-direction: row;
    background-color: var(--color-background);
    justify-content: space-between;
    align-items: center;
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
    width: 30%;
  }
  .additional-info {
    font-size: 12px;
    color: gray;
  }
</style>
