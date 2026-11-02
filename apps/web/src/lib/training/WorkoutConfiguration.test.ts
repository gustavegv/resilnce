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
        targetSets: [
          { id: 'target-1', position: 1, targetReps: 12, targetWeight: '40.25' },
          { id: 'target-2', position: 2, targetReps: 10, targetWeight: '50.00' },
        ],
      },
      sets: [
        {
          id: 'set-1',
          targetSetId: 'target-1',
          position: 1,
          kind: 'prescribed',
          state: 'pending',
          targetReps: 12,
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
const field = (name: string) => {
  const label = [...document.querySelectorAll('label')].find(
    (l) => l.textContent?.trim() === name
  )!;
  return (document.getElementById(label.htmlFor) ??
    label.querySelector('input')) as HTMLInputElement;
};
const enter = (name: string, value: string) => {
  const input = field(name);
  input.value = value;
  input.dispatchEvent(new Event('input', { bubbles: true }));
};
async function open(fetchMock: ReturnType<typeof vi.fn>) {
  user.set(account);
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutScreen, { target: document.body, props: { workoutId: fixture.id } });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Bench press'));
}

test('shared edit distinguishes names and mixed weights, and ordinary save patches only changed settings', async () => {
  const saved = structuredClone(fixture);
  saved.revision = '9007199254740994';
  saved.exercises[0].currentVariant.repThreshold = 10;
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(response(saved));
  await open(fetchMock);
  button('Edit exercise').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('every session using this variant')
  );
  expect(field('Exercise name').value).toBe('Bench press');
  expect(field('Variant name').value).toBe('Heavy');
  expect(document.body.textContent).toContain('Mixed target weights');
  expect(field('Weight (kg)').value).toBe('');
  enter('Rep threshold', '10');
  button('Save edit').click();
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  expect(fetchMock.mock.calls[1][0]).toContain('/workouts/configuration');
  expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toMatchObject({
    workoutId: fixture.id,
    attemptId: 'attempt-1',
    exerciseId: 'exercise',
    variantId: 'variant-1',
    expectedWorkoutRevision: '9007199254740993',
    expectedExerciseRevision: '1',
    expectedVariantRevision: '1',
    patch: { repThreshold: 10 },
  });
  expect(JSON.parse(fetchMock.mock.calls[1][1].body).patch).toEqual({ repThreshold: 10 });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Active workout · Saved'));
});

test('count reduction discloses recorded draft deletion while explicit all-sets action is the only weight intent', async () => {
  const initial = structuredClone(fixture);
  Object.assign(initial.exercises[0].sets[1], {
    state: 'performed',
    recordedReps: 8,
    recordedWeight: '50.00',
  });
  const saved = structuredClone(initial);
  saved.revision = '2';
  saved.exercises[0].sets = saved.exercises[0].sets.slice(0, 1);
  saved.exercises[0].sets[0].targetWeight = '45.50';
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(initial))
    .mockResolvedValueOnce(response(saved));
  await open(fetchMock);
  button('Edit exercise').click();
  await vi.waitFor(() => expect(field('Sets').value).toBe('2'));
  enter('Sets', '1');
  enter('Weight (kg)', '45.50');
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('including recorded reps and weights')
  );
  expect(document.body.textContent).toContain('Completed history is kept');
  button('Apply weight to all sets').click();
  button('Save edit').click();
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  expect(JSON.parse(fetchMock.mock.calls[1][1].body).patch).toEqual({
    prescribedSets: 1,
    applyWeightToAllSets: '45.50',
  });
  await vi.waitFor(() => expect(document.querySelectorAll('.set')).toHaveLength(1));
  expect(document.body.textContent).toContain('45.50 kg target');
});

test('an entered common weight alone never equalizes mixed weights on count or name save', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(response(fixture));
  await open(fetchMock);
  button('Edit exercise').click();
  await vi.waitFor(() => expect(field('Sets').value).toBe('2'));
  enter('Weight (kg)', '60');
  enter('Sets', '3');
  enter('Exercise name', 'Barbell press');
  enter('Variant name', 'Light');
  button('Save edit').click();
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  expect(JSON.parse(fetchMock.mock.calls[1][1].body).patch).toEqual({
    prescribedSets: 3,
    exerciseName: 'Barbell press',
    variantName: 'Light',
  });
});

test('unresolved configuration retains fields, pauses recording and retries exact JSON through the shared guard', async () => {
  let acknowledge!: (response: Response) => void;
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new Error('Configuration response lost'))
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          acknowledge = resolve;
        })
    );
  await open(fetchMock);
  button('Edit exercise').click();
  await vi.waitFor(() => expect(field('Sets').value).toBe('2'));
  enter('Variant name', 'New heavy');
  button('Save edit').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('Configuration response lost')
  );
  expect(field('Variant name').value).toBe('New heavy');
  expect(document.body.textContent).toContain('Not saved');
  expect(document.querySelector<HTMLButtonElement>('.stepper-count')!.disabled).toBe(true);
  expect(button('Edit exercise').disabled).toBe(true);
  const cancel = vi.fn();
  const navigate = vi.mocked(beforeNavigate).mock.calls.at(-1)![0];
  navigate({ cancel } as unknown as Parameters<typeof navigate>[0]);
  expect(cancel).toHaveBeenCalledOnce();
  const unload = new Event('beforeunload', { cancelable: true });
  window.dispatchEvent(unload);
  expect(unload.defaultPrevented).toBe(true);
  button('Retry').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Saving'));
  expect(fetchMock.mock.calls[2][1].body).toBe(fetchMock.mock.calls[1][1].body);
  expect(field('Variant name').matches(':disabled')).toBe(true);
  acknowledge(response(fixture));
  await vi.waitFor(() => expect(document.body.textContent).toContain('Active workout · Saved'));
  expect(document.querySelector<HTMLButtonElement>('.stepper-count')!.disabled).toBe(false);
});

test('configuration cannot overtake an unresolved main-control recording', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockRejectedValueOnce(new Error('Recording response lost'));
  await open(fetchMock);
  document.querySelector<HTMLButtonElement>('.stepper-count')!.click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Recording response lost'));
  expect(button('Edit exercise').disabled).toBe(true);
  button('Edit exercise').click();
  expect(fetchMock).toHaveBeenCalledTimes(2);
  expect(document.body.textContent).not.toContain('Editing exercise');
});

test.each(['accept', 'reapply'])(
  'stale configuration requires explicit %s and refreshes all reviewed revisions',
  async (choice) => {
    const current = structuredClone(fixture);
    current.revision = '9007199254740995';
    current.exercises[0].exercise.revision = '4';
    current.exercises[0].currentVariant.revision = '7';
    current.exercises[0].currentVariant.repThreshold = 8;
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(response(fixture))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ code: 'stale-revision', message: 'Configuration changed' }), {
          status: 409,
        })
      )
      .mockResolvedValueOnce(response(current))
      .mockResolvedValueOnce(response(current));
    await open(fetchMock);
    button('Edit exercise').click();
    await vi.waitFor(() => expect(field('Variant name').value).toBe('Heavy'));
    enter('Variant name', 'New name');
    button('Save edit').click();
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain('Current server revision 9007199254740995')
    );
    expect(field('Variant name').value).toBe('New name');
    expect(document.body.textContent).toContain('variant revision 7');
    expect(document.body.textContent).toContain('threshold 8');
    expect(fetchMock).toHaveBeenCalledTimes(3);
    if (choice === 'accept') {
      button('Accept server state').click();
      await vi.waitFor(() => expect(document.body.textContent).not.toContain('Editing exercise'));
      expect(fetchMock).toHaveBeenCalledTimes(3);
    } else {
      button('Reapply local input to reviewed revision').click();
      await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(4));
      const original = JSON.parse(fetchMock.mock.calls[1][1].body),
        reapplied = JSON.parse(fetchMock.mock.calls[3][1].body);
      expect(reapplied).toMatchObject({
        expectedWorkoutRevision: '9007199254740995',
        expectedExerciseRevision: '4',
        expectedVariantRevision: '7',
        patch: { variantName: 'New name' },
      });
      expect(reapplied.operationId).not.toBe(original.operationId);
    }
  }
);

test('name conflict keeps edit fields and permits a new corrected intent without retrying rejected JSON', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({ code: 'name-conflict', message: 'That variant name is reserved' }),
        { status: 409 }
      )
    )
    .mockResolvedValueOnce(response(fixture));
  await open(fetchMock);
  button('Edit exercise').click();
  await vi.waitFor(() => expect(field('Variant name').value).toBe('Heavy'));
  enter('Variant name', 'Reserved');
  button('Save edit').click();
  await vi.waitFor(() =>
    expect(document.body.textContent).toContain('That variant name is reserved')
  );
  expect(field('Variant name').value).toBe('Reserved');
  button('Edit rejected configuration').click();
  await vi.waitFor(() => expect(button('Save edit').disabled).toBe(false));
  enter('Variant name', 'Available');
  button('Save edit').click();
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
  const original = JSON.parse(fetchMock.mock.calls[1][1].body),
    corrected = JSON.parse(fetchMock.mock.calls[2][1].body);
  expect(corrected.operationId).not.toBe(original.operationId);
  expect(corrected.patch).toEqual({ variantName: 'Available' });
});

test('finalized configuration conflict retains the form and disables reapply until server state is accepted', async () => {
  const ended = structuredClone(fixture);
  ended.status = 'ended';
  ended.exercises[0].state = 'completed';
  ended.revision = '2';
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(response(fixture))
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 'resource-finalized', message: 'Already completed' }), {
        status: 409,
      })
    )
    .mockResolvedValueOnce(response(ended));
  await open(fetchMock);
  button('Edit exercise').click();
  await vi.waitFor(() => expect(field('Variant name').value).toBe('Heavy'));
  enter('Variant name', 'My input');
  button('Save edit').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Current server revision 2'));
  expect(field('Variant name').value).toBe('My input');
  expect(button('Reapply local input to reviewed revision').disabled).toBe(true);
  button('Accept server state').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Workout ended'));
  expect(fetchMock).toHaveBeenCalledTimes(3);
  expect(document.querySelectorAll('input')).toHaveLength(0);
});
