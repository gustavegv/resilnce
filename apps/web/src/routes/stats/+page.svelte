<script lang="ts">
  import ChartLineDefault, {
    type ChartData,
  } from '$lib/components/ui/chart/chart-line-default.svelte';
  import * as Card from '$lib/components/ui/card/index.js';
  import { onMount } from 'svelte';
  import * as Tabs from '$lib/components/ui/tabs/index.js';
  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { getMe } from '../me';

  import type { SessionMetaData } from '../tracker/dbFetches';
  import { GetSessionExercises, GetSessions } from '../tracker/dbFetches';
  import { fetchHistoricData, hasDataOlderThan } from './statistics';
    import Skeleton from '$lib/components/ui/skeleton/skeleton.svelte';
    import { fade } from 'svelte/transition';

  type SessionExerciseChart = {
    id: number;
    name: string;
    data: ChartData[];
  };

  let isAuth = $state(false);
  let exDa: SessionMetaData[] = $state([]);
  let sessionExercises = $state<Record<number, SessionExerciseChart[]>>({});
  let loadingSessions = $state<Record<number, boolean>>({});
  let selectedSessionID = $state('');

  let selectedWPS = $state(true);
  let selected1RM = $state(true);
  let selectedTVL = $state(false);
  let montComp = $state(false);

  let sessionsLoading = $state(true)

  onMount(async () => {
    const value = get(user);

    if (value?.name || (await getMe())) {
      isAuth = true;
      exDa = await GetSessions();
      selectedSessionID = exDa[0] ? String(exDa[0].id) : '';
      sessionsLoading = false
    } else {
      goto(resolve('/account'));
    }
  });

  $effect(() => {
    if (!selectedSessionID) return;

    void loadSessionExercises(Number(selectedSessionID));
  });

  $effect(() => {
    if (!selectedSessionID || selectedRange !== '1m') return;

    if (!tabHasMonthComparison(Number(selectedSessionID))) {
      montComp = false;
    }
  });

  let selectedRange: string = $state('all');

  function adjustRange(range: string) {
    selectedRange = range;

    if (range !== '1m') {
      montComp = false;
    }
  }

  function toggleMetric(metric: 'wps' | '1rm' | 'tvl' | 'monthComp') {
    if (metric === 'wps') {
      selectedWPS = !selectedWPS;
      return;
    }

    if (metric === '1rm') {
      selected1RM = !selected1RM;
      return;
    }

    if (metric === 'tvl') {
      selectedTVL = !selectedTVL;
      return;
    }

    if (selectedRange === '1m') {
      montComp = !montComp;
    }
  }

  async function loadSessionExercises(sessionID: number) {
    if (
      Number.isNaN(sessionID) ||
      loadingSessions[sessionID] ||
      sessionExercises[sessionID] !== undefined
    ) {
      return;
    }

    loadingSessions = { ...loadingSessions, [sessionID]: true };

    try {
      const exercises = await GetSessionExercises(sessionID);
      const charts = await Promise.all(
        exercises.map(async (exercise) => {
          const exerciseID = Number(exercise.id);

          return {
            id: exerciseID,
            name: exercise.name,
            data: Number.isFinite(exerciseID) ? await fetchHistoricData(exerciseID) : [],
          };
        })
      );

      sessionExercises = { ...sessionExercises, [sessionID]: charts };
    } catch (error) {
      console.error(`Failed to load exercise stats for session ${sessionID}`, error);
      sessionExercises = { ...sessionExercises, [sessionID]: [] };
    } finally {
      loadingSessions = { ...loadingSessions, [sessionID]: false };
    }
  }

  function tabHasMonthComparison(sessionID: number): boolean {
    const charts = sessionExercises[sessionID] ?? [];
    return charts.some((exercise) => hasDataOlderThan(exercise.data, 2, 'm'));
  }

  function formatRange(range:string):string{
    if (range == "all") return "(all time)"
    if (range == "6m") return "(6 months)"
    if (range == "1m") return "(1 month)"
    return ""
  }
</script>

<main class="page">
  <eyebrow>Statistics</eyebrow>
  <h2>Your stats</h2>
  <subtitle
    >View historical workout data, spot trends, and see how your performance changes over time.
    Start by selecting a session below to inspect.
  </subtitle>

{#if isAuth}

    <Tabs.Root bind:value={selectedSessionID} class="mt-4 w-full max-w-[900px]">
      <div class="category-carousel">
        {#if sessionsLoading}
          <Skeleton class="h-4 w-[70px]" />
          <Skeleton class="h-4 w-[70px]" />
          <Skeleton class="h-4 w-[70px]" />
          <Skeleton class="h-4 w-[70px]" />
          <Skeleton class="h-4 w-[70px]" />


        {:else}
          {#each exDa as ex}
            <Tabs.Trigger class="bg-card" value={String(ex.id)}>{ex.name}</Tabs.Trigger>
          {/each}
        {/if}
      </div>

      {#each exDa as ex}
        <Tabs.Content value={String(ex.id)}>
        <div class="flex flex-col justify-center border border-border p-4 mb-4 mt-2 rounded-lg bg-card">
          <p class="text-neutral-400 text-sm">Choose a time horizon</p>
          <div class="flex w-full justify-baseline gap-2 mt-2">
            <button
              class:selected={selectedRange === 'all'}
              class="selector-button"
              onclick={() => adjustRange('all')}
            >
              All time
            </button>

            <button
              class:selected={selectedRange === '6m'}
              class="selector-button"
              onclick={() => adjustRange('6m')}
            >
              6 months
            </button>

            <button
              class:selected={selectedRange === '1m'}
              class="selector-button"
              onclick={() => adjustRange('1m')}
            >
              1 month
            </button>
          </div>

          <p class="text-neutral-400 text-sm mt-4">Select graph overlays</p>

          <div class="category-carousel mt-2">
            <button
              class:active={selectedWPS}
              class="selector-button metric"
              onclick={() => toggleMetric('wps')}
            >
              Weight per set
            </button>

            <button
              class:active={selected1RM}
              class="selector-button metric"
              onclick={() => toggleMetric('1rm')}
            >
              1RM
            </button>

            <button
              class:active={selectedTVL}
              class="selector-button metric"
              onclick={() => toggleMetric('tvl')}
            >
              Total volume lifted
            </button>

            {#if selectedRange == '1m' && tabHasMonthComparison(ex.id)}
              <button
                class:active={montComp}
                class="selector-button metric"
                onclick={() => toggleMetric('monthComp')}
              >
                This month vs Last
              </button>
            {/if}
          </div>
        </div>


          {#if loadingSessions[ex.id]}
            <Card.Root class="border-border w-full gap-4">
                <Skeleton class="h-6 w-[100px] mx-4 bg-neutral-700"/>
                <Skeleton class="h-4 w-[170px] mx-4 mt-0 bg-neutral-700"/>
                <Skeleton class="h-50 w-[40] mx-4 bg-neutral-700"/>

            </Card.Root>
          {:else if (sessionExercises[ex.id]?.length ?? 0) === 0}
            <Card.Root class="border-border w-full">
              <Card.Content class="text-muted-foreground py-6 text-center text-sm">
                No exercises with historic data found for this session yet.
              </Card.Content>
            </Card.Root>
          {:else}
            <div class="stats-grid">
              {#each sessionExercises[ex.id] ?? [] as exercise, index (exercise.id)}
              <div in:fade={{delay:index*150}}>
                <Card.Root class="border-border w-full">
                  <ChartLineDefault
                    title={exercise.name + ' statistics ' + formatRange(selectedRange)}
                    desc="Toggle between weight per set, 1RM, total volume, and monthly comparison layers."
                    data={exercise.data}
                    range={selectedRange}
                    {selectedWPS}
                    {selected1RM}
                    {selectedTVL}
                    {montComp}
                  />
                </Card.Root>
              </div>
              {/each}
            </div>
          {/if}
        </Tabs.Content>
      {/each}
    </Tabs.Root>
    {/if}
</main>

<style>
  .page {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 0 1rem;
    margin-top: 4rem;
    overflow-x: auto;
  }

  .category-carousel {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    overflow-x: auto;
    scroll-snap-type: x proximity;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  .category-carousel.center {
    justify-content: center;
  }

  .selector-button {
    background: var(--color-background);
    border-radius: 8px;
    border: 1px solid var(--border);
    padding: 0.2rem 0.5rem;
    font-size: 14px;
  }

  .selector-button.selected {
    background: var(--surface-middle);
    color: hsl(var(--primary-foreground));
  }

  .selector-button.metric {
    width: fit-content;
    white-space: nowrap;
  }

  .selector-button.active {
    background-color: rgba(70, 130, 72, 0.49);
    border: 1px solid rgba(105, 195, 105, 0.628);

    width: fit-content;
    white-space: nowrap;
  }

  .stats-grid {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
</style>
