<script lang="ts">
  import { onMount } from 'svelte';
  import SessionSlug from '../../components/SessionSlug.svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { listSessions, readSession } from './client';
  import type { Session } from './types';
  import { categories, matchesCategory } from './sessionCategories';
  import SessionPlanEditor from './SessionPlanEditor.svelte';
  import Icon from '@iconify/svelte';
  let sessions = $state.raw<Session[]>([]);
  let selected = $state.raw<Session | null>(null);
  let loading = $state(true);
  let reading = $state(false);
  let error = $state('');
  let activeCategory = $state('All');
  let editing = $state(false);
  let maintenanceBusy = $state(false);
  let starting = $state(false);
  let deleting = $state(false);
  const recentBoundary = Date.now() - 14 * 24 * 60 * 60 * 1000;
  let filtered = $derived(
    sessions.filter((session) => matchesCategory(session.name, activeCategory))
  );
  let recent = $derived(
    filtered.filter(
      (session) =>
        session.lastPerformedAt !== null && Date.parse(session.lastPerformedAt) >= recentBoundary
    )
  );
  let older = $derived(
    filtered.filter(
      (session) =>
        session.lastPerformedAt === null || Date.parse(session.lastPerformedAt) < recentBoundary
    )
  );
  async function load() {
    loading = true;
    error = '';
    try {
      sessions = (await listSessions()).sessions;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to load sessions.';
    } finally {
      loading = false;
    }
  }
  onMount(() => {
    void load();
  });
  async function showPlan(id: string, edit = false, remove = false) {
    if (maintenanceBusy || starting || reading) return;
    if (!edit) {
      starting = true;
      try {
        await goto(resolve('/workout/start/[sessionId]', { sessionId: id }));
      } finally {
        starting = false;
      }
      return;
    }
    editing = edit;
    deleting = remove;
    reading = true;
    error = '';
    selected = null;
    try {
      selected = (await readSession(id)).session;
    } catch (failure) {
      error = failure instanceof Error ? failure.message : 'Unable to load the plan.';
    } finally {
      reading = false;
    }
  }
</script>

<div class="main">
  <eyebrow>Sessions</eyebrow>
  <h2>Start a new session</h2>
  <subtitle>Select the session you want to run.</subtitle>
  <hr />
  {#if error}<p role="alert">{error}</p>
    <button onclick={load}>Retry</button>{/if}
  {#if loading}<div role="status" aria-label="Loading sessions">
      <Icon icon="svg-spinners:3-dots-bounce" width="30" />
    </div>
  {:else if !sessions.length}<p>Nothing here yet. Add a session to get started.</p>
  {:else}
    <div class="category-carousel">
      {#each categories as category (category)}
        <button
          disabled={maintenanceBusy || starting || reading}
          class={[
            'category-pill',
            category.toLowerCase().replace(' ', '-'),
            { active: activeCategory === category },
          ]}
          onclick={() => {
            activeCategory = category;
            selected = null;
          }}>{category}</button
        >
      {/each}
    </div>
    <div class="btn-container">
      {#if recent.length}<h3>Recent</h3>{/if}
      {#each recent as session (session.id)}
        <SessionSlug
          disabled={maintenanceBusy || starting || reading}
          slug={{
            name: session.name,
            date: session.lastPerformedAt ? new Date(session.lastPerformedAt) : undefined,
            undatedLabel:
              session.lastPerformedStatus === 'unknown' ? 'Last run unavailable' : 'Never run',
          }}
          onPress={() => showPlan(session.id)}
          onEdit={() => showPlan(session.id, true)}
          onDel={() => showPlan(session.id, true, true)}
        />
      {/each}
      {#if older.length}<h3>Older</h3>{/if}
      {#each older as session (session.id)}
        <SessionSlug
          disabled={maintenanceBusy || starting || reading}
          slug={{
            name: session.name,
            date: session.lastPerformedAt ? new Date(session.lastPerformedAt) : undefined,
            undatedLabel:
              session.lastPerformedStatus === 'unknown' ? 'Last run unavailable' : 'Never run',
          }}
          onPress={() => showPlan(session.id)}
          onEdit={() => showPlan(session.id, true)}
          onDel={() => showPlan(session.id, true, true)}
        />
      {/each}
    </div>
  {/if}
  {#if reading}<p role="status">Loading plan…</p>{/if}
  {#if selected}
    {#if editing}
      {#key selected.id}
        <SessionPlanEditor
          session={selected}
          confirmDeletion={deleting}
          onbusy={(value) => {
            maintenanceBusy = value;
          }}
          onsaved={(session) => {
            selected = null;
            sessions = sessions.map((item) => (item.id === session.id ? session : item));
            editing = false;
          }}
          ondeleted={(id) => {
            sessions = sessions.filter((item) => item.id !== id);
            selected = null;
            editing = false;
          }}
          oncancel={() => {
            selected = null;
            editing = false;
          }}
        />
      {/key}
    {/if}
  {/if}
</div>

<style>
  .main {
    width: 100%;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: left;
    padding: 4rem 1rem;
    overflow-y: auto;
  }
  hr {
    height: 0.1px;
    background-color: #fff;
    margin: 20px 0;
    width: 100%;
  }
  .btn-container {
    display: flex;
    flex-direction: column;
    align-items: baseline;
    width: 100%;
    gap: 0.5rem;
  }
  .category-carousel {
    display: flex;
    gap: 0.5rem;
    width: 100%;
    overflow-x: auto;
    padding: 0.25rem 1rem 0.75rem 1rem;
    scroll-snap-type: x proximity;
    scrollbar-width: none;
  }
  .category-pill {
    flex: 0 0 auto;
    padding: 0.4rem 0.5rem;
    min-width: 5rem;
    border: 0;
    border-radius: var(--border-middle);
    font: inherit;
    font-weight: 500;
    color: #fff;
    white-space: nowrap;
    cursor: pointer;
    scroll-snap-align: start;
    transition:
      transform 0.2s ease,
      opacity 0.2s ease,
      box-shadow 0.2s ease;
    background: var(--surface-middle);
    font-size: 12px;
    opacity: 0.65;
  }
  .category-pill.all {
    background: var(--surface-middle);
  }
  .category-pill.upper-body {
    background: var(--chart-2);
  }
  .category-pill.lower-body {
    background: var(--chart-4);
  }
  .category-pill.full-body {
    background: var(--chart-5);
  }
  .category-pill.other {
    background: linear-gradient(135deg, #64748b, #808790);
  }
  .category-pill.active {
    opacity: 1;
    box-shadow:
      0 10px 24px rgba(0, 0, 0, 0.18),
      0 0px 2px rgba(255, 255, 255, 0.539);
    transform: translateY(-2px);
    font-weight: 600;
  }
</style>
