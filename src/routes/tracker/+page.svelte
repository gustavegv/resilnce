<script lang="ts">
  import { goto } from "$app/navigation";
  import { getAllSessionMeta } from "$lib/firebaseDataHandler";
  import { onMount } from "svelte";

type SessionMetaData = {
  id: string,
  name: string,
}

// TODO: Put in db or smt
let slugs: SessionMetaData[];

onMount(async () => {
  slugs = await getAllSessionMeta();

  });
  
</script>

<div class="btn-container">
  {#each slugs as slug}
    <button class="base-btn sesh" on:click={() => goto(`/tracker/${slug.id}`)}>{slug.name}</button>

  {/each}
</div>

<style>


  
  .btn-container{
      display: flex;
      flex-direction: column;
      align-items: center;
  }
  
  .base-btn{
      margin: 10px;
      padding: 10px 40px;
  
  }
  
  .base-btn.sesh{
      background-color: var(--color-secondary);
  }
  
  .base-btn.alt{
      background-color: var(--color-alt);
      color: var(--color-secondary);
  
  
  }
</style>