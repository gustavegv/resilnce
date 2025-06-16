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
} from 'firebase/firestore';

import type { ExerciseInfo } from './firebaseCreation';

export interface Set {
  id: number;
  reps: number;
}

export interface Historic {
  avgSet: number;
  weightH: number;
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

export interface ExInfoPackage {
  name: string;
  weight: number;
  sets: number;
}

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
  uID: string,
): Promise<{ slugs: SessionMetaData[]; active: boolean }> {
  const colRef = collection(db, 'users', uID, 'sessions');

  const snapshot = await getDocs(colRef);

  const slugs: SessionMetaData[] = snapshot.docs.map((doc) => {
    const data = doc.data();
    return {
      id: doc.id,
      name: data.title || '(untitled)', // fallback om .title saknas
    };
  });

  return { slugs: slugs, active: true };
}

export async function checkActiveSession(
  uID: string,
): Promise<{ active: boolean; session: string } | null> {
  const docRef = doc(db, 'users', uID);
  const docSnap = await getDoc(docRef);

  if (!docSnap.exists()) {
    console.warn('No such document!');
    return null;
  }

  const data = docSnap.data();
  console.log('ahha', data);
  return {
    active: data.hasActiveSession,
    session: data.activeSessionName,
  };
}

// samla och gör om inkommande data till en historiskt sesh
function createHistoricData(blocks: number[], weight: number) {
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

  return { avgSet: avgSet, weightH: weight, date: formatted };
}

export async function saveRecordedLift(
  repArray: number[],
  weight: number,
  exTag: string,
): Promise<number[]> {
  const hisData = createHistoricData(repArray, weight);

  // push historic data and update sets
  const exerciseRef = doc(db, 'push-day', exTag);

  let curProg;

  if (tryAutoIncrease(hisData)) {
    const increment = 5;
    const wps = new Array(repArray.length).fill(weight + increment);

    curProg = {
      repsPerSet: repArray,
      weightPerSet: wps,
    };
  } else {
    curProg = {
      repsPerSet: repArray,
    };
  }

  const updatedStats = {
    currentProgress: curProg,
    history: arrayUnion(hisData),
  };

  updateDoc(exerciseRef, updatedStats);

  return repArray;
}

function tryAutoIncrease(history: any): boolean {
  if (history.avgSet > 11) {
    return true;
  } else {
    return false;
  }
}
