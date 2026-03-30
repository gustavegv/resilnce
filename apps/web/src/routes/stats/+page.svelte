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
  import { GetSessions } from '../tracker/dbFetches';
  import { formatDemoStatistics, hasDataOlderThan } from './statistics';

  let isAuth = $state(false);
  let formattedData: ChartData[] = $state(formatDemoStatistics());
  let exDa: SessionMetaData[] = $state([]);

  let selectedWPS = $state(true);
  let selected1RM = $state(true);
  let selectedTVL = $state(false);
  let montComp = $state(false);

  onMount(async () => {
    const value = get(user);

    if (value?.name || (await getMe())) {
      isAuth = true;
      exDa = await GetSessions();
    } else {
      goto(resolve('/account'));
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

  function dataOlderThan(amount: number, unit: 'd' | 'm' | 'y'): boolean {
    return hasDataOlderThan(formattedData, amount, unit);
  }
</script>

{#if isAuth}
  <main class="mt-16 flex flex-col items-center justify-center px-4">
    <eyebrow>Statistics</eyebrow>
    <h2>Your stats</h2>
    <subtitle
      >View historical workout data, spot trends, and see how your performance changes over time.
      Start by selecting a session below to inspect.
    </subtitle>

    <Tabs.Root value="account" class="mt-4 w-[400px]">
      <div class="category-carousel">
        {#each exDa as ex}
          <Tabs.Trigger class="bg-card" value={ex.name}>{ex.name}</Tabs.Trigger>
        {/each}
      </div>

      {#each exDa as ex}
        <Tabs.Content value={ex.name}>
          <Card.Root class="border-border w-full">
            <ChartLineDefault
              title={ex.name + ' session statistics'}
              desc="Toggle between weight per set, 1RM, total volume, and monthly comparison layers."
              data={formattedData}
              range={selectedRange}
              {selectedWPS}
              {selected1RM}
              {selectedTVL}
              {montComp}
            />
            <div class="flex w-full justify-center gap-2">
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
            <div class="category-carousel" class:center={selectedRange != '1m'}>
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

              {#if selectedRange == '1m' && dataOlderThan(2, 'm')}
                <button
                  class:active={montComp}
                  class="selector-button metric"
                  onclick={() => toggleMetric('monthComp')}
                >
                  This month vs Last
                </button>
              {/if}
            </div>
          </Card.Root>
        </Tabs.Content>
      {/each}
    </Tabs.Root>
  </main>
{/if}

<style>
  .category-carousel {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    overflow-x: auto;
    padding: 0.25rem 1rem 0.75rem 1rem;
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
</style>
