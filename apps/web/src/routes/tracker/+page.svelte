<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import SessionSlug from '../../components/SessionSlug.svelte';

  import * as Alert from '$lib/components/alert/index.js';

  import { fade, scale } from 'svelte/transition';
  import Icon from '@iconify/svelte';
  import { resolve } from '$app/paths';

  import type { SessionMetaData } from './dbFetches';
  import { CheckActiveSession, DeleteSession, GetSessions, SetActiveSession } from './dbFetches';
  import { toast, Toaster } from 'svelte-sonner';
  import EditScreen from './EditScreen.svelte';

  let slugs: SessionMetaData[] = $state([]);
  let activeSlugs: SessionMetaData[] = $state([]);
  let recentSlugs: SessionMetaData[] = $state([]);
  let otherSlugs: SessionMetaData[] = $state([]);

  let isAnotherSessionActive: boolean = $state(false);
  let sessionsLoaded: boolean = $state(false);

  let itemToRemove: [string, number] = $state(['', 0]);

  let sessionToStart: number = $state(-1);
  let sessionToEditID: number = $state(-1);

  let activeSessionPopupShowing: boolean = $state(false);
  let deletePopupShowing: boolean = $state(false);
  let editSessionPopupShowing: boolean = $state(false);

  let toEditSessionName: string = $state('sessiontoedit');
  let newSessionName: string = $state('');

  const activeTimespan = new Date();
  activeTimespan.setDate(activeTimespan.getDate() - 14);

  onMount(async () => {
    slugs = await GetSessions();

    const [activeID, activeName] = await CheckActiveSession();
    isAnotherSessionActive = activeID != '';
    sessionsLoaded = true;

    partitionSlugs(slugs);
  });

  function partitionSlugs(slugs: SessionMetaData[]) {
    activeSlugs = slugs.filter((slug) => !slug.deleted);

    recentSlugs = activeSlugs.filter((slug) => {
      if (!slug.date) return false;
      return new Date(slug.date) >= activeTimespan;
    });

    otherSlugs = activeSlugs.filter((slug) => {
      if (!slug.date) return true;
      return new Date(slug.date) < activeTimespan;
    });
  }

  function startSes(id: number) {
    if (isAnotherSessionActive) {
      activeSessionPopupShowing = true;
      sessionToStart = id;
    } else {
      goto(resolve(`/tracker/${id}`));
    }
  }

  function confirmStartSession() {
    const id = sessionToStart;
    if (id != -1) {
      SetActiveSession(String(id));
      goto(resolve(`/tracker/${id}`));
    }
  }

  function getSlugFromID(id: number): SessionMetaData | null {
    return slugs.find((slug) => slug.id === id) || null;
  }

  function editSes(id: number) {
    let editSlug: SessionMetaData | null = getSlugFromID(id);

    if (editSlug == null) {
      toast.error('Session to edit not found');
      return;
    }
    toEditSessionName = editSlug.name;
    newSessionName = editSlug.name;
    sessionToEditID = editSlug.id;
    editSessionPopupShowing = true;
  }

  function deleteSession(SessionTitle: string, SesID: number) {
    deletePopupShowing = true;
    itemToRemove = [SessionTitle, SesID];
  }

  async function confirmDeleteSession(toRemove: [string, number]) {
    const SessionTitle: string = toRemove[0];
    const SessionID: number = toRemove[1];

    console.log(SessionTitle, 'deleted.');
    deletePopupShowing = false;
    if (await DeleteSession(SessionID)) {
      await new Promise((r) => setTimeout(r, 100));
      deleteLocally(SessionTitle);
      toast.success(`Session ${SessionTitle} deleted!`);
    } else {
      toast.error('Deletion failed. Could not delete. Try again later.', {
        duration: 5000,
        style: 'background: red;',
      });
    }
  }

  function deleteLocally(id: string) {
    for (const item of slugs) {
      if (item.name === id) {
        item.deleted = true;
        break;
      }
    }
  }

  function shuffleEdit() {
    const trimmedSessionName = newSessionName.trim();

    const editedSlug = getSlugFromID(sessionToEditID);
    if (editedSlug) {
      editedSlug.name = trimmedSessionName;
      sortByCategory(activeCategory);
    }

    // cleanup
    editSessionPopupShowing = false;
    toEditSessionName = trimmedSessionName;
    newSessionName = '';
    sessionToEditID = -1;
  }

  const keywordsByCategory: Record<string, readonly string[]> = {
    'Upper body': [
      'upper',
      'upper body',
      'pull',
      'push',
      'chest',
      'pecs',
      'back',
      'lats',
      'traps',
      'delts',
      'shoulders',
      'biceps',
      'triceps',
      'arms',
      'bench',
      'press',
      'overhead press',
      'ohp',
      'incline',
      'dip',
      'row',
      'pulldown',
      'pullup',
      'pull-up',
      'chinup',
      'chin-up',
    ],
    'Lower body': [
      'lower',
      'lower body',
      'legs',
      'leg day',
      'quads',
      'quadriceps',
      'hamstrings',
      'hams',
      'calves',
      'adductors',
      'abductors',
      'squat',
      'front squat',
      'back squat',
      'deadlift',
      'rdl',
      'romanian deadlift',
      'hinge',
      'lunge',
      'split squat',
      'bulgarian split squat',
      'leg press',
      'leg curl',
      'leg extension',
      'hip thrust',
      "booty",
      "glute",
      "glutes",
      "peach",
      "cake",
      "wheels",
      "quad",
      "hamstring",
      "hamstrings",
      "thighs",
      "posterior",
      "squats",
    ],
    'Full body': [
      'full body',
      'full-body',
      'fullbody',
      'total body',
      'total-body',
      'totalbody',
      'whole body',
      'whole-body',
      'compound',
      'compound day',
    ],
  };

  function sortByCategory(category: string): void {
    activeCategory = category;

    if (category === 'All') {
      partitionSlugs(slugs);
      return;
    }

    if (category === 'Other') {
      const allKnownKeywords = Object.values(keywordsByCategory)
        .flat()
        .map((keyword) => keyword.toLowerCase());

      const filteredSlugs = slugs.filter(({ name = '' }) => {
        const normalizedName = name.toLowerCase();
        return !allKnownKeywords.some((keyword) => normalizedName.includes(keyword));
      });

      partitionSlugs(filteredSlugs);
      return;
    }

    const matches = (keywordsByCategory[category] ?? []).map((k) => k.toLowerCase());

    const filteredSlugs = slugs.filter(({ name = '' }) => {
      const normalizedName = name.toLowerCase();
      return matches.some((keyword) => normalizedName.includes(keyword));
    });

    partitionSlugs(filteredSlugs);
  }

  const categories = [
    { label: 'All', tone: 'all' },
    { label: 'Upper body', tone: 'upper-body' },
    { label: 'Lower body', tone: 'lower-body' },
    { label: 'Full body', tone: 'full-body' },
    { label: 'Other', tone: 'other' },
  ];

  let activeCategory = $state(categories[0].label);
</script>

<div class="main">
  <eyebrow>Sessions</eyebrow>
  <h2>Start a new session</h2>
  <subtitle>Select the session you want to run.</subtitle>
  <Toaster theme="dark"></Toaster>

  <Alert.Root bind:open={deletePopupShowing}>
    <Alert.Content title="Are you absolutely sure?" class="border-border border">
      <Alert.Description>
        This action cannot be undone. This will permanently delete the session and remove it from
        our servers.
      </Alert.Description>

      <div class="mt-5 flex justify-evenly gap-3">
        <Alert.Cancel>No, take me back</Alert.Cancel>
        <Alert.Action
          class="bg-destructive text-white"
          onclick={() => confirmDeleteSession(itemToRemove)}
        >
          Remove {itemToRemove[0]}
        </Alert.Action>
      </div>
    </Alert.Content>
  </Alert.Root>

  <Alert.Root bind:open={activeSessionPopupShowing}>
    <Alert.Content title="Another session is already active!" class="border-border border">
      <Alert.Description>Are you sure you want to start a new one?</Alert.Description>

      <div class="mt-5 flex justify-end gap-3">
        <Alert.Cancel>No, take me back.</Alert.Cancel>
        <Alert.Action class="bg-accent" onclick={() => confirmStartSession()}>
          Start new session!
        </Alert.Action>
      </div>
    </Alert.Content>
  </Alert.Root>

  <EditScreen
    bind:open={editSessionPopupShowing}
    bind:newSessionName
    {sessionToEditID}
    sessionName={toEditSessionName}
    onConfirm={shuffleEdit}
  />

  <hr />

  {#if slugs.length && sessionsLoaded}
    <div class="category-carousel">
      {#each categories as category}
        <button
          type="button"
          class={`category-pill ${category.tone}`}
          class:active-category={activeCategory === category.label}
          onclick={() => sortByCategory(category.label)}
        >
          {category.label}
        </button>
      {/each}
    </div>

    <div class="btn-container">
      {#if recentSlugs.length}
        <h3>Recent</h3>
        {#each recentSlugs as slug, i}
          <div style="width: inherit" in:fade|global={{ delay: i * 50 }} out:scale>
            <SessionSlug
              onPress={() => startSes(slug.id)}
              onEdit={() => editSes(slug.id)}
              onDel={() => deleteSession(slug.name, slug.id)}
              {slug}
            />
          </div>
        {/each}
      {/if}

      {#if otherSlugs.length}
        <h3 in:fade|global={{ delay: 50 }} class="mt-4">Older</h3>
        {#each otherSlugs as slug, i}
          <div style="width: inherit" in:fade|global={{ delay: 50 + i * 50 }} out:scale>
            <SessionSlug
              onPress={() => startSes(slug.id)}
              onEdit={() => editSes(slug.id)}
              onDel={() => deleteSession(slug.name, slug.id)}
              {slug}
            />
          </div>
        {/each}
      {/if}
    </div>
  {:else if !sessionsLoaded}
    <Icon icon="svg-spinners:3-dots-bounce" width="30" />
  {:else}
    <subtitle>Nothing here yet. Add a session to get started.</subtitle>
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
    overflow-y: scroll;
  }

  .title {
    margin: 0 0 0.25rem;
    font-size: 2.25rem;
    line-height: 1.1;
    font-weight: 700;
    letter-spacing: -0.02em;
    text-align: left;
    width: 100%;
  }

  hr {
    height: 0.1px;
    background-color: #ffffff;
    margin: 20px 0;
    width: 100%;
  }

  .btn-container {
    display: flex;
    flex-direction: column;
    align-items: baseline;
    justify-content: center;
    box-sizing: border-box;
    width: 100%;
    text-align: left;
    gap: 0.5rem;
    overflow: hidden;
  }

  .category-carousel {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    overflow-x: auto;
    padding: 0.25rem 1rem 0.75rem 1rem;
    scroll-snap-type: x proximity;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  .category-carousel::-webkit-scrollbar {
    display: none;
  }

  .category-pill {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 auto;
    padding: 0.5rem 1rem;
    min-width: max(5rem);
    border: 0;
    border-radius: var(--border-middle);
    font: inherit;
    font-weight: 500;
    font-size: 14px;
    color: #fff;
    white-space: nowrap;
    cursor: pointer;
    scroll-snap-align: start;
    opacity: 0.65;
    transition:
      transform 0.2s ease,
      opacity 0.2s ease,
      box-shadow 0.2s ease;
  }

  .active-category {
    opacity: 1;
    box-shadow:
      0 10px 24px rgba(0, 0, 0, 0.18),
      0 0px 2px rgba(255, 255, 255, 0.539);
    transform: translateY(-2px);
    font-weight: 600;
  }

  .category-pill.all {
    background: var(--surface-middle);
  }

  .category-pill.favorites {
    background: linear-gradient(135deg, #f5c60b, #efb044);
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
</style>
