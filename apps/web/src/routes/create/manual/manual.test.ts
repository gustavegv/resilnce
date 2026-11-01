import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import Manual from './+page.svelte';
import { user } from '$lib/stores/appState';
import { goto } from '../../../test/navigation';

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
  const fetchMock = vi
    .fn()
    .mockRejectedValueOnce(new TypeError('response lost'))
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          operation: { operationId: 'accepted', acceptedRevision: '1' },
          session: { id: '20000000-0000-4000-8000-000000000001', revision: '1' },
        }),
        { status: 200 }
      )
    );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(Manual, { target: document.body });
  await tick();
  input('input[placeholder="Untitled session"]', ' Upper body ');
  input('input[type="text"]', ' Bench press ');
  input('input[type="number"]', '2');
  input('input[inputmode="decimal"]', '40.50');
  await tick();
  button('Finish and save session').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Not saved'));
  expect(document.body.textContent).toContain('Bench press');
  expect(goto).not.toHaveBeenCalled();
  const [url, options] = fetchMock.mock.calls[0];
  expect(url).toBe('https://backend.example/api/training/sessions/create');
  const command = JSON.parse(options.body);
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
  expect(fetchMock.mock.calls[1][1].body).toBe(options.body);
  expect(fetchMock.mock.calls[1][1].credentials).toBe('include');
});
