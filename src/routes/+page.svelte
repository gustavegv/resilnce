<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { checkActiveSession, getOrderedExercises2 } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import { testDB } from '$lib/firebaseCreation';

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

  function gas() {
    console.log('Results from fetch:', getOrderedExercises2('user1', 'upper'));
  }
</script>

<div class="btn-container">
  <button class="base-btn sesh" onclick={() => goto('/tracker')}>Start new session</button>
  <button class="base-btn sesh" onclick={() => goto('/create')}>Create new session</button>
  <button class="base-btn sesh" onclick={testDB}>Test DB!</button>
  <button class="base-btn sesh" onclick={gas}>Test Data fetching!</button>

  {#if existingSession}
    <button class="base-btn alt" onclick={() => goto(`/tracker/${existingID}`)}>
      <h2>Continue session:</h2>
      <h3>{existingID}</h3>
    </button>
  {/if}
</div>

<style>
  .btn-container {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .base-btn {
    margin: 10px;
    padding: 10px 40px;
  }

  .base-btn.sesh {
    background-color: var(--color-secondary);
  }

  .base-btn.alt {
    background-color: var(--color-alt);
    color: var(--color-secondary);
  }
</style>
