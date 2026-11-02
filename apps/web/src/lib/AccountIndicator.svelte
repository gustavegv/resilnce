<script lang="ts">
  import { user, upgradeRequired } from './stores/appState';
</script>

{#if $user?.developmentMode}
  <div class="development-account" role="status" aria-label="Development account" title={$user.email}>
    Development account<span class="sr-only"> · {$user.email}</span>
  </div>
{/if}

{#if $upgradeRequired}
  <div class="application-status" role="alert">
    <span>Refresh the application before saving.</span>
    <button type="button" onclick={() => window.location.reload()}>Refresh</button>
  </div>
{/if}

<style>
  .development-account,
  .application-status {
    position: fixed;
    top: 3.1rem;
    right: 0.75rem;
    z-index: 50;
    max-width: calc(100vw - 1.5rem);
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    padding: 0.25rem 0.5rem;
    background: var(--surface-middle, #242424);
    color: var(--foreground, white);
    font-size: 0.7rem;
    overflow-wrap: anywhere;
  }
  .development-account {
    top: 0.4rem;
    left: 3rem;
    right: auto;
    max-width: calc(50vw - 5rem);
    font-size: 0.6rem;
  }
  .application-status {
    top: 5.5rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  button {
    border: 1px solid var(--border);
    border-radius: 0.25rem;
    padding: 0.25rem 0.5rem;
  }
</style>
