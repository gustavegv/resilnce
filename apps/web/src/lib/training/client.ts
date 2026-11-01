import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
import { upgradeRequired } from '$lib/stores/appState';
import type { CreateSessionCommand, ExerciseSelection, Session, SessionResult } from './types';

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

export const searchExercises = (query: string) =>
  request<{ exercises: ExerciseSelection[] }>(`/exercises?q=${encodeURIComponent(query.trim())}`);
export const readExercise = (id: string) =>
  request<ExerciseSelection>(`/exercises/${encodeURIComponent(id)}`);
export const readCatalog = () =>
  request<{ muscleGroups: { code: string; label: string }[] }>('/catalog');
