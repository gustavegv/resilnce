import type { ChartData } from '$lib/components/ui/chart/chart-line-default.svelte';
import { GetExerciseHistory, type HistoricEntry } from '../tracker/dbFetches';

export interface DemoStatisticEntry {
  day: Date;
  reps: number;
  sets: number;
  weight: number;
}

export function est1RM(weight: number, reps: number) {
  if (reps < 1) throw new Error('Reps must be at least 1');
  return weight * (1 + 0.0333 * reps);
}

export function totalVolume(weight: number, reps: number, sets: number) {
  return weight * reps * sets;
}

export const demoStatistics: DemoStatisticEntry[] = [
  { day: new Date('2026-01-05'), reps: 8, sets: 2, weight: 65 },
  { day: new Date('2026-01-12'), reps: 8, sets: 2, weight: 66 },
  { day: new Date('2026-01-19'), reps: 8, sets: 2, weight: 67.5 },
  { day: new Date('2026-01-26'), reps: 8, sets: 2, weight: 69 },
  { day: new Date('2026-02-02'), reps: 8, sets: 2, weight: 70 },
  { day: new Date('2026-02-09'), reps: 8, sets: 2, weight: 71 },
  { day: new Date('2026-02-16'), reps: 8, sets: 2, weight: 72.5 },
  { day: new Date('2026-02-23'), reps: 8, sets: 2, weight: 74 },
  { day: new Date('2026-03-02'), reps: 8, sets: 2, weight: 75 },
  { day: new Date('2026-03-09'), reps: 8, sets: 2, weight: 76 },
  { day: new Date('2026-03-16'), reps: 8, sets: 2, weight: 77.5 },
  { day: new Date('2026-03-23'), reps: 8, sets: 2, weight: 79 },
];

export function formatDemoStatistics(entries: DemoStatisticEntry[] = demoStatistics): ChartData[] {
  return entries
    .map((entry) => ({
      date: entry.day,
      oneRM: Math.round(est1RM(entry.weight, entry.reps) * 10) / 10,
      weightPerSet: entry.weight,
      totalVolume: totalVolume(entry.weight, entry.reps, entry.sets),
    }))
    .sort((a, b) => a.date.getTime() - b.date.getTime());
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

/* 
Commented out to remove old Firebase dependencies, but no timeplan of when to look at statistics again.



export function historicTo1RM(d: Historic[]): ChartData[] {
  let newA: ChartData[] = [];

  for (var entry of d) {
    const val = est1RM(entry.weightH, entry.avgSet);
    const n: ChartData = {
      date: entry.date,
      oneRM: val,
      weightPerSet: entry.weightH,
      totalVolume: entry.weightH * entry.avgSet
    };
    newA = [...newA, n];
  }

  return newA;
}

export async function calling(sesID: string): Promise<ChartData[]> {
  let uID = get(user);
  if (!uID) return [];

  const data: Historic[] = await fetchHistoricData(uID, sesID);
  if (!data.length) {
  }
  console.log(data);

  const chartData = historicTo1RM(data);

  console.log(chartData);

  return chartData;
}
 */
