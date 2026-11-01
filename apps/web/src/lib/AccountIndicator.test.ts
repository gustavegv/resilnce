import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import { user, upgradeRequired } from './stores/appState';
import { getMe } from '../routes/me';
import AccountIndicator from './AccountIndicator.svelte';
import { EditSessionName } from '../routes/tracker/dbFetches';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  user.set(null);
  upgradeRequired.set(false);
  vi.unstubAllGlobals();
});

test('shows the server-selected development account persistently and clears it after identity expires', async () => {
  // A browser username cookie cannot select the account displayed by the app.
  document.cookie = 'username=Other account; path=/';
  const account = {
    id: 'development@resilnce.invalid',
    email: 'development@resilnce.invalid',
    name: 'Development account',
    developmentMode: true,
    capabilities: {
      applicationModel: 'exercise-centric',
      trainingWrites: true,
      writeMaintenance: false,
    },
  };
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce({ ok: true, json: async () => account })
    .mockResolvedValueOnce({ ok: false, status: 401 });
  vi.stubGlobal('fetch', fetchMock);
  component = mount(AccountIndicator, { target: document.body });
  expect(document.body.textContent).not.toContain('Development account');
  expect(await getMe()).toBe(true);
  await tick();
  expect(document.body.textContent).toContain('Development account');
  expect(document.body.textContent).toContain('development@resilnce.invalid');
  expect(document.body.textContent).not.toContain('Other account');
  expect(fetchMock).toHaveBeenCalledWith(
    'https://backend.example/api/me',
    expect.objectContaining({ credentials: 'include' })
  );
  expect(await getMe()).toBe(false);
  await tick();
  expect(document.body.textContent).not.toContain('Development account');
});

test('normal login never displays a development indicator', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        id: 'owner@example.com',
        email: 'owner@example.com',
        name: 'Owner',
        developmentMode: false,
        capabilities: { applicationModel: 'legacy', trainingWrites: true, writeMaintenance: false },
      }),
    })
  );
  component = mount(AccountIndicator, { target: document.body });
  await getMe();
  await tick();
  expect(document.body.textContent).not.toContain('Development account');
});

test('requires a refresh when an old-model save is rejected without retrying a fallback write', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValue(
      new Response(
        JSON.stringify({
          code: 'upgrade-required',
          message: 'Refresh the application before saving',
        }),
        { status: 426, headers: { 'Content-Type': 'application/json' } }
      )
    );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(AccountIndicator, { target: document.body });
  expect(await EditSessionName(1, 'Changed plan')).toBe(false);
  await tick();
  expect(document.body.textContent).toContain('Refresh the application before saving');
  expect(document.querySelector('button')?.textContent).toContain('Refresh');
  expect(fetchMock).toHaveBeenCalledTimes(1);
});
