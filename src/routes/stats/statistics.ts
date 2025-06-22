import type { ChartData } from '$lib/components/ui/chart/chart-line-default.svelte';
import { fetchHistoricData, type Exercise, type Historic } from '$lib/firebaseDataHandler';
import { get } from 'svelte/store';
import { user } from '../account/user';

export interface Unformat {
  day: Date;
  reps: number;
  weight: number;
}

export function est1RM(weight: number, reps: number) {
  if (reps < 1) throw new Error('Reps must be at least 1');
  return weight * (1 + 0.0333 * reps);
}

export function historicTo1RM(d: Historic[]): ChartData[] {
  let newA: ChartData[] = [];

  for (var entry of d) {
    const val = est1RM(entry.weightH, entry.avgSet);
    const n: ChartData = { date: entry.date, oneRM: val, second: entry.weightH };
    newA = [...newA, n];
  }

  return newA;
}

export async function calling(): Promise<ChartData[]> {
  let uID = get(user);
  if (!uID) return [];

  const data: Historic[] = await fetchHistoricData(uID, 'Link');

  const chartData = historicTo1RM(data);

  console.log(chartData);
  return chartData;
}
