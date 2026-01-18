import { db } from './firebase';
import {
  collection,
  query,
  orderBy,
  getDocs,
  getDoc,
  doc,
  updateDoc,
  arrayUnion,
  writeBatch,
  WriteBatch,
  type DocumentData,
  Timestamp,
  limit,
} from 'firebase/firestore';

import type { ExerciseInfo } from './firebaseCreation';

export interface EditData {
  user: string;
  sesID: string;
  exID: string;
  oldName: string;
  newName?: string;
  newW?: number;
  addedSets: number;
}

export interface Set {
  id: number;
  reps: number;
}

export interface Historic {
  avgSet: number;
  weightH: number;
  date: Date;
}

export interface Exercise {
  name: string;
  sets: Set[];
  weight: number;
  historic: Historic[];
  order?: number;
  tag: string;
}

export interface SessionMetaData {
  id: string;
  name: string;
  date?: Date;
  deleted?: boolean;
}

type ExInfo = {
  name: string;
  weight: number;
  sets: number;
};

type SessionInfo = {
  name: string;
  exercises: ExInfo[];
};

export async function getOrderedExercises(uID: string, sesID: string): Promise<ExerciseInfo[]> {
  const colRef = collection(db, 'users', uID, 'sessions', sesID, 'exercises');
  const q = query(colRef, orderBy('order', 'asc'));

  const snapshot = await getDocs(q);

  if (snapshot.empty) {
    console.error('Empty snapshot');
    return [];
  }

  return snapshot.docs.map((doc) => doc.data() as ExerciseInfo);
}

export async function getAllSessionMeta(
  uID: string
): Promise<{ slugs: SessionMetaData[]; active: boolean }> {
  const colRef = collection(db, 'users', uID, 'sessions');

  const snapshot = await getDocs(colRef);

  const slugs: SessionMetaData[] = snapshot.docs.map((doc) => {
    const data = doc.data();

    let formattedDate: Date | undefined = undefined;
    if (data.date) {
      const sec = data.date.seconds;
      const nano = data.date.nanoseconds;
      const ts = new Timestamp(sec, nano);
      formattedDate = ts.toDate();
    }

    return {
      id: doc.id,
      name: data.title || '(untitled)', // fallback om .title saknas
      deleted: data.sessionDeleted || false,
      date: formattedDate || undefined,
    };
  });

  const docRef = doc(db, 'users', uID);
  const docSnap = await getDoc(docRef);
  const data = docSnap.data();

  if (data) {
    const active = data.hasActiveSession ?? false;
    return { slugs: slugs, active: active };
  } else {
    return { slugs: slugs, active: false };
  }
}

export async function checkActiveSession(
  uID: string
): Promise<{ active: boolean; session: string } | null> {
  const docRef = doc(db, 'users', uID);
  const docSnap = await getDoc(docRef);

  if (!docSnap.exists()) {
    console.warn('No such document!');
    return null;
  }

  const data = docSnap.data();
  return {
    active: data.hasActiveSession,
    session: data.activeSessionName,
  };
}

// samla och gör om inkommande data till en historiskt sesh
function createHistoricData(blocks: number[], weight: number): Historic {
  const totalReps = blocks.reduce((total, num) => total + num, 0);
  const avgSet = totalReps / blocks.length;

  const now = new Date();
  const formatted =
    now.getFullYear() +
    '-' +
    String(now.getMonth() + 1).padStart(2, '0') +
    '-' +
    String(now.getDate()).padStart(2, '0') +
    ' ' +
    String(now.getHours()).padStart(2, '0') +
    ':' +
    String(now.getMinutes()).padStart(2, '0') +
    ':' +
    String(now.getSeconds()).padStart(2, '0');

  const date = new Date(formatted);

  return { avgSet: avgSet, weightH: weight, date: date };
}

export async function saveRecordedLift(
  uID: string,
  sesID: string,
  repArray: number[],
  weight: number,
  exTag: string
): Promise<number[]> {
  if (exTag == 'error') {
    console.error('Special ex-ID not found in DB.');
  }

  const hisData = createHistoricData(repArray, weight);

  // push historic data and update sets
  const exRef = doc(db, 'users', uID, 'sessions', sesID, 'exercises', String(exTag));

  let curProg;

  let updatedStats: DocumentData;

  const docSnap = await getDoc(exRef);
  const data = docSnap.data();
  if (!data) {
    return [0];
  } // TODO: Bad
  const repThreshold = data.repThreshold ?? 12;

  if (tryAutoIncrease(hisData, repThreshold)) {
    let autoInc: number = 2.5; //standard 2.5

    if (data) {
      autoInc = data.autoIncrease ?? 2.5; //standard 2.5
    }

    const wps = new Array(repArray.length).fill(weight + autoInc);
    const rps = new Array(repArray.length).fill(Math.round(repThreshold / 1.6)); //TODO: Magic number

    updatedStats = {
      'currentProgress.repsPerSet': rps,
      'currentProgress.weightPerSet': wps,
      history: arrayUnion(hisData),
    };
  } else {
    updatedStats = {
      'currentProgress.repsPerSet': repArray,
      history: arrayUnion(hisData),
    };
  }

  updateDoc(exRef, updatedStats);

  return repArray;
}

function tryAutoIncrease(history: any, threshold: number): boolean {
  if (history.avgSet >= threshold) {
    return true;
  } else {
    return false;
  }
}

export async function fakeDeleteSession(uID: string, sesID: string) {
  const globalRef = doc(db, 'users', uID);
  const localRef = doc(db, 'users', uID, 'sessions', sesID);

  await updateDoc(localRef, {
    sessionDeleted: true,
  });
}

export async function editExercise(eData: EditData) {
  const uID = eData.user;
  const sesID = eData.sesID;
  const exID = eData.exID;

  const exRef = doc(db, 'users', uID, 'sessions', sesID, 'exercises', exID);

  const updatedStats: any = {};

  if (eData.newName !== undefined) {
    updatedStats.name = eData.newName;
  }
  if (eData.newW !== undefined) {
    updatedStats.weight = eData.newW;
  }

  if (Object.keys(updatedStats).length > 0) {
    await updateDoc(exRef, updatedStats);
  }
}

export async function fetchHistoricData(uID: string, sesID: string): Promise<Historic[]> {
  const colRef = collection(db, 'users', uID, 'sessions', sesID, 'exercises');
  const q = query(colRef, orderBy('order', 'asc'), limit(1));

  const snapshot = await getDocs(q);

  if (snapshot.empty) {
    console.error('Empty snapshot');
    return [];
  }

  const firstDoc = snapshot.docs[0];
  console.log(firstDoc);

  const firstDocData = firstDoc?.data().history as Historic[];

  return firstDocData;
}
