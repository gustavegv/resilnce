<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import SessionSlug from '../../components/SessionSlug.svelte';
  import * as AlertDialog from "$lib/components/ui/alert-dialog/index.js";

  import { fade, scale } from 'svelte/transition';
  import Icon from '@iconify/svelte';
  import { resolve } from '$app/paths';

  import type { SessionMetaData } from './dbFetches';
  import { CheckActiveSession, GetSessions, SetActiveSession } from './dbFetches';
  import { toast } from 'svelte-sonner';

  let slugs: SessionMetaData[] = $state([]);
  let isAnotherSessionActive: boolean = $state(false);
  let sessionsLoaded: boolean = $state(false);

  let deletePopupShowing: boolean = $state(false);
  let itemToRemove: string = $state('');
  
  let activeSessionPopupShowing: boolean = $state(false);
  let sessionToStart: number = $state(-1)

  onMount(async () => {
    slugs = await GetSessions();

    const [activeID, activeName] = await CheckActiveSession();
    isAnotherSessionActive = activeID != '';
    sessionsLoaded = true;
  });


  function startSes(id: number) {
    if (isAnotherSessionActive) {
      activeSessionPopupShowing = true
      sessionToStart = id
    } else {
      goto(resolve(`/tracker/${id}`));
    }
  }

  function confirmStartSession(){
    const id = sessionToStart
    if (id != -1) {
        SetActiveSession(String(id));
        goto(resolve(`/tracker/${id}`));
    }
  }

  function editSes(id: number) {
    console.log(id, 'edited');
    alert('Edit not yet implemented.');
  }

  function deleteSession(SessionTitle: string) {
    deletePopupShowing = true;
    itemToRemove = SessionTitle
  }

  async function confirmDeleteSession(SessionTitle: string){
      console.log(SessionTitle, 'deleted.');
      deletePopupShowing = false
      await new Promise((r) => setTimeout(r, 300));
      deleteLocally(SessionTitle)
      toast.success(`Session ${SessionTitle} deleted!`)
  }

  // todo: Add real deletion too.
  function deleteLocally(id: string) {
    for (const item of slugs) {
      if (item.name === id) {
        item.deleted = true;
        break;
      }
    }
  }
</script>

<div class="main">
  <h1 class="mb-2 text-3xl leading-snug font-bold">Sessions</h1>
  <AlertDialog.Root bind:open={deletePopupShowing}>
  <AlertDialog.Content>
    <AlertDialog.Header>
    <AlertDialog.Title>Are you absolutely sure?</AlertDialog.Title>
    <AlertDialog.Description>
      This action cannot be undone. This will permanently delete the session
      and remove it from our servers.
    </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
    <AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
    <AlertDialog.Action 
      class="bg-red-500 text-white-500" 
      onclick={() => confirmDeleteSession(itemToRemove)}>
        Remove {itemToRemove}
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
  </AlertDialog.Root>

    <AlertDialog.Root bind:open={activeSessionPopupShowing}>
  <AlertDialog.Content>
    <AlertDialog.Header>
    <AlertDialog.Title>Another session is already active!</AlertDialog.Title>
    <AlertDialog.Description>
      Are you sure you want to start a new one?
    </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
    <AlertDialog.Cancel>No, take me back</AlertDialog.Cancel>
    <AlertDialog.Action 
      class="bg-green-600 text-white-500" 
      onclick={() => confirmStartSession()}>
        Yes, start new session
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
  </AlertDialog.Root>

  <hr />

  {#if slugs.length && sessionsLoaded}
    <div class="btn-container">
      {#each slugs as slug, i}
        {#if !slug.deleted}
          <div style="width: inherit" in:fade|global={{ delay: i * 50 }} out:scale>
            <SessionSlug
              onPress={() => startSes(slug.id)}
              onEdit={() => editSes(slug.id)}
              onDel={() => deleteSession(slug.name)}
              {slug}
            />
          </div>
        {/if}
      {/each}
    </div>
  {:else if !sessionsLoaded}
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
