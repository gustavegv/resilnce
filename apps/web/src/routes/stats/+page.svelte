<script lang="ts">
  import ChartLineDefault, {
    type ChartData,
  } from '$lib/components/ui/chart/chart-line-default.svelte';
  import * as Card from '$lib/components/ui/card/index.js';
  import { onMount } from 'svelte';
  import Input from '$lib/components/ui/input/input.svelte';
  import * as Tabs from '$lib/components/ui/tabs/index.js';
  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { getMe } from '../me';

  import type { SessionMetaData } from '../tracker/dbFetches';
  import { GetSessions } from '../tracker/dbFetches';

  let isAuth = $state(false);
  let formattedData: ChartData[] = $state([]);
  let exDa: SessionMetaData[] = $state([]);
  let uID = get(user) ?? '';

  onMount(async () => {
    const value = get(user);

    if (value?.name || (await getMe())) {
      isAuth = true;
      exDa = await GetSessions();

    } else {
      goto(resolve('/account'));
    }
  });

  formattedData = [
  { date: new Date('2026-01-05'), oneRM: 82.5, second: 65 },
  { date: new Date('2026-01-12'), oneRM: 84, second: 66 },
  { date: new Date('2026-01-19'), oneRM: 85.5, second: 67.5 },
  { date: new Date('2026-01-26'), oneRM: 87, second: 69 },
  { date: new Date('2026-02-02'), oneRM: 88, second: 70 },
  { date: new Date('2026-02-09'), oneRM: 89.5, second: 71 },
  { date: new Date('2026-02-16'), oneRM: 91, second: 72.5 },
  { date: new Date('2026-02-23'), oneRM: 92.5, second: 74 },
  { date: new Date('2026-03-02'), oneRM: 94, second: 75 },
  { date: new Date('2026-03-09'), oneRM: 95, second: 76 },
  { date: new Date('2026-03-16'), oneRM: 96.5, second: 77.5 },
  { date: new Date('2026-03-23'), oneRM: 98, second: 79 },
];



let selectedRange: string = $state('all');

function adjustRange(range: string){
  selectedRange = range;

}

function filterChartDataByRange(data: ChartData[]): ChartData[] {

  if (selectedRange === 'all') return data;

  const now = new Date();
  const cutoff = new Date(now);

  if (selectedRange === '6m') {
    cutoff.setMonth(cutoff.getMonth() - 6);
  }

  if (selectedRange === '1m') {
    cutoff.setMonth(cutoff.getMonth() - 1);
  }
  return data.filter((item) => item.date >= cutoff);
}
</script>


{#if isAuth}
<main class="flex flex-col items-center justify-center px-4 mt-16">
  <eyebrow>Statistics</eyebrow>
  <h2>Your stats</h2>
  <subtitle>View historical workout data, spot trends, and see how your performance changes over time.
    Start by selecting a session below to inspect.
  </subtitle>

  <Tabs.Root value="account" class="w-[400px] mt-4">
    <div class="category-carousel">
      {#each exDa as ex}
        <Tabs.Trigger class="bg-card" value={ex.name}>{ex.name}</Tabs.Trigger>
      {/each}
    </div>

    {#each exDa as ex}
      <Tabs.Content value={ex.name}>
        <Card.Root class="w-full border-border">
          <ChartLineDefault
            title={ex.name + ' Latest session'}
            desc="One Rep Max Progression layered with the current weight"
            data={filterChartDataByRange(formattedData)}
            range={selectedRange}
          />
            <div class="flex gap-2 w-full justify-center">
              <button
                class:selected={selectedRange === 'all'}
                class="rangeSelector"
                onclick={() => adjustRange('all')}
              >
                All time
              </button>

              <button
                class:selected={selectedRange === '6m'}
                class="rangeSelector"
                onclick={() => adjustRange('6m')}
              >
                6 months
              </button>

              <button
                class:selected={selectedRange === '1m'}
                class="rangeSelector"
                onclick={() => adjustRange('1m')}
              >
                1 month
              </button>
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

  .rangeSelector{
    background: var(--color-background);
    border-radius: 8px;
    border: 1px solid var(--border);
    padding: 0.2rem 0.5rem ;
    font-size: 14px;
  }

  .rangeSelector.selected {
    background: var(--surface-middle);
    color: hsl(var(--primary-foreground));
  }
</style>