<script lang="ts">
  import { goto } from '$app/navigation';
  import { getAllSessionMeta } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';

  type SessionMetaData = {
    id: string;
    name: string;
  };

  // TODO: Put in db or smt
  let slugs: SessionMetaData[];

  onMount(async () => {
    slugs = await getAllSessionMeta('user1');
  });
</script>

<div class="btn-container">
  {#each slugs as slug}
    <button class="base-btn sesh" onclick={() => goto(`/tracker/${slug.id}`)}>{slug.id}</button>
  {/each}
</div>

<style>
  .btn-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    box-sizing: border-box;
    padding: 4rem 0;
    width: 100%;
  }
</style>
