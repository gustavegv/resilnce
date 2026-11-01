import { afterEach, expect, test, vi } from 'vitest';
import { mount, unmount, tick } from 'svelte';
import SessionPlanEditor from './SessionPlanEditor.svelte';
import type { Session } from './types';

let component: ReturnType<typeof mount> | undefined;
afterEach(async () => {
  if (component) await unmount(component);
  component = undefined;
  document.body.innerHTML = '';
  vi.unstubAllGlobals();
});
const plan: Session = {
  id: '20000000-0000-4000-8000-000000000001',
  name: 'Upper body',
  revision: '5',
  createdAt: '2026-11-01T10:00:00Z',
  lastPerformedAt: null,
  lastPerformedStatus: 'never',
  placements: ['Bench', 'Row'].map((name, i) => ({
    id: `30000000-0000-4000-8000-00000000000${i + 1}`,
    variantId: `40000000-0000-4000-8000-00000000000${i + 1}`,
    position: i + 1,
    exercise: {
      id: `50000000-0000-4000-8000-00000000000${i + 1}`,
      name,
      revision: '1',
      muscleGroupCodes: [],
      archived: false,
    },
    variant: {
      id: `40000000-0000-4000-8000-00000000000${i + 1}`,
      name: 'Heavy',
      revision: '1',
      progressionMode: 'average',
      repThreshold: 12,
      autoIncrement: '2.50',
      restSeconds: 30,
      prescribedSets: 1,
      archived: false,
      targetSets: [
        {
          id: `60000000-0000-4000-8000-00000000000${i + 1}`,
          position: 1,
          targetReps: 12,
          targetWeight: '40.50',
        },
      ],
    },
  })),
};
const button = (text: string) =>
  [...document.querySelectorAll('button')].find((el) => el.textContent?.trim() === text)!;

test('maintains placement identities and ordered input through an identical failed-save retry', async () => {
  const saved = vi.fn();
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 'stale-revision', message: 'Session changed' }), {
        status: 409,
      })
    )
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          operation: {
            operationId: 'accepted',
            acceptedRevision: '6',
            committedAt: '2026-11-01T12:00:00Z',
          },
          session: { ...plan, name: 'Push', revision: '6' },
        })
      )
    );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(SessionPlanEditor, {
    target: document.body,
    props: { session: plan, onsaved: saved, ondeleted: vi.fn(), oncancel: vi.fn() },
  });
  const name = document.querySelector<HTMLInputElement>('[aria-label="Session name"]')!;
  name.value = 'Push';
  name.dispatchEvent(new Event('input', { bubbles: true }));
  document.querySelector<HTMLButtonElement>('[aria-label="Move Row up"]')!.click();
  await tick();
  button('Save session').click();
  await vi.waitFor(() => expect(document.body.textContent).toContain('Not saved'));
  expect(name.value).toBe('Push');
  expect(name.disabled || name.closest('fieldset')?.disabled).toBe(true);
  const first = JSON.parse(fetchMock.mock.calls[0][1].body);
  expect(first.expectedRevision).toBe('5');
  expect(first.items).toEqual([
    { kind: 'retainPlacement', placementId: plan.placements[1].id },
    { kind: 'retainPlacement', placementId: plan.placements[0].id },
  ]);
  button('Retry').click();
  await vi.waitFor(() => expect(saved).toHaveBeenCalled());
  expect(fetchMock.mock.calls[1][1].body).toBe(fetchMock.mock.calls[0][1].body);
});

test('can save an empty plan and requires a deliberate delete confirmation', async () => {
  const deleted = vi.fn();
  const fetchMock = vi
    .fn()
    .mockResolvedValue(
      new Response(JSON.stringify({ operation: {}, sessionId: plan.id, deleted: true }))
    );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(SessionPlanEditor, {
    target: document.body,
    props: { session: plan, onsaved: vi.fn(), ondeleted: deleted, oncancel: vi.fn() },
  });
  document.querySelector<HTMLButtonElement>('[aria-label="Remove Bench"]')!.click();
  await tick();
  document.querySelector<HTMLButtonElement>('[aria-label="Remove Row"]')!.click();
  await tick();
  expect(document.body.textContent).toContain('This plan is empty');
  button('Delete session').click();
  await tick();
  expect(fetchMock).not.toHaveBeenCalled();
  button('Confirm delete').click();
  await vi.waitFor(() => expect(deleted).toHaveBeenCalledWith(plan.id));
  expect(fetchMock.mock.calls[0][0]).toContain(`/sessions/${plan.id}/delete`);
});

test('uses reviewed personal variants for additions and rejects a duplicate of a retained placement', async () => {
  const bench = plan.placements[0];
  const light = {
    ...bench.variant,
    id: '40000000-0000-4000-8000-000000000003',
    name: 'Light',
    revision: '9',
  };
  const fetchMock = vi.fn(
    async (url: string) =>
      new Response(
        JSON.stringify(
          url.includes('/exercises?')
            ? { exercises: [{ exercise: bench.exercise, variants: [bench.variant, light] }] }
            : { operation: {}, session: plan }
        )
      )
  );
  vi.stubGlobal('fetch', fetchMock);
  component = mount(SessionPlanEditor, {
    target: document.body,
    props: { session: plan, onsaved: vi.fn(), ondeleted: vi.fn(), oncancel: vi.fn() },
  });
  const search = document.querySelector<HTMLInputElement>('[aria-label="Exercise name"]')!;
  search.value = 'Bench';
  search.dispatchEvent(new Event('input', { bubbles: true }));
  await vi.waitFor(() => expect(button('Bench')).toBeDefined());
  button('Bench').click();
  await tick();
  button('Reuse Heavy unchanged').click();
  await tick();
  expect(document.body.textContent).toContain('already in this session');
  button('Reuse Light unchanged').click();
  await tick();
  button('Save session').click();
  await vi.waitFor(() =>
    expect(fetchMock.mock.calls.some((call) => call[0].includes('/update'))).toBe(true)
  );
  const call = fetchMock.mock.calls.find((call) => call[0].includes('/update'))!;
  const command = JSON.parse((call as unknown as [string, { body: string }])[1].body);
  expect(command.items).toEqual([
    { kind: 'retainPlacement', placementId: plan.placements[0].id },
    { kind: 'retainPlacement', placementId: plan.placements[1].id },
    {
      kind: 'reuseVariant',
      exerciseId: bench.exercise.id,
      variantId: light.id,
      expectedVariantRevision: '9',
    },
  ]);
});
