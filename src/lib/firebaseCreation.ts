import {
  getFirestore,
  doc,
  setDoc,
  addDoc,
  collection,
  writeBatch,
  runTransaction,
  FirestoreError,
} from 'firebase/firestore';

import { db } from './firebase';

import type { ExInfoPackage } from './firebaseDataHandler';

/**
 * Data shapes for Firestore operations
 */
export interface UserInfo {
  name: string;
  email?: string;
  signupDate?: Date;
}

function simpleUserType(name: string): UserInfo {
  const simple: UserInfo = { name: name };
  return simple;
}

export interface SessionInfo {
  name: string;
  date?: Date;
  notes?: string;
}

function simpleSessionType(name: string): SessionInfo {
  const simple: SessionInfo = { name: name };
  return simple;
}

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
  id?: string;
  finished?: boolean;
}

function simpleExerciseType(exif: ExInfoPackage): ExerciseInfo {
  const name: string = exif.name;
  const sets: number = exif.sets;
  const w: number = exif.weight;

  const rps = new Array(sets).fill(7);
  const wps = new Array(sets).fill(w);

  const prog = {
    sets: sets,
    repsPerSet: rps,
    weightPerSet: wps,
    restSeconds: 0,
  };

  const simple: ExerciseInfo = {
    name: name,
    currentProgress: prog,
  };
  return simple;
}

function simpleExerciseTypeBatch(pcA: ExInfoPackage[]): ExerciseInfo[] {
  let result: ExerciseInfo[] = [];
  pcA.forEach((pck) => {
    const next = simpleExerciseType(pck);
    result = [...result, next];
  });
  return result;
}

export interface HistoryEntryInfo {
  date: Date;
  sets: number;
  repsPerSet: number[];
  weightPerSet: number[];
  notes?: string;
}

function simpleHistoryType(name: string, sets: number, w: number): HistoryEntryInfo {
  const rps = new Array(sets).fill(7);
  const wps = new Array(sets).fill(w);

  const prog = {
    sets: sets,
    repsPerSet: rps,
    weightPerSet: wps,
    restSeconds: 0,
  };

  const simple: HistoryEntryInfo = {
    date: new Date(),
    sets: sets,
    repsPerSet: rps,
    weightPerSet: wps,
  };
  return simple;
}

// 1) UserInfo
const userDummy: UserInfo = {
  name: 'Dummy User',
  email: 'dummy@example.com',
  signupDate: new Date('2025-01-01T10:00:00Z'),
};

// 2) SessionInfo
const sessionDummy: SessionInfo = {
  name: 'Leg Day',
  date: new Date('2025-06-15T08:30:00Z'),
  notes: 'Focus on squats and lunges',
};

// 3) ExerciseInfo
const exerciseDummy: ExerciseInfo = {
  name: 'Squat',
  muscleGroup: 'Legs',
  currentProgress: {
    sets: 4,
    repsPerSet: [8, 8, 6, 6],
    weightPerSet: [80, 80, 85, 85],
    restSeconds: 120,
  },
};

const exerciseDummy2: ExerciseInfo = {
  name: 'Leg Press',
  muscleGroup: 'Legs',
  currentProgress: {
    sets: 3,
    repsPerSet: [12, 10, 8],
    weightPerSet: [200, 220, 240],
    restSeconds: 90,
  },
};

// 4) Batch of ExerciseInfo (for batchAddExercises)
const exercisesBatchDummy: ExerciseInfo[] = [exerciseDummy, exerciseDummy2];

// 5) HistoryEntryInfo
const historyEntryDummy: HistoryEntryInfo = {
  date: new Date('2025-06-01T07:00:00Z'),
  sets: 4,
  repsPerSet: [8, 8, 8, 8],
  weightPerSet: [75, 75, 80, 80],
  notes: 'Felt strong today!',
};

/**
 * Adds a new user document under 'users/{userId}'
 */
export async function addUser(search: string, info: UserInfo): Promise<void> {
  const userRef = doc(db, 'users', search);
  await setDoc(userRef, {
    name: info.name,
    email: info.email || '',
    // signupDate: info.signupDate,
    hasActiveSession: false,
  });
}

/**
 * Adds (or overwrites) a session under 'users/{uID}/sessions/{sessionName}'
 * Uses the session name as the document ID.
 * Returns the sessionName used as the ID.
 */
export async function addSessionByName(uID: string, info: SessionInfo): Promise<string> {
  // Use the session name directly as the document ID
  const sessionId = info.name;
  const sessionRef = doc(db, 'users', uID, 'sessions', sessionId);

  await setDoc(sessionRef, {
    name: info.name,
    // date: info.date,
    notes: info.notes || '',
    exCount: 0,
  });

  return sessionId;
}

/**
 * Adds one exercise to an existing session
 * Returns the generated exerciseId
 */
export async function addExercise(
  search: { userId: string; sessionId: string },
  info: ExerciseInfo,
): Promise<string> {
  const { userId, sessionId } = search;
  const sessionRef = doc(db, 'users', userId, 'sessions', sessionId);
  const exercisesCol = collection(db, 'users', userId, 'sessions', sessionId, 'exercises');
  // autogenererad ID
  const newExerciseRef = doc(exercisesCol);

  await runTransaction(db, async (tx) => {
    const sessionSnap = await tx.get(sessionRef);
    if (!sessionSnap.exists()) {
      throw new Error(`firestore error 69: Session ${sessionId} for user ${userId} does not exist`);
    }

    // läs exCount
    const prevCount = (sessionSnap.data().exCount as number) || 0;
    const newCount = prevCount + 1;

    tx.set(newExerciseRef, {
      name: info.name,
      muscleGroup: info.muscleGroup || '',
      currentProgress: {
        sets: info.currentProgress.sets,
        repsPerSet: info.currentProgress.repsPerSet,
        weightPerSet: info.currentProgress.weightPerSet,
        restSeconds: info.currentProgress.restSeconds,
      },
      order: newCount,
      id: newExerciseRef.id,
    });

    // uppdatera session exCount
    tx.update(sessionRef, { exCount: newCount });
  });

  return newExerciseRef.id;
}

/**
 * Batch adds multiple exercises to an existing session
 */
export async function batchAddExercises(
  search: { userId: string; sessionId: string },
  infos: ExerciseInfo[],
): Promise<void> {
  const batch = writeBatch(db);
  const exercisesCol = collection(
    db,
    'users',
    search.userId,
    'sessions',
    search.sessionId,
    'exercises',
  );

  let exCount = 0;
  infos.forEach((info, index) => {
    const exerciseRef = doc(exercisesCol);

    batch.set(exerciseRef, {
      name: info.name,
      //muscleGroup: info.muscleGroup,
      currentProgress: {
        sets: info.currentProgress.sets,
        repsPerSet: info.currentProgress.repsPerSet,
        weightPerSet: info.currentProgress.weightPerSet,
        restSeconds: info.currentProgress.restSeconds,
      },
      order: index,
      id: exerciseRef.id,
    });

    exCount++;
  });
  await batch.commit();

  const sessionRef = doc(db, 'users', search.userId, 'sessions', search.sessionId);

  await setDoc(sessionRef, {
    exCount: exCount,
  });
}

/**
 * Pushes a history entry under '.../exercises/{exerciseId}/history'
 * Returns the generated entryId
 */
export async function pushHistoryEntry(
  search: { userId: string; sessionId: string; exerciseId: string },
  info: HistoryEntryInfo,
): Promise<string> {
  const historyCol = collection(
    db,
    'users',
    search.userId,
    'sessions',
    search.sessionId,
    'exercises',
    search.exerciseId,
    'history',
  );
  const historyRef = await addDoc(historyCol, {
    date: info.date,
    sets: info.sets,
    repsPerSet: info.repsPerSet,
    weightPerSet: info.weightPerSet,
    notes: info.notes || '',
  });
  return historyRef.id;
}

export async function testDB() {
  await addUser('user1', simpleUserType('user1'));

  const sif = simpleSessionType('upper');
  await addSessionByName('user1', sif);

  const search = { userId: 'user1', sessionId: 'upper' };
  const pck: ExInfoPackage = { name: 'row', sets: 3, weight: 40 };
  const eif = simpleExerciseType(pck);
  // await addExercise(search, eif)

  await batchAddExercises(search, exercisesBatchDummy);

  console.log('Added files to DB.');
}

export async function betterAdd(sessionName: string, exif: ExInfoPackage[]) {
  const s: SessionInfo = {
    name: sessionName,
  };
  await addSessionByName('user1', s);

  const exinfo = simpleExerciseTypeBatch(exif);

  const search = { userId: 'user1', sessionId: sessionName };
  await batchAddExercises(search, exinfo);
}
