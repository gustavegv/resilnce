<script lang="ts">
  import { onMount } from 'svelte';
  import '../../app.css';
  import Settings from './settings.svelte';
  import { goto } from '$app/navigation';

  onMount(() => {
    goto('/account');
  });
</script>

<div class="settings-frame">
  <Settings />
</div>

<style>
  .settings-frame {
    margin: 30px 60px;
  }
</style>
