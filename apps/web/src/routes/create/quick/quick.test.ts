import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import Quick from './+page.svelte';
import Manual from '../manual/+page.svelte';
import { goto } from '../../../test/navigation';
import { page } from '../../../test/page-state';
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
function button(text: string) {
  return [...document.querySelectorAll('button')].find(
    (el) => el.getAttribute('aria-label') === text || el.textContent?.includes(text)
  )!;
}
async function generate() {
  component = mount(Quick, { target: document.body });
  await tick();
  const notes = document.querySelector('textarea')!;
  notes.value = 'Bench press 3 sets at 40 kg';
  notes.dispatchEvent(new Event('input', { bubbles: true }));
  await tick();
  button('Continue').click();
}
test('Quick fill distinguishes authoritative quota denial from a failed generation', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(
          JSON.stringify({ code: 'ai-quota-exceeded', message: 'AI call count exceeded' }),
          { status: 403 }
        )
    )
  );
  await generate();
  await vi.waitFor(() => expect(document.body.textContent).toContain('AI usage limit reached'));
  expect(sessionStorage.getItem('workoutPayload')).toBeNull();
  expect(goto).not.toHaveBeenCalled();
});
test('Quick fill hands suggestions to explicit review and cancel leaves no training writes', async () => {
  const draft = JSON.stringify([{ name: 'Bench press', sets: 3, weight: 40 }]);
  const http = vi.fn(async (url: string) =>
    url.endsWith('/ai/quick')
      ? new Response(draft)
      : new Response(JSON.stringify({ exercises: [] }))
  );
  vi.stubGlobal('fetch', http);
  await generate();
  await vi.waitFor(() => expect(button('Review generated session')).toBeDefined());
  button('Review generated session').click();
  await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/create/manual?quickload=1'));
  await unmount(component!);
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
  page.url = new URL('https://app.example/create/manual?quickload=1');
  component = mount(Manual, { target: document.body });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Review suggestion 1 of 1'));
  expect(document.querySelectorAll('li.sortable-item')).toHaveLength(0);
  button('Cancel generated session').click();
  await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/create'));
  expect(http.mock.calls.some(([url]) => url.includes('/sessions/create'))).toBe(false);
});
