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
const personal = {
  exercise: {
    id: 'private-bench',
    name: 'Bench press',
    revision: '2',
    muscleGroupCodes: [],
    archived: false,
  },
  variants: [
    {
      id: 'heavy',
      name: 'Heavy',
      revision: '3',
      prescribedSets: 2,
      repThreshold: 8,
      autoIncrement: '2.50',
      restSeconds: 60,
      progressionMode: 'average',
      archived: false,
      targetSets: [],
    },
  ],
};
const names = ['Bench press', 'Bench dips', 'Barbell bench press', 'Squat'];
const field = () => document.querySelector<HTMLInputElement>('[aria-label="Exercise name"]')!;
function enter(value: string) {
  const input = field();
  input.focus();
  input.value = value;
  input.dispatchEvent(new Event('input', { bubbles: true }));
  input.setSelectionRange(value.length, value.length);
  input.dispatchEvent(new Event('select'));
}
function key(key: string) {
  field().dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }));
}
const options = () => [...document.querySelectorAll<HTMLElement>('[role="option"]')];
const button = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>('button')].find(
    (el) => el.getAttribute('aria-label') === name || el.textContent?.includes(name)
  )!;

test('combines a deduplicated private-library dropdown with a muted prefix completion', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async (url: string) =>
        new Response(
          JSON.stringify(
            url.endsWith('/catalog') ? { exerciseNames: names } : { exercises: [personal] }
          )
        )
    )
  );
  component = mount(DraftItemResolver, { target: document.body, props: { onresolve: () => true } });
  enter('Be');
  await vi.waitFor(() => expect(document.querySelector('[role="combobox"]')).not.toBeNull());
  await vi.waitFor(() =>
    expect(document.querySelector('[data-completion]')?.textContent).toBe('nch press')
  );
  expect(options().filter((el) => el.dataset.name?.toLowerCase() === 'bench press')).toHaveLength(
    1
  );
  expect(options().every((option) => !/\s/.test(option.id))).toBe(true);
  expect(
    options()
      .map((el) => el.dataset.name)
      .filter(Boolean)
  ).toEqual(['Bench press', 'Bench dips', 'Barbell bench press']);
  key('ArrowDown');
  await tick();
  expect(field().getAttribute('aria-activedescendant')).toBe(options()[0].id);
  key('Enter');
  await tick();
  expect(field().value).toBe('Bench press');
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(button('Reuse Heavy unchanged')).toBeDefined();
});

test('Tab accepts a suggested name into a personal Standard draft without writing or inventing an ID', async () => {
  const resolved: DraftSessionItem[] = [];
  const http = vi.fn(
    async (url: string, _options?: RequestInit) =>
      new Response(
        JSON.stringify(url.endsWith('/catalog') ? { exerciseNames: names } : { exercises: [] })
      )
  );
  vi.stubGlobal('fetch', http);
  component = mount(DraftItemResolver, {
    target: document.body,
    props: {
      onresolve: (item) => {
        resolved.push(item);
        return true;
      },
    },
  });
  enter('Sq');
  await vi.waitFor(() =>
    expect(document.querySelector('[data-completion]')?.textContent).toBe('uat')
  );
  key('Tab');
  await tick();
  expect(field().value).toBe('Squat');
  expect(document.body.textContent).toContain('New exercise · Standard');
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  const setValue = (selector: string, value: string) => {
    const input = document.querySelector<HTMLInputElement>(selector)!;
    input.value = value;
    input.dispatchEvent(new Event('input', { bubbles: true }));
  };
  setValue('input[type="number"]', '3');
  setValue('input[inputmode="decimal"]', '50');
  await tick();
  button('Add to session').click();
  await tick();
  expect(resolved[0].intent).toEqual({
    kind: 'createExercise',
    name: 'Squat',
    muscleGroupCodes: [],
    configuration: {
      prescribedSets: 3,
      initialWeight: '50',
      repThreshold: 12,
      autoIncrement: '2.5',
      restSeconds: 0,
    },
  });
  expect(http.mock.calls.every(([, options]) => !options || options.method === 'GET')).toBe(true);
  expect(http.mock.calls.filter(([url]) => url.endsWith('/catalog'))).toHaveLength(1);
  expect(field().value).toBe('');
  expect(document.querySelector('[data-completion]')).toBeNull();
  expect(document.querySelector('[role="listbox"]')).toBeNull();
});

test('keeps completion at the caret and respects composition, dismissal and modified navigation', async () => {
  const http = vi.fn(
    async (url: string) =>
      new Response(
        JSON.stringify(url.endsWith('/catalog') ? { exerciseNames: names } : { exercises: [] })
      )
  );
  vi.stubGlobal('fetch', http);
  component = mount(DraftItemResolver, { target: document.body, props: { onresolve: () => true } });
  enter('Sq');
  await vi.waitFor(() => expect(document.querySelector('[data-completion]')).not.toBeNull());
  field().setSelectionRange(0, 0);
  field().dispatchEvent(new Event('select'));
  await tick();
  expect(document.querySelector('[data-completion]')).toBeNull();
  key('ArrowRight');
  await tick();
  expect(field().value).toBe('Sq');
  field().setSelectionRange(2, 2);
  field().dispatchEvent(new Event('select'));
  await tick();
  const modified = new KeyboardEvent('keydown', {
    key: 'ArrowRight',
    ctrlKey: true,
    bubbles: true,
    cancelable: true,
  });
  field().dispatchEvent(modified);
  expect(modified.defaultPrevented).toBe(false);
  expect(field().value).toBe('Sq');
  const requests = http.mock.calls.length;
  field().dispatchEvent(new CompositionEvent('compositionstart', { bubbles: true }));
  field().dispatchEvent(new InputEvent('input', { bubbles: true, isComposing: true }));
  await tick();
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(document.querySelector('[data-completion]')).toBeNull();
  expect(http).toHaveBeenCalledTimes(requests);
  field().dispatchEvent(new CompositionEvent('compositionend', { bubbles: true }));
  await vi.waitFor(() => expect(document.querySelector('[data-completion]')).not.toBeNull());
  key('Escape');
  await tick();
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(field().value).toBe('Sq');
  field().dispatchEvent(new MouseEvent('click', { bubbles: true }));
  await tick();
  key('ArrowRight');
  await tick();
  expect(field().value).toBe('Squat');
  expect(document.body.textContent).toContain('New exercise · Standard');
});

test('waits for personal results before accepting names and ignores a late response after selection', async () => {
  let finishSearch!: (response: Response) => void;
  const late = new Promise<Response>((resolve) => {
    finishSearch = resolve;
  });
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => {
      if (url.endsWith('/catalog')) return new Response(JSON.stringify({ exerciseNames: names }));
      if (url.endsWith('q=Be')) return late;
      return new Response(JSON.stringify({ exercises: [] }));
    })
  );
  component = mount(DraftItemResolver, { target: document.body, props: { onresolve: () => true } });
  enter('Be');
  await vi.waitFor(() => expect(options().length).toBeGreaterThan(0));
  expect(options().every((el) => (el as HTMLButtonElement).disabled)).toBe(true);
  expect(document.querySelector('[data-completion]')).toBeNull();
  (options()[0] as HTMLButtonElement).click();
  await tick();
  expect(document.body.textContent).not.toContain('New exercise · Standard');
  enter('Sq');
  await vi.waitFor(() => expect(document.querySelector('[data-completion]')).not.toBeNull());
  key('ArrowDown');
  await tick();
  key('Enter');
  await tick();
  expect(field().value).toBe('Squat');
  finishSearch(new Response(JSON.stringify({ exercises: [personal] })));
  await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
  expect(field().value).toBe('Squat');
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(document.body.textContent).not.toContain('Reuse Heavy unchanged');
});

test('empty-input browsing checks the personal library before a matching built-in can be selected', async () => {
  let finishSearch!: (response: Response) => void;
  const pending = new Promise<Response>((resolve) => {
    finishSearch = resolve;
  });
  const http = vi.fn(async (url: string) =>
    url.endsWith('/catalog') ? new Response(JSON.stringify({ exerciseNames: names })) : pending
  );
  vi.stubGlobal('fetch', http);
  component = mount(DraftItemResolver, { target: document.body, props: { onresolve: () => true } });
  field().focus();
  await vi.waitFor(() => expect(options().length).toBeGreaterThan(0));
  expect(http.mock.calls.some(([url]) => url.endsWith('/exercises?q='))).toBe(true);
  expect(options().every((el) => (el as HTMLButtonElement).disabled)).toBe(true);
  (options()[0] as HTMLButtonElement).click();
  await tick();
  expect(document.body.textContent).not.toContain('New exercise · Standard');
  finishSearch(new Response(JSON.stringify({ exercises: [personal] })));
  await vi.waitFor(() =>
    expect(options().filter((el) => el.dataset.name === 'Bench press')).toHaveLength(1)
  );
  await vi.waitFor(() => expect((options()[0] as HTMLButtonElement).disabled).toBe(false));
  (options()[0] as HTMLButtonElement).click();
  await tick();
  expect(button('Reuse Heavy unchanged')).toBeDefined();
  expect(document.body.textContent).not.toContain('New exercise · Standard');
});
