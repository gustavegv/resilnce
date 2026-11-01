import type { ChartData } from '$lib/components/ui/chart/chart-line-default.svelte';
import type { StatsEntry } from '$lib/training/types';

export function metric(value: string | null): number | undefined {
  if (value === null || value.trim() === '') return undefined;
  const number = Number(value);
  return Number.isFinite(number) && number >= 0 ? number : undefined;
}

export function est1RM(weight: number, reps: number) {
  return weight * (1 + 0.0333 * reps);
}

export function summariesToChartData(entries: StatsEntry[]): ChartData[] {
  return entries
    .flatMap((entry) => {
      if (!entry.occurredAt) return [];
      const date = new Date(entry.occurredAt);
      if (!Number.isFinite(date.getTime())) return [];
      const weight = metric(entry.averageWeight);
      const reps = metric(entry.averageReps);
      return [
        {
          id: entry.id,
          date,
          representation: entry.representation,
          volumeEstimated: entry.volumeEstimated,
          volumeExact: entry.volume ?? undefined,
          oneRM:
            weight !== undefined && reps !== undefined && reps > 0
              ? Math.round(est1RM(weight, reps) * 10) / 10
              : undefined,
          weightPerSet: weight,
          totalVolume: metric(entry.volume),
        },
      ];
    })
    .sort((a, b) => a.date.getTime() - b.date.getTime() || a.id.localeCompare(b.id));
}

function shiftDate(date: Date, amount: number, unit: 'd' | 'm' | 'y') {
  const shiftedDate = new Date(date);

  if (unit === 'd') {
    shiftedDate.setDate(shiftedDate.getDate() - amount);
  }

  if (unit === 'm') {
    shiftedDate.setMonth(shiftedDate.getMonth() - amount);
  }

  if (unit === 'y') {
    shiftedDate.setFullYear(shiftedDate.getFullYear() - amount);
  }

  return shiftedDate;
}

export function hasDataOlderThan(
  data: ChartData[],
  amount: number,
  unit: 'd' | 'm' | 'y',
  referenceDate?: Date
) {
  if (!data.length) return false;

  const latestDate = referenceDate
    ? new Date(referenceDate)
    : data.reduce((latest, entry) => (entry.date > latest ? entry.date : latest), data[0].date);
  const cutoffDate = shiftDate(latestDate, amount, unit);
  const oldestDate = data.reduce(
    (oldest, entry) => (entry.date < oldest ? entry.date : oldest),
    data[0].date
  );

  return oldestDate <= cutoffDate;
}
