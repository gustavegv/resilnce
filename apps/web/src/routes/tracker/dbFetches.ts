import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
import { toast } from 'svelte-sonner';

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
  id?: string | number;
  finished?: boolean;
  auto_increase?: number;
  rep_threshold?: number;
}

export interface SessionMetaData {
  id: number;
  name: string;
  date?: Date;
  deleted?: boolean;
}

export interface ExerciseEdit {
  id: string;
  name?: string;
  sets?: number;
  weight?: number;
  repThreshold?: number;
  autoIncrease?: number;
}

export interface HistoricEntry {
  exID: number;
  saveDate: Date;
  avgRep: number;
  avgWeight: number;
  noOfSets: number;
}

const baseURL: string = PUBLIC_BACKEND_BASE_URL;

function address(dir: string): string {
  return baseURL + dir;
}

function statusToasterHandler(code: number) {
  switch (code) {
    case 400:
      toast.error(
        'The request could not be processed. Please check your credentials or try again.'
      );
      break;
    case 401:
      toast.error('Your session has expired. Please log in again.');
      break;
    case 403:
      toast.error("You don't have permission to access this feature.");
      break;
    case 429:
      toast.error('Too many statistics requests. Please wait a moment and try again.');
      break;

    default:
      toast.error('Network response was not ok. Try again.');
      break;
  }
}

async function fetchDB(dir: string): Promise<any> {
  const res = await fetch(address(`/db/` + dir), {
    method: 'GET',
    credentials: 'include',
  });
  if (!res.ok) {
    statusToasterHandler(res.status);
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
    return false;
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

export async function CheckActiveSession(): Promise<[string, string]> {
  const data = await fetchDB(`active`);
  if (!data.ok) {
    console.log(data);
  }
  return [data.ActiveSession, data.ActiveSessionName];
}

export async function GetSessionExercises(sesID: number): Promise<ExerciseInfo[]> {
  const data = await fetchDB(`getExercises?sesID=${sesID}`);

  var exs: ExerciseInfo[] = [];
  console.log('From DB:');
  console.log(data);
  for (let i = 0; data[i] != undefined; i++) {
    const ex: ExerciseInfo = {
      ...data[i],
      id: Number(data[i].id),
      name: data[i].ex_name,
      currentProgress: {
        sets: data[i].set_count,
        repsPerSet: data[i].rep_per_set,
        weightPerSet: data[i].weight_per_set,
        restSeconds: 0,
      },
    };

    exs.push(ex);
  }

  return exs;
}

export async function GetExerciseHistory(exID: number): Promise<HistoricEntry[]> {
  const data = await fetchDB(`history?exID=${exID}`);
  const history: HistoricEntry[] = [];

  for (let i = 0; data[i] != undefined; i++) {
    history.push({
      exID: data[i].ex_id,
      saveDate: new Date(data[i].save_date),
      avgRep: Number(data[i].avg_rep ?? 0),
      avgWeight: Number(data[i].avg_weight ?? 0),
      noOfSets: Number(data[i].no_of_sets ?? 0),
    });
  }

  return history;
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
  return await postDB(`update?sesID=${sesID}`, info);
}

export async function SendEdit(info: ExerciseEdit, sesID: string) {
  return await postDB(`edit?sesID=${sesID}`, info);
}

export async function CompleteSession(sesID: string) {
  await postDB(`complete?sesID=${sesID}`, null);
}

export async function SetActiveSession(sesID: string) {
  await postDB(`setActive?sesID=${sesID}`, null);
}

export async function DeleteSession(sesID: number): Promise<boolean> {
  return await postDB(`deleteSession?sesID=${sesID}`, null);
}

export async function EditSessionName(sesID: number, name: string): Promise<boolean> {
  return await postDB(`editSession?sesID=${sesID}`, { name });
}

export async function AddSessionExercises(
  sesID: number,
  exercises: ExerciseInfo[]
): Promise<boolean> {
  if (!exercises.length) {
    return true;
  }

  return await postDB(`addSessionExercises?sesID=${sesID}`, { exercises });
}

function getExerciseIDs(exercises: ExerciseInfo[]): number[] | null {
  const exerciseIDs = exercises
    .map((exercise) => exercise.id)
    .filter((id): id is string | number => id != null)
    .map((id) => Number(id));

  if (exerciseIDs.length !== exercises.length) {
    console.error(
      'One or more exercises were missing ids when trying to remove them from a session.'
    );
    return null;
  }

  if (exerciseIDs.some((id) => !Number.isInteger(id) || id <= 0)) {
    console.error(
      'One or more exercise ids were invalid when trying to remove them from a session.'
    );
    return null;
  }

  return exerciseIDs;
}

export async function RemoveSessionExercises(
  sesID: number,
  exercises: ExerciseInfo[]
): Promise<boolean> {
  if (!exercises.length) {
    return true;
  }

  const exerciseIDs = getExerciseIDs(exercises);
  if (!exerciseIDs) {
    return false;
  }

  return await postDB(`removeSessionExercises?sesID=${sesID}`, { exerciseIDs });
}
