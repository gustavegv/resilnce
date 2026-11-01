import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import Manual from '../manual/+page.svelte';
import { page } from '../../../test/page-state';
import { goto } from '../../../test/navigation';
import { user } from '$lib/stores/appState';

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
test('Program Edit and add opens the same explicit suggestion review before saving', async () => {
  vi.stubGlobal('matchMedia', () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
  const { default: Program } = await import('./+page.svelte');
  const http = vi.fn(async (_url: string) => new Response(JSON.stringify({ exercises: [] })));
  vi.stubGlobal('fetch', http);
  component = mount(Program, { target: document.body });
  await tick();
  const card = [...document.querySelectorAll('article')].find((el) =>
    el.textContent?.includes('Quick Compound Upper Lower')
  )!;
  card.querySelector<HTMLButtonElement>('button.program-card__trigger')!.click();
  await tick();
  const edit = [...card.querySelectorAll('button')].find((el) =>
    el.textContent?.includes('Edit and add')
  )!;
  edit.click();
  await vi.waitFor(() =>
    expect(goto).toHaveBeenCalledWith('/create/manual?quickload=1&title=Compound+Upper+A')
  );
  expect(http).not.toHaveBeenCalled();
  await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
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
  page.url = new URL('https://app.example/create/manual?quickload=1&title=Compound+Upper+A');
  component = mount(Manual, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Review suggestion 1 of 5'));
  expect(document.querySelector<HTMLInputElement>('[aria-label="Exercise name"]')!.value).toBe(
    'Bench Press'
  );
  expect(document.querySelectorAll('li.sortable-item')).toHaveLength(0);
  [...document.querySelectorAll('button')]
    .find((el) => el.textContent?.includes('Finish and save'))!
    .click();
  await tick();
  expect(http.mock.calls.some(([url]) => url.includes('/sessions/create'))).toBe(false);
  expect(document.body.textContent).toContain('Review every suggestion before saving');
}, 20000);
