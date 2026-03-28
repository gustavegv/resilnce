<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, scale } from 'svelte/transition';

  import '../../app.css';
  import { goto } from '$app/navigation';
  import Icon from '@iconify/svelte';
  import ExerciseTrackScreen from './ExerciseTrackScreen.svelte';
  import LoadingSkeleton from './LoadingSkeleton.svelte';
  import * as Alert from '$lib/components/alert/index.js';

  import ExerciseCompleteScreen from '../../components/ExerciseCompleteScreen.svelte';
  import { resolve } from '$app/paths';
  import { cubicOut } from 'svelte/easing';

  import {
    CompleteSession,
    GetSessionExercises,
    SendUpdate,
    SetActiveSession,
    type ExerciseInfo,
  } from './dbFetches';

  import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
  import { Portal } from 'bits-ui';

  // Exercise ID som blir tilldelad när man callar komponenten:
  //   Exempel:  <Track sesID="push" />
  let { sesID }: { sesID: string } = $props();

  // State variabler
  let exercises: ExerciseInfo[] = $state([]);

  let currentExerciseIndex = $state(0);
  let loading: boolean = $state(true);

  let showOverlay: boolean = $state(false);
  let error: string | null = $state(null);
  let allFinished: boolean = $state(false);

  let editMode: boolean = $state(false);

  // Derived variabler
  const currentExercise = $derived(
    !loading && exercises[currentExerciseIndex] ? exercises[currentExerciseIndex] : undefined
  );

  const finished: boolean = $derived(currentExercise?.finished ?? false);

  onMount(async () => {
    try {
      exercises = await GetSessionExercises(parseInt(sesID));
      await SetActiveSession(sesID);
    } catch (e) {
      error = (e as Error).message;
      console.error(error);
    } finally {
      loading = false;
    }
    monitorOutOfBoundsMovement();
  });

  $effect(() => {
    if (currentExercise) {
      console.log('Exercise index', currentExerciseIndex, 'rendered');
    }
  });

  //Functions

  function handleRepCountIncrementation({ id, count }: { id: number; count: number }) {
    if (currentExercise) {
      currentExercise.currentProgress.repsPerSet[id - 1] = count;
    }
  }

  function flashLoadingScreen(speed: number) {
    showOverlay = true;
    setTimeout(() => (showOverlay = false), speed);
  }

  async function loadNextExercise() {
    if (currentExerciseIndex + 1 < exercises.length) {
      currentExerciseIndex += 1;
      console.log('loaded next, index is: ' + currentExerciseIndex);
    } else {
      console.log('Workout finished?');
    }
  }

  function checkRepThresholdMet(r: number[], threshold: number): boolean {
    if (r.length === 0) return false;
    const t = r.reduce((acc, c) => acc + c, 0) / r.length;
    return t > threshold ? true : false;
  }

  function packageUpdatedProgress() {
    if (currentExercise) {
      const finalReps = currentExercise.currentProgress.repsPerSet;
      let finalWeight: number = currentExercise?.currentProgress.weightPerSet[0];
      const threshold = currentExercise.rep_threshold ?? 12;

      var packagedReps: number[] = finalReps.slice(); // kopiera array

      if (checkRepThresholdMet(finalReps, threshold)) {
        const baseRep = Math.floor(threshold / 1.7);
        packagedReps.fill(baseRep, 0, finalReps.length);
        finalWeight += currentExercise.auto_increase ?? 2.5;
      }

      var packagedWeight: number[] = new Array(packagedReps.length);
      packagedWeight.fill(finalWeight, 0, packagedReps.length);

      const updateInfo = {
        reps: packagedReps,
        weights: packagedWeight,
        id: String(currentExercise.id ?? -1),
      };
      return updateInfo;
    }
    return null;
  }

  async function submitExercise() {
    const savedProgress = packageUpdatedProgress();
    if (savedProgress == null) {
      console.error('Updated info packaging error');
      return;
    }

    SendUpdate(savedProgress, sesID);

    exercises[currentExerciseIndex].finished = true;
    flashLoadingScreen(150);

    if (checkAllFinished()) {
      allFinished = true;
      await CompleteSession(sesID);

      console.log('Workout finished, active set to false');
    } else {
      await new Promise((resolve) => setTimeout(resolve, 100));
      await loadNextExercise();
      monitorOutOfBoundsMovement();
    }
  }

  function checkAllFinished(): boolean {
    let finished: boolean = true;

    exercises.forEach((ex) => {
      if (ex.finished == undefined || ex.finished == false) {
        finished = false;
      }
    });
    return finished;
  }

  let prevExists = $state(false);
  let nextExists = $state(true);

  function monitorOutOfBoundsMovement() {
    console.log('my index: ' + currentExerciseIndex);
    if (currentExerciseIndex - 1 < 0) {
      prevExists = false;
    } else {
      prevExists = true;
    }
    if (currentExerciseIndex + 1 >= exercises.length) {
      nextExists = false;
    } else {
      nextExists = true;
    }
  }

  function prevExercise() {
    if (currentExerciseIndex > 0) {
      currentExerciseIndex--;
    }
    monitorOutOfBoundsMovement();
  }

  function skipExercise() {
    if (exercises.length > currentExerciseIndex + 1) {
      currentExerciseIndex++;
    }
    monitorOutOfBoundsMovement();
  }

  async function quitSession() {
    await CompleteSession(sesID);
    goto(resolve('/'));
  }

  function enterEditMode() {
    editMode = true;
  }

  function getProgressRatio(): number {
    if (allFinished) {
      return 100;
    }
    const ratio: number = currentExerciseIndex / exercises.length;
    return 100 * ratio;
  }

  function getProgressFraction(): string {
    if (allFinished) {
      return exercises.length + ' of ' + exercises.length;
    }

    return currentExerciseIndex + 1 + ' of ' + exercises.length;
  }

  let showActionMenu: boolean = $state(false);

  function growDown(node: Element, { duration = 180 } = {}) {
    return {
      duration,
      easing: cubicOut,
      css: (t: number) => `
      opacity: ${t};
      transform: scaleY(${0.72 + 0.28 * t}) scaleX(${0.94 + 0.06 * t});
    `,
    };
  }

  function toggleActionMenu() {
    showActionMenu = !showActionMenu;
  }

  function closeActionMenu() {
    showActionMenu = false;
  }

  function handleWindowKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && showActionMenu) {
      closeActionMenu();
    }
  }

  let alertVisible = $state(false);

  function toggleAlert() {
    alertVisible = true;
  }

  async function handleQuitAction() {
    closeActionMenu();
    toggleAlert();
  }

  function handleEditAction() {
    closeActionMenu();
    enterEditMode();
  }

  async function finishSessionNow() {
    closeActionMenu();

    if (confirm('Finish the session now?\nYou will go to the session summary.')) {
    }
  }

  function getNextUnfinishedIndex(): number {
    return exercises.findIndex((ex) => !ex.finished);
  }

  function checkNextUnfinished(): boolean {
    const index = getNextUnfinishedIndex();
    return index !== -1 && exercises[index].id === currentExercise?.id;
  }

  function gotoNextUnfinished() {
    const index = getNextUnfinishedIndex();
    if (index !== -1) {
      currentExerciseIndex = index;
      monitorOutOfBoundsMovement();
    }
  }
</script>

<Alert.Root bind:open={alertVisible}>
  <Alert.Content title="Quit session?" class="border-border border">
    <Alert.Description>All completed exercise are already saved.</Alert.Description>

    <div class="mt-5 flex justify-evenly gap-3">
      <Alert.Cancel>No, take me back</Alert.Cancel>
      <Alert.Action class="bg-destructive text-white" onclick={() => quitSession()}>
        Quit session.
      </Alert.Action>
    </div>
  </Alert.Content>
</Alert.Root>

<svelte:window onkeydown={handleWindowKeydown} />

{#if showActionMenu}
  <button
    type="button"
    class="menu-backdrop"
    aria-label="Close session actions"
    onclick={closeActionMenu}
    transition:fade={{ duration: 140 }}
  ></button>
{/if}

<main class="app-container">
  {#if loading || error}
    <LoadingSkeleton />
    <h1 class="text-2xl">{error ?? ''}</h1>
  {:else}
    <div class="progress">
      <div class="progress-header">
        <span class="label">Session Progress</span>
        <span class="label label--accent">Exercise {getProgressFraction()}</span>
      </div>

      <div class="progress-track">
        <div class="progress-fill" style="width: {getProgressRatio()}%"></div>
      </div>
    </div>

    {#if allFinished}
      <h1 class="title" class:is-celebrating={allFinished}>Session finished!</h1>
      <h2 class="subtitle">Session overview</h2>
      <hr />
      <ExerciseCompleteScreen {exercises} />
      <button class="buttonClass w-full" onclick={() => goto(resolve(`/`))}
        >Return to homepage</button
      >
    {:else}
      <div class="session-menu-anchor">
        <button
          type="button"
          onclick={toggleActionMenu}
          class="action-trigger buttonClass"
          aria-haspopup="menu"
          aria-expanded={showActionMenu}
          aria-controls="session-action-menu"
        >
          <Icon icon="material-symbols:more-horiz" color="grey" width={24} />
        </button>

        {#if showActionMenu}
          <div class="session-action-menu-shell">
            <div
              id="session-action-menu"
              class="session-action-menu"
              aria-label="Session actions"
              transition:growDown
            >
              <button
                type="button"
                class="session-action"
                aria-label="Quit session"
                onclick={handleQuitAction}
              >
                <Icon icon="material-symbols:logout-rounded" width={20} />
              </button>

              <button
                type="button"
                class="session-action"
                aria-label="Edit"
                onclick={handleEditAction}
              >
                <Icon icon="material-symbols:edit-outline-rounded" width={20} />
              </button>

              <button
                type="button"
                class="session-action"
                aria-label="Finish session"
                onclick={finishSessionNow}
              >
                <Icon icon="material-symbols:check-circle-outline-rounded" width={20} />
              </button>
            </div>
          </div>
        {/if}
      </div>
      <ExerciseTrackScreen
        {finished}
        exIndex={currentExerciseIndex}
        onCount={handleRepCountIncrementation}
        onSubmit={submitExercise}
        exData={currentExercise}
      />

      <div class="movement-cont">
        <button class="movement-b" class:inactive={!prevExists} onclick={() => prevExercise()}>
          <Icon icon="material-symbols:arrow-left-alt-rounded" />
          <span>Prev</span>
        </button>
        <button
          class="movement-b long"
          class:inactive={checkNextUnfinished()}
          onclick={() => gotoNextUnfinished()}
        >
          <span>Go to current</span>
        </button>
        <button class="movement-b" class:inactive={!nextExists} onclick={() => skipExercise()}>
          <span>Skip</span>
          <Icon icon="material-symbols:arrow-right-alt-rounded" />
        </button>
      </div>
    {/if}
  {/if}
</main>

<style>
  hr {
    border: 1px solid var(--text-muted);
    margin: 10px 0;
    width: 100%;
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

  .title.is-celebrating {
    animation: session-finished-celebration 900ms ease both;
  }

  @keyframes session-finished-celebration {
    0% {
      transform: scale(0.92);
    }

    22% {
      transform: scale(1.05);
    }

    38% {
      transform: scale(1.06);
    }

    58% {
      transform: scale(1.09);
    }

    78% {
      transform: scale(0.99);
    }

    100% {
      transform: scale(1);
    }
  }

  .subtitle {
    margin-top: 2rem;
    font-size: 1.125rem;
    letter-spacing: 0.03em;
    color: var(--color-gray);
    text-align: left;
    width: 100%;
  }

  .but {
    width: 80%;
    background: var(--color-alt);
  }

  .progress {
    margin-bottom: 2rem;
    width: 100%;
  }

  .progress-header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    margin-bottom: 0.5rem;
  }

  .label {
    font-family: var(--font-label);
    font-size: 0.75rem;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    color: var(--text-muted);
  }

  .label--accent {
    font-weight: 600;
    color: var(--color-alt);
  }

  .progress-track {
    width: 100%;
    height: 2px;
    background: var(--border);
    overflow: hidden;
  }

  .progress-fill {
    transition: all ease-in-out 280ms;
    height: 100%;
    background: var(--color-alt);
  }

  .app-container {
    display: flex;
    flex-direction: column;
    padding: 4rem 2rem;
    max-width: 30rem;
    margin: 0 auto;
    text-align: center;
    align-items: center;
    background: var(--gradient-prim);
    overflow-y: auto;
    min-height: 100vh;
    overflow-x: hidden;
  }

  .overlay {
    position: fixed;
    inset: 0;
    background: #fff;
    opacity: 0.2;
    pointer-events: none;
  }
  .movement-cont {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    width: 100%;
    margin-top: 2rem;
  }

  .movement-b {
    display: flex;

    flex-direction: row;
    justify-content: space-evenly;
    align-items: center;
    width: 5rem;
    height: 2rem;
    border-radius: 20px;
    border: none;
    outline: none;
    box-shadow: var(--shadow-dark);
    transition-property: opacity;
    transition-timing-function: ease;
    transition-duration: 700ms;
    transition-delay: 150ms;
  }

  .movement-b.inactive {
    filter: brightness(0.5);
    transition: opacity ease 200ms;
    opacity: 0;
  }

  .movement-b.long {
    width: 40%;
  }

  .movement-b span {
    font-size: 12px;
    color: var(--color-text);
  }

  .menu-backdrop {
    position: fixed;
    inset: 0;
    z-index: 1;
    border: none;
    padding: 0;
    margin: 0;
    background: linear-gradient(180deg, rgba(28, 28, 28, 0.596), transparent);
    mask: linear-gradient(black, transparent);
    backdrop-filter: blur(2px);

    cursor: default;
  }

  .session-menu-anchor {
    position: absolute;
    right: 0;
    top: 0;
    transform: translate(1rem, -1.3rem); /* visually offset without creating overflow */
    z-index: 20;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .action-trigger {
    width: fit-content;
    height: fit-content;
    background-color: rgba(240, 248, 255, 0);
    box-shadow: none;
    z-index: 21;
  }

  .session-action-menu-shell {
    position: absolute;
    top: calc(100% + 0.6rem);
    left: 50%;
    transform: translateX(-50%);
  }

  .session-action-menu {
    transform-origin: top center;
    padding: var(--spacing-xs);
    display: flex;
    flex-direction: column;
    border-radius: 20px 0 0 20px;
    background: var(--color-background);
    border: 1px solid rgba(255, 255, 255, 0.15);
    backdrop-filter: blur(14px);
    box-shadow:
      0 18px 40px rgba(15, 23, 42, 0.18),
      var(--shadow-dark);
  }

  .session-action {
    display: grid;
    place-items: center;
    width: 3rem;
    height: 3.45rem;
    border: none;
    border-radius: 999px;
    background: transparent;
    color: var(--color-text);
    transition:
      transform 140ms ease,
      background-color 140ms ease,
      box-shadow 140ms ease;
  }

  .session-action:active {
    transform: scale(0.96);
  }

  .session-action:focus-visible {
    outline: 2px solid var(--color-alt);
    outline-offset: 2px;
  }
</style>
