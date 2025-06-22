<script lang="ts">
  import ChartLineDefault from '$lib/components/ui/chart/chart-line-default.svelte';
  import * as Card from '$lib/components/ui/card/index.js';

  import type { ChartData } from '$lib/components/ui/chart/chart-line-default.svelte';

  interface Unformat {
    day: Date;
    reps: number;
    weight: number;
  }

  const unformatted: Unformat[] = [
    { day: new Date('2024-01-01'), reps: 7, weight: 80 },
    { day: new Date('2024-01-02'), reps: 8, weight: 80 },
    { day: new Date('2024-01-03'), reps: 8, weight: 80 },
    { day: new Date('2024-01-04'), reps: 9, weight: 80 },
    { day: new Date('2024-01-05'), reps: 12, weight: 80 },
    { day: new Date('2024-01-06'), reps: 8, weight: 82.5 },
    { day: new Date('2024-01-08'), reps: 9, weight: 82.5 },
    { day: new Date('2024-01-10'), reps: 9, weight: 82.5 },
    { day: new Date('2024-01-12'), reps: 10, weight: 82.5 },
    { day: new Date('2024-01-13'), reps: 11, weight: 82.5 },
    { day: new Date('2024-01-13'), reps: 8, weight: 85 },
  ];

  function est1RM(weight: number, reps: number) {
    if (reps < 1) throw new Error('Reps must be at least 1');
    return weight * (1 + 0.0333 * reps);
  }

  function historicTo1RM(d: Unformat[]): ChartData[] {
    let newA: ChartData[] = [];

    for (var entry of d) {
      const val = est1RM(entry.weight, entry.reps);
      const n: ChartData = { date: entry.day, oneRM: val, second: entry.weight };
      newA = [...newA, n];
    }
    let last = newA[newA.length - 1];
    const date = last.date;
    date.setDate(date.getDate() + 1);
    last.date = date;
    newA = [...newA, last];

    return newA;
  }
</script>

<br />
<br />
<br />
<br />
<br />
<main class="flex flex-col items-center justify-center gap-4 px-4">
  <Card.Root class="w-full">
    <ChartLineDefault
      title="Latest session"
      desc="One Rep Max Progression layered with the current weight"
      data={historicTo1RM(unformatted)}
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
