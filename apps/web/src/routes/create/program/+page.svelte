<script lang="ts">
  import { base } from '$app/paths';
  import * as Collapsible from '$lib/components/ui/collapsible/index.js';

  import Icon from '@iconify/svelte';

  type Program = {
    id: string;
    slug: string;
    name: string;
    tagline: string;
    blockLength: string;
    cadence: string;
    level: string;
    type: string;
    focus: string;
    tags: string[];
    accent: 'violet' | 'sky' | 'emerald' | 'amber';
    overview: string;
    howItRuns: string[];
    bestFor: string[];
  };

  const programs: Program[] = [
    {
      id: '531',
      slug: '531',
      name: '5/3/1',
      tagline: 'Steady long-term strength progression',
      blockLength: '4-week block',
      cadence: '3-4 days / week',
      level: 'Beginner',
      type: 'Complete',
      focus: 'Main lifts + sustainable progress',
      tags: ['Strength', 'Classic'],
      accent: 'violet',
      overview:
        'A classic strength template built around squat, bench, deadlift, and press using conservative training maxes and repeatable monthly progression.',
      howItRuns: [
        'Each block runs across four weeks, with the main work progressing through 5s, 3s, and 5/3/1 style loading before a lighter reset or transition week.',
        'You train one main lift each session, then add supplemental work and a small amount of assistance based on your goal.',
        'The pace is intentionally sustainable, which makes it easy to run for multiple blocks without burning out.',
      ],
      bestFor: [
        'Building strength over months, not just one peak',
        'Lifters who want structure without excessive complexity',
      ],
    },
    {
      id: 'bbb',
      slug: 'boring-but-big',
      name: '5/3/1 Boring But Big',
      tagline: '5/3/1 with a big hypertrophy push',
      blockLength: '12-week block',
      cadence: '4 days / week',
      level: 'Intermediate',
      type: 'Complete',
      focus: 'Strength + size',
      tags: ['Hypertrophy', 'High volume', '5/3/1'],
      accent: 'sky',
      overview:
        'A higher-volume 5/3/1 variation that keeps the main strength work and adds 5 sets of 10 on supplemental lifts to drive muscle gain.',
      howItRuns: [
        'Each session starts with the regular 5/3/1 main lift work.',
        'After that, you perform 5x10 on the same lift or a paired lift depending on the setup you choose.',
        'Accessories stay simple so the extra volume can do the heavy lifting for growth and work capacity.',
      ],
      bestFor: [
        'Lifters who want size without abandoning strength work',
        'People who tolerate moderate to high barbell volume well',
      ],
    },
    {
      id: 'russian-squat',
      slug: 'russian-squat',
      name: 'Russian Strength Program',
      tagline: 'Short overreach cycle focused on one main lift',
      blockLength: '6-week block',
      cadence: '3 sessions / week',
      level: 'Intermediate',
      type: 'Partial',
      focus: 'Rapid peak for a chosen lift',
      tags: ['Strength', 'Specialization', 'Intensity'],
      accent: 'amber',
      overview:
        'A concentrated six-week strength cycle that can be applied to a single main lift, such as squat or bench, starting with repeated work around 80% and then tapering volume while intensity climbs toward a max test.',
      howItRuns: [
        'The first half builds a lot of volume on one chosen lift, commonly through repeated sets at roughly the same percentage while reps increase session to session.',
        'The second half shifts toward heavier, lower-rep work so you can express the strength built during the volume wave.',
        'Because the frequency and specificity are high, the rest of training usually needs to stay simple and fatigue-aware.',
      ],
      bestFor: [
        'Short-term specialization on a single lift',
        'Lifters who recover well from frequent practice on the same movement',
      ],
    },
    {
      id: 'candito-6-week',
      slug: 'candito-6-week',
      name: 'Candito 6 Week',
      tagline: 'Block-style strength cycle with changing weekly emphasis',
      blockLength: '6-week block',
      cadence: '4 days / week',
      level: 'Intermediate',
      type: 'Complete',
      focus: 'Strength peak across the big three',
      tags: ['Powerlifting', 'Periodized', '6 weeks'],
      accent: 'emerald',
      overview:
        'A six-week strength cycle that rotates emphasis from conditioning and hypertrophy into heavier strength and peaking work.',
      howItRuns: [
        'Each week has a distinct role, moving from higher workload into heavier, more specific strength work.',
        'The structure is more block-oriented than a static weekly template, so the feel of training changes across the cycle.',
        'It works well when you want a defined short block with a clear performance focus at the end.',
      ],
      bestFor: [
        'Intermediates who want a sharper peak',
        'Lifters who enjoy week-to-week variation',
      ],
    },
  ];

  let openPrograms = $state<Record<string, boolean>>(
    Object.fromEntries(programs.map((program, index) => [program.id, index === -1]))
  );
</script>

<section class="program-page">
  <header class="program-page__header">
    <div>
      <eyebrow>Program sessions</eyebrow>
      <h2>Start from a proven program</h2>
      <subtitle>
        Pick a training template and build your session from its structure. Expand any program to
        see what the block targets and how it is usually run.
      </subtitle>
    </div>

    <div class="program-page__helper">
      <span class="program-page__helper-badge">Structured templates</span>
      <p>
        Great for proven progressions like <strong>5/3/1</strong>, <strong>Russian Squat</strong>,
        and other fixed blocks.
      </p>
    </div>
  </header>

  <div class="program-list" role="list">
    {#each programs as program (program.id)}
      <article
        class={`program-card program-card--${program.accent}`}
        class:program-card--expanded={openPrograms[program.id]}
        role="listitem"
      >
        <Collapsible.Root bind:open={openPrograms[program.id]}>
          <Collapsible.Trigger class="program-card__trigger w-[100%]">
            <div class="program-card__main">
              <div class="program-card__row program-card__row--top">
                <div class="w-[100%]">
                  <h2 class="program-card__title">{program.name}</h2>
                  <p class="program-card__tagline">{program.tagline}</p>
                </div>
                <div class="program-card__tags">
                  <span class="program-card__length">{program.blockLength}</span>
                  <span class="program-card__length">{program.type}</span>
                </div>
              </div>

              <div class="program-card__meta">
                <span>{program.cadence}</span>
              </div>

              <div class="program-card__tags mt-4" aria-label="Program tags">
                <span class="program-card__tag {program.level.toLowerCase()}">{program.level}</span>
                {#each program.tags as tag}
                  <span class="program-card__tag">{tag}</span>
                {/each}
              </div>
            </div>

            <div class="program-card__chevron-wrap" aria-hidden="true">
              {#if openPrograms[program.id]}
                <Icon icon="si:expand-less-fill" class="program-card__chevron-icon" />
              {:else}
                <Icon icon="si:expand-more-fill" class="program-card__chevron-icon" />
              {/if}
            </div>
          </Collapsible.Trigger>

          <Collapsible.Content>
            <div class="program-card__panel">
              <div class="program-card__panel-grid">
                <div class="program-card__section">
                  <h3>Overview</h3>
                  <p>{program.overview}</p>
                </div>

                <div class="program-card__section">
                  <h3>How it runs</h3>
                  <ul>
                    {#each program.howItRuns as item}
                      <li>{item}</li>
                    {/each}
                  </ul>
                </div>

                <div class="program-card__section">
                  <h3>Best for</h3>
                  <ul>
                    {#each program.bestFor as item}
                      <li>{item}</li>
                    {/each}
                  </ul>
                </div>
              </div>

              <div class="program-card__actions">
                <button class="program-card__cta program-card__cta--ghost" type="button">
                  Preview week 1
                </button>
                <a
                  class="program-card__cta program-card__cta--primary"
                  href={`${base}/create/program/${program.slug}`}
                >
                  Use template
                </a>
              </div>
            </div>
          </Collapsible.Content>
        </Collapsible.Root>
      </article>
    {/each}
  </div>
</section>

<style>
  :global(:root) {
    --program-white-rgb: 255 255 255;
    --program-black-rgb: 0 0 0;

    --program-surface-top: var(--surface-middle);
    --program-surface-bottom: var(--surface-low);

    --program-text-primary: var(--color-text);
    --program-text-secondary: var(--text-muted);
    --program-text-muted: var(--program-text-secondary);

    --program-line-strong: rgb(255 255 255 / 0.12);
    --program-line-soft: rgb(255 255 255 / 0.08);
    --program-line-faint: rgb(255 255 255 / 0.06);

    --program-surface-soft: rgb(255 255 255 / 0.08);
    --program-surface-faint: rgb(255 255 255 / 0.05);
    --program-surface-pill: rgb(255 255 255 / 0.07);
    --program-surface-strong: var(--color-secondary);

    --program-shadow-soft: rgb(0 0 0 / 0.32);
    --program-shadow-strong: rgb(0 0 0 / 0.44);

    --program-focus: rgb(92 133 255 / 0.42);
  }

  .program-page {
    display: grid;
    gap: 1rem;
    color: var(--program-text-primary);
    padding: 1rem;
    padding-top: 4rem;
    overflow-x: auto;
  }

  .program-page__header {
    display: grid;
    gap: 1rem;
    padding: 0.35rem 0 0.4rem;
  }

  .program-page__title {
    font-size: 1.875rem;
    font-weight: 600;
    letter-spacing: -0.025em;
    color: var(--color-text);
  }

  .program-page__subtitle {
    margin: 0.9rem 0 0;
    max-width: 42rem;
    font-size: 0.98rem;
    line-height: 1.65;
    color: var(--program-text-secondary);
  }

  .program-page__helper {
    position: relative;
    overflow: hidden;
    border-radius: 1.5rem;
    padding: 1rem 1rem 1rem 1.05rem;
    border: 1px solid var(--program-line-soft);
    box-shadow:
      0 18px 54px var(--program-shadow-soft),
      inset 0 1px 0 rgb(255 255 255 / 0.08);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
  }

  .program-page__helper > * {
    position: relative;
    z-index: 1;
  }

  .program-page__helper p {
    margin: 0;
    font-size: 0.93rem;
    line-height: 1.55;
    color: var(--program-text-secondary);
  }

  .program-page__helper-badge {
    display: inline-flex;
    margin-bottom: 0.7rem;
    border-radius: 999px;
    padding: 0.4rem 0.72rem;
    border: 1px solid var(--program-line-faint);
    background: var(--program-surface-pill);
    font-size: 0.74rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    color: var(--program-text-primary);
  }

  .program-list {
    display: grid;
    gap: 0.9rem;
  }

  .program-card {
    position: relative;
    overflow: hidden;
    padding: 1rem;
    border-radius: 1.55rem;
    border: 1px solid var(--program-line-soft);
    box-shadow:
      0 18px 56px var(--program-shadow-soft),
      inset 0 1px 0 rgb(255 255 255 / 0.08);
    backdrop-filter: blur(18px);
    -webkit-backdrop-filter: blur(18px);
    transition:
      transform 220ms ease,
      border-color 220ms ease,
      box-shadow 220ms ease;
  }

  .program-card--expanded {
    border-color: var(--program-line-strong);
    box-shadow:
      0 24px 72px var(--program-shadow-strong),
      inset 0 1px 0 rgb(255 255 255 / 0.09);
  }

  /* Visible focus now lands on the card, not the Collapsible trigger itself */
  .program-card:has(.program-card__trigger:focus-visible),
  .program-card:has(.program-card__cta:focus-visible) {
    outline: 2px solid var(--program-focus);
    outline-offset: 2px;
  }

  /* Neutral layout wrapper only — no visible trigger styling */
  .program-card__trigger {
    position: relative;
    z-index: 1;
    display: grid;
    grid-template-columns: 0.3rem 1fr auto;
    gap: 0.9rem;
    width: 100%;
    padding: 0;
    margin: 0;
    border: 0;
    background: none;
    color: inherit;
    text-align: left;
    font: inherit;
    cursor: pointer;
    appearance: none;
    -webkit-appearance: none;
    box-shadow: none;
    border-radius: 0;
  }

  .program-card__trigger:focus-visible,
  .program-card__cta:focus-visible {
    outline: none;
  }

  .program-card__main {
  }

  .program-card__row--top {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }

  .program-card__title {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    line-height: 1.1;
    letter-spacing: -0.03em;
    text-align: left;
  }

  .program-card__tagline {
    margin: 0.35rem 0 0;
    font-size: 0.9rem;
    line-height: 1.45;
    text-align: left;
    color: var(--program-text-secondary);
  }

  .program-card__length {
    flex-shrink: 0;
    align-self: flex-start;
    border-radius: 999px;
    padding: 0.38rem 0.7rem;
    border: 1px solid var(--program-line-faint);
    background: var(--program-surface-pill);
    font-size: 0.76rem;
    font-weight: 700;
    white-space: nowrap;
    color: var(--program-text-primary);
  }

  .program-card__meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.42rem;
    margin-top: 0.78rem;
    font-size: 0.8rem;
    color: var(--program-text-muted);
  }

  .program-card__tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.45rem;
  }

  .program-card__tag {
    display: inline-flex;
    align-items: center;
    border-radius: 999px;
    padding: 0.34rem 0.64rem;
    border: 1px solid var(--program-line-faint);
    background: var(--program-surface-pill);
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--program-text-secondary);
  }

  .program-card__tag.beginner {
    border: 1px solid rgb(103, 198, 66);
    background-color: rgb(63, 90, 56);
  }

  .program-card__tag.intermediate {
    border: 1px solid rgb(198, 176, 66);
    background-color: rgb(89, 90, 56);
  }

  .program-card__tag.advanced {
    border: 1px solid rgb(198, 66, 66);
    background-color: rgb(90, 56, 56);
  }

  .program-card__chevron-wrap {
    display: grid;
    place-items: center;
    width: 2.5rem;
    margin: 1rem 1rem 1rem 0;
    color: var(--program-text-primary);
  }

  .program-card__chevron-icon {
    font-size: 1.15rem;
  }

  .program-card__panel {
    position: relative;
    z-index: 1;
    border-top: 1px solid var(--program-line-soft);
  }

  .program-card__panel-grid {
    display: grid;
    gap: 0.9rem;
    padding-top: 0.95rem;
  }

  .program-card__section {
    border-radius: 1.1rem;
    border: 1px solid var(--program-line-faint);
    background: var(--program-surface-faint);
    padding: 0.9rem 0.95rem;
  }

  .program-card__section h3 {
    margin: 0 0 0.55rem;
    font-size: 0.82rem;
    font-weight: 800;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--program-text-muted);
  }

  .program-card__section p,
  .program-card__section li {
    margin: 0;
    font-size: 0.92rem;
    line-height: 1.6;
    color: var(--program-text-secondary);
  }

  .program-card__section ul {
    display: grid;
    gap: 0.55rem;
    margin: 0;
    padding-left: 1rem;
  }

  .program-card__actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    padding-top: 0.95rem;
  }

  .program-card__cta {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 2.8rem;
    border-radius: 999px;
    padding: 0.72rem 1rem;
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 700;
    transition:
      transform 180ms ease,
      background-color 180ms ease,
      border-color 180ms ease;
  }

  .program-card__cta:hover {
    transform: translateY(-1px);
  }

  .program-card__cta--primary {
    color: var(--program-text-primary);
    border: 1px solid var(--program-line-soft);
    background: var(--program-surface-strong);
  }

  .program-card__cta--ghost {
    color: var(--program-text-secondary);
    border: 1px solid var(--program-line-faint);
  }

  @media (min-width: 820px) {
    .program-page__header {
      grid-template-columns: minmax(0, 1fr) minmax(18rem, 22rem);
      align-items: end;
    }

    .program-card__panel-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (max-width: 640px) {
    .program-page {
      gap: 0.85rem;
    }

    .program-page__helper {
      border-radius: 1.3rem;
      padding: 0.95rem;
    }

    .program-card {
      border-radius: 1.35rem;
    }

    .program-card__trigger {
      grid-template-columns: 0.24rem 1fr auto;
      gap: 0.75rem;
    }

    .program-card__row--top {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.7rem;
    }

    .program-card__length {
      align-self: flex-start;
    }

    .program-card__chevron-wrap {
      width: 2.2rem;
      margin: 0.95rem 0.95rem 0.95rem 0;
    }

    .program-card__actions {
      flex-direction: column;
    }

    .program-card__cta {
      width: 100%;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .program-card,
    .program-card__cta {
      transition: none;
    }
  }
</style>
