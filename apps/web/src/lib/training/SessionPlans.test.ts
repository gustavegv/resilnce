import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import SessionPlans from './SessionPlans.svelte';
import { goto } from '$app/navigation';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
});

test('opens the original session edit dialog from the sliding three-dot menu', async () => {
  const session = {
    id: 'plan',
    name: 'Push',
    revision: '1',
    createdAt: '2026-11-01T00:00:00Z',
    lastPerformedAt: null,
    lastPerformedStatus: 'never',
    placements: [],
  };
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async (url: string) =>
        new Response(
          JSON.stringify(url.endsWith('/sessions') ? { sessions: [session] } : { session })
        )
    )
  );
  component = mount(SessionPlans, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Never run'));
  const options = document.querySelector<HTMLButtonElement>('[aria-label="Options for Push"]');
  expect(options).not.toBeNull();
  options!.click();
  document.querySelector<HTMLButtonElement>('[aria-label="Edit Push"]')!.click();
  await vi.waitFor(() => expect(document.querySelector('[role="alertdialog"]')).not.toBeNull());
  expect(document.querySelector('[aria-label="Session name"]')).not.toBeNull();
  expect(document.querySelector('[aria-label="Delete Push"]')).not.toBeNull();
  document.querySelector<HTMLButtonElement>('[aria-label="Close dialog"]')!.click();
  await tick();
  expect(document.querySelector('[role="alertdialog"]')).toBeNull();
  document.querySelector<HTMLButtonElement>('[aria-label="Delete Push"]')!.click();
  await vi.waitFor(() =>
    expect(document.querySelector('[role="alertdialog"]')?.textContent).toContain(
      'Are you absolutely sure?'
    )
  );
  expect(document.querySelector('[aria-label="Session name"]')).toBeNull();
});

test('keeps the inclusive fourteen-day boundary, unknown imported dates and categories distinct', async () => {
  vi.spyOn(Date, 'now').mockReturnValue(Date.parse('2026-11-01T12:00:00Z'));
  const fixture = (
    id: string,
    name: string,
    lastPerformedAt: string | null,
    lastPerformedStatus: string
  ) => ({
    id,
    name,
    revision: '1',
    createdAt: '2026-01-01T00:00:00Z',
    lastPerformedAt,
    lastPerformedStatus,
    placements: [],
  });
  const sessions = [
    fixture('a', 'Push boundary', '2026-10-18T12:00:00Z', 'dated'),
    fixture('b', 'Push older', '2026-10-18T11:59:59.999Z', 'dated'),
    fixture('c', 'Unknown plan', null, 'unknown'),
    fixture('d', 'Never plan', null, 'never'),
  ];
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ sessions }))));
  component = mount(SessionPlans, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Last run unavailable'));
  const content = document.body.textContent!;
  expect(content.indexOf('Recent')).toBeLessThan(content.indexOf('Push boundary'));
  expect(content.indexOf('Push boundary')).toBeLessThan(content.indexOf('Older'));
  expect(content.indexOf('Older')).toBeLessThan(content.indexOf('Push older'));
  expect(content).toContain('Never run');
  [...document.querySelectorAll('button')]
    .find((el) => el.textContent?.trim() === 'Upper body')!
    .click();
  await tick();
  expect(document.body.textContent).toContain('Push boundary');
  expect(document.body.textContent).not.toContain('Unknown plan');
});
test('starts a never-run plan directly using its UUID and current revision', async () => {
  const session = {
    id: '20000000-0000-4000-8000-000000000001',
    name: 'Upper body',
    revision: '9007199254740993',
    createdAt: '2026-11-01T10:00:00Z',
    lastPerformedAt: null,
    placements: [
      {
        id: '30000000-0000-4000-8000-000000000001',
        variantId: '40000000-0000-4000-8000-000000000001',
        position: 1,
        exercise: {
          id: '50000000-0000-4000-8000-000000000001',
          name: 'Bench press',
          revision: '1',
          muscleGroupCodes: [],
          archived: false,
        },
        variant: {
          id: '40000000-0000-4000-8000-000000000001',
          name: 'Standard',
          revision: '1',
          progressionMode: 'average',
          prescribedSets: 2,
          repThreshold: 12,
          autoIncrement: '2.50',
          restSeconds: 0,
          archived: false,
          targetSets: [
            {
              id: '60000000-0000-4000-8000-000000000001',
              position: 1,
              targetReps: 12,
              targetWeight: '40.25',
            },
            {
              id: '60000000-0000-4000-8000-000000000002',
              position: 2,
              targetReps: 10,
              targetWeight: '50.00',
            },
          ],
        },
      },
    ],
  };
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({ sessions: [session] })))
    .mockResolvedValueOnce(new Response(JSON.stringify({ session })))
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          workout: {
            id: 'workout',
            status: 'active',
            sourceSession: { id: session.id, name: session.name },
          },
        })
      )
    );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(SessionPlans, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Never run'));
  expect(document.body.textContent).toContain('Older');
  [...document.querySelectorAll('button')]
    .find((el) => el.textContent?.includes('Upper body') && el.textContent?.includes('Never run'))!
    .click();
  expect(goto).toHaveBeenCalledWith('/workout/start/' + session.id);
  expect(fetchMock).toHaveBeenCalledTimes(1);
  expect(document.body.textContent).not.toContain('Loading plan');
  expect(document.body.textContent).not.toContain('Start workout');
  expect(document.body.textContent).not.toContain('Continue workout');
});

test('an unresolved maintenance save disables workout start and session switching', async () => {
  const session = {
    id: '20000000-0000-4000-8000-000000000001',
    name: 'Push',
    revision: '1',
    createdAt: '2026-11-01T00:00:00Z',
    lastPerformedAt: null,
    lastPerformedStatus: 'never',
    placements: [
      {
        id: 'placement',
        variantId: 'variant',
        position: 1,
        exercise: {
          id: 'exercise',
          name: 'Bench',
          revision: '1',
          muscleGroupCodes: [],
          archived: false,
        },
        variant: {
          id: 'variant',
          name: 'Heavy',
          revision: '1',
          progressionMode: 'average',
          repThreshold: 12,
          autoIncrement: '2.50',
          restSeconds: 0,
          prescribedSets: 1,
          archived: false,
          targetSets: [],
        },
      },
    ],
  };
  const fetchMock = vi.fn(
    async (url: string) =>
      new Response(
        JSON.stringify(
          url.endsWith('/sessions')
            ? { sessions: [session] }
            : url.endsWith('/update')
              ? { code: 'internal-error', message: 'Connection lost' }
              : { session }
        ),
        { status: url.endsWith('/update') ? 500 : 200 }
      )
  );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(SessionPlans, { target: document.body });
  const button = (text: string) =>
    [...document.querySelectorAll('button')].find((el) => el.textContent?.trim() === text)!;
  await vi.waitFor(() => expect(document.body.textContent).toContain('Never run'));
  document.querySelector<HTMLButtonElement>('[aria-label="Edit Push"]')!.click();
  await vi.waitFor(() => expect(button('Confirm edits')).toBeDefined());
  button('Confirm edits').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Not saved'));
  expect(document.querySelector<HTMLButtonElement>('.start-side')!.disabled).toBe(true);
  expect(document.querySelector<HTMLButtonElement>('[aria-label="Close dialog"]')!.disabled).toBe(
    true
  );
  expect(button('Lower body').disabled).toBe(true);
  expect(fetchMock.mock.calls.filter((call) => call[0].includes('/workouts/start'))).toHaveLength(
    0
  );
});
