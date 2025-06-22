<script lang="ts">
  import ChartLineDefault, {
    type ChartData,
  } from '$lib/components/ui/chart/chart-line-default.svelte';
  import * as Card from '$lib/components/ui/card/index.js';
  import { calling, historicTo1RM, type Unformat } from './statistics';
  import type { Historic } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import Input from '$lib/components/ui/input/input.svelte';

  onMount(async () => {
    formattedData = await calling();
  });

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
</script>

<br />
<br />
<br />
<br />
<br />

<main class="flex flex-col items-center justify-center gap-4 px-4">
  <div class="grid grid-cols-2 items-center">
    <Input class="" placeholder="Session name" />
    <button class="buttonClass m-4" onclick={calling}>Get session</button>
  </div>

  <Card.Root class="w-full">
    <ChartLineDefault
      title="Latest session"
      desc="One Rep Max Progression layered with the current weight"
      data={formattedData}
    />
  </Card.Root>
  <div class="grid grid-cols-2 gap-4">
    <Card.Root class="max-w-96">
      <ChartLineDefault
        title="Squat progress"
        desc="One Rep Max Progression layered with the current weight"
        data={historicTo1RM(unformatted)}
      />
    </Card.Root>
    <Card.Root class="max-w-96">
      <ChartLineDefault
        title="Bench Progress"
        desc="One Rep Max Progression layered with the current weight"
        data={historicTo1RM(unformatted)}
      />
    </Card.Root>
  </div>
</main>
