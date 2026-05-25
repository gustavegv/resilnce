<script lang="ts">
  import { base } from '$app/paths';
  import * as Collapsible from '$lib/components/ui/collapsible/index.js';


  import Icon from '@iconify/svelte';
    import { advancedPrograms, hypertrophyPrograms } from './programs';
    import { Button } from '$lib/components/ui/button';

 import { LineChart } from "layerchart";
  import { curveLinearClosed } from "d3-shape";
  import { scaleBand } from "d3-scale";
  import * as Chart from "$lib/components/ui/chart/index.js";
  import * as Card from "$lib/components/ui/card/index.js";


  const chartConfig = {
    volume: {
      label: "Training Volume",
      color: "var(--color-secondary)",
    },
  } satisfies Chart.ChartConfig;

  let hypActv = $state(true)

  function getSessions(){
    return hypActv ? hypertrophyPrograms : advancedPrograms
  }

  let programs = $derived(getSessions())

let openOverrides = $state<Record<string, boolean>>({});

let openPrograms = $derived.by(() =>
  Object.fromEntries(
    programs.map((p) => [p.id, openOverrides[p.id] ?? false])
  )
);

  function setProgramOpen(id: string, open: boolean) {
    openOverrides[id] = open;
  }

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

  <div class="program-type-selector">
    <button class="program-type" class:selected={hypActv} onclick={() => {hypActv = true}}>
      Hyperthrophy programs
    </button>
    <button class="program-type" class:selected={!hypActv} onclick={() => {hypActv = false}}>
      Powerlifting programs
    </button>
  </div>
  
  {#if !hypActv}
  <div class="bg-yellow-500/10 border border-yellow-500/15 rounded-lg py-4 px-2 w-full">
    <p class="w-full text-yellow-100 text-sm text-center"> Powerlifting programs are not availible yet...</p>
  </div>
  {/if}

  <div class="program-list" role="list" class:inactive={!hypActv}>
    {#each programs as program (program.id)}
      <article
        class={`program-card program-card--${program.accent}`}
        class:program-card--expanded={openPrograms[program.id]}
        role="listitem"
      >
        <Collapsible.Root 
          open={openPrograms[program.id]} 
          onOpenChange={(open) => setProgramOpen(program.id, open)}>
          
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
          <div class="program-card__panel pt-3">
            <div class="program-card__panel-grid grid gap-3.5">
              <!-- Overview -->
                <p class="p-card__section">
                  {program.overview}
                </p>

              <!-- Sessions -->
              <section class="grid gap-2.5">
                <div class="flex items-end justify-between gap-3 px-0.5">
                  <div>
                    <h3 class="text-[0.95rem] font-[650] tracking-[-0.01em]">
                      Sessions
                    </h3>

                    <p class="text-[var(--text-muted)] text-[0.8rem]">
                      This program consists of {program.sessions?.length} sessions.
                    </p>
                  </div>
                </div>

                <div class="grid gap-2 bg-[hsl(var(--muted)/0.2)] rounded-2xl">
                  {#each program.sessions as item, index}
                    <Card.Root class="border border-border py-4">
                      <Card.Content class="grid gap-3 p-3">
                        <div class="flex items-start justify-between gap-3">
                          <div class="flex min-w-0 gap-2.5">
                            <span class="grid place-items-center w-8 h-8 min-w-8 rounded-full bg-[hsl(var(--background))] text-[hsl(var(--foreground))] p-0 text-[0.8rem] font-semibold leading-none whitespace-nowrap">
                              {index + 1}
                            </span>

                            <div class="min-w-0">
                              <Card.Title class="text-[0.925rem] leading-[1.25] tracking-[-0.01em]">
                                {item}
                              </Card.Title>

                              <Card.Description class="mt-1 text-[0.775rem] leading-[1.35]">
                                Session {index + 1}
                              </Card.Description>
                            </div>
                          </div>

                          <Button
                            variant="outline"
                            disabled
                            size="sm"
                            type="button"
                            class="h-8 px-2.5 rounded-full text-[0.8rem]"
                          >
                            <Icon icon="mdi:eye" fill="white"/>
                          </Button>
                        </div>

                        <div class="grid grid-cols-2 gap-2">
                          <Button
                            disabled

                            variant="outline"
                            size="sm"
                            class="text-[0.825rem] font-semibold"
                          >
                            <a href={`${base}/create/program/${program.slug}`}>
                              Edit and add
                            </a>
                          </Button>

                          <Button
                          disabled
                            size="sm"
                            class="text-[0.825rem] font-semibold"
                          >
                            <a href={`${base}/create/program/${program.slug}`}>
                              Quick add
                            </a>
                          </Button>
                        </div>
                      </Card.Content>
                    </Card.Root>
                  {/each}
                </div>
              </section>
            </div>
          </div>

<Card.Root class="border-border mt-4">
  <Card.Header class="items-center">
    <Card.Title>Radar Chart - Muscle Groups</Card.Title>
    <Card.Description>
      Showing relative training volume across major muscle groups
    </Card.Description>
  </Card.Header>

  <Card.Content class="flex-1 ">
    <Chart.Container config={chartConfig} class="mx-auto aspect-square max-h-[320px] w-[70%]">
      <LineChart
        data={program.chartData}
        series={[
          {
            key: "volume",
            label: "Training Volume",
            color: chartConfig.volume.color,
          },
        ]}
        radial
        x="muscleGroup"
        xScale={scaleBand()}
        points={{ r: 4 }}
        padding={12}
        props={{
          spline: {
            curve: curveLinearClosed,
            fill: "var(--color-volume)",
            fillOpacity: 0.6,
            stroke: "0",
            motion: "tween",
          },
          xAxis: {
            tickLength: 0,
          },
          yAxis: {
            format: () => "",
          },
          grid: {
            radialY: "linear",
          },
          tooltip: {
            context: {
              mode: "voronoi",
            },
          },
          highlight: {
            lines: false,
            points: false,
          },
        }}
      >
        {#snippet tooltip()}
          <Chart.Tooltip />
        {/snippet}
      </LineChart>
    </Chart.Container>
  </Card.Content>
</Card.Root>



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
    padding-bottom: 10rem;
  }

  .program-type-selector {
    display: flex;
    width: 100%; 
    gap: 1rem;
  }

  .program-type {
    padding: 0.3rem 1rem;
    background-color: var(--surface-low);
    border-radius: 10px;
    border: 1px solid var(--border);
    text-align: center;
    font-size: 14px;
  }

  .program-type.selected {
    background-color: var(--surface-middle);
    box-shadow: var(--shadow-dark);
    font-weight: 600;
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

  .program-list.inactive {
    opacity: 0.6;
    pointer-events: none;
    cursor: not-allowed;
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

  h3 {
    margin: 1rem 0 0 0;
    font-size: 1rem;
    font-weight: 700;
    letter-spacing: 0.04em;
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
  .program-card__section li,
  .p-card__section {
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
    min-height: 2rem;
    border-radius: 999px;
    text-decoration: none;
    font-size: 0.7rem;
    font-weight: 700;
    transition:
      transform 180ms ease,
      background-color 180ms ease,
      border-color 180ms ease;
  }

  .program-card__cta--primary {
    color: var(--program-text-primary);
    border: 1px solid var(--program-line-soft);
    background: var(--program-surface-strong);
    border-radius: 100px;
    box-shadow: var(--shadow-liquid);
  }

  .program-content-holder {
    display: flex;
    flex-direction: column;
    padding: 0.5rem;
    background-color: var(--surface-middle);
    border-radius: 10px;
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
