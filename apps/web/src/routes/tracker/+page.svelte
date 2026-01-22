<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import ErrorPopup from '../../components/ErrorPopup.svelte';
  import SessionSlug from '../../components/SessionSlug.svelte';

  import Popup from '../../components/Popup.svelte';
  import { fade } from 'svelte/transition';
  import Icon from '@iconify/svelte';
  import { resolve } from '$app/paths';

  import type { SessionMetaData } from './dbFetches';
  import { CheckActiveSession, GetSessions, SetActiveSession } from './dbFetches';

  let slugs: SessionMetaData[] = $state([]);
  let activeSession: boolean = $state(false);
  let loaded: boolean = $state(false);

  let showPopup: boolean = $state(false);

  let showError: string = $state('');

  onMount(async () => {
    slugs = await GetSessions();

    const [activeID, activeName] = await CheckActiveSession();
    activeSession = activeID != '';

    loaded = true;
  });

  function closePopup() {
    showPopup = false;
  }

  function openPopup(reason: string) {
    // showPopup = true;
    // popupResponse = reason
  }

  function startSes(id: number) {
    console.log('Exercise ID: (' + id, +') called for start.');

    if (activeSession) {
      openPopup('active');

      if (confirm(`You have an unfinished session, do you really want to start a new one?`)) {
        // todo switch popup to custom popup
        SetActiveSession(String(id));
        goto(resolve(`/tracker/${id}`));
      }
    } else {
      goto(resolve(`/tracker/${id}`));
    }
  }

  function editSes(id: number) {
    console.log(id, 'edited');
    alert('Edit not yet implemented.');
    // popup edit?
  }

  async function delSes(SessionTitle: string) {
    openPopup('delete');

    if (confirm(`Are you sure you want to delete ${SessionTitle}?`)) {
      alert('Delete not implemented');
      console.log(SessionTitle, 'deleted.');
    } else {
      console.log('Delete cancelled.');
    }
  }

  function deleteLocalSlug(id: number) {
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
              onDel={() => delSes(slug.name)}
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
