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

  // ---------------------------
  // State (Svelte 5 runes)
  // ---------------------------
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
    // Use client-side nav for speed; <a> still included for semantics & fallback
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

<!--
  Layout notes:
  - The screen is split into two equal, stacked regions.
  - Each region is a large, tappable card (link + JS nav).
  - Works great on tall, vertical displays.
-->
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
            <Icon icon={mode.icon} class="icon" width="26" />
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
    --bg: #0b0c10;
    --card: rgb(26, 26, 26);
    --card-hover: rgba(255, 255, 255, 0.09);
    --border: rgba(255, 255, 255, 0.18);
    --text: #e8edf3;
    --muted: #b7c0cc;
    --accent: var(--color-secondary); /* soft azure */
    --accent-2: var(--color-gray-o); /* lavender */
    --ring: #9bb7ff;
    --shadow: 0 20px 60px rgba(0, 0, 0, 0.45);
  }

  .page {
    min-height: 100svh;
    background:
      radial-gradient(1200px 800px at 75% -10%, rgba(122, 162, 255, 0.18), transparent 60%),
      radial-gradient(1000px 700px at 20% 110%, rgba(160, 119, 255, 0.18), transparent 55%),
      linear-gradient(180deg, #0b0c10 0%, #0e1117 100%);
    display: grid;
    place-items: stretch;
  }

  .stack {
    height: 96svh;
    margin-top: 4svh;
    display: grid;
    grid-template-rows: 1fr 1fr; /* each box takes 50% */
  }

  .mode-card {
    position: relative;
    display: grid;
    place-items: stretch;
    text-decoration: none;
    isolation: isolate;
    outline: none;
  }

  .mode-card:focus-visible .card-surface,
  .mode-card:focus-visible .card-surface {
    transform: translateY(-2px) scale(1.01);
  }

  .card-surface {
    position: relative;
    inset: 0;
    display: grid;
    grid-template-columns: 1fr auto;
    align-items: center;
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

  .mode-card:hover .card-surface {
    background: var(--card-hover);
    transform: translateY(-3px);
    border-color: color-mix(in lab, var(--border) 60%, var(--accent) 40%);
  }

  .mode-card:active .card-surface {
    transform: translateY(0);
  }

  .icon-wrap {
    display: grid;
    place-items: center;
    width: clamp(64px, 12vw, 100px);
    height: clamp(64px, 12vw, 100px);
    border-radius: 18px;
    background: linear-gradient(
      135deg,
      color-mix(in lab, var(--accent) 65%, #fff 0%),
      color-mix(in lab, var(--accent-2) 65%, #fff 0%)
    );
    border: 1px solid rgba(255, 255, 255, 0.24);
    box-shadow:
      inset 0 1px 0 rgba(255, 255, 255, 0.28),
      0 10px 30px rgba(122, 162, 255, 0.22);
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
