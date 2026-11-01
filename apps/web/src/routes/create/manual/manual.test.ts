import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import Manual from './+page.svelte';
import { user } from '$lib/stores/appState';
import { goto } from '../../../test/navigation';
import type { CreateSessionCommand } from '$lib/training/types';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  user.set(null);
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});
function input(selector: string, value: string) {
  const el = document.querySelector<HTMLInputElement>(selector)!;
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true }));
}
function button(text: string) {
  return [...document.querySelectorAll('button')].find((el) => el.textContent?.includes(text))!;
}

test('finishing includes the unadded exercise once and retries the identical atomic command after a lost response', async () => {
  user.set({
    id: 'development@resilnce.invalid',
    email: 'development@resilnce.invalid',
    name: 'Development account',
    developmentMode: true,
    capabilities: {
      applicationModel: 'exercise-centric',
      trainingWrites: true,
      writeMaintenance: false,
    },
  });
  let posts = 0;
  const fetchMock = vi.fn(async (url: string, _options?: RequestInit) => {
    if (url.includes('/exercises'))
      return new Response(JSON.stringify({ exercises: [] }), { status: 200 });
    if (++posts === 1) throw new TypeError('response lost');
    return new Response(
      JSON.stringify({
        operation: { operationId: 'accepted', acceptedRevision: '1' },
        session: { id: '20000000-0000-4000-8000-000000000001', revision: '1' },
      }),
      { status: 200 }
    );
  });
  vi.stubGlobal('fetch', fetchMock);
  component = mount(Manual, { target: document.body });
  await tick();
  input('input[placeholder="Untitled session"]', ' Upper body ');
  input('input[type="text"]', ' Bench press ');
  await vi.waitFor(() => expect(button('Create new exercise')).toBeDefined());
  button('Create new exercise').click();
  await tick();
  input('input[type="number"]', '2');
  input('input[inputmode="decimal"]', '40.50');
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Not saved'));
  expect(document.body.textContent).toContain('Bench press');
  expect(goto).not.toHaveBeenCalled();
  const [url, options] = fetchMock.mock.calls.find((call) => call[0].includes('/sessions/create'))!;
  expect(url).toBe('https://backend.example/api/training/sessions/create');
  const command = JSON.parse(options!.body as string);
  expect(command.operationId).toMatch(/^[0-9a-f-]{36}$/);
  expect(command.name).toBe('Upper body');
  expect(command.items).toEqual([
    {
      kind: 'createExercise',
      name: 'Bench press',
      muscleGroupCodes: [],
      configuration: {
        prescribedSets: 2,
        initialWeight: '40.50',
        repThreshold: 12,
        autoIncrement: '2.5',
        restSeconds: 0,
      },
    },
  ]);
  button('Retry').click();
  await vi.waitFor(() => expect(goto).toHaveBeenCalled());
  const postCalls = fetchMock.mock.calls.filter((call) => call[0].includes('/sessions/create'));
  expect(postCalls[1][1]!.body).toBe(options!.body);
  expect(postCalls[1][1]!.credentials).toBe('include');
});

test('manual creation keeps variant IDs distinct and preserves an ordered reviewed draft after a conflict', async () => {
  user.set({
    id: 'development@resilnce.invalid',
    email: 'development@resilnce.invalid',
    name: 'Development',
    developmentMode: true,
    capabilities: {
      applicationModel: 'exercise-centric',
      trainingWrites: true,
      writeMaintenance: false,
    },
  });
  const exercise = {
    id: 'e1',
    name: 'Bench press',
    revision: '1',
    muscleGroupCodes: [],
    archived: false,
  };
  const standard = {
    id: 'v1',
    name: 'Standard',
    revision: '3',
    prescribedSets: 2,
    repThreshold: 12,
    autoIncrement: '2.50',
    restSeconds: 0,
    progressionMode: 'average',
    archived: false,
    targetSets: [
      { id: 't1', position: 1, targetReps: 12, targetWeight: '40.50' },
      { id: 't2', position: 2, targetReps: 12, targetWeight: '37.50' },
    ],
  };
  const heavy = { ...standard, id: 'v2', name: 'Heavy', revision: '7' };
  const commands: CreateSessionCommand[] = [];
  const http = vi.fn(async (url: string, options?: RequestInit) => {
    if (url.includes('/exercises'))
      return new Response(
        JSON.stringify({ exercises: [{ exercise, variants: [standard, heavy] }] }),
        { status: 200 }
      );
    commands.push(JSON.parse(options!.body as string));
    return new Response(
      JSON.stringify({
        code: 'stale-revision',
        message: 'The variant changed. Review its current configuration before saving',
      }),
      { status: 409 }
    );
  });
  vi.stubGlobal('fetch', http);
  component = mount(Manual, { target: document.body });
  await tick();
  input('input[placeholder="Untitled session"]', 'Upper');
  input('input[type="text"]', 'Bench');
  await tick();
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  expect(http.mock.calls[0][0]).toBe('https://backend.example/api/training/exercises?q=Bench');
  button('Bench press').click();
  await tick();
  button('Reuse Standard unchanged').click();
  await tick();
  input('input[type="text"]', 'Bench');
  await tick();
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  button('Bench press').click();
  await tick();
  button('Reuse Standard unchanged').click();
  await tick();
  expect(document.body.textContent).toContain('already in this session');
  button('Reuse Heavy unchanged').click();
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Not saved'));
  expect(commands[0].items).toEqual([
    { kind: 'reuseVariant', exerciseId: 'e1', variantId: 'v1', expectedVariantRevision: '3' },
    { kind: 'reuseVariant', exerciseId: 'e1', variantId: 'v2', expectedVariantRevision: '7' },
  ]);
  expect([...document.querySelectorAll('li.sortable-item h3')].map((el) => el.textContent)).toEqual(
    ['Bench press', 'Bench press']
  );
  expect(document.body.textContent).toContain('37.50');
  button('Retry').click();
  await vi.waitFor(() => expect(commands).toHaveLength(2));
  expect(commands[1]).toEqual(commands[0]);
});

test('a stale reuse stays editable and requires an explicit review of current targets before a new command', async () => {
  user.set({
    id: 'development@resilnce.invalid',
    email: 'development@resilnce.invalid',
    name: 'Development',
    developmentMode: true,
    capabilities: {
      applicationModel: 'exercise-centric',
      trainingWrites: true,
      writeMaintenance: false,
    },
  });
  const exercise = {
    id: 'e1',
    name: 'Bench press',
    revision: '1',
    muscleGroupCodes: [],
    archived: false,
  };
  const variant = {
    id: 'v1',
    name: 'Standard',
    revision: '1',
    prescribedSets: 1,
    repThreshold: 12,
    autoIncrement: '2.50',
    restSeconds: 0,
    progressionMode: 'average',
    archived: false,
    targetSets: [{ id: 't1', position: 1, targetReps: 12, targetWeight: '40.50' }],
  };
  const current = {
    ...variant,
    revision: '2',
    targetSets: [{ id: 't1', position: 1, targetReps: 10, targetWeight: '45.00' }],
  };
  const commands: CreateSessionCommand[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, options?: RequestInit) => {
      if (url.includes('/exercises/'))
        return new Response(JSON.stringify({ exercise, variants: [current] }), { status: 200 });
      if (url.includes('/exercises?'))
        return new Response(JSON.stringify({ exercises: [{ exercise, variants: [variant] }] }), {
          status: 200,
        });
      commands.push(JSON.parse(options!.body as string));
      return commands.length === 1
        ? new Response(
            JSON.stringify({
              code: 'stale-revision',
              message: 'The variant changed. Review its current configuration before saving',
            }),
            { status: 409 }
          )
        : new Response(JSON.stringify({ session: { id: 'saved' } }), { status: 200 });
    })
  );
  component = mount(Manual, { target: document.body });
  await tick();
  input('input[placeholder="Untitled session"]', 'Upper');
  input('input[type="text"]', 'Bench');
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  button('Bench press').click();
  await tick();
  button('Reuse Standard unchanged').click();
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(button('Edit draft')).toBeDefined());
  button('Edit draft').click();
  await tick();
  document
    .querySelector<HTMLButtonElement>('button[aria-label="Options for Bench press"]')!
    .click();
  await tick();
  document.querySelector<HTMLButtonElement>('button[aria-label="Edit Bench press"]')!.click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('45.00'));
  expect(commands).toHaveLength(1);
  button('Reuse Standard unchanged').click();
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(commands).toHaveLength(2));
  expect(commands[1].operationId).not.toBe(commands[0].operationId);
  expect(commands[1].items).toEqual([
    { kind: 'reuseVariant', exerciseId: 'e1', variantId: 'v1', expectedVariantRevision: '2' },
  ]);
});
