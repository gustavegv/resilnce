<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { checkActiveSession } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import DbIcon from '../components/icons/DBIcon.svelte';
  import AddIcon from '../components/icons/AddIcon.svelte';

  let existingSession = $state(false);
  let existingID = $state('');

  onMount(async () => {
    const prevSession = await checkActiveSession();
    if (prevSession != null) {
      existingSession = prevSession.active;
      console.log('exists', existingSession);
      if (existingSession) {
        existingID = prevSession.session;
      }
    }
  });
</script>

<img src="books.png" alt="a" class="books" />

<div class="body">
  <h1 class="wid">The only tracker you need.</h1>
  <hr />
  <div class="btn-container">
    <button class="base-btn sesh" onclick={() => goto('/tracker')}>
      <g>Begin a workout</g>
      <DbIcon />
    </button>

    <button class="base-btn sesh" onclick={() => goto('/create')}>
      <g>Add new session</g>
      <AddIcon />
    </button>

    {#if existingSession}
      <button class="base-btn alt" onclick={() => goto(`/tracker/${existingID}`)}>
        <h2>Continue session:</h2>
        <h3>{existingID}</h3>
      </button>
    {/if}
  </div>
</div>

<style>
  .books {
    position: fixed; /* Stay fixed to the viewport */
    top: 30%;
    left: 50%;
    transform: translate(-50%, -50%); /* Adjust this to where you want it */
    z-index: 0; /* Sit behind interactive content */
    pointer-events: none; /* Let clicks pass through */
    user-select: none;
    width: 17rem; /* Adjust to taste */
    height: auto;
  }

  hr {
    height: 0.1px;
    background-color: #ffffff;
    margin: 20px 0;
    width: 80%;
  }

  .wid {
    width: 80%;
    font-size: 2.5rem;
    margin: 1rem 0;
  }

  .body {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-end;
    box-sizing: border-box;
    padding: 7rem 0;
    height: 100vh;
    background: var(--gradient-prim);
  }

  h1 {
    font-weight: 600;
    color: white;
  }

  .btn-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 90%;
  }

  .base-btn {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    -webkit-text-stroke: 0.6px rgb(255, 255, 255);
  }

  button {
    background-color: #151515;
    width: 90%;
  }

  .base-btn.sesh {
    box-shadow: rgba(0, 0, 0, 0.2) 0px 2px 8px 0px;
  }

  .base-btn.alt {
    background-color: var(--color-alt);
    color: var(--color-secondary);
  }
</style>
