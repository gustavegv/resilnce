import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import WorkoutScreen from './WorkoutScreen.svelte';
import { beforeNavigate } from '$app/navigation';
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
  id: '91000000-0000-4000-8000-000000000001',
  status: 'active',
  revision: '9007199254740993',
  origin: 'session',
  sourceSession: { id: 'plan', name: 'Upper body' },
  startedAt: '2026-11-01T10:00:00Z',
  endedAt: null,
  exercises: [
    {
      id: 'attempt-1',
      variantId: 'variant-1',
      position: 1,
      state: 'draft',
      origin: 'planned',
      configRevision: '1',
      progressionMode: 'average',
      repThreshold: 12,
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
        id: 'variant-1',
        name: 'Heavy',
        revision: '1',
        progressionMode: 'average',
        repThreshold: 12,
        autoIncrement: '2.50',
        restSeconds: 0,
        prescribedSets: 2,
        archived: false,
        targetSets: [],
      },
      sets: [
        {
          id: 'set-1',
          targetSetId: 'target-1',
          position: 1,
          kind: 'prescribed',
          state: 'pending',
          targetReps: 9,
          targetWeight: '40.25',
          recordedReps: null,
          recordedWeight: null,
          recordedAt: null,
          lateWeightOverride: false,
        },
        {
          id: 'set-2',
          targetSetId: 'target-2',
          position: 2,
          kind: 'prescribed',
          state: 'pending',
          targetReps: 10,
          targetWeight: '50.00',
          recordedReps: null,
          recordedWeight: null,
          recordedAt: null,
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
        committedAt: '2026-11-01T10:00:00Z',
      },
      workout,
    })
  );
const button = (text: string) =>
  [...document.querySelectorAll('button')].find((b) => b.textContent?.trim() === text)!;
const count = (index = 0) => document.querySelectorAll<HTMLButtonElement>('.stepper-count')[index];
async function open(fetchMock: ReturnType<typeof vi.fn>) {
  user.set(account);
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutScreen, { target: document.body, props: { workoutId: fixture.id } });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Bench press'));
}

test('only an explicit tap records chosen zero; acknowledgment confirms actuals and permits the next ordered save', async () => {
  let acknowledge!: (response: Response) => void;
  const initial = structuredClone(fixture);
  initial.exercises[0].sets[0].targetReps = 1;
  const saved = structuredClone(initial);
  saved.revision = '9007199254740994';
  Object.assign(saved.exercises[0].sets[0], {
    state: 'performed',
    recordedReps: 0,
    recordedWeight: '40.25',
    recordedAt: '2026-11-01T10:00:01Z',
  });
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(initial))
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          acknowledge = resolve;
        })
    )
    .mockResolvedValueOnce(response(saved));
  await open(fetchMock);
  expect(fetchMock).toHaveBeenCalledTimes(1);
  document.querySelector<HTMLButtonElement>('.stepper')!.click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Saving'));
  expect(count()?.textContent).toBe('0');
  expect(
    [...document.querySelectorAll<HTMLButtonElement>('.stepper-count')].every((b) => b.disabled)
  ).toBe(true);
  expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toMatchObject({
    workoutId: fixture.id,
    attemptId: 'attempt-1',
    setId: 'set-1',
    expectedWorkoutRevision: '9007199254740993',
    state: 'performed',
    recordedReps: 0,
  });
  expect(JSON.parse(fetchMock.mock.calls[1][1].body)).not.toHaveProperty('recordedWeight');
  acknowledge(response(saved));
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Performed: 0 reps · 40.25 kg')
  );
  expect(document.body.textContent).toContain('Saved');
  count(1).click();
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
  expect(JSON.parse(fetchMock.mock.calls[2][1].body)).toMatchObject({
    setId: 'set-2',
    recordedReps: 10,
    expectedWorkoutRevision: '9007199254740994',
  });
});

test('expired account keeps unresolved input mounted and prevents retry under another identity', async () => {
  const { default: WorkoutPage } = await import('../../routes/workout/[id]/+page.svelte');
  const { page } = await import('$app/state');
  Object.assign(page, { params: { id: fixture.id } });
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new Error('Response lost'));
  user.set(account);
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutPage, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Bench press'));
  count().click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Response lost'));
  user.set(null);
  await vi.waitFor(() => expect(count()?.textContent).toBe('9'));
  expect(document.body.textContent).toContain('account changed or expired');
  user.set({ ...account, id: 'different-owner' });
  await vi.waitFor(() => expect(button('Retry').disabled).toBe(true));
  button('Retry').click();
  expect(fetchMock).toHaveBeenCalledTimes(2);
  expect(count()?.textContent).toBe('9');
});

test('lost response retains input, pauses recording, guards navigation/unload, and retries the identical intent', async () => {
  const saved = structuredClone(fixture);
  saved.revision = '9007199254740994';
  Object.assign(saved.exercises[0].sets[0], {
    state: 'performed',
    recordedReps: 9,
    recordedWeight: '40.25',
    recordedAt: '2026-11-01T10:00:01Z',
  });
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new Error('Response lost'))
    .mockResolvedValueOnce(response(saved));
  await open(fetchMock);
  count().click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Response lost'));
  expect(document.body.textContent).toContain('Not saved');
  expect(count()?.textContent).toBe('9');
  const cancel = vi.fn();
  const navigate = vi.mocked(beforeNavigate).mock.calls.at(-1)![0];
  navigate({ cancel } as unknown as Parameters<typeof navigate>[0]);
  await vi.waitFor(() => expect(document.body.textContent).toContain('Stay and resolve it before leaving.'));
  expect(cancel).toHaveBeenCalledOnce();
  const unload = new Event('beforeunload', { cancelable: true });
  window.dispatchEvent(unload);
  expect(unload.defaultPrevented).toBe(true);
  expect(
    [...document.querySelectorAll<HTMLButtonElement>('.stepper-count')].every((b) => b.disabled)
  ).toBe(true);
  button('Stay').click();
  button('Retry').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Performed: 9 reps · 40.25 kg')
  );
  expect(fetchMock.mock.calls[2][1].body).toBe(fetchMock.mock.calls[1][1].body);
  const afterSave = new Event('beforeunload', { cancelable: true });
  window.dispatchEvent(afterSave);
  expect(afterSave.defaultPrevented).toBe(false);
});

test.each(['accept', 'reapply'])(
  'stale input remains visible until explicit %s of reviewed server state',
  async (choice) => {
    const current = structuredClone(fixture);
    current.revision = '9007199254740995';
    Object.assign(current.exercises[0].sets[0], {
      state: 'performed',
      recordedReps: 4,
      recordedWeight: '40.25',
      targetWeight: '45.00',
    });
    const reapplied = structuredClone(current);
    reapplied.revision = '9007199254740996';
    reapplied.exercises[0].sets[0].recordedReps = 9;
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(response(fixture))
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ code: 'stale-revision', message: 'Changed on another device' }),
          { status: 409 }
        )
      )
      .mockResolvedValueOnce(response(current))
      .mockResolvedValueOnce(response(reapplied));
    await open(fetchMock);
    count().click();
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain('Current server revision 9007199254740995')
    );
    expect(count()?.textContent).toBe('9');
    expect(document.body.textContent).toContain('4 reps · 40.25 kg');
    expect(document.body.textContent).toContain('45.00 kg');
    expect(fetchMock).toHaveBeenCalledTimes(3);
    if (choice === 'accept') {
      button('Accept server state').click();
      await vi.waitFor(() => expect(count()?.textContent).toBe('4'));
      expect(document.body.textContent).toContain('Performed: 4 reps · 40.25 kg');
      expect(fetchMock).toHaveBeenCalledTimes(3);
    } else {
      button('Reapply local input to reviewed revision').click();
      await vi.waitFor(() =>
        expect(document.body.textContent).toContain('Performed: 9 reps · 40.25 kg')
      );
      const original = JSON.parse(fetchMock.mock.calls[1][1].body),
        reapplied = JSON.parse(fetchMock.mock.calls[3][1].body);
      expect(reapplied).toMatchObject({
        setId: 'set-1',
        workoutId: fixture.id,
        recordedReps: 9,
        expectedWorkoutRevision: '9007199254740995',
      });
      expect(reapplied.operationId).not.toBe(original.operationId);
    }
  }
);

test('reviewing a finalized or removed attempt retains local reps and forbids reapply', async () => {
  const current = structuredClone(fixture);
  current.revision = '9007199254740994';
  current.status = 'ended';
  current.exercises[0].state = 'completed';
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 'resource-finalized', message: 'Already complete' }), {
        status: 409,
      })
    )
    .mockResolvedValueOnce(response(current));
  await open(fetchMock);
  count().click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Current server revision'));
  expect(count()?.textContent).toBe('9');
  expect(button('Reapply local input to reviewed revision').disabled).toBe(true);
  button('Accept server state').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended'));
  expect(document.querySelectorAll('input')).toHaveLength(0);
  expect(fetchMock).toHaveBeenCalledTimes(3);
});

test('correction, explicit skip, and reset keep distinct states and exact set identity', async () => {
  const initial = structuredClone(fixture);
  Object.assign(initial.exercises[0].sets[0], {
    state: 'performed',
    recordedReps: 8,
    recordedWeight: '40.25',
    targetWeight: '45.00',
  });
  const corrected = structuredClone(initial);
  corrected.revision = '2';
  corrected.exercises[0].sets[0].recordedReps = 9;
  const skipped = structuredClone(corrected);
  skipped.revision = '3';
  Object.assign(skipped.exercises[0].sets[0], {
    state: 'skipped',
    recordedReps: null,
    recordedWeight: null,
  });
  const reset = structuredClone(skipped);
  reset.revision = '4';
  reset.exercises[0].sets[0].state = 'pending';
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(initial))
    .mockResolvedValueOnce(response(corrected))
    .mockResolvedValueOnce(response(skipped))
    .mockResolvedValueOnce(response(reset));
  await open(fetchMock);
  document.querySelector<HTMLButtonElement>('[aria-label="Increase reps for set 1"]')!.click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Performed: 9 reps · 40.25 kg')
  );
  button('Skip set').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Skipped · no performance recorded')
  );
  button('Reset to pending').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).not.toContain('Skipped · no performance recorded')
  );
  expect(document.body.textContent).toContain('Pending · no performance recorded');
  expect(JSON.parse(fetchMock.mock.calls[2][1].body)).toMatchObject({
    setId: 'set-1',
    state: 'skipped',
  });
  expect(JSON.parse(fetchMock.mock.calls[2][1].body)).not.toHaveProperty('recordedReps');
  expect(JSON.parse(fetchMock.mock.calls[3][1].body)).toMatchObject({
    setId: 'set-1',
    state: 'pending',
  });
  expect(JSON.parse(fetchMock.mock.calls[3][1].body)).not.toHaveProperty('recordedReps');
});

test('retry after an uncertain response restores accepted state before recording resumes', async () => {
  const accepted = structuredClone(fixture);
  accepted.revision = '9007199254740994';
  Object.assign(accepted.exercises[0].sets[0], {
    state: 'performed',
    recordedReps: 9,
    recordedWeight: '40.25',
  });
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new Error('Lost acknowledgment'))
    .mockResolvedValueOnce(response(accepted));
  await open(fetchMock);
  count().click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Lost acknowledgment'));
  button('Retry').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Performed: 9 reps · 40.25 kg')
  );
  expect(fetchMock.mock.calls[2][1].body).toBe(fetchMock.mock.calls[1][1].body);
  expect(fetchMock.mock.calls[2][1].method).toBe('POST');
  expect(count()?.textContent).toBe('9');
});

test('mount, exercise navigation, and another-device restored state perform no recording or completion', async () => {
  const restored = structuredClone(fixture);
  restored.exercises.push({
    ...structuredClone(restored.exercises[0]),
    id: 'attempt-2',
    position: 2,
    sets: [{ ...structuredClone(restored.exercises[0].sets[0]), id: 'set-3' }],
  });
  Object.assign(restored.exercises[0].sets[0], {
    state: 'performed',
    recordedReps: 0,
    recordedWeight: '40.25',
  });
  const fetchMock = vi.fn().mockResolvedValueOnce(response(restored));
  await open(fetchMock);
  expect(document.body.textContent).toContain('Performed: 0 reps · 40.25 kg');
  button('Skip').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Exercise 2 of 2'));
  expect(fetchMock).toHaveBeenCalledTimes(1);
  expect(document.body.textContent).toContain('Draft');
  expect(document.body.textContent).toContain('Pending · no performance recorded');
});

test('retains the original tracking shell and set steppers with variant identity', async () => {
  await open(vi.fn().mockResolvedValue(response(fixture)));
  expect(document.querySelector('main.app-container')).not.toBeNull();
  expect(document.querySelector('.progress-track')).not.toBeNull();
  expect(document.body.textContent).toContain('Session Progress');
  expect(document.body.textContent).toContain('Exercise 1 of 1');
  expect(document.querySelectorAll('.counter-container')).toHaveLength(2);
  expect(document.querySelectorAll('.stepper')).toHaveLength(4);
  expect(document.querySelector('.confirm-button')?.textContent).toContain('Confirm');
  expect(document.body.textContent).toContain('Heavy');
});

test('original count tap explicitly records the displayed target and steppers save the chosen reps', async () => {
  const saved = structuredClone(fixture);
  saved.revision = '2';
  Object.assign(saved.exercises[0].sets[0], {
    state: 'performed',
    recordedReps: 9,
    recordedWeight: '40.25',
  });
  const corrected = structuredClone(saved);
  corrected.revision = '3';
  corrected.exercises[0].sets[0].recordedReps = 8;
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(response(saved))
    .mockResolvedValueOnce(response(corrected));
  await open(fetchMock);
  expect(fetchMock).toHaveBeenCalledTimes(1);
  const count = document.querySelector<HTMLButtonElement>('.stepper-count')!;
  expect(count.textContent).toBe('9');
  expect(document.body.textContent).toContain('40.25 kg target');
  expect(document.body.textContent).toContain('50.00 kg target');
  count.click();
  await vi.waitFor(() =>
    expect(document.querySelector('.counter-container.set--active')).not.toBeNull()
  );
  expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toMatchObject({
    setId: 'set-1',
    state: 'performed',
    recordedReps: 9,
  });
  document.querySelector<HTMLButtonElement>('.stepper')!.click();
  await vi.waitFor(() => expect(document.querySelector('.stepper-count')?.textContent).toBe('8'));
  expect(JSON.parse(fetchMock.mock.calls[2][1].body)).toMatchObject({
    setId: 'set-1',
    state: 'performed',
    recordedReps: 8,
    expectedWorkoutRevision: '2',
  });
});

test.each(['pending', 'performed'] as const)(
  'set options for %s sets contain only skip and reset alongside performance details',
  async (state) => {
    const snapshot = structuredClone(fixture);
    if (state === 'performed')
      Object.assign(snapshot.exercises[0].sets[0], {
        state,
        recordedReps: 0,
        recordedWeight: '40.25',
      });
    await open(vi.fn().mockResolvedValue(response(snapshot)));
    const options = document.querySelector('.set-options')!;
    expect(options.querySelectorAll('input')).toHaveLength(0);
    expect([...options.querySelectorAll('button')].map((item) => item.textContent?.trim())).toEqual(
      ['Skip set', 'Reset to pending']
    );
    expect(document.querySelector('.stepper-count')).not.toBeNull();
    expect(document.querySelectorAll('.stepper')).toHaveLength(4);
    if (state === 'performed')
      expect(options.textContent).toContain('Performed: 0 reps · 40.25 kg');
  }
);
