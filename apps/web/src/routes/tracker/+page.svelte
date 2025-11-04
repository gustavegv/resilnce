<script lang="ts">
  import { goto } from '$app/navigation';
  import { getAllSessionMeta, fakeDeleteSession } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import ErrorPopup from '../../components/ErrorPopup.svelte';
  import SessionSlug from '../../components/SessionSlug.svelte';

  import type { SessionMetaData } from '$lib/firebaseDataHandler';

  import Popup from '../../components/Popup.svelte';
  import { fade } from 'svelte/transition';
  import Icon from '@iconify/svelte';
  import { resolve } from '$app/paths';

  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';
  import { updateDoc } from 'firebase/firestore';

  let slugs: SessionMetaData[] = $state([]);
  let activeSession: boolean = $state(false);
  let loaded: boolean = $state(false);

  let showPopup: boolean = $state(false);

  let showError: string = $state('');

  onMount(async () => {
    const uData = get(user);

    if (!uData || !uData.id) {
      goto(resolve(`/`));
      return;
    }

    const data = await getAllSessionMeta(uData.id);
    slugs = data.slugs;

    slugs = sortByDateSafe(slugs, 'desc');

    activeSession = data.active;
    loaded = true;
  });

  function sortByDateSafe<T extends { date?: string | Date }>(
    array: T[],
    direction: 'asc' | 'desc' = 'asc'
  ): T[] {
    return array.sort((a, b) => {
      const dateA = a.date ? new Date(a.date).getTime() : null;
      const dateB = b.date ? new Date(b.date).getTime() : null;

      if (dateA === null && dateB === null) return 0;
      if (dateA === null) return direction === 'asc' ? 1 : -1; // undefined dates go last in asc, first in desc
      if (dateB === null) return direction === 'asc' ? -1 : 1;

      return direction === 'asc' ? dateA - dateB : dateB - dateA;
    });
  }

  function delay(ms: number) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  function closePopup() {
    showPopup = false;
  }

  function openPopup(reason: string) {
    // showPopup = true;
    // popupResponse = reason
  }

  function editWorkout(sesID: string) {
    console.log('Edit', sesID + '!');
  }

  function startSes(id: string) {
    console.log(id, 'started');

    if (activeSession) {
      openPopup('active');

      if (confirm(`You have an unfinished exercise, do you really want to start a new one?`)) {
        goto(resolve(`/tracker/${id}`));
      } else {
        goto(resolve(`/`));
      }
    } else {
      goto(resolve(`/tracker/${id}`));
    }
  }
  function editSes(id: string) {
    console.log(id, 'edited');
    alert('Edit not yet implemented.');
    // popup edit?
  }

  async function delSes(SessionTitle: string) {
    openPopup('delete');
    const uData = get(user);
    if (!uData || !uData.id) return;

    if (confirm(`Are you sure you want to delete ${SessionTitle}?`)) {
      await fakeDeleteSession(uData.id, SessionTitle);
      deleteLocalSlug(SessionTitle);
      console.log(SessionTitle, 'deleted.');
    } else {
      console.log('Delete cancelled.');
    }

    // popup are you sure
  }

  function deleteLocalSlug(id: string) {
    for (const item of slugs) {
      if (item.id === id) {
        item.deleted = true;
        console.log('Found local copy and delted it');

        break;
      }
    }
  }

  function handlePop(accept: boolean) {
    if (accept) {
      console.log('accepted');
    } else {
      console.log('declined');
    }
    closePopup();
  }
</script>

<div class="main">
  <h1 class="mb-2 text-3xl leading-snug font-bold">Sessions</h1>
  <ErrorPopup message={showError}></ErrorPopup>
  <Popup show={showPopup} onAccept={() => handlePop(true)} onDecline={() => handlePop(false)}
  ></Popup>

  <hr />

  {#if slugs.length && loaded}
    <div class="btn-container">
      {#each slugs as slug, i}
        {#if !slug.deleted}
          <div style="width: inherit" in:fade|global={{ delay: i * 50 }}>
            <SessionSlug
              onPress={() => startSes(slug.id)}
              onEdit={() => editSes(slug.id)}
              onDel={() => delSes(slug.id)}
              {slug}
            />
          </div>
        {/if}
      {/each}
    </div>
  {:else if !loaded}
    <Icon icon="svg-spinners:3-dots-bounce" width="30" />
  {:else}
    <h2>No sessions created yet.</h2>
  {/if}
</div>

<style>
  .main {
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: left;
    padding: 5rem 1rem;
    overflow: scroll;
  }

  hr {
    height: 0.1px;
    background-color: #ffffff;
    margin: 20px 0;
    width: 100%;
  }

  .btn-container {
    display: flex;
    flex-direction: column;
    align-items: baseline;
    justify-content: center;
    box-sizing: border-box;
    width: 100%;
    text-align: left;
    gap: 0.5rem;
    overflow: hidden;
  }
</style>
