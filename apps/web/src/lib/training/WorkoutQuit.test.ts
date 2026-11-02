import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import { beforeNavigate } from '$app/navigation';
import WorkoutScreen from './WorkoutScreen.svelte';
import { user, type Account } from '$lib/stores/appState';
import type { Workout } from './types';
const account: Account = {
  id: 'owner',
  email: 'owner@example.com',
  name: 'Owner',
  developmentMode: true,
  capabilities: {
    applicationModel: 'exercise-centric',
    trainingWrites: true,
    writeMaintenance: false,
  },
};
const fixture: Workout = {
  id: 'workout',
  status: 'active',
  revision: '9007199254740993',
  origin: 'session',
  sourceSession: { id: 'session', name: 'Upper body' },
  startedAt: '2026-11-01T10:00:00Z',
  endedAt: null,
  exercises: [
    {
      id: 'attempt',
      variantId: 'variant',
      position: 1,
      state: 'draft',
      origin: 'planned',
      configRevision: '2',
      progressionMode: 'average',
      repThreshold: 11,
      autoIncrement: '2.50',
      restSeconds: 0,
      completedAt: null,
      recordedRpe: null,
      notes: null,
      exercise: {
        id: 'exercise',
        name: 'Bench press',
        revision: '1',
        muscleGroupCodes: [],
        archived: false,
      },
      currentVariant: {
        id: 'variant',
        name: 'Standard',
        revision: '2',
        progressionMode: 'average',
        repThreshold: 11,
        autoIncrement: '2.50',
        restSeconds: 0,
        prescribedSets: 2,
        archived: false,
        targetSets: [
          { id: 'target1', position: 1, targetReps: 11, targetWeight: '40.00' },
          { id: 'target2', position: 2, targetReps: 11, targetWeight: '50.00' },
        ],
      },
      sets: [
        {
          id: 'set1',
          targetSetId: 'target1',
          position: 1,
          kind: 'prescribed',
          state: 'performed',
          targetReps: 11,
          targetWeight: '40.00',
          recordedReps: 12,
          recordedWeight: '40.00',
          recordedAt: '2026-11-01T10:01:00Z',
          lateWeightOverride: false,
        },
        {
          id: 'set2',
          targetSetId: 'target2',
          position: 2,
          kind: 'prescribed',
          state: 'performed',
          targetReps: 11,
          targetWeight: '50.00',
          recordedReps: 10,
          recordedWeight: '50.00',
          recordedAt: '2026-11-01T10:02:00Z',
          lateWeightOverride: false,
        },
      ],
    },
  ],
};
let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  user.set(null);
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});
const response = (workout: Workout) =>
  new Response(
    JSON.stringify({
      operation: {
        operationId: 'op',
        acceptedRevision: workout.revision,
        committedAt: '2026-11-01T10:03:00Z',
      },
      workout,
    })
  );
const button = (text: string) =>
  [...document.querySelectorAll('button')].find(
    (b) => b.textContent?.trim() === (text === 'Finish' ? 'Confirm' : text)
  )!;
async function open(fetchMock: ReturnType<typeof vi.fn>) {
  user.set(account);
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutScreen, { target: document.body, props: { workoutId: fixture.id } });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Bench press'));
}
test('Quit requires explicit confirmation and retains completed work while removing drafts', async () => {
  const mixed = structuredClone(fixture);
  mixed.exercises[0].state = 'completed';
  const draft = structuredClone(fixture.exercises[0]);
  draft.id = 'draft';
  draft.exercise.name = 'Squat';
  mixed.exercises.push(draft);
  const endedWorkout = structuredClone(mixed);
  endedWorkout.status = 'ended';
  endedWorkout.revision = '9007199254740994';
  endedWorkout.endedAt = '2026-11-01T10:03:00Z';
  endedWorkout.exercises.pop();
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(mixed))
    .mockResolvedValueOnce(response(endedWorkout));
  await open(fetchMock);
  button('Quit workout').click();
  await vi.waitFor(() =>
    expect(document.body.textContent?.replace(/\s+/g, ' ')).toContain(
      'Completed work and progression are kept. Unfinished attempts and their sets are discarded.'
    )
  );
  expect(fetchMock).toHaveBeenCalledTimes(1);
  button('Keep training').click();
  await vi.waitFor(() => expect(button('Confirm quit')).toBeUndefined());
  button('Quit workout').click();
  await vi.waitFor(() => expect(button('Confirm quit')).toBeDefined());
  button('Confirm quit').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended · Saved'));
  expect(fetchMock.mock.calls[1][0]).toContain('/workouts/quit');
  expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toEqual({
    operationId: expect.any(String),
    workoutId: 'workout',
    expectedWorkoutRevision: '9007199254740993',
  });
  expect(document.body.textContent).toContain('Performed: 12 reps · 40.00 kg');
  expect(document.body.textContent).not.toContain('Squat');
  expect(button('Quit workout')).toBeUndefined();
});

test('lost quit response keeps exact intent and prevents leaving until retry confirms the ended workout', async () => {
  const endedWorkout = structuredClone(fixture);
  endedWorkout.status = 'ended';
  endedWorkout.revision = '9007199254740994';
  endedWorkout.endedAt = '2026-11-01T10:03:00Z';
  endedWorkout.exercises = [];
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new TypeError('Connection lost'))
    .mockResolvedValueOnce(response(endedWorkout));
  await open(fetchMock);
  button('Quit workout').click();
  await vi.waitFor(() => expect(button('Confirm quit')).toBeDefined());
  button('Confirm quit').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Quit is unconfirmed'));
  expect(button('Quit workout').disabled).toBe(true);
  expect(button('Finish').disabled).toBe(true);
  expect(button('Edit exercise').disabled).toBe(true);
  const navigate = vi.mocked(beforeNavigate).mock.calls.at(-1)![0];
  const cancel = vi.fn();
  navigate({ cancel } as unknown as Parameters<typeof navigate>[0]);
  expect(cancel).toHaveBeenCalledTimes(1);
  const unload = new Event('beforeunload', { cancelable: true });
  window.dispatchEvent(unload);
  expect(unload.defaultPrevented).toBe(true);
  user.set({ ...account, id: 'other' });
  await vi.waitFor(() => expect(button('Retry').disabled).toBe(true));
  button('Retry').click();
  expect(fetchMock).toHaveBeenCalledTimes(2);
  user.set(account);
  await vi.waitFor(() => expect(button('Retry').disabled).toBe(false));
  button('Retry').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended · Saved'));
  expect(fetchMock.mock.calls[2][1].body).toBe(fetchMock.mock.calls[1][1].body);
  const resolvedCancel = vi.fn();
  navigate({ cancel: resolvedCancel } as unknown as Parameters<typeof navigate>[0]);
  expect(resolvedCancel).not.toHaveBeenCalled();
});

test('quit cannot overtake an uncertain completion or unresolved recording save', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new TypeError('Connection lost'));
  await open(fetchMock);
  button('Finish').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Completion is unconfirmed'));
  expect(button('Quit workout').disabled).toBe(true);
  button('Quit workout').click();
  expect(button('Confirm quit')).toBeUndefined();
  expect(fetchMock).toHaveBeenCalledTimes(2);
  await unmount(component!);
  component = undefined;
  document.body.innerHTML = '';
  let acknowledge!: (r: Response) => void;
  const recording = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          acknowledge = resolve;
        })
    );
  await open(recording);
  document.querySelector<HTMLButtonElement>('.stepper-count')!.click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Saving…'));
  expect(button('Quit workout').disabled).toBe(true);
  button('Quit workout').click();
  expect(recording).toHaveBeenCalledTimes(2);
  const saved = structuredClone(fixture);
  saved.revision = '9007199254740994';
  acknowledge(response(saved));
  await vi.waitFor(() => expect(button('Quit workout').disabled).toBe(false));
});

test('stale quit requires explicit review before reapplying a new intent against the reviewed revision', async () => {
  const reviewed = structuredClone(fixture);
  reviewed.revision = '9007199254740995';
  reviewed.exercises[0].state = 'completed';
  const draft = structuredClone(fixture.exercises[0]);
  draft.id = 'draft';
  draft.exercise.name = 'Squat';
  reviewed.exercises.push(draft);
  const endedWorkout = structuredClone(reviewed);
  endedWorkout.revision = '9007199254740996';
  endedWorkout.status = 'ended';
  endedWorkout.endedAt = '2026-11-01T10:03:00Z';
  endedWorkout.exercises.pop();
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 'stale-revision', message: 'Workout changed' }), {
        status: 409,
      })
    )
    .mockResolvedValueOnce(response(reviewed))
    .mockResolvedValueOnce(response(endedWorkout));
  await open(fetchMock);
  button('Quit workout').click();
  await vi.waitFor(() => expect(button('Confirm quit')).toBeDefined());
  button('Confirm quit').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Current server revision 9007199254740995')
  );
  expect(document.body.textContent).toContain('completed');
  expect(fetchMock).toHaveBeenCalledTimes(3);
  button('Reapply local input to reviewed revision').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended · Saved'));
  const first = JSON.parse(fetchMock.mock.calls[1][1].body),
    next = JSON.parse(fetchMock.mock.calls[3][1].body);
  expect(next.expectedWorkoutRevision).toBe('9007199254740995');
  expect(next.operationId).not.toBe(first.operationId);
  expect(document.body.textContent).not.toContain('Squat');
});

test('closing the screen sends no quit and server active state remains resumable', async () => {
  const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(response(fixture)));
  await open(fetchMock);
  await unmount(component!);
  component = undefined;
  document.body.innerHTML = '';
  expect(fetchMock).toHaveBeenCalledTimes(1);
  await open(fetchMock);
  expect(document.body.textContent).toContain('Active workout · Saved');
  expect(fetchMock.mock.calls.every(([, options]) => options.method === 'GET')).toBe(true);
});

test('an unresolved stepper correction and an open shared editor block quit until resolved', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new Error('Correction response lost'))
    .mockResolvedValue(response(fixture));
  await open(fetchMock);
  document.querySelector<HTMLButtonElement>('.stepper')!.click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Correction response lost'));
  await vi.waitFor(() => expect(button('Quit workout').disabled).toBe(true));
  button('Retry').click();
  await vi.waitFor(() => expect(button('Quit workout').disabled).toBe(false));
  button('Edit exercise').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Editing exercise'));
  expect(button('Quit workout').disabled).toBe(true);
  button('Cancel').click();
  await vi.waitFor(() => expect(button('Quit workout').disabled).toBe(false));
});
