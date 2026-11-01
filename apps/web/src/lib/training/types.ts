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
// Other explicit intent tags extend this union in the library-selection slice.
export type SessionItem = CreateExerciseIntent;
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
}
