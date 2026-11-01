import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import DraftItemResolver from './DraftItemResolver.svelte';
import type { DraftSessionItem } from './types';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
});
const exercise = {
  id: 'exercise-1',
  name: 'Bench press',
  revision: '4',
  muscleGroupCodes: [],
  archived: false,
};
const variant = {
  id: 'variant-1',
  name: 'Heavy',
  revision: '9007199254740993',
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
function button(text: string) {
  return [...document.querySelectorAll('button')].find((el) => el.textContent?.includes(text))!;
}
function input(label: string, value: string) {
  const el = document.querySelector<HTMLInputElement>(`input[aria-label="${label}"]`)!;
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true }));
}

test('personal autocomplete resolves a reviewed variant identity without applying suggested targets', async () => {
  const resolved: DraftSessionItem[] = [];
  const http = vi
    .fn()
    .mockResolvedValue(
      new Response(JSON.stringify({ exercises: [{ exercise, variants: [variant] }] }), {
        status: 200,
      })
    );
  vi.stubGlobal('fetch', http);
  component = mount(DraftItemResolver, {
    target: document.body,
    props: {
      suggestion: {
        name: 'Bench press',
        configuration: { prescribedSets: 3, initialWeight: '40.00' },
      },
      onresolve: (item: DraftSessionItem) => {
        resolved.push(item);
        return true;
      },
    },
  });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Bench press'));
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  button('Bench press').click();
  await tick();
  expect(document.body.textContent).toContain('Suggested');
  expect(document.body.textContent).toContain('40.00');
  expect(document.body.textContent).toContain('80.00');
  expect(document.body.textContent).toContain('77.50');
  button('Reuse Heavy unchanged').click();
  await tick();
  expect(resolved).toHaveLength(1);
  expect(resolved[0].intent).toEqual({
    kind: 'reuseVariant',
    exerciseId: 'exercise-1',
    variantId: 'variant-1',
    expectedVariantRevision: '9007199254740993',
  });
  expect(http.mock.calls[0][0]).toContain('/api/training/exercises?q=Bench%20press');
});

test('creating a named variant preserves the reviewed suggestion even when another variant has identical settings', async () => {
  const resolved: DraftSessionItem[] = [];
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(JSON.stringify({ exercises: [{ exercise, variants: [variant] }] }), {
          status: 200,
        })
    )
  );
  component = mount(DraftItemResolver, {
    target: document.body,
    props: {
      suggestion: {
        name: 'Bench press',
        configuration: {
          prescribedSets: 2,
          initialWeight: '80.00',
          repThreshold: 8,
          autoIncrement: '5.00',
          restSeconds: 90,
        },
      },
      onresolve: (item: DraftSessionItem) => {
        resolved.push(item);
        return true;
      },
    },
  });
  await vi.waitFor(() => expect(button('Bench press')).toBeDefined());
  button('Bench press').click();
  await tick();
  expect(document.body.textContent).toContain('5.00 kg increase');
  expect(document.body.textContent).toContain('90s rest');
  button('Create named variant').click();
  await tick();
  input('Variant name', ' Heavy again ');
  await tick();
  button('Add to session').click();
  await tick();
  expect(resolved[0].intent).toEqual({
    kind: 'createVariant',
    exerciseId: 'exercise-1',
    expectedExerciseRevision: '4',
    name: 'Heavy again',
    configuration: {
      prescribedSets: 2,
      initialWeight: '80.00',
      repThreshold: 8,
      autoIncrement: '5',
      restSeconds: 90,
    },
  });
});
