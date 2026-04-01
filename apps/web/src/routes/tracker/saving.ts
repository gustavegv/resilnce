import type { ExerciseInfo } from './dbFetches';

export type ProgressUpdate = {
  id: string;
  reps: number[];
  weights: number[];
};

const DEFAULT_REP_THRESHOLD = 12;
const DEFAULT_AUTO_INCREASE = 2.5;
const REP_RESET_DIVISOR = 1.7;

function checkRepThresholdMet(reps: number[], threshold: number): boolean {
  if (reps.length === 0) {
    return false;
  }

  const average = reps.reduce((sum, rep) => sum + rep, 0) / reps.length;
  return average > threshold;
}

export function buildProgressUpdate(
  currentExercise: ExerciseInfo | undefined
): ProgressUpdate | null {
  if (!currentExercise || currentExercise.id == null) {
    return null;
  }

  const reps = [...currentExercise.currentProgress.repsPerSet];
  const threshold = currentExercise.rep_threshold ?? DEFAULT_REP_THRESHOLD;
  let nextWeight = currentExercise.currentProgress.weightPerSet[0] ?? 0;

  if (checkRepThresholdMet(reps, threshold)) {
    const resetRepTarget = Math.floor(threshold / REP_RESET_DIVISOR);
    reps.fill(resetRepTarget);
    nextWeight += currentExercise.auto_increase ?? DEFAULT_AUTO_INCREASE;
  }

  return {
    id: String(currentExercise.id),
    reps,
    weights: Array(reps.length).fill(nextWeight),
  };
}

export function areAllExercisesFinished(exercises: ExerciseInfo[]): boolean {
  return exercises.every((exercise) => exercise.finished === true);
}
