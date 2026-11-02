import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import { goto } from '$app/navigation';
import WorkoutStart from './WorkoutStart.svelte';
import type { Session } from './types';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

test('a session-card start navigates only after confirmation and retries the same operation', async () => {
  const session = {
    id: 'plan',
    name: 'Upper body',
    revision: '7',
    placements: [{ exercise: { archived: false }, variant: { archived: false } }],
  } as Session;
  const fetchMock = vi
    .fn()
    .mockRejectedValueOnce(new Error('Connection lost'))
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          workout: { id: 'active-workout', sourceSession: { id: 'plan', name: 'Upper body' } },
        })
      )
    );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(WorkoutStart, {
    target: document.body,
    props: { session, autostart: true },
  });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Not saved'));
  expect(goto).not.toHaveBeenCalled();
  [...document.querySelectorAll('button')]
    .find((button) => button.textContent === 'Retry')!
    .click();
  await vi.waitFor(() =>
    expect(goto).toHaveBeenCalledWith('/workout/active-workout', { replaceState: true })
  );
  expect(fetchMock).toHaveBeenCalledTimes(2);
  expect(fetchMock.mock.calls[1][1].body).toBe(fetchMock.mock.calls[0][1].body);
  expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toMatchObject({
    sessionId: 'plan',
    expectedSessionRevision: '7',
  });
});

test('automatic starts show only workout loading until the destination replaces the start route', async () => {
  const session = {
    id: 'plan',
    name: 'Upper body',
    revision: '7',
    placements: [{ exercise: { archived: false }, variant: { archived: false } }],
  } as Session;
  let acknowledge!: (response: Response) => void;
  vi.stubGlobal(
    'fetch',
    vi.fn().mockImplementation(
      () =>
        new Promise<Response>((resolve) => {
          acknowledge = resolve;
        })
    )
  );
  component = mount(WorkoutStart, { target: document.body, props: { session, autostart: true } });
  await vi.waitFor(() => expect(acknowledge).toBeDefined());
  expect(document.body.textContent).toContain('Loading workout');
  expect(document.body.textContent).not.toContain('Start workout');
  acknowledge(
    new Response(
      JSON.stringify({
        workout: { id: 'active-workout', sourceSession: { id: 'plan', name: 'Upper body' } },
      })
    )
  );
  await vi.waitFor(() => expect(goto).toHaveBeenCalled());
  expect(document.body.textContent).toContain('Loading workout');
  expect(document.body.textContent).not.toContain('Saved');
  expect(document.body.textContent).not.toContain('Workout ready');
  expect(document.querySelector('a')).toBeNull();
});
