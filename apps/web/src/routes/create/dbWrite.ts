import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';

const baseURL: string = PUBLIC_BACKEND_BASE_URL;

function address(dir: string): string {
  return baseURL + dir;
}

async function postDB(dir: string, data: any): Promise<boolean> {
  const res = await fetch(address(`/db/` + dir), {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const text = await res.text();
    console.error(`Request failed: ${res.status} ${res.statusText}`);
    console.error('Response body:', text);
    return false;
  } else {
    return true;
  }
}

export interface ExpandedExerciseData {
  id: string;
  ex_name: string;
  rep_threshold: string;
  auto_increase: string;
  set_count: string;
  weight_per_set: number[];
  rep_per_set: number[];
  order: string;
  finished: boolean;
}

export interface ExerciseDataPackaged {
  name: string;
  weight: number;
  sets: number;
  autoIncrease?: number;
  repThreshold?: number;
}

// A -> B
function exercisePackageToExpanded(
  a: ExerciseDataPackaged,
  order: number,
  id = ''
): ExpandedExerciseData {
  const repThreshold = a.repThreshold ?? 0;
  const autoIncrease = a.autoIncrease ?? 0;

  return {
    id,
    ex_name: a.name,

    rep_threshold: String(repThreshold),
    auto_increase: String(autoIncrease),
    set_count: String(a.sets),
    order: String(order),

    weight_per_set: Array(a.sets).fill(a.weight),
    rep_per_set: Array(a.sets).fill(repThreshold),

    finished: false,
  };
}

function expandExerciseData(list: ExerciseDataPackaged[]): ExpandedExerciseData[] {
  return list.map((a, idx) => exercisePackageToExpanded(a, idx + 1));
}

export async function CreateSession(
  sesID: string,
  exInfoPack: ExerciseDataPackaged[]
): Promise<boolean> {
  const ExpandedExInfo = expandExerciseData(exInfoPack);

  const payload: { sesID: string; exI: ExpandedExerciseData[] } = {
    sesID: sesID,
    exI: ExpandedExInfo,
  };

  return await postDB(`newSession`, payload);
}
