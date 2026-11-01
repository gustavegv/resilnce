import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import Manual from './+page.svelte';
import { user } from '$lib/stores/appState';
import { page } from '../../../test/page-state';
import { goto } from '../../../test/navigation';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  sessionStorage.clear();
  page.url = new URL('https://app.example/create/manual');
  user.set(null);
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});
function button(text: string) {
  return [...document.querySelectorAll('button')].find((el) => el.textContent?.trim() === text)!;
}
function input(label: string, value: string) {
  const el = document.querySelector<HTMLInputElement>(`[aria-label="${label}"]`)!;
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true }));
}
function loadDraft(draft: string) {
  user.set({
    id: 'dev',
    email: 'dev',
    name: 'Development',
    developmentMode: true,
    capabilities: {
      applicationModel: 'exercise-centric',
      trainingWrites: true,
      writeMaintenance: false,
    },
  });
  page.url = new URL('https://app.example/create/manual?quickload=1&title=Upper');
  sessionStorage.setItem('workoutPayload', draft);
  component = mount(Manual, { target: document.body });
}
const suggestion = { name: 'Bench press', sets: 3, weight: 40, repThreshold: 12, autoIncrease: 5 };
const exercise = {
  id: 'e1',
  name: 'Bench press',
  revision: '4',
  muscleGroupCodes: [],
  archived: false,
};
const variant = {
  id: 'v1',
  name: 'Heavy',
  revision: '7',
  prescribedSets: 2,
  repThreshold: 8,
  autoIncrement: '2.50',
  restSeconds: 60,
  progressionMode: 'average',
  archived: false,
  targetSets: [
    { id: 't1', position: 1, targetReps: 8, targetWeight: '80.00' },
    { id: 't2', position: 2, targetReps: 7, targetWeight: '77.50' },
  ],
};

test('generated suggestions require explicit current-config review and retry one exact atomic save', async () => {
  let posts = 0;
  const http = vi.fn(async (url: string, options?: RequestInit) => {
    if (url.includes('/exercises'))
      return new Response(JSON.stringify({ exercises: [{ exercise, variants: [variant] }] }));
    if (++posts === 1) throw new TypeError('response lost');
    return new Response(JSON.stringify({ session: { id: 'saved' } }));
  });
  vi.stubGlobal('fetch', http);
  loadDraft(JSON.stringify([suggestion]));
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  button('Finish and save session').click();
  await tick();
  expect(posts).toBe(0);
  button('Bench press').click();
  await tick();
  expect(document.body.textContent).toContain('Suggested: 3 sets');
  expect(document.body.textContent).toContain('77.50');
  button('Reuse Heavy unchanged').click();
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Not saved'));
  const calls = () => http.mock.calls.filter(([url]) => url.includes('/sessions/create'));
  const body = calls()[0][1]!.body as string;
  expect(JSON.parse(body).items).toEqual([
    { kind: 'reuseVariant', exerciseId: 'e1', variantId: 'v1', expectedVariantRevision: '7' },
  ]);
  expect(
    document.querySelector<HTMLInputElement>('[aria-label="Session name"]')!.matches(':disabled')
  ).toBe(true);
  button('Retry').click();
  await vi.waitFor(() => expect(goto).toHaveBeenCalled());
  expect(calls()[1][1]!.body).toBe(body);
  expect(sessionStorage.getItem('workoutPayload')).toBeNull();
});

test.each([
  ['broken JSON', '[{'],
  ['empty draft', '[]'],
  ['invalid sets', JSON.stringify([{ ...suggestion, sets: 2.5 }])],
  ['invalid precision', JSON.stringify([{ ...suggestion, weight: 40.001 }])],
  ['invalid name', JSON.stringify([{ ...suggestion, name: '   ' }])],
  ['null threshold', JSON.stringify([{ ...suggestion, repThreshold: null }])],
  ['unknown muscle code', JSON.stringify([{ ...suggestion, muscleGroupCodes: ['pecs'] }])],
  ['unknown field', JSON.stringify([{ ...suggestion, weightPerSet: [10, 20] }])],
])(
  'invalid generated %s stays correctable and cancellation performs no domain writes',
  async (_case, raw) => {
    const http = vi.fn(
      async (_url: string) => new Response(JSON.stringify({ muscleGroups: [{ code: 'chest' }] }))
    );
    vi.stubGlobal('fetch', http);
    loadDraft(raw);
    await vi.waitFor(() =>
      expect(document.querySelector('[aria-label="Suggested draft JSON"]')).not.toBeNull()
    );
    expect(
      (document.querySelector('[aria-label="Suggested draft JSON"]') as HTMLTextAreaElement).value
    ).toBe(raw);
    button('Finish and save session').click();
    await tick();
    expect(http.mock.calls.some(([url]) => url.includes('/sessions/create'))).toBe(false);
    button('Cancel generated session').click();
    await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/create'));
    expect(sessionStorage.getItem('workoutPayload')).toBeNull();
  }
);

test('repeated suggested names are reviewed independently and duplicate placements stay editable', async () => {
  const http = vi.fn(
    async () =>
      new Response(
        JSON.stringify({
          exercises: [{ exercise, variants: [variant, { ...variant, id: 'v2', name: 'Light' }] }],
        })
      )
  );
  vi.stubGlobal('fetch', http);
  loadDraft(JSON.stringify([suggestion, { ...suggestion, weight: 25 }]));
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  button('Bench press').click();
  await tick();
  button('Reuse Heavy unchanged').click();
  await tick();
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  expect(document.body.textContent).toContain('Review suggestion 2 of 2');
  button('Bench press').click();
  await tick();
  button('Reuse Heavy unchanged').click();
  await tick();
  expect(document.body.textContent).toContain('already in this session');
  expect(document.body.textContent).toContain('Suggested: 3 sets · 25 kg');
  button('Reuse Light unchanged').click();
  await tick();
  expect(document.querySelectorAll('li.sortable-item')).toHaveLength(2);
  button('Cancel generated session').click();
  await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/create'));
});

test('a corrected generated draft can explicitly create a named variant with its suggested settings', async () => {
  const http = vi.fn(async (url: string, options?: RequestInit) => {
    if (url.endsWith('/catalog'))
      return new Response(JSON.stringify({ muscleGroups: [{ code: 'chest' }] }));
    if (url.includes('/exercises'))
      return new Response(JSON.stringify({ exercises: [{ exercise, variants: [variant] }] }));
    return new Response(JSON.stringify({ session: { id: 'saved' } }));
  });
  vi.stubGlobal('fetch', http);
  loadDraft('[{');
  await vi.waitFor(() => expect(button('Review corrected draft')).toBeDefined());
  input('Suggested draft JSON', JSON.stringify([{ ...suggestion, muscleGroupCodes: ['chest'] }]));
  await tick();
  button('Review corrected draft').click();
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  button('Bench press').click();
  await tick();
  button('Create named variant').click();
  await tick();
  input('Variant name', 'Suggested');
  await tick();
  button('Add to session').click();
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(goto).toHaveBeenCalled());
  const body = JSON.parse(
    http.mock.calls.find(([url]) => url.includes('/sessions/create'))![1]!.body as string
  );
  expect(body.items).toEqual([
    {
      kind: 'createVariant',
      exerciseId: 'e1',
      expectedExerciseRevision: '4',
      name: 'Suggested',
      configuration: {
        prescribedSets: 3,
        initialWeight: '40',
        repThreshold: 12,
        autoIncrement: '5',
        restSeconds: 0,
      },
    },
  ]);
});

test('a stale generated review preserves its choices and requires explicit re-review before a new save', async () => {
  const commands: { operationId: string; items: unknown[] }[] = [];
  const current = {
    ...variant,
    revision: '8',
    targetSets: [{ id: 't1', position: 1, targetReps: 6, targetWeight: '85.00' }],
  };
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, options?: RequestInit) => {
      if (url.includes('/exercises/'))
        return new Response(JSON.stringify({ exercise, variants: [current] }));
      if (url.includes('/exercises?'))
        return new Response(JSON.stringify({ exercises: [{ exercise, variants: [variant] }] }));
      commands.push(JSON.parse(options!.body as string));
      return commands.length === 1
        ? new Response(
            JSON.stringify({ code: 'stale-revision', message: 'Review the changed variant' }),
            { status: 409 }
          )
        : new Response(JSON.stringify({ session: { id: 'saved' } }));
    })
  );
  loadDraft(JSON.stringify([suggestion]));
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  button('Bench press').click();
  await tick();
  button('Reuse Heavy unchanged').click();
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(button('Edit draft')).toBeDefined());
  expect(document.querySelectorAll('li.sortable-item')).toHaveLength(1);
  button('Edit draft').click();
  await tick();
  document
    .querySelector<HTMLButtonElement>('button[aria-label="Options for Bench press"]')!
    .click();
  await tick();
  document.querySelector<HTMLButtonElement>('button[aria-label="Edit Bench press"]')!.click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('85.00'));
  expect(commands).toHaveLength(1);
  button('Reuse Heavy unchanged').click();
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(commands).toHaveLength(2));
  expect(commands[1].operationId).not.toBe(commands[0].operationId);
  expect(commands[1].items).toEqual([
    { kind: 'reuseVariant', exerciseId: 'e1', variantId: 'v1', expectedVariantRevision: '8' },
  ]);
});

test('repeated new-exercise suggestions cannot silently merge and valid groups follow only explicit Standard creation', async () => {
  const http = vi.fn(async (url: string, options?: RequestInit) => {
    if (url.endsWith('/catalog'))
      return new Response(JSON.stringify({ muscleGroups: [{ code: 'chest' }] }));
    if (url.includes('/exercises')) return new Response(JSON.stringify({ exercises: [] }));
    return new Response(JSON.stringify({ session: { id: 'saved' } }));
  });
  vi.stubGlobal('fetch', http);
  loadDraft(
    JSON.stringify([
      { ...suggestion, muscleGroupCodes: ['chest'] },
      { ...suggestion, name: ' BENCH PRESS ', weight: 25 },
    ])
  );
  await vi.waitFor(() =>
    expect(button('Create new exercise “Bench press” with Standard')).toBeDefined()
  );
  button('Create new exercise “Bench press” with Standard').click();
  await tick();
  expect(document.body.textContent).toContain('New exercise muscle groups: chest');
  button('Add to session').click();
  await tick();
  await vi.waitFor(() =>
    expect(button('Create new exercise “BENCH PRESS” with Standard')).toBeDefined()
  );
  button('Create new exercise “BENCH PRESS” with Standard').click();
  await tick();
  button('Add to session').click();
  await tick();
  expect(document.body.textContent).toContain('creation name is already');
  expect(document.querySelectorAll('li.sortable-item')).toHaveLength(1);
  input('Exercise name', 'Incline bench press');
  await tick();
  button('Create new exercise “Incline bench press” with Standard').click();
  await tick();
  button('Add to session').click();
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(goto).toHaveBeenCalled());
  const command = JSON.parse(
    http.mock.calls.find(([url]) => url.includes('/sessions/create'))![1]!.body as string
  );
  expect(
    command.items.map((item: { name: string; muscleGroupCodes: string[] }) => [
      item.name,
      item.muscleGroupCodes,
    ])
  ).toEqual([
    ['Bench press', ['chest']],
    ['Incline bench press', []],
  ]);
});
