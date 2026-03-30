import type { ExerciseEdit, ExerciseInfo } from './dbFetches';

export type EditFieldValues = {
  name: string;
  sets: string;
  weight: string;
  repThreshold: string;
  autoIncrease: string;
};

export function createEmptyEditFields(): EditFieldValues {
  return {
    name: '',
    sets: '',
    weight: '',
    repThreshold: '',
    autoIncrease: '',
  };
}

export function buildExerciseEdit(
  currentExercise: ExerciseInfo | undefined,
  fields: EditFieldValues
): ExerciseEdit | null {
  if (!currentExercise?.id) {
    return null;
  }

  const edit: ExerciseEdit = {
    id: String(currentExercise.id),
  };

  if (fields.name !== '') {
    edit.name = fields.name;
  }

  if (fields.sets !== '') {
    edit.sets = Number(fields.sets);
  }

  if (fields.weight !== '') {
    edit.weight = Number(fields.weight);
  }

  if (fields.repThreshold !== '') {
    edit.repThreshold = Number(fields.repThreshold);
  }

  if (fields.autoIncrease !== '') {
    edit.autoIncrease = Number(fields.autoIncrease);
  }

  return edit;
}

export function hasEditChanges(edit: ExerciseEdit): boolean {
  return (
    edit.name !== undefined ||
    edit.sets !== undefined ||
    edit.weight !== undefined ||
    edit.repThreshold !== undefined ||
    edit.autoIncrease !== undefined
  );
}

export function validateExerciseEdit(edit: ExerciseEdit): string | null {
  if (edit.name !== undefined && edit.name.length > 100) {
    return 'Exercise name cannot be longer than 100 characters.';
  }

  if (edit.sets !== undefined) {
    if (!Number.isInteger(edit.sets)) {
      return 'Sets must be a whole number.';
    }

    if (edit.sets < 1 || edit.sets > 20) {
      return 'Sets must be between 1 and 20.';
    }
  }

  if (edit.weight !== undefined) {
    if (Number.isNaN(edit.weight)) {
      return 'Weight must be a number.';
    }

    if (edit.weight < 0 || edit.weight > 9999) {
      return 'Weight must be between 0 and 9999.';
    }
  }

  if (edit.repThreshold !== undefined) {
    if (!Number.isInteger(edit.repThreshold)) {
      return 'Rep threshold must be a whole number.';
    }

    if (edit.repThreshold < 1 || edit.repThreshold > 99) {
      return 'Rep threshold must be between 1 and 99.';
    }
  }

  if (edit.autoIncrease !== undefined) {
    if (Number.isNaN(edit.autoIncrease)) {
      return 'Auto increase must be a number.';
    }

    if (edit.autoIncrease < 0.25 || edit.autoIncrease > 99) {
      return 'Auto increase must be between 0.25 and 99.';
    }
  }

  return null;
}

function resizeReps(reps: number[], nextSetCount: number): number[] {
  const resizedReps = reps.slice(0, nextSetCount);

  while (resizedReps.length < nextSetCount) {
    resizedReps.push(0);
  }

  return resizedReps;
}

export function applyExerciseEdit(
  currentExercise: ExerciseInfo | undefined,
  edit: ExerciseEdit
): void {
  if (!currentExercise) {
    return;
  }

  if (edit.name !== undefined) {
    currentExercise.name = edit.name;
  }

  if (edit.repThreshold !== undefined) {
    currentExercise.rep_threshold = edit.repThreshold;
  }

  if (edit.autoIncrease !== undefined) {
    currentExercise.auto_increase = edit.autoIncrease;
  }

  const nextSetCount = edit.sets ?? currentExercise.currentProgress.sets;
  const nextWeight = edit.weight ?? currentExercise.currentProgress.weightPerSet[0] ?? 0;

  if (edit.sets !== undefined) {
    currentExercise.currentProgress.sets = nextSetCount;
    currentExercise.currentProgress.repsPerSet = resizeReps(
      currentExercise.currentProgress.repsPerSet,
      nextSetCount
    );
  }

  if (edit.sets !== undefined || edit.weight !== undefined) {
    currentExercise.currentProgress.weightPerSet = Array(nextSetCount).fill(nextWeight);
  }
}
