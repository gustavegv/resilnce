import { afterEach, expect, test, vi } from 'vitest';
import { createRawSnippet, mount, unmount } from 'svelte';
import { goto } from '$app/navigation';
import { page } from '$app/state';
import Layout from './+layout.svelte';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

test.each(['/workout/workout-id', '/workout/start/session-id'])(
  'the workout header Back button returns to tracker from %s',
  async (pathname) => {
    Object.assign(page, {
      url: new URL(pathname, 'https://app.example'),
      route: { id: pathname.includes('/start/') ? '/workout/start/[sessionId]' : '/workout/[id]' },
    });
    vi.stubGlobal('matchMedia', () => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 401 })));
    component = mount(Layout, {
      target: document.body,
      props: {
        children: createRawSnippet(() => ({ render: () => '<p>Workout page</p>' })),
      },
    });
    document.querySelector<HTMLElement>('[aria-label="Back"]')!.click();
    expect(goto).toHaveBeenCalledWith('/tracker');
  }
);
