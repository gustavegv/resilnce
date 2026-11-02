import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import { goto } from '$app/navigation';
import { user } from '$lib/stores/appState';
import WorkoutStartScreen from './WorkoutStartScreen.svelte';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  user.set(null);
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

const account = {
  id: 'owner',
  email: 'owner@example.com',
  name: 'Owner',
  developmentMode: false,
  capabilities: {
    applicationModel: 'exercise-centric' as const,
    trainingWrites: true,
    writeMaintenance: false,
  },
};
const session = {
  id: 'plan',
  name: 'Upper body',
  revision: '9007199254740993',
  placements: [{ exercise: { archived: false }, variant: { archived: false } }],
};

test('the workout destination loads the current plan and starts it under one loading state', async () => {
  let finishPlan!: (response: Response) => void;
  const fetchMock = vi
    .fn()
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finishPlan = resolve;
        })
    )
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          workout: { id: 'active', sourceSession: { id: 'plan', name: 'Upper body' } },
        })
      )
    );
  user.set(account);
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutStartScreen, { target: document.body, props: { sessionId: 'plan' } });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
  expect(document.querySelector('main.app-container')).not.toBeNull();
  expect(document.body.textContent).toContain('Loading workout');
  expect(fetchMock.mock.calls[0][0]).toBe('https://backend.example/api/training/sessions/plan');
  finishPlan(new Response(JSON.stringify({ session })));
  await vi.waitFor(() =>
    expect(goto).toHaveBeenCalledWith('/workout/active', { replaceState: true })
  );
  expect(fetchMock).toHaveBeenCalledTimes(2);
  expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toMatchObject({
    sessionId: 'plan',
    expectedSessionRevision: '9007199254740993',
  });
  expect(document.body.textContent).not.toContain('Workout ready');
  expect(document.querySelector('a')).toBeNull();
});

test('a failed plan read retries on the workout screen without starting until it succeeds', async () => {
  const fetchMock = vi
    .fn()
    .mockRejectedValueOnce(new Error('Connection lost'))
    .mockResolvedValueOnce(new Response(JSON.stringify({ session })))
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          workout: { id: 'active', sourceSession: { id: 'plan', name: 'Upper body' } },
        })
      )
    );
  user.set(account);
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutStartScreen, { target: document.body, props: { sessionId: 'plan' } });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Connection lost'));
  expect(fetchMock).toHaveBeenCalledTimes(1);
  expect(goto).not.toHaveBeenCalled();
  document.querySelector('button')!.click();
  await vi.waitFor(() => expect(goto).toHaveBeenCalled());
  expect(fetchMock).toHaveBeenCalledTimes(3);
});

test('switching accounts during plan loading cannot start the previous account’s session', async () => {
  let finishPlan!: (response: Response) => void;
  const fetchMock = vi.fn().mockImplementation(
    () =>
      new Promise<Response>((resolve) => {
        finishPlan = resolve;
      })
  );
  user.set(account);
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutStartScreen, { target: document.body, props: { sessionId: 'plan' } });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
  user.set({ ...account, id: 'another-account' });
  finishPlan(new Response(JSON.stringify({ session })));
  await vi.waitFor(() => expect(document.body.textContent).toContain('Your account changed'));
  expect(fetchMock).toHaveBeenCalledTimes(1);
  expect(goto).not.toHaveBeenCalled();
});
