import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
import { extendTailwindMerge } from 'tailwind-merge';

/**
 * export interface SessionMetaData {
  id: string;       // mail
  name: string;     // ses_name
  date?: Date;      // date_last_ran
  deleted?: boolean;
}
 */

export interface ExerciseInfo {
  name: string;
  muscleGroup?: string;
  currentProgress: {
    sets: number;
    repsPerSet: number[];
    weightPerSet: number[];
    restSeconds: number;
  };
  order?: number;
  id?: number;
  finished?: boolean;
  autoIncrease?: number;
  repThreshold?: number;
}

export interface SessionMetaData {
  id: number;
  name: string;
  date?: Date;
  deleted?: boolean;
}

const baseURL: string = PUBLIC_BACKEND_BASE_URL;

function address(dir: string): string {
  return baseURL + dir;
}

async function fetchDB(dir: string): Promise<any> {
  const res = await fetch(address(`/db/` + dir), {
    method: 'GET',
    credentials: 'include',
  });
  if (!res.ok) {
    throw new Error('Network response was not ok');
  } else {
    const data = await res.json();
    return data;
  }
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
    throw new Error('Network response was not ok');
  } else {
    return true;
  }
}

export async function GetSessions(): Promise<SessionMetaData[]> {
  var b: SessionMetaData[] = [];
  const data = await fetchDB(`mySessions`);

  for (let i = 0; data[i] != undefined; i++) {
    const fb: SessionMetaData = {
      id: data[i].id,
      name: data[i].name,
      date: new Date(data[i].date),
    };
    b.push(fb);
  }

  return b;
}

export async function CheckActiveSession(): Promise<string> {
  const data = await fetchDB(`active`);
  if (!data.ok) {
    console.log(data);
  }
  return data.ActiveSession;
}

export async function GetSessionExercises(sesID: number): Promise<ExerciseInfo[]> {
  const data = await fetchDB(`getExercises?sesID=${sesID}`);

  var exs: ExerciseInfo[] = [];
  console.log('From DB:');
  console.log(data);
  for (let i = 0; data[i] != undefined; i++) {
    var ex: ExerciseInfo = data[i];
    ex.name = data[i].ex_name;
    ex.currentProgress = {
      sets: data[i].set_count,
      repsPerSet: data[i].rep_per_set,
      weightPerSet: data[i].weight_per_set,
      restSeconds: 0,
    };

    exs.push(ex);
  }

  return exs;
}

export async function GetFinishedExercises(sesID: string): Promise<{
  finExs: number[];
  ongoing: boolean;
}> {
  const data = await fetchDB(`finished?sesID=${sesID}`);
  const fins: number[] = data.Finished;
  if (!fins || fins.length == 0) {
    return { finExs: [], ongoing: false };
  }
  return { finExs: fins, ongoing: true };
}

export async function SendUpdate(
  info: {
    reps: number[];
    weights: number[];
    id: string;
  },
  sesID: string
) {
  const success = await postDB(`update?sesID=${sesID}`, info);
}

export async function CompleteSession(sesID: string) {
  await postDB(`complete?sesID=${sesID}`, null);
}

export async function SetActiveSession(sesID: string) {
  await postDB(`setActive?sesID=${sesID}`, null);
}
