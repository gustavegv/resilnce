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

type SessionMetaData = {
  id: string;
  name: string;
};

type ExInfo = {
  name: string;
  weight: number;
  sets: number;
};

type SessionInfo = {
  name: string;
  exercises: ExInfo[];
};

type RecNum = Record<number, number>;

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

export async function getAllSessionMeta(uID: string): Promise<SessionMetaData[]> {
  const colRef = collection(db, 'users', uID, 'sessions');

  const snapshot = await getDocs(colRef);

  const slugs: SessionMetaData[] = snapshot.docs.map((doc) => {
    const data = doc.data();
    return {
      id: doc.id,
      name: data.title || '(untitled)', // fallback om .title saknas
    };
  });

  console.log('Session meta:', slugs);
  return slugs;
}

export async function checkActiveSession(): Promise<{ active: boolean; session: string } | null> {
  const docRef = doc(db, 'sessions', 'session-mangager');
  const docSnap = await getDoc(docRef);

  if (!docSnap.exists()) {
    console.warn('No such document!');
    return null;
  }

  const data = docSnap.data();
  return {
    active: data.active,
    session: data.session,
  };
}

function transformBlocksToSets(blocks: RecNum): Set[] {
  return Object.entries(blocks).map(([id, reps]) => ({
    id: Number(id),
    reps,
  }));
}

// samla och gör om inkommande data till en historiskt sesh
function createHistoricData(blocks: Record<number, number>, weight: number) {
  const entries = Object.entries(blocks);
  const totalReps = entries.reduce((sum, [_, reps]) => sum + reps, 0);
  const avgSet = totalReps / entries.length;

  return { avgSet, weightH: weight };
}

export async function saveRecordedLift(
  blocks: RecNum,
  weight: number,
  exTag: string,
): Promise<void> {
  const hisData = createHistoricData(blocks, weight);

  // push historic data and update sets
  const exerciseRef = doc(db, 'push-day', exTag);

  return; // obs todo remove
  if (tryAutoIncrease(hisData)) {
    await updateDoc(exerciseRef, {
      historic: arrayUnion(hisData),
      sets: resetSetCount(blocks),
      weight: weight + 5,
    });
  } else {
    await updateDoc(exerciseRef, {
      historic: arrayUnion(hisData),
      sets: transformBlocksToSets(blocks),
    });
  }
}

function tryAutoIncrease(history: any): boolean {
  console.log(history);
  if (history.avgSet > 11) {
    return true;
  } else {
    return false;
  }
}

function resetSetCount(blocks: RecNum) {
  return Object.entries(blocks).map(([id, reps]) => ({
    id: Number(id),
    reps: 6,
  }));
}

export async function addNewSession(newSession: SessionInfo) {
  const batch = writeBatch(db);

  const sessionsRef = doc(db, 'sessions', newSession.name);
  batch.set(sessionsRef, {
    title: newSession.name,
  });

  for (const [index, ex] of newSession.exercises.entries()) {
    // this creates a document at: /<session-name>/<exercise-name>

    const setsArray = Array.from({ length: ex.sets }, (_, i) => ({
      id: i + 1,
      reps: 6,
    }));

    const ref = doc(db, newSession.name, ex.name);
    batch.set(ref, {
      weight: ex.weight,
      sets: setsArray,
      order: index,
      name: ex.name,
    });
  }

  await batch.commit();
}
