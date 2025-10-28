<script lang="ts">
  import Icon from '@iconify/svelte';
  import { base } from '$app/paths';
  import { goto } from '$app/navigation';

  // ---------------------------
  // Types
  // ---------------------------
  type Mode = {
    id: 'manual' | 'quick';
    title: string;
    description: string;
    icon: string;
    href: string;
  };

  const modes: Mode[] = $state([
    {
      id: 'manual',
      title: 'Manual mode',
      description:
        'Start from scratch and enter everything yourself. Full control, perfect for creating new sessions.',
      icon: 'mdi:pencil-outline',
      href: `${base}/create/manual`,
    },
    {
      id: 'quick',
      title: 'Quick fill mode',
      description:
        'Import a workout from your notes and let our AI tool get you set up in seconds.',
      icon: 'mdi:wand',
      href: `${base}/create/quick`,
    },
  ]);

  // ---------------------------
  // Navigation helpers
  // ---------------------------
  function openRoute(href: string) {
    goto(href);
  }

  function onKeyActivate(event: KeyboardEvent, href: string) {
    const key = event.key;
    if (key === 'Enter' || key === ' ') {
      event.preventDefault();
      openRoute(href);
    }
  }
</script>

<div class="page">
  <div class="stack">
    {#each modes as mode (mode.id)}
      <a
        class="mode-card"
        href={mode.href}
        onclick={() => openRoute(mode.href)}
        role="button"
        tabindex="0"
        aria-label={`${mode.title}: ${mode.description}`}
        onkeydown={(e) => onKeyActivate(e, mode.href)}
      >
        <div class="card-surface">
          <div class="icon-wrap">
            <Icon icon={mode.icon} class="icon" width="28" />
          </div>

          <div class="content">
            <h2 class="title">{mode.title}</h2>
            <p class="description">{mode.description}</p>
          </div>
        </div>
      </a>
    {/each}
  </div>
</div>

<style>
  :global(:root) {
    --card: rgb(43, 43, 43);
    --card-hover: rgba(255, 255, 255, 0.09);
    --ring: #9bb7ff;
    --shadow: var(--shadow-dark);
  }

  .page {
    min-height: 100svh;
    background-color: var(--color-background);
    display: grid;
    place-items: stretch;
    padding-top: 1rem;
  }

  .stack {
    height: 75svh;
    padding-top: 6svh;
    display: grid;
    grid-template-rows: 1fr 1fr;
  }

  .mode-card {
    position: relative;
    display: grid;
    grid-template-rows: 1fr;

    place-items: stretch;
    text-decoration: none;
    isolation: isolate;
    outline: none;
    margin: 0.5rem 1rem;
  }

  .mode-card:focus-visible .card-surface,
  .mode-card:focus-visible .card-surface {
    transform: translateY(-2px) scale(1.01);
  }

  .card-surface {
    position: relative;
    inset: 0;
    display: grid;
    grid-template-columns: auto;
    border-radius: var(--radius-lg);

    align-items: baseline;

    gap: clamp(1rem, 2.5vw, 2rem);
    padding: clamp(1.25rem, 4vw, 2.5rem);
    background: var(--card);
    backdrop-filter: saturate(140%) blur(8px);
    transition:
      transform 260ms cubic-bezier(0.2, 0.8, 0.2, 1),
      background 220ms ease,
      border-color 220ms ease;
    box-shadow: var(--shadow);
  }

  .mode-card:active .card-surface {
    transform: translateY(0);
  }

  .icon-wrap {
    display: grid;
    place-items: center;
    width: clamp(64px, 16vw, 100px);
    height: clamp(64px, 16vw, 100px);
    border-radius: var(--radius-lg);
    background: linear-gradient(
      135deg,
      color-mix(in lab, var(--accent) 65%, #fff 0%),
      color-mix(in lab, var(--accent-2) 65%, #fff 0%)
    );
    border: 1px solid rgba(255, 255, 255, 0.24);
    box-shadow:
      inset 0 1px 0 rgba(163, 163, 163, 0.28),
      0 5px 20px rgba(122, 162, 255, 0.12);
  }

  .content {
    display: grid;
    gap: 0.4rem;
    align-content: center;
  }

  .title {
    font-size: clamp(1.4rem, 2.4vw, 1.75rem);
    line-height: 1.15;
    font-weight: 700;
    letter-spacing: 0.2px;
  }

  .description {
    font-size: clamp(0.975rem, 1.6vw, 1.05rem);
    line-height: 1.5;
    color: var(--accent-2);
    max-width: 70ch;
  }

  @media (max-height: 640px) {
    .description {
      display: -webkit-box;
      -webkit-line-clamp: 2;
      line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .card-surface {
      transition: none;
    }
  }
</style>
