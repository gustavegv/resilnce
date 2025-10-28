<script lang="ts">
  import Track from '../track.svelte';
  export let data: { exID: string };
</script>

<Track sesID={data.exID} />
