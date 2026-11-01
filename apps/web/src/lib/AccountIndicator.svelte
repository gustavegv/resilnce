<script lang="ts">
  import { user, upgradeRequired } from './stores/appState';
</script>

{#if $user?.developmentMode}
  <div class="development-account" role="status" aria-label="Development account">
    Development account · {$user.email}
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
