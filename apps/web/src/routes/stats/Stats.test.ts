import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount } from 'svelte';
vi.hoisted(() => {
  Object.defineProperty(globalThis, 'ResizeObserver', {
    configurable: true,
    value: class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  });
  Object.defineProperty(HTMLElement.prototype, 'clientWidth', {
    configurable: true,
    get: () => 800,
  });
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get: () => 300,
  });
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: () => ({ matches: false, addEventListener() {}, removeEventListener() {} }),
  });
});
import Stats from './+page.svelte';
import { user, type Account } from '$lib/stores/appState';

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
let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  user.set(null);
  vi.unstubAllGlobals();
});
const entry = {
  id: 'import-1',
  occurredAt: null,
  sourceTimestamp: '2001-01-01T12:00:00.000000',
  sourceTimezone: null,
  setCount: 3,
  averageReps: '7',
  averageWeight: '42.25',
  volume: '887.25',
  representation: 'legacy_aggregate',
  volumeEstimated: true,
};
test('exercise-only imported history remains reachable with no saved sessions and no invented date', async () => {
  user.set(account);
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string) => {
      if (input.endsWith('/api/training/sessions'))
        return new Response(JSON.stringify({ sessions: [] }));
      if (input.endsWith('/api/training/stats'))
        return new Response(
          JSON.stringify({
            tracks: [
              {
                exerciseId: 'exercise',
                exerciseName: 'Current press',
                variantId: null,
                variantName: null,
                entries: [entry],
              },
            ],
          })
        );
      throw new Error(`Unexpected request ${input}`);
    })
  );
  component = mount(Stats, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Current press'));
  expect(document.body.textContent).toContain('Imported history (unassigned variant)');
  expect(document.body.textContent).toContain('Occurrence time unavailable');
  expect(document.body.textContent).toContain('887.25');
  expect(document.body.textContent).toContain('estimated');
});

function button(text: string) {
  const result = [...document.querySelectorAll('button')].find(
    (b) => b.textContent?.trim() === text
  );
  if (!result) throw new Error(`Missing button ${text}`);
  return result;
}
function statsFetch(sessions: unknown[], tracks: unknown[]) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string) => {
      if (input.endsWith('/api/training/sessions'))
        return new Response(JSON.stringify({ sessions }));
      if (input.endsWith('/api/training/stats')) return new Response(JSON.stringify({ tracks }));
      throw new Error(`Unexpected request ${input}`);
    })
  );
}
const actual = {
  ...entry,
  id: 'actual',
  occurredAt: '2026-11-01T12:00:00Z',
  sourceTimestamp: null,
  representation: 'set_derived',
  volumeEstimated: false,
};
test('session navigation reuses the same UUID track with current labels and preserves range and overlay controls', async () => {
  user.set(account);
  statsFetch(
    [
      { id: 'one', name: 'Upper one', placements: [{ variantId: 'heavy' }] },
      { id: 'two', name: 'Upper two', placements: [{ variantId: 'heavy' }] },
    ],
    [
      {
        exerciseId: 'exercise',
        exerciseName: 'Current press',
        variantId: 'heavy',
        variantName: 'Heavy',
        entries: [
          { ...actual, id: 'old', occurredAt: '2026-07-01T12:00:00Z', averageWeight: '30' },
          actual,
        ],
      },
    ]
  );
  component = mount(Stats, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Current press — Heavy'));
  expect(document.querySelectorAll('[data-entry-id]').length).toBe(2);
  await vi.waitFor(() => expect(document.querySelectorAll('svg .lc-point').length).toBe(4));
  button('1 month').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('(1 month)'));
  await vi.waitFor(() => expect(document.querySelectorAll('svg .lc-point').length).toBe(2));
  button('This month vs Last').click();
  await vi.waitFor(() =>
    expect(button('This month vs Last').classList.contains('active')).toBe(true)
  );
  await vi.waitFor(() => expect(document.querySelectorAll('svg .lc-point').length).toBe(3));
  button('Total volume lifted').click();
  await vi.waitFor(() =>
    expect(button('Total volume lifted').classList.contains('active')).toBe(true)
  );
  button('Upper two').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Current press — Heavy'));
  await vi.waitFor(() => expect(document.querySelectorAll('[data-entry-id]').length).toBe(2));
  expect(button('Total volume lifted').classList.contains('active')).toBe(true);
  button('6 months').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('(6 months)'));
  expect(document.body.textContent).not.toContain('This month vs Last');
  await vi.waitFor(() => expect(document.querySelectorAll('svg .lc-point').length).toBe(6));
}, 20000);

test('equal-time entries retain IDs and missing metrics do not invent 1RM or volume while actual zero stays visible', async () => {
  user.set(account);
  statsFetch(
    [],
    [
      {
        exerciseId: 'exercise',
        exerciseName: 'Press',
        variantId: null,
        variantName: null,
        entries: [
          {
            ...actual,
            id: 'zero',
            averageReps: '0',
            averageWeight: '0',
            volume: '0.00',
            setCount: 1,
          },
          {
            ...actual,
            id: 'skip',
            averageReps: null,
            averageWeight: null,
            volume: '0.00',
            setCount: 0,
          },
          {
            ...actual,
            id: 'fractional',
            averageReps: '0.5',
            averageWeight: '40.5',
            volume: '40.50',
            setCount: 2,
          },
          {
            ...actual,
            id: 'invalid',
            representation: 'legacy_aggregate',
            volumeEstimated: true,
            averageReps: '-1',
            averageWeight: '42.25',
            volume: null,
          },
        ],
      },
    ]
  );
  component = mount(Stats, { target: document.body });
  await vi.waitFor(() => expect(document.querySelectorAll('[data-entry-id]').length).toBe(4));
  const text = (id: string) => document.querySelector(`[data-entry-id="${id}"]`)?.textContent;
  expect(text('zero')).toContain('Weight per set: 0 kg');
  expect(text('zero')).toContain('1RM: unavailable');
  expect(text('skip')).toContain('Weight per set: unavailable');
  expect(text('fractional')).toContain('1RM: 41.2');
  expect(text('fractional')).toContain('40.50 kg-reps');
  expect(text('invalid')).toContain('Volume: unavailable');
  expect(text('invalid')).toContain('1RM: unavailable');
  await vi.waitFor(() => expect(document.querySelectorAll('svg .lc-point').length).toBe(4));
  button('Weight per set').click();
  await vi.waitFor(() => expect(document.querySelectorAll('svg .lc-point').length).toBe(1));
  button('1RM').click();
  button('Total volume lifted').click();
  await vi.waitFor(() => expect(document.querySelectorAll('svg .lc-point').length).toBe(3));
  expect(document.querySelector('svg')?.innerHTML).not.toContain('NaN');
});

test('late results are discarded when the selected account changes', async () => {
  user.set(account);
  let release!: (response: Response) => void;
  vi.stubGlobal(
    'fetch',
    vi.fn((input: string) => {
      if (input.endsWith('/api/training/sessions'))
        return Promise.resolve(new Response('{"sessions":[]}'));
      return new Promise<Response>((resolve) => {
        release = resolve;
      });
    })
  );
  component = mount(Stats, { target: document.body });
  await vi.waitFor(() => expect(release).toBeTypeOf('function'));
  user.set({
    ...account,
    id: 'other',
    email: 'other@example.com',
    capabilities: { ...account.capabilities, applicationModel: 'legacy' },
  });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Statistics are unavailable'));
  release(
    new Response(
      JSON.stringify({
        tracks: [
          {
            exerciseId: 'private',
            exerciseName: 'Private lift',
            variantId: null,
            variantName: null,
            entries: [entry],
          },
        ],
      })
    )
  );
  await new Promise((resolve) => setTimeout(resolve, 20));
  expect(document.body.textContent).not.toContain('Private lift');
});

test('an unavailable account never falls back to legacy Stats reads', async () => {
  const fetch = vi.fn(async (input: string) => {
    if (input.endsWith('/api/me')) return new Response('', { status: 401 });
    throw new Error(`Unexpected request ${input}`);
  });
  vi.stubGlobal('fetch', fetch);
  component = mount(Stats, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Statistics are unavailable'));
  expect(fetch.mock.calls.map((c) => c[0])).toEqual(['https://backend.example/api/me']);
});
