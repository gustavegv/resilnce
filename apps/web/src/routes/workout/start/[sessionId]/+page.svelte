<script lang="ts">
  import { page } from '$app/state';
  import WorkoutStartScreen from '$lib/training/WorkoutStartScreen.svelte';
</script>

{#if page.params.sessionId}
  {#key page.params.sessionId}
    <WorkoutStartScreen sessionId={page.params.sessionId} />
  {/key}
{/if}
