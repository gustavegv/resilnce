import type { ChartData } from '$lib/components/ui/chart/chart-line-default.svelte';
import { GetExerciseHistory, type HistoricEntry } from '../tracker/dbFetches';

export function est1RM(weight: number, reps: number) {
  if (reps < 1) throw new Error('Reps must be at least 1');
  return weight * (1 + 0.0333 * reps);
}

export function totalVolume(weight: number, reps: number, sets: number) {
  return weight * reps * sets;
}

export function historicToChartData(entries: HistoricEntry[]): ChartData[] {
  return entries
    .map((entry) => ({
      date: entry.saveDate,
      oneRM: Math.round(est1RM(entry.avgWeight, Math.max(entry.avgRep, 1)) * 10) / 10,
      weightPerSet: entry.avgWeight,
      totalVolume: totalVolume(entry.avgWeight, entry.avgRep, entry.noOfSets),
    }))
    .sort((a, b) => a.date.getTime() - b.date.getTime());
}

export async function fetchHistoricData(exID: number): Promise<ChartData[]> {
  const history = await GetExerciseHistory(exID);
  return historicToChartData(history);
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
