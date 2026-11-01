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
  [...document.querySelectorAll('button')].find((b) => b.textContent?.trim() === text)!;
async function open(fetchMock: ReturnType<typeof vi.fn>) {
  user.set(account);
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutScreen, { target: document.body, props: { workoutId: fixture.id } });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Bench press'));
}
function ended() {
  const w = structuredClone(fixture);
  w.status = 'ended';
  w.endedAt = '2026-11-01T10:03:00Z';
  w.revision = '9007199254740994';
  w.exercises[0].state = 'completed';
  w.exercises[0].completedAt = w.endedAt;
  w.exercises[0].currentVariant.revision = '3';
  w.exercises[0].currentVariant.targetSets[0].targetReps = 6;
  w.exercises[0].currentVariant.targetSets[0].targetWeight = '42.50';
  w.exercises[0].currentVariant.targetSets[1].targetReps = 6;
  w.exercises[0].currentVariant.targetSets[1].targetWeight = '52.50';
  return w;
}

test('Finish sends reviewed identity only and renders frozen actuals separately from authoritative future targets', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(response(ended()));
  await open(fetchMock);
  button('Finish').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended · Saved'));
  expect(fetchMock.mock.calls[1][0]).toContain('/workouts/complete');
  expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toEqual({
    operationId: expect.any(String),
    workoutId: 'workout',
    attemptId: 'attempt',
    exerciseId: 'exercise',
    variantId: 'variant',
    expectedWorkoutRevision: '9007199254740993',
    expectedVariantRevision: '2',
  });
  expect(document.body.textContent).toContain('Performed: 12 reps · 40.00 kg');
  expect(document.body.textContent).toContain('Performed: 10 reps · 50.00 kg');
  expect(document.body.textContent).toContain('Current next targets');
  expect(document.body.textContent).toContain('6 reps · 42.50 kg');
  expect(document.body.textContent).toContain('6 reps · 52.50 kg');
  expect(button('Finish')).toBeUndefined();
  expect(document.querySelectorAll('input')).toHaveLength(0);
});

test('Finish waits for explicit pending resolution and successful preceding saves', async () => {
  const pending = structuredClone(fixture);
  Object.assign(pending.exercises[0].sets[1], {
    state: 'pending',
    recordedReps: null,
    recordedWeight: null,
    recordedAt: null,
  });
  const resolved = structuredClone(pending);
  resolved.revision = '9007199254740994';
  resolved.exercises[0].sets[1].state = 'skipped';
  let acknowledge!: (response: Response) => void;
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(pending))
    .mockImplementationOnce(
      () =>
        new Promise<Response>((r) => {
          acknowledge = r;
        })
    )
    .mockResolvedValueOnce(response(ended()));
  await open(fetchMock);
  expect(button('Finish').disabled).toBe(true);
  expect(document.body.textContent).toContain('Resolve every pending set');
  const skipButtons = [...document.querySelectorAll('button')].filter(
    (b) => b.textContent?.trim() === 'Skip set'
  );
  skipButtons[1].click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Saving…'));
  expect(button('Finish').disabled).toBe(true);
  button('Finish').click();
  expect(fetchMock).toHaveBeenCalledTimes(2);
  acknowledge(response(resolved));
  await vi.waitFor(() => expect(button('Finish').disabled).toBe(false));
  button('Finish').click();
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
  expect(JSON.parse(fetchMock.mock.calls[2][1].body).expectedWorkoutRevision).toBe(
    '9007199254740994'
  );
});

test('unsent correction and open shared editor each block Finish', async () => {
  const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(response(fixture)));
  await open(fetchMock);
  const input = document.querySelector('input')!;
  input.value = '0';
  input.dispatchEvent(new Event('input', { bubbles: true }));
  await vi.waitFor(() => expect(button('Finish').disabled).toBe(true));
  expect(fetchMock).toHaveBeenCalledTimes(1);
  button('Discard local input').click();
  await vi.waitFor(() => expect(button('Finish').disabled).toBe(false));
  button('Edit exercise').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Editing exercise'));
  expect(button('Finish').disabled).toBe(true);
  button('Cancel').click();
  await vi.waitFor(() => expect(button('Finish').disabled).toBe(false));
});

test('lost completion response retains exact intent and blocks discard, navigation and account overtaking until retry resolves', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new TypeError('Connection lost'))
    .mockResolvedValueOnce(response(ended()));
  await open(fetchMock);
  button('Finish').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Completion is unconfirmed'));
  expect(button('Discard local input')).toBeUndefined();
  expect(button('Finish').disabled).toBe(true);
  expect(button('Edit exercise').disabled).toBe(true);
  expect(document.body.textContent).toContain('Performed: 12 reps · 40.00 kg');
  const cancel = vi.fn();
  const navigate = vi.mocked(beforeNavigate).mock.calls.at(-1)![0];
  navigate({ cancel } as unknown as Parameters<typeof navigate>[0]);
  expect(cancel).toHaveBeenCalledTimes(1);
  const unload = new Event('beforeunload', { cancelable: true });
  window.dispatchEvent(unload);
  expect(unload.defaultPrevented).toBe(true);
  user.set({ ...account, id: 'other', email: 'other@example.com' });
  await vi.waitFor(() => expect(button('Retry').disabled).toBe(true));
  button('Retry').click();
  expect(fetchMock).toHaveBeenCalledTimes(2);
  user.set(account);
  await vi.waitFor(() => expect(button('Retry').disabled).toBe(false));
  button('Retry').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended · Saved'));
  expect(fetchMock.mock.calls[2][1].body).toBe(fetchMock.mock.calls[1][1].body);
  expect(button('Discard local input')).toBeUndefined();
  const resolvedCancel = vi.fn();
  navigate({ cancel: resolvedCancel } as unknown as Parameters<typeof navigate>[0]);
  expect(resolvedCancel).not.toHaveBeenCalled();
});

test('completion conflict retains intent and reapplies only after current configuration is reviewed', async () => {
  const reviewed = structuredClone(fixture);
  reviewed.revision = '9007199254740995';
  reviewed.exercises[0].currentVariant.revision = '4';
  reviewed.exercises[0].configRevision = '4';
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 'stale-revision', message: 'Configuration changed' }), {
        status: 409,
      })
    )
    .mockResolvedValueOnce(response(reviewed))
    .mockResolvedValueOnce(response(ended()));
  await open(fetchMock);
  button('Finish').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Current server revision 9007199254740995')
  );
  expect(button('Finish').disabled).toBe(true);
  expect(button('Reapply local input to reviewed revision').disabled).toBe(false);
  button('Reapply local input to reviewed revision').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended · Saved'));
  const initial = JSON.parse(fetchMock.mock.calls[1][1].body),
    next = JSON.parse(fetchMock.mock.calls[3][1].body);
  expect(next.operationId).not.toBe(initial.operationId);
  expect(next.expectedWorkoutRevision).toBe('9007199254740995');
  expect(next.expectedVariantRevision).toBe('4');
});

test('another-device finality can be accepted without a second completion or reopening actuals', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 'stale-revision', message: 'Workout changed' }), {
        status: 409,
      })
    )
    .mockResolvedValueOnce(response(ended()));
  await open(fetchMock);
  button('Finish').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Current server revision 9007199254740994')
  );
  expect(button('Reapply local input to reviewed revision').disabled).toBe(true);
  button('Accept server state and discard local input').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended · Saved'));
  expect(fetchMock).toHaveBeenCalledTimes(3);
  expect(document.querySelectorAll('input')).toHaveLength(0);
});

test('actionable bounds rejection permits reviewing configuration before a new completion intent', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          code: 'invalid-field',
          message:
            'Automatic increase would exceed 9999 kg. Adjust the weight or increment, then finish again',
        }),
        { status: 422 }
      )
    )
    .mockResolvedValueOnce(response(fixture));
  await open(fetchMock);
  button('Finish').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Adjust the weight or increment')
  );
  expect(button('Discard local input')).toBeDefined();
  button('Discard local input').click();
  await vi.waitFor(() => expect(button('Edit exercise').disabled).toBe(false));
  button('Edit exercise').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Editing exercise'));
});
