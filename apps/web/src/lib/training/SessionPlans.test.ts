import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import SessionPlans from './SessionPlans.svelte';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
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
test('lists never-run plans honestly and reads current ordered variant targets by UUID', async () => {
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
    .mockResolvedValueOnce(new Response(JSON.stringify({ session })));
  vi.stubGlobal('fetch', fetchMock);
  component = mount(SessionPlans, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Never run'));
  expect(document.body.textContent).toContain('Older');
  [...document.querySelectorAll('button')]
    .find((el) => el.textContent?.includes('Upper body') && el.textContent?.includes('Never run'))!
    .click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('40.25'));
  expect(document.body.textContent).toContain('Standard');
  expect(document.body.textContent).toContain('50.00');
  expect(fetchMock.mock.calls[1][0]).toBe(
    'https://backend.example/api/training/sessions/' + session.id
  );
  [...document.querySelectorAll('button')]
    .find((el) => el.textContent?.trim() === 'Lower body')!
    .click();
  await tick();
  expect(document.body.textContent).not.toContain('Upper bodyNever run');
});
