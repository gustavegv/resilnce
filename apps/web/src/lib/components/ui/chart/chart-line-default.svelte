<script lang="ts">
  import { AreaChart } from 'layerchart';
  import TrendingUpIcon from '@lucide/svelte/icons/trending-up';
  import { curveBumpX } from 'd3-shape';
  import { scaleUtc } from 'd3-scale';
  import * as Chart from '$lib/components/ui/chart/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import type { DomainType } from 'layerchart/utils/scales.svelte';

  let {
    title,
    desc,
    data,
    range,
    selectedWPS,
    selected1RM,
    selectedTVL,
    montComp,
  }: {
    title: string;
    desc: string;
    data: ChartData[];
    range: string;
    selectedWPS: boolean;
    selected1RM: boolean;
    selectedTVL: boolean;
    montComp: boolean;
  } = $props();

  export interface ChartData {
    date: Date;
    oneRM: number;
    weightPerSet: number;
    totalVolume: number;
  }

  function getRangeMonths(range: string) {
    if (range === '1m') return 1;
    if (range === '6m') return 6;
    return 100;
  }

  function getRangeLabel(range: string) {
    if (range === '1m') return 'this month';
    if (range === '6m') return 'during the last 6 months';
    return 'all time';
  }

  const rangeMonths = $derived(getRangeMonths(range));
  const rangeLabel = $derived(getRangeLabel(range));

  const chartConfig = {
    oneRM: { label: 'EST 1RM', color: 'var(--color-secondary)' },
    weightPerSet: { label: 'Weight per set', color: 'var(--chart-2)' },
    totalVolume: { label: 'Total volume lifted', color: 'var(--chart-4)' },
    monthComparison: { label: 'Last month weight per set', color: 'var(--chart-5)' },
  } satisfies Chart.ChartConfig;

  function filterChartDataByRange(data: ChartData[], range: string) {
    if (range === 'all' || !data.length) return data;

    const latestDate = data.reduce(
      (latest, entry) => (entry.date > latest ? entry.date : latest),
      data[0].date
    );
    const cutoffDate = new Date(latestDate);

    if (range === '6m') {
      cutoffDate.setMonth(cutoffDate.getMonth() - 6);
    }

    if (range === '1m') {
      cutoffDate.setMonth(cutoffDate.getMonth() - 1);
    }

    return data.filter((item) => item.date >= cutoffDate);
  }

  function getClosestEntry(entries: ChartData[], targetDate: Date) {
    let closestEntry = entries[0];
    let closestDiff = Math.abs(entries[0].date.getTime() - targetDate.getTime());

    for (const entry of entries) {
      const diff = Math.abs(entry.date.getTime() - targetDate.getTime());
      if (diff < closestDiff) {
        closestDiff = diff;
        closestEntry = entry;
      }
    }

    return closestEntry;
  }

  function getMonthComparisonData(data: ChartData[], visibleData: ChartData[]) {
    if (range !== '1m' || !visibleData.length || !data.length) {
      return [];
    }

    return visibleData.map((entry) => {
      const targetDate = new Date(entry.date);
      targetDate.setMonth(targetDate.getMonth() - 1);

      const comparisonEntry = getClosestEntry(data, targetDate);

      return {
        ...comparisonEntry,
        date: entry.date,
      };
    });
  }

  function getYDomain(
    visibleData: ChartData[],
    comparisonData: ChartData[],
    {
      selectedWPS,
      selected1RM,
      selectedTVL,
      montComp,
      range,
    }: {
      selectedWPS: boolean;
      selected1RM: boolean;
      selectedTVL: boolean;
      montComp: boolean;
      range: string;
    }
  ): DomainType {
    const values: number[] = [];

    if (selectedWPS) {
      values.push(...visibleData.map((entry) => entry.weightPerSet));
    }

    if (selected1RM) {
      values.push(...visibleData.map((entry) => entry.oneRM));
    }

    if (selectedTVL) {
      values.push(...visibleData.map((entry) => entry.totalVolume));
    }

    if (range === '1m' && montComp) {
      values.push(...comparisonData.map((entry) => entry.weightPerSet));
    }

    if (!values.length) {
      return [0, 100];
    }

    const lower = Math.min(...values);
    const upper = Math.max(...values);
    const span = upper - lower || Math.max(upper, 10);

    return [Math.max(0, lower - span * 0.15), upper + span * 0.2];
  }

  export function getTrend(data: ChartData[]) {
    if (!data || data.length === 0) {
      return null;
    }

    const sorted = data.slice().sort((a, b) => a.date.getTime() - b.date.getTime());
    const latestEntry = sorted[sorted.length - 1];
    const latestDate = latestEntry.date;
    const latestValue = latestEntry.weightPerSet;

    const targetDate = new Date(
      latestDate.getFullYear(),
      latestDate.getMonth() - rangeMonths,
      latestDate.getDate(),
      latestDate.getHours()
    );

    const pastEntry = getClosestEntry(sorted, targetDate);

    const pastValue = pastEntry.weightPerSet;

    const percentageChange = pastValue === 0 ? null : ((latestValue - pastValue) / pastValue) * 100;
    const rounded = Math.round((percentageChange ?? 1) * 10) / 10;

    if (!percentageChange || percentageChange < 0) {
      return null;
    } else {
      return `Weight per set trending up by ${rounded}% ${rangeLabel}`;
    }
  }

  const chartData = $derived.by(() => filterChartDataByRange(data, range));
  const monthComparisonData = $derived.by(() => getMonthComparisonData(data, chartData));
  const visibleSeries = $derived.by(() => {
    const series = [];

    if (selected1RM) {
      series.push({
        key: 'oneRM',
        label: '1RM',
        color: chartConfig.oneRM.color,
      });
    }

    if (selectedWPS) {
      series.push({
        key: 'weightPerSet',
        label: 'Weight per set',
        color: chartConfig.weightPerSet.color,
      });
    }

    if (selectedTVL) {
      series.push({
        key: 'totalVolume',
        label: 'Total volume lifted',
        color: chartConfig.totalVolume.color,
      });
    }

    if (range === '1m' && montComp) {
      series.push({
        key: 'monthComparison',
        label: 'Last month 1RM',
        color: chartConfig.monthComparison.color,
        value: (entry: ChartData) => entry.weightPerSet,
        data: monthComparisonData,
      });
    }

    return series;
  });
  const yDomain = $derived(
    getYDomain(chartData, monthComparisonData, {
      selectedWPS,
      selected1RM,
      selectedTVL,
      montComp,
      range,
    })
  );
  const trend = $derived(selectedWPS ? getTrend(chartData) : null);
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
      series={visibleSeries}
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
          labelFormatter={(v: Date) =>
            v.toLocaleDateString('en-US', {
              year: 'numeric',
              month: 'short',
              day: 'numeric',
            })}
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
