export type ID = string;
export type Revision = string;
export type Decimal = string;
export interface InitialConfiguration {
  prescribedSets: number;
  initialWeight: Decimal;
  repThreshold?: number;
  autoIncrement?: Decimal;
  restSeconds?: number;
}
export interface CreateExerciseIntent {
  kind: 'createExercise';
  name: string;
  muscleGroupCodes: string[];
  configuration: InitialConfiguration;
}
export interface CreateVariantIntent {
  kind: 'createVariant';
  exerciseId: ID;
  expectedExerciseRevision: Revision;
  name: string;
  configuration: InitialConfiguration;
}
export interface ReuseVariantIntent {
  kind: 'reuseVariant';
  exerciseId: ID;
  variantId: ID;
  expectedVariantRevision: Revision;
}
export type SessionItem = CreateExerciseIntent | CreateVariantIntent | ReuseVariantIntent;
export interface ExerciseSuggestion {
  name: string;
  configuration: InitialConfiguration;
}
export interface ExerciseSelection {
  exercise: Exercise;
  variants: Variant[];
}
export interface CreateSessionCommand {
  operationId: ID;
  name: string;
  items: SessionItem[];
}
export interface TargetSet {
  id: ID;
  position: number;
  targetReps: number;
  targetWeight: Decimal;
}
export interface Exercise {
  id: ID;
  name: string;
  revision: Revision;
  muscleGroupCodes: string[];
  archived: boolean;
}
export interface Variant {
  id: ID;
  name: string;
  revision: Revision;
  progressionMode: 'average' | 'all_sets' | 'off';
  repThreshold: number;
  autoIncrement: Decimal;
  restSeconds: number;
  prescribedSets: number;
  archived: boolean;
  targetSets: TargetSet[];
}
export interface Placement {
  id: ID;
  variantId: ID;
  position: number;
  exercise: Exercise;
  variant: Variant;
}
export interface Session {
  id: ID;
  name: string;
  revision: Revision;
  createdAt: string;
  lastPerformedAt: string | null;
  placements: Placement[];
}
export interface Acknowledgment {
  operationId: ID;
  acceptedRevision: Revision;
  committedAt: string;
}
export interface SessionResult {
  operation: Acknowledgment;
  session: Session;
}
export interface DraftSessionItem {
  draftId: ID;
  intent: SessionItem;
  exercise?: Exercise;
  reviewedVariant?: Variant;
}

export interface StartWorkoutCommand {
  operationId: ID;
  sessionId: ID;
  expectedSessionRevision: Revision;
}
export interface WorkoutSource {
  id: ID;
  name: string;
}
export interface WorkoutSet {
  id: ID;
  targetSetId: ID | null;
  position: number;
  kind: 'prescribed' | 'bonus';
  state: 'pending' | 'performed' | 'skipped';
  targetReps: number;
  targetWeight: Decimal;
  recordedReps: number | null;
  recordedWeight: Decimal | null;
  recordedAt: string | null;
  lateWeightOverride: boolean;
}
export interface WorkoutExercise {
  id: ID;
  variantId: ID;
  position: number;
  state: 'draft' | 'completed';
  origin: 'planned' | 'added';
  configRevision: Revision;
  progressionMode: Variant['progressionMode'];
  repThreshold: number;
  autoIncrement: Decimal;
  restSeconds: number;
  completedAt: string | null;
  recordedRpe: number | null;
  notes: string | null;
  exercise: Exercise;
  currentVariant: Variant;
  sets: WorkoutSet[];
}
export interface Workout {
  id: ID;
  status: 'active' | 'ended';
  revision: Revision;
  origin: 'session' | 'free';
  sourceSession: WorkoutSource | null;
  startedAt: string;
  endedAt: string | null;
  exercises: WorkoutExercise[];
}
export interface WorkoutResult {
  operation: Acknowledgment;
  workout: Workout;
}

export type RecordSetCommand = {
  operationId: ID;
  workoutId: ID;
  attemptId: ID;
  setId: ID;
  expectedWorkoutRevision: Revision;
} & ({ state: 'performed'; recordedReps: number } | { state: 'pending' | 'skipped' });
