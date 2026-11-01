import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import ResumeWorkout from './ResumeWorkout.svelte';
import WorkoutStart from './WorkoutStart.svelte';
import WorkoutScreen from './WorkoutScreen.svelte';
import type { Session } from './types';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
});

test('resume reads workout UUID and shows captured targets separately from actual performance', async () => {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(
      JSON.stringify({
        workout: {
          id: '90000000-0000-4000-8000-000000000001',
          status: 'active',
          revision: '7',
          sourceSession: { id: 'plan', name: 'Upper body' },
          exercises: [
            {
              id: 'attempt',
              state: 'draft',
              exercise: { name: 'Bench press' },
              currentVariant: {
                name: 'Heavy',
                targetSets: [{ id: 'future', targetWeight: '99.00' }],
              },
              sets: [
                {
                  id: 'set-1',
                  position: 1,
                  state: 'pending',
                  targetReps: 12,
                  targetWeight: '40.25',
                  recordedReps: null,
                  recordedWeight: null,
                },
                {
                  id: 'set-2',
                  position: 2,
                  state: 'performed',
                  targetReps: 10,
                  targetWeight: '50.00',
                  recordedReps: 0,
                  recordedWeight: '50.00',
                },
              ],
            },
          ],
        },
      })
    )
  );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutScreen, {
    target: document.body,
    props: { workoutId: '90000000-0000-4000-8000-000000000001' },
  });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Bench press'));
  expect(document.body.textContent).toContain('Heavy');
  expect(document.body.textContent).toContain('12 reps · 40.25 kg target');
  expect(document.body.textContent).toContain('Pending · no performance recorded');
  expect(document.body.textContent).toContain('Performed: 0 reps · 50.00 kg');
  expect(document.body.textContent).not.toContain('99.00');
  expect(fetchMock.mock.calls[0][0]).toBe(
    'https://backend.example/api/training/workouts/90000000-0000-4000-8000-000000000001'
  );
});

test('starting a different plan identifies the existing workout and retries identical intent', async () => {
  const session: Session = {
    id: '20000000-0000-4000-8000-000000000002',
    name: 'Lower body',
    revision: '9007199254740993',
    createdAt: '2026-11-01T10:00:00Z',
    lastPerformedAt: null,
    lastPerformedStatus: 'never',
    placements: [
      {
        id: 'placement',
        variantId: 'variant',
        position: 1,
        exercise: {
          id: 'exercise',
          name: 'Squat',
          revision: '1',
          muscleGroupCodes: [],
          archived: false,
        },
        variant: {
          id: 'variant',
          name: 'Heavy',
          revision: '1',
          archived: false,
          progressionMode: 'average',
          prescribedSets: 1,
          repThreshold: 12,
          autoIncrement: '2.50',
          restSeconds: 0,
          targetSets: [],
        },
      },
    ],
  };
  const fetchMock = vi
    .fn()
    .mockRejectedValueOnce(new Error('Connection lost'))
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          operation: {
            operationId: 'op',
            acceptedRevision: '1',
            committedAt: '2026-11-01T10:00:00Z',
          },
          workout: {
            id: '90000000-0000-4000-8000-000000000001',
            status: 'active',
            sourceSession: { id: '20000000-0000-4000-8000-000000000001', name: 'Upper body' },
          },
        })
      )
    );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutStart, { target: document.body, props: { session } });
  [...document.querySelectorAll('button')]
    .find((b) => b.textContent?.includes('Start workout'))!
    .click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Not saved'));
  [...document.querySelectorAll('button')].find((b) => b.textContent?.trim() === 'Retry')!.click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Upper body'));
  expect(fetchMock.mock.calls[1][1].body).toBe(fetchMock.mock.calls[0][1].body);
  expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toMatchObject({
    sessionId: session.id,
    expectedSessionRevision: '9007199254740993',
  });
  expect(document.body.textContent).toContain('already active');
  expect(document.querySelector('a')?.getAttribute('href')).toBe(
    '/workout/90000000-0000-4000-8000-000000000001'
  );
});

test('home Continue uses active workout identity and source from the server', async () => {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(
      JSON.stringify({
        workout: {
          id: '90000000-0000-4000-8000-000000000001',
          status: 'active',
          sourceSession: { id: '20000000-0000-4000-8000-000000000001', name: 'Upper body' },
        },
      })
    )
  );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(ResumeWorkout, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Upper body'));
  expect(document.querySelector('a')?.getAttribute('href')).toBe(
    '/workout/90000000-0000-4000-8000-000000000001'
  );
  expect(fetchMock.mock.calls[0][0]).toBe('https://backend.example/api/training/workouts/active');
});
