import {
  getFirestore,
  doc,
  setDoc,
  addDoc,
  collection,
  writeBatch,
  runTransaction,
  FirestoreError,
  updateDoc,
  getDoc,
} from 'firebase/firestore';

import { db } from './firebase';

interface ExInfoPackage {
  name: string;
  weight: number;
  sets: number;
  autoIncrease?: number;
  repThreshold?: number;
}

/**
 * Data shapes for Firestore operations
 */
export interface UserInfo {
  name: string;
  pass: string;
  email?: string;
  signupDate?: Date;
}

function simpleUserType(name: string, pass: string): UserInfo {
  const simple: UserInfo = { name: name, pass: pass };
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
  id?: number;
  finished?: boolean;
  autoIncrease?: number;
  repThreshold?: number;
}

function simpleExerciseType(exif: ExInfoPackage): ExerciseInfo {
  const name: string = exif.name;
  const sets: number = exif.sets;
  const w: number = exif.weight;
  const autoInc: number = exif.autoIncrease ?? 2.5; //standard 2.5kg
  const repThres: number = exif.repThreshold ?? 12; //standard 12reps
  console.log('😃\n', exif.repThreshold, '\n'); //TODO: Remove log

  const rps = new Array(sets).fill(Math.round(repThres / 1.75));
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
    autoIncrease: autoInc,
    repThreshold: repThres,
  };
  return simple;
}

function simpleExerciseTypeBatch(pcA: ExInfoPackage[]): ExerciseInfo[] {
  let result: ExerciseInfo[] = [];
  pcA.forEach((pck) => {
    const next = simpleExerciseType(pck);
    console.log('THIS:', next.repThreshold); //TODO: remove log
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
export async function addUser(uID: string, pass: string, info: UserInfo): Promise<void> {
  const userRef = doc(db, 'users', uID);

  await setDoc(userRef, {
    name: info.name,
    pass: pass,
    email: info.email || '',
    signupDate: new Date(),
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

  console.log('ADDING SEESION');

  await setDoc(sessionRef, {
    name: info.name,
    date: new Date(),
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
  info: ExerciseInfo
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

    tx.set(
      newExerciseRef,
      {
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
        autoIncrease: info.autoIncrease,
        repThreshold: info.repThreshold,
      },
      { merge: true }
    );

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
  infos: ExerciseInfo[]
): Promise<void> {
  const batch = writeBatch(db);
  const exercisesCol = collection(
    db,
    'users',
    search.userId,
    'sessions',
    search.sessionId,
    'exercises'
  );

  let exCount = 0;
  infos.forEach((info, index) => {
    const exerciseRef = doc(exercisesCol);

    batch.set(
      exerciseRef,
      {
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
        autoIncrease: info.autoIncrease,
        repThreshold: info.repThreshold,
      },
      { merge: true }
    );

    exCount++;
  });
  await batch.commit();

  const sessionRef = doc(db, 'users', search.userId, 'sessions', search.sessionId);

  await setDoc(
    sessionRef,
    {
      exCount: exCount,
    },
    { merge: true }
  );
}

/**
 * Pushes a history entry under '.../exercises/{exerciseId}/history'
 * Returns the generated entryId
 */
export async function pushHistoryEntry(
  search: { userId: string; sessionId: string; exerciseId: string },
  info: HistoryEntryInfo
): Promise<string> {
  const historyCol = collection(
    db,
    'users',
    search.userId,
    'sessions',
    search.sessionId,
    'exercises',
    search.exerciseId,
    'history'
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

export async function addUserByForm(name: string, pass: string, mail: string) {
  let info: UserInfo = {
    name: name,
    pass: pass,
    email: mail,
    signupDate: new Date(),
  };

  await addUser(name, 'pass', info);
}

export async function betterAdd(uID: string, sessionName: string, exif: ExInfoPackage[]) {
  const s: SessionInfo = {
    name: sessionName,
  };
  console.log(exif);

  await addSessionByName(uID, s);

  const exinfo = simpleExerciseTypeBatch(exif);

  const search = { userId: uID, sessionId: sessionName };
  await batchAddExercises(search, exinfo);
}

export async function setActivityStatus(
  uID: string,
  sesID: string,
  activating: boolean,
  fin?: number[]
) {
  // Use the session name directly as the document ID
  const globalRef = doc(db, 'users', uID);
  const localRef = doc(db, 'users', uID, 'sessions', sesID);

  const finished = fin ?? [];

  console.log('Activity status changed, now:', activating);

  if (activating) {
    await updateDoc(globalRef, {
      hasActiveSession: activating,
      activeSessionName: sesID,
    });

    await updateDoc(localRef, {
      isSessionActive: activating,
      finished: finished,
    });
  } else {
    await updateDoc(globalRef, { hasActiveSession: activating });
    await updateDoc(localRef, { isSessionActive: activating });
  }
}

export async function loadFinishedExercises(
  uID: string,
  sesID: string
): Promise<{ finishedIDXS: number[]; unfinished: boolean }> {
  let idxs: number[];
  let fin: boolean;

  const localRef = doc(db, 'users', uID, 'sessions', sesID);

  const snap = await getDoc(localRef);

  if (snap.exists()) {
    const data = snap.data();
    idxs = data.finished ?? [];
    fin = data.isSessionActive ?? false;

    console.log('finsihed indexes:', idxs);
  } else {
    console.error('Document does not exist.');
    fin = false;
    idxs = [];
  }

  return { unfinished: fin, finishedIDXS: idxs };
}

export async function signInOrSignUp(
  username: string,
  password: string,
  signUpAvailible: boolean
): Promise<boolean> {
  const nameKey = username.toLowerCase().trim();
  const userRef = doc(db, 'users', nameKey);

  try {
    return await runTransaction(db, async (tx) => {
      const userSnap = await tx.get(userRef);

      if (userSnap.exists()) {
        // User already exists, log in
        if (userSnap.data().pass == password) {
          return true;
        }
        return false;
      } else if (signUpAvailible) {
        if (confirm(`Createa a new user with username: ${username}`)) {
          const d = simpleUserType(username, password);
          await addUser(username, password, d);
          return true;
        }
        return false;
      } else {
        return false;
      }
    });
  } catch (e) {
    if ((e as FirestoreError).code === 'aborted') {
      console.error('Transaction repeatedly conflicted.');
    }
    throw e;
  }
}
