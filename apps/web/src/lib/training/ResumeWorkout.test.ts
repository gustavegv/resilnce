import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import ResumeWorkout from './ResumeWorkout.svelte';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
});

test('active workouts use the original home resume card and update the home state', async () => {
  const workout = {
    id: 'active-workout',
    status: 'active',
    sourceSession: { id: 'plan', name: 'Upper body' },
  };
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ workout }))));
  const onloaded = vi.fn();
  component = mount(ResumeWorkout, { target: document.body, props: { onloaded } });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Upper body'));
  const card = document.querySelector('a')!;
  expect(card.classList.contains('base-btn')).toBe(true);
  expect(card.classList.contains('alt')).toBe(true);
  expect(card.querySelector('.font-bold')?.textContent).toBe('Resume session');
  expect(card.querySelector('.truncate')?.textContent).toBe('Upper body');
  expect(card.getAttribute('href')).toBe('/workout/active-workout');
  expect(onloaded).toHaveBeenCalledWith(workout);
});

test('a failed active-workout lookup can be retried without showing a stale resume card', async () => {
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockRejectedValueOnce(new Error('Connection lost'))
      .mockResolvedValueOnce(new Response(JSON.stringify({ workout: null })))
  );
  const onloaded = vi.fn();
  component = mount(ResumeWorkout, { target: document.body, props: { onloaded } });
  await vi.waitFor(() => expect(document.body.textContent).toContain('Connection lost'));
  document.querySelector('button')!.click();
  await vi.waitFor(() => expect(onloaded).toHaveBeenCalledWith(null));
  expect(document.querySelector('a')).toBeNull();
});
