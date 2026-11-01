import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
import { upgradeRequired } from '$lib/stores/appState';
import type {
  CreateSessionCommand,
  CompleteAttemptCommand,
  QuitWorkoutCommand,
  UpdateConfigurationCommand,
  ConfigurationResult,
  DeleteSessionCommand,
  DeleteSessionResult,
  UpdateSessionCommand,
  ExerciseSelection,
  Session,
  SessionResult,
  RecordSetCommand,
  StartWorkoutCommand,
  Workout,
  WorkoutResult,
} from './types';

export class TrainingError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public details?: unknown
  ) {
    super(message);
  }
}
async function request<T>(path: string, command?: unknown): Promise<T> {
  const response = await fetch(`${PUBLIC_BACKEND_BASE_URL}/api/training${path}`, {
    method: command === undefined ? 'GET' : 'POST',
    credentials: 'include',
    headers: command === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: command === undefined ? undefined : JSON.stringify(command),
  });
  if (!response.ok) {
    if (response.status === 426) upgradeRequired.set(true);
    const error = await response.json().catch(() => ({}));
    throw new TrainingError(
      response.status,
      error.code ?? 'request-failed',
      error.message ?? 'Unable to save or load training data. Retry the request',
      error.details
    );
  }
  return response.json();
}
export const createSession = (command: CreateSessionCommand) =>
  request<SessionResult>('/sessions/create', command);
export const listSessions = () => request<{ sessions: Session[] }>('/sessions');
export const readSession = (id: string) =>
  request<{ session: Session }>(`/sessions/${encodeURIComponent(id)}`);
export const updateSession = (id: string, command: UpdateSessionCommand) =>
  request<SessionResult>(`/sessions/${encodeURIComponent(id)}/update`, command);
export const deleteSession = (id: string, command: DeleteSessionCommand) =>
  request<DeleteSessionResult>(`/sessions/${encodeURIComponent(id)}/delete`, command);

export const searchExercises = (query: string) =>
  request<{ exercises: ExerciseSelection[] }>(`/exercises?q=${encodeURIComponent(query.trim())}`);
export const readExercise = (id: string) =>
  request<ExerciseSelection>(`/exercises/${encodeURIComponent(id)}`);
export const readCatalog = () =>
  request<{ muscleGroups: { code: string; label: string }[] }>('/catalog');

export const startWorkout = (command: StartWorkoutCommand) =>
  request<WorkoutResult>('/workouts/start', command);
export const activeWorkout = () => request<{ workout: Workout | null }>('/workouts/active');
export const readWorkout = (id: string) =>
  request<{ workout: Workout }>(`/workouts/${encodeURIComponent(id)}`);

export const recordSet = (command: RecordSetCommand) =>
  request<WorkoutResult>('/workouts/sets/record', command);

export const completeAttempt = (command: CompleteAttemptCommand) =>
  request<WorkoutResult>('/workouts/complete', command);

export const updateConfiguration = (command: UpdateConfigurationCommand) =>
  request<ConfigurationResult>('/workouts/configuration', command);

export const quitWorkout = (command: QuitWorkoutCommand) =>
  request<WorkoutResult>('/workouts/quit', command);
