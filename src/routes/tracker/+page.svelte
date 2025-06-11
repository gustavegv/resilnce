<script lang="ts">
  import { goto } from "$app/navigation";

type SessionMetaData = {
  id: string,
  name: string,
}

// TODO: Put in db or smt
const slugs: SessionMetaData[] = [
  {id: "push", name: "Push-day"}, 
  {id: "pull", name: "Pull-day"}, 
  {id: "legs", name: "Leg-day"}, 
]

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