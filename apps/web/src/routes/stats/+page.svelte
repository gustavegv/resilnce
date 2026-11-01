<script lang="ts">
  import ChartLineDefault from '$lib/components/ui/chart/chart-line-default.svelte';
  import * as Card from '$lib/components/ui/card/index.js';
  import { onMount } from 'svelte';
  import * as Tabs from '$lib/components/ui/tabs/index.js';
  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';
  import { getMe } from '../me';
  import { listSessions, readStats } from '$lib/training/client';
  import type { Session, StatsTrack } from '$lib/training/types';
  import { summariesToChartData, hasDataOlderThan, metric } from './statistics';
  import Skeleton from '$lib/components/ui/skeleton/skeleton.svelte';
  import { fade } from 'svelte/transition';

  let mounted = $state(false);
  let sessions = $state.raw<Session[]>([]);
  let tracks = $state.raw<StatsTrack[]>([]);
  let selectedSessionID = $state('');
  let selectedWPS = $state(true);
  let selected1RM = $state(true);
  let selectedTVL = $state(false);
  let montComp = $state(false);
  let sessionsLoading = $state(true);
  let loadError = $state('');
  let reload = $state(0);
  let selectedRange = $state('all');

  onMount(() => {
    mounted = true;
    if (!get(user)) void getMe();
  });

  // This effect owns an asynchronous read and its cancellation across accounts.
  $effect(() => {
    if (!mounted) return;
    const account = $user;
    reload;
    sessions = [];
    tracks = [];
    selectedSessionID = '';
    montComp = false;
    loadError = '';
    sessionsLoading = false;
    if (account?.capabilities.applicationModel !== 'exercise-centric') return;
    let cancelled = false;
    sessionsLoading = true;
    void Promise.all([listSessions(), readStats()])
      .then(([plans, stats]) => {
        if (cancelled) return;
        sessions = plans.sessions;
        tracks = stats.tracks;
        selectedSessionID =
          plans.sessions[0]?.id ??
          (stats.tracks.some((t) => t.entries.length) ? 'other-history' : '');
      })
      .catch(() => {
        if (!cancelled) loadError = 'Unable to load statistics. Retry to load your history.';
      })
      .finally(() => {
        if (!cancelled) sessionsLoading = false;
      });
    return () => {
      cancelled = true;
    };
  });

  const groups = $derived.by(() => {
    const byVariant = new Map(tracks.filter((t) => t.variantId).map((t) => [t.variantId, t]));
    const placed = new Set(sessions.flatMap((s) => s.placements.map((p) => p.variantId)));
    const groups = sessions.map((s) => ({
      id: s.id,
      name: s.name,
      tracks: s.placements.flatMap((p) => {
        const track = byVariant.get(p.variantId);
        return track ? [track] : [];
      }),
    }));
    const other = tracks.filter(
      (t) => t.entries.length && (!t.variantId || !placed.has(t.variantId))
    );
    if (other.length) groups.push({ id: 'other-history', name: 'Other history', tracks: other });
    return groups;
  });

  function adjustRange(range: string) {
    selectedRange = range;
    if (range !== '1m') montComp = false;
  }
  function toggleMetric(metric: 'wps' | '1rm' | 'tvl' | 'monthComp') {
    if (metric === 'wps') selectedWPS = !selectedWPS;
    else if (metric === '1rm') selected1RM = !selected1RM;
    else if (metric === 'tvl') selectedTVL = !selectedTVL;
    else if (selectedRange === '1m') montComp = !montComp;
  }
  function tabHasMonthComparison(id: string) {
    return (
      groups
        .find((g) => g.id === id)
        ?.tracks.some((t) => hasDataOlderThan(summariesToChartData(t.entries), 2, 'm')) ?? false
    );
  }
  const effectiveMonthComparison = $derived(montComp && tabHasMonthComparison(selectedSessionID));
  function formatRange(range: string) {
    return range === 'all' ? '(all time)' : range === '6m' ? '(6 months)' : '(1 month)';
  }
  function trackName(track: StatsTrack) {
    return (
      track.exerciseName + ' — ' + (track.variantName ?? 'Imported history (unassigned variant)')
    );
  }
  function formatMetric(value: string | null) {
    return metric(value) === undefined ? 'unavailable' : value;
  }
  function formatAverage(value: string | null) {
    const number = metric(value);
    return number === undefined
      ? 'unavailable'
      : new Intl.NumberFormat(undefined, { maximumFractionDigits: 4 }).format(number);
  }
</script>

<main class="page">
  <eyebrow>Statistics</eyebrow>
  <h2>Your stats</h2>
  <subtitle
    >View historical workout data, spot trends, and see how your performance changes over time.
    Start by selecting a session below to inspect.
  </subtitle>

  {#if !sessionsLoading && $user?.capabilities.applicationModel !== 'exercise-centric'}
    <p role="status">Statistics are unavailable until an exercise-centric account is loaded.</p>
  {:else if loadError}
    <p role="alert">{loadError}</p>
    <button class="selector-button" onclick={() => reload++}>Retry</button>
  {:else if $user?.capabilities.applicationModel === 'exercise-centric'}
    <Tabs.Root bind:value={selectedSessionID} class="mt-4 w-full max-w-[900px]">
      <div class="category-carousel">
        {#if sessionsLoading}
          <Skeleton class="h-4 w-[70px]" />
          <Skeleton class="h-4 w-[70px]" />
          <Skeleton class="h-4 w-[70px]" />
          <Skeleton class="h-4 w-[70px]" />
          <Skeleton class="h-4 w-[70px]" />
        {:else}
          {#each groups as ex (ex.id)}
            <Tabs.Trigger class="bg-card" value={ex.id}>{ex.name}</Tabs.Trigger>
          {/each}
        {/if}
      </div>

      {#each groups as ex (ex.id)}
        <Tabs.Content value={ex.id}>
          {#if selectedSessionID === ex.id}
            <div
              class="border-border bg-card mt-2 mb-4 flex flex-col justify-center rounded-lg border p-4"
            >
              <p class="text-sm text-neutral-400">Choose a time horizon</p>
              <div class="mt-2 flex w-full justify-baseline gap-2">
                <button
                  class={['selector-button', { selected: selectedRange === 'all' }]}
                  onclick={() => adjustRange('all')}
                >
                  All time
                </button>

                <button
                  class={['selector-button', { selected: selectedRange === '6m' }]}
                  onclick={() => adjustRange('6m')}
                >
                  6 months
                </button>

                <button
                  class={['selector-button', { selected: selectedRange === '1m' }]}
                  onclick={() => adjustRange('1m')}
                >
                  1 month
                </button>
              </div>

              <p class="mt-4 text-sm text-neutral-400">Select graph overlays</p>

              <div class="category-carousel mt-2">
                <button
                  aria-pressed={selectedWPS}
                  class={['selector-button metric', { active: selectedWPS }]}
                  onclick={() => toggleMetric('wps')}
                >
                  Weight per set
                </button>

                <button
                  aria-pressed={selected1RM}
                  class={['selector-button metric', { active: selected1RM }]}
                  onclick={() => toggleMetric('1rm')}
                >
                  1RM
                </button>

                <button
                  aria-pressed={selectedTVL}
                  class={['selector-button metric', { active: selectedTVL }]}
                  onclick={() => toggleMetric('tvl')}
                >
                  Total volume lifted
                </button>

                {#if selectedRange == '1m' && tabHasMonthComparison(ex.id)}
                  <button
                    aria-pressed={effectiveMonthComparison}
                    class={['selector-button metric', { active: effectiveMonthComparison }]}
                    onclick={() => toggleMetric('monthComp')}
                  >
                    This month vs Last
                  </button>
                {/if}
              </div>
            </div>

            {#if sessionsLoading}
              <Card.Root class="border-border w-full gap-4">
                <Skeleton class="mx-4 h-6 w-[100px] bg-neutral-700" />
                <Skeleton class="mx-4 mt-0 h-4 w-[170px] bg-neutral-700" />
                <Skeleton class="mx-4 h-50 w-[40] bg-neutral-700" />
              </Card.Root>
            {:else if ex.tracks.length === 0}
              <Card.Root class="border-border w-full">
                <Card.Content class="text-muted-foreground py-6 text-center text-sm">
                  No exercises with historic data found for this session yet.
                </Card.Content>
              </Card.Root>
            {:else}
              <div class="stats-grid">
                {#each ex.tracks as exercise, index (exercise.variantId ?? exercise.exerciseId)}
                  <div in:fade={{ delay: index * 150 }}>
                    <Card.Root class="border-border w-full">
                      <ChartLineDefault
                        title={trackName(exercise) + ' statistics ' + formatRange(selectedRange)}
                        desc="Toggle between weight per set, 1RM, total volume, and monthly comparison layers."
                        data={summariesToChartData(exercise.entries)}
                        range={selectedRange}
                        {selectedWPS}
                        {selected1RM}
                        {selectedTVL}
                        montComp={effectiveMonthComparison}
                      />
                      <div class="text-muted-foreground px-6 pb-4 text-sm">
                        {#if exercise.entries.some((e) => e.representation === 'legacy_aggregate')}
                          <p>
                            Imported history uses original aggregates; imported volume is estimated.
                          </p>
                        {/if}
                        {#if !exercise.entries.length}<p>No completed performance yet.</p>{/if}
                        <details>
                          <summary>Performance summaries ({exercise.entries.length})</summary>
                          <ul class="mt-2 grid gap-2">
                            {#each exercise.entries as entry (entry.id)}
                              {@const chartEntry = summariesToChartData([entry])[0]}
                              <li data-entry-id={entry.id}>
                                {entry.occurredAt
                                  ? new Date(entry.occurredAt).toLocaleString()
                                  : 'Occurrence time unavailable'}
                                · {entry.representation === 'legacy_aggregate'
                                  ? 'Imported'
                                  : 'Recorded'}
                                · Weight per set: {formatAverage(entry.averageWeight)} kg · Average reps:
                                {formatAverage(entry.averageReps)}
                                · Sets: {entry.setCount !== null && entry.setCount >= 0
                                  ? entry.setCount
                                  : 'unavailable'}
                                · 1RM: {chartEntry?.oneRM ?? 'unavailable'}
                                · Volume: {formatMetric(entry.volume)}{entry.volume !== null
                                  ? ' kg-reps'
                                  : ''}{entry.volumeEstimated ? ' (estimated)' : ' (exact)'}
                                {#if entry.sourceTimestamp}
                                  <span
                                    >· Source time: {entry.sourceTimestamp}{entry.sourceTimezone
                                      ? ` (${entry.sourceTimezone})`
                                      : ' (timezone unavailable)'}</span
                                  >
                                {/if}
                              </li>
                            {/each}
                          </ul>
                        </details>
                      </div>
                    </Card.Root>
                  </div>
                {/each}
              </div>
            {/if}
          {/if}
        </Tabs.Content>
      {/each}
    </Tabs.Root>
    {#if !sessionsLoading && !groups.length}<p>
        No completed performance or saved sessions yet.
      </p>{/if}
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
