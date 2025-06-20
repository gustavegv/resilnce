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

  let slugs: SessionMetaData[] = $state([]);
  let activeSession: boolean = $state(false);

  let showPopup: boolean = $state(false);

  let showError: string = $state('');

  onMount(async () => {
    const data = await getAllSessionMeta('user1');
    slugs = data.slugs;
    activeSession = data.active;
  });

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
        goto(`/tracker/${id}`);
      } else {
        goto(`/`);
      }
    } else {
      goto(`/tracker/${id}`);
    }
  }
  function editSes(id: string) {
    console.log(id, 'edited');

    // popup edit?
  }

  async function delSes(id: string) {
    openPopup('delete');

    if (confirm(`Are you sure you want to delete ${id}?`)) {
      await fakeDeleteSession('user1', id);
      deleteLocalSlug(id);
      console.log(id, 'deleted.');
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
  <h2 style="width: 100%;" class="text-2xl font-bold">Sessions:</h2>
  <ErrorPopup message={showError}></ErrorPopup>
  <Popup show={showPopup} onAccept={() => handlePop(true)} onDecline={() => handlePop(false)}
  ></Popup>

  <hr />

  {#if slugs.length}
    <div class="btn-container">
      {#each slugs as slug, i}
        {#if !slug.deleted}
          <div in:fade|global={{ delay: i * 50 }}>
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
  {:else}
    <Icon icon="svg-spinners:3-dots-bounce" width="30" />
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
    box-sizing: border-box;
    width: 100%;
    text-align: left;
  }
</style>
