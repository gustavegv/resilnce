<script lang="ts">
  import { AreaChart } from 'layerchart';
  import TrendingUpIcon from '@lucide/svelte/icons/trending-up';
  import { curveBumpX, curveNatural } from 'd3-shape';
  import { scaleLinear, scaleUtc } from 'd3-scale';
  import * as Chart from '$lib/components/ui/chart/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { max } from 'd3-array';
  import type { DomainType } from 'layerchart/utils/scales.svelte';

  let {
    title,
    desc,
    data,
  }: {
    title: string;
    desc: string;
    data: ChartData[];
  } = $props();

  export interface ChartData {
    date: Date;
    oneRM: number;
    second: number;
  }

  const chartData: ChartData[] = data;

  const chartConfig = {
    oneRM: { label: 'oneRM', color: 'var(--chart-1)' },
    second: { label: 'Weight', color: 'var(--chart-2)' },
  } satisfies Chart.ChartConfig;

  const yDomain: DomainType = getYDomain(data);

  function getYDomain(data: ChartData[]): DomainType {
    let lower: number | undefined = undefined;
    let upper: number = 0;

    data.forEach((d) => {
      if (!lower) {
        lower = d.second;
      } else {
        lower = Math.min(d.second, lower);
      }
      upper = Math.max(d.oneRM, upper);
    });
    const avg = upper - (lower ?? 0);
    return [(lower ?? 0) - 10, upper + avg / 2];
  }

  export function getTrend(data: ChartData[]) {
    if (!data || data.length === 0) {
      return null;
    }

    // 1. Sort ascending by date
    const sorted = data.slice().sort((a, b) => a.date.getTime() - b.date.getTime());

    // 2. Latest entry
    const latestEntry = sorted[sorted.length - 1];
    const latestDate = latestEntry.date;
    const latestValue = latestEntry.oneRM;

    // 3. Build a “one month ago” target date, anchored to latestDate
    const targetDate = new Date(
      latestDate.getFullYear(),
      latestDate.getMonth() - 1,
      latestDate.getDate(),
      latestDate.getHours()
    );

    // 4. Find the entry closest in time to targetDate
    let pastEntry = sorted[0];
    let closestDiff = Math.abs(pastEntry.date.getTime() - targetDate.getTime());
    for (const entry of sorted) {
      const diff = Math.abs(entry.date.getTime() - targetDate.getTime());
      if (diff < closestDiff) {
        closestDiff = diff;
        pastEntry = entry;
      }
    }

    const pastValue = pastEntry.oneRM;

    // 5. Compute % change (guarding against division by zero)
    const percentageChange = pastValue === 0 ? null : ((latestValue - pastValue) / pastValue) * 100;
    const rounded = Math.round((percentageChange ?? 1) * 10) / 10;

    if (!percentageChange || percentageChange < 0) {
      return null;
    } else {
      return `Trending up by ${rounded}% this month`;
    }
  }

  const trend: string | null = getTrend(data);
</script>

<Card.Header>
  <Card.Title>{title}</Card.Title>
  <Card.Description>{desc}</Card.Description>
</Card.Header>
<Card.Content>
  <Chart.Container config={chartConfig}>
    <AreaChart
      data={chartData}
      x="date"
      xScale={scaleUtc()}
      {yDomain}
      series={[
        {
          key: 'second',
          label: 'Weight',
          color: chartConfig.second.color,
        },
        {
          key: 'oneRM',
          label: '1RM',
          color: chartConfig.oneRM.color,
        },
      ]}
      axis="x"
      props={{
        area: {
          curve: curveBumpX,
          'fill-opacity': 0.2,
          line: { class: 'stroke-1' },
          motion: 'tween',
        },
        xAxis: {
          format: (v: Date) => v.toLocaleDateString('en-US', { dateStyle: 'short' }),
        },
      }}
    >
      {#snippet tooltip()}
        <Chart.Tooltip
          labelFormatter={(v: Date) => v.toLocaleDateString('en-US', { month: 'long' })}
          indicator="line"
        />
      {/snippet}
    </AreaChart>
  </Chart.Container>
</Card.Content>
<Card.Footer>
  <div class="flex w-full items-start gap-2 text-sm">
    <div class="grid gap-2">
      {#if trend}
        <div class="flex items-center gap-2 leading-none font-medium">
          {trend}
          <TrendingUpIcon class="size-4" />
        </div>
      {/if}
    </div>
  </div>
</Card.Footer>
