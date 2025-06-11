<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { checkActiveSession } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';

  let existingSession = $state(false)
  let existingID = $state("")


  onMount(async () => {
    const prevSession = await checkActiveSession();
    if (prevSession != null){
      existingSession = prevSession.active
      console.log("exists",existingSession)
      if (existingSession){
        existingID = prevSession.session
      }
    
    }

  });

</script>

<div class="btn-container">
  <button class="base-btn sesh" on:click={() => goto('/tracker')}>Start new session</button>
  <button class="base-btn sesh" on:click={() => goto('/tracker')}>Create new session</button>

  {#if existingSession}
    <button class="base-btn alt" on:click={() => goto(`/tracker/${existingID}`)}>
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
