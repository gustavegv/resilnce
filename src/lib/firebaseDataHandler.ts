import { db } from './firebase';
import {
  collection,
  query,
  orderBy,
  getDocs,
  doc,
  updateDoc,
  arrayUnion
} from 'firebase/firestore';

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

type RecNum = Record<number, number>;

export async function getOrderedExercises(): Promise<Exercise[]> {
  const q = query(collection(db, 'push-day'), orderBy('order', 'asc'));
  const snapshot = await getDocs(q);
  return snapshot.docs.map(doc => doc.data() as Exercise);
}

function transformBlocksToSets(blocks: RecNum): Set[] {
  return Object.entries(blocks).map(([id, reps]) => ({
    id: Number(id),
    reps
  }));
}

// samla och gör om inkommande data till en historiskt sesh
function createHistoricData(blocks: Record<number,number>, weight: number){
  const entries = Object.entries(blocks);
  const totalReps = entries.reduce((sum, [_, reps]) => sum + reps, 0);
  const avgSet = totalReps / entries.length;

  return { avgSet, weightH: weight };
}

export async function saveRecordedLift(
  blocks: RecNum,
  weight: number,
  exTag: string
): Promise<void> {
  
  const hisData = createHistoricData(blocks, weight)

  // push historic data and update sets
  const exerciseRef = doc(db, 'push-day', exTag);

  if (tryAutoIncrease(hisData)){
   await updateDoc(exerciseRef, {
      historic: arrayUnion(hisData),
      sets: resetSetCount(blocks),
      weight: (weight+5)
    });
  } else {
    await updateDoc(exerciseRef, {
      historic: arrayUnion(hisData),
      sets: transformBlocksToSets(blocks)
    }); 
  }
}

function tryAutoIncrease(history: any):boolean {
  console.log(history)
  if (history.avgSet > 11){
    return true
  } else {
    return false
  }
}

function resetSetCount(blocks: RecNum){
    return Object.entries(blocks).map(([id, reps]) => ({
    id: Number(id),
    reps: 6
  }));
}