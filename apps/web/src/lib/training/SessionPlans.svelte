<script lang="ts">
  import { onMount } from 'svelte';
  import SessionSlug from '../../components/SessionSlug.svelte';
  import { listSessions, readSession } from './client';
  import type { Session } from './types';
  import { categories, matchesCategory } from './sessionCategories';
  let sessions = $state.raw<Session[]>([]);
  let selected = $state.raw<Session | null>(null);
  let loading = $state(true);
  let reading = $state(false);
  let error = $state('');
  let activeCategory = $state('All');
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
  async function showPlan(id: string) {
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
  <span>Sessions</span>
  <h2>Start a new session</h2>
  <p>Select a session to view its plan.</p>
  <hr />
  {#if error}<p role="alert">{error}</p>
    <button onclick={load}>Retry</button>{/if}
  {#if loading}<p role="status">Loading sessions…</p>
  {:else if !sessions.length}<p>Nothing here yet. Add a session to get started.</p>
  {:else}
    <div class="category-carousel">
      {#each categories as category (category)}
        <button
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
          slug={{
            name: session.name,
            date: session.lastPerformedAt ? new Date(session.lastPerformedAt) : undefined,
          }}
          onPress={() => showPlan(session.id)}
        />
      {/each}
      {#if older.length}<h3>Older</h3>{/if}
      {#each older as session (session.id)}
        <SessionSlug
          slug={{
            name: session.name,
            date: session.lastPerformedAt ? new Date(session.lastPerformedAt) : undefined,
          }}
          onPress={() => showPlan(session.id)}
        />
      {/each}
    </div>
  {/if}
  {#if reading}<p role="status">Loading plan…</p>{/if}
  {#if selected}
    <section class="plan" aria-label={`${selected.name} plan`}>
      <h3>{selected.name}</h3>
      {#if !selected.placements.length}<p>Add a variant before starting this session.</p>{/if}
      {#each selected.placements as placement (placement.id)}
        <div class="exercise">
          <h4>{placement.position}. {placement.exercise.name} · {placement.variant.name}</h4>
          {#if placement.exercise.archived || placement.variant.archived}<p>
              Unavailable — remove this placement before starting.
            </p>{/if}
          <p>
            {placement.variant.prescribedSets} sets · threshold {placement.variant.repThreshold} · {placement
              .variant.autoIncrement} kg increase
          </p>
          {#each placement.variant.targetSets as target (target.id)}<p>
              Set {target.position}: {target.targetReps} reps · {target.targetWeight} kg target
            </p>{/each}
        </div>
      {/each}
    </section>
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
    padding: 0.25rem 0 0.75rem;
  }
  .category-pill {
    flex: 0 0 auto;
    padding: 0.4rem 0.5rem;
    border-radius: var(--border-middle);
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
    box-shadow: var(--shadow-dark);
  }
  .plan {
    width: 100%;
    margin-top: 1rem;
  }
  .exercise {
    background: var(--surface-low);
    border-radius: 15px;
    padding: 1rem;
    margin: 0.5rem 0;
  }
</style>
