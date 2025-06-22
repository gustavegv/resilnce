<script lang="ts">
  import ChartLineDefault, {
    type ChartData,
  } from '$lib/components/ui/chart/chart-line-default.svelte';
  import * as Card from '$lib/components/ui/card/index.js';
  import { calling, historicTo1RM, type Unformat } from './statistics';
  import { getOrderedExercises, type Historic } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import Input from '$lib/components/ui/input/input.svelte';
  import * as Tabs from '$lib/components/ui/tabs/index.js';
  import type { ExerciseInfo } from '$lib/firebaseCreation';
  import { user } from '../account/user';
  import { get } from 'svelte/store';

  onMount(async () => {});

  const unformatted: Historic[] = [
    { date: new Date('2024-01-01'), avgSet: 7, weightH: 80 },
    { date: new Date('2024-01-02'), avgSet: 8, weightH: 80 },
    { date: new Date('2024-01-03'), avgSet: 8, weightH: 80 },
    { date: new Date('2024-01-04'), avgSet: 9, weightH: 80 },
    { date: new Date('2024-01-05'), avgSet: 12, weightH: 80 },
    { date: new Date('2024-01-06'), avgSet: 8, weightH: 82.5 },
    { date: new Date('2024-01-08'), avgSet: 9, weightH: 82.5 },
    { date: new Date('2024-01-10'), avgSet: 9, weightH: 82.5 },
    { date: new Date('2024-01-12'), avgSet: 10, weightH: 82.5 },
    { date: new Date('2024-01-13'), avgSet: 11, weightH: 82.5 },
    { date: new Date('2024-01-13'), avgSet: 8, weightH: 85 },
  ];

  let formattedData: ChartData[] = $state([]);
  let sessionInput: string = $state('Link');
  let exDa: ExerciseInfo[] = $state([]);
  let uID = get(user) ?? '';

  async function as(u: string, s: string) {
    exDa = await getOrderedExercises(u, s);
    const cD = await calling(s);
    formattedData = cD;
  }
</script>

<br />
<br />
<br />
<br />
<br />

<main class="flex flex-col items-center justify-center gap-4 px-4">
  <div class="grid grid-cols-2 items-center">
    <Input bind:value={sessionInput} class="" placeholder="Session name" />
    <button class="buttonClass m-4" onclick={() => as(uID, sessionInput)}>Get session</button>
  </div>

  <Tabs.Root value="account" class="w-[400px]">
    <Tabs.List>
      {#each exDa as ex}
        <Tabs.Trigger value={ex.name}>{ex.name}</Tabs.Trigger>
      {/each}
    </Tabs.List>
    {#each exDa as ex}
      <Tabs.Content value={ex.name}>
        <Card.Root class="w-full">
          <ChartLineDefault
            title="Latest session"
            desc="One Rep Max Progression layered with the current weight"
            data={formattedData}
          />
        </Card.Root>
      </Tabs.Content>
    {/each}
  </Tabs.Root>
</main>
