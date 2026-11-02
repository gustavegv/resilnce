<script lang="ts">
  import Icon from '@iconify/svelte';
  import { slide } from 'svelte/transition';

  let { href, name }: { href: string; name: string } = $props();
</script>

<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Callers pass a path already resolved for the legacy or exercise-centric route. -->
<a {href} in:slide|global={{ duration: 600 }} class="base-btn alt">
  <div class="flex min-w-0 flex-1 flex-col text-left">
    <span class="font-bold">Resume session</span>
    <i class="truncate">{name}</i>
  </div>
  <Icon icon="material-symbols:resume-rounded" width={40} />
</a>

<style>
  .base-btn {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    width: 100%;
    min-height: 3.5rem;
    margin: 0;
    padding: 0.875rem 1rem;

    border: 1px solid rgb(255 255 255 / 0.08);
    border-radius: 12px;
    background: var(--color-secondary);
    color: white;

    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.22),
      inset 0 -1px 0 rgb(0 0 0 / 0.08),
      0 10px 24px rgb(0 0 0 / 0.18);

    transition:
      transform 120ms ease,
      box-shadow 180ms ease,
      filter 180ms ease,
      background-color 180ms ease;

    background:
      linear-gradient(to bottom, rgb(255 255 255 / 0.08), rgb(255 255 255 / 0)),
      var(--color-secondary);
  }

  .base-btn:hover {
    filter: brightness(1.03) saturate(1.02);
    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.24),
      inset 0 -1px 0 rgb(0 0 0 / 0.08),
      0 12px 28px rgb(0 0 0 / 0.2);
  }

  .base-btn:active {
    transform: scale(0.985);
    filter: brightness(0.98);
    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.16),
      inset 0 -1px 0 rgb(0 0 0 / 0.12),
      0 6px 14px rgb(0 0 0 / 0.16);
  }

  .base-btn:focus-visible {
    outline: 2px solid rgb(255 255 255 / 0.35);
    outline-offset: 2px;
  }

  .base-btn.alt {
    color: rgb(255 255 255 / 0.7);
    border: 1px solid rgb(255 255 255 / 0.1);
    background: rgb(255 255 255 / 0.05);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);

    width: 50%;
    font-size: 12px;
    border-radius: 16px;
    min-height: 4rem;
    opacity: 90%;
  }

  .base-btn.alt:hover {
    background: rgb(255 255 255 / 0.04);
    border-color: rgb(255 255 255 / 0.09);
  }

  .base-btn.alt:active {
    box-shadow: inset 0 1px 0 rgb(255 255 255 / 0.03);
  }
</style>
