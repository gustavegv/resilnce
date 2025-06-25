<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { checkActiveSession } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import DbIcon from '../components/icons/DBIcon.svelte';
  import AddIcon from '../components/icons/AddIcon.svelte';
  import { user } from './account/user';
  import Icon from '@iconify/svelte';
  import { blur, crossfade, fade, scale, slide } from 'svelte/transition';
  import { get } from 'svelte/store';
  import { base } from '$app/paths';
  const linkBase = import.meta.env.BASE_URL;

  let existingSession = $state(false);
  let existingID = $state('');

  onMount(async () => {
    const userID = get(user);
    if (userID) {
      const prevSession = await checkActiveSession(userID);

      if (prevSession != null) {
        existingSession = prevSession.active;
        console.log('exists', existingSession);
        if (existingSession) {
          existingID = prevSession.session;
        }
      }
    } else {
      goto(`${base}/account`);
    }
  });
</script>

<img src="{linkBase}books.png" alt="a" class="books {existingSession}" />
{#if $user}
  <div class="body">
    <h1 class="wid">Good evening <span class="toUpper">{$user}</span>.</h1>

    <hr />

    <div class="btn-container">
      {#if existingSession}
        <button
          in:slide|global={{ duration: 600 }}
          class="base-btn alt buttonClass"
          onclick={() => goto(`${base}/tracker/${existingID}`)}
        >
          <g>Continue session:</g>
          <h4>{existingID}</h4>
        </button>
      {/if}

      <button class="base-btn sesh buttonClass" onclick={() => goto(`${base}/tracker`)}>
        <g>Begin a workout</g>
        <DbIcon />
      </button>

      <button class="base-btn sesh buttonClass" onclick={() => goto(`${base}/create`)}>
        <g>Add new session</g>
        <AddIcon />
      </button>
    </div>
  </div>
{:else}
  <div class="body">
    <h1 class="wid">The only tracker you need.</h1>
    <hr />
    <button class="base-btn sesh buttonClass" onclick={() => goto(`${base}/account`)}>
      <g>Log in</g>
      <Icon icon="material-symbols:login-rounded" width="35" />
    </button>
  </div>
{/if}

<style>
  .books {
    position: fixed; /* Stay fixed to the viewport */
    top: 31%;
    left: 50%;
    transform: translate(-50%, -50%); /* Adjust this to where you want it */
    z-index: 0; /* Sit behind interactive content */
    pointer-events: none; /* Let clicks pass through */
    user-select: none;
    width: 16rem; /* Adjust to taste */
    height: auto;
  }

  .toUpper {
    text-transform: capitalize;
  }

  .books.true {
    top: 27%;
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
    padding: 5rem 0;
    height: 100vh;
  }

  h1 {
    font-weight: 600;
    color: var(--color-contrast);
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
    box-shadow: var(--shadow-dark);
    width: 90%;
  }

  .base-btn.sesh {
    -webkit-text-stroke: 0.6px rgb(255, 255, 255);
  }

  .base-btn.alt {
    background-color: var(--color-alt);
    color: var(--color-contrast);
    box-shadow: var(--color-alt) 0px 0px 20px 1px;
    height: 5rem;
  }
</style>
