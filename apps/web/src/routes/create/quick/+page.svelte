<script lang="ts">
  import { Textarea } from '$lib/components/ui/textarea/index.js';
  import * as Collapsible from '$lib/components/ui/collapsible/index.js';
  import { Checkbox } from '$lib/components/ui/checkbox/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { goto } from '$app/navigation';
  import Icon from '@iconify/svelte';
  import { toast } from 'svelte-sonner';
  import { resolve } from '$app/paths';
  import { QuickGeneration } from '../dbWrite';
  import { fade, fly } from 'svelte/transition';

  var userText = $state<string>('');
  var isAdditionalInfoEnabled = $state<boolean>(false);

  let cbSpellcheck = $state(false);
  let cbRestart = $state(false);
  let cbAutoChoose = $state(false);
  let cbStrength = $state(false);

  let collapsibleOpen = $state(false);

  let advancedOpen = $state(false);
  let loadStateText = $state('');
  let isLoadingStatesRunning = false;

  let dataPromise = $state<Promise<string> | null>(null);

  async function quickfillAPICall(prompt: string) {
    const choices: boolean[] = [cbSpellcheck, cbRestart, cbAutoChoose, cbStrength];

    // load
    // dataPromise = QuickGeneration(prompt, choices);
    // finish loading
    console.log(await dataPromise);
  }

  function validateForm(): true | string {
    if (userText.trim().length < 10) return 'Please enter at least 10 characters of data.';
    return true;
  }

  function handleContinue() {
    const validation = validateForm();
    collapsibleOpen = false;

    if (validation !== true) {
      toast.error(validation);
      return;
    } else {
      collapsibleOpen = true;
      runLoadingStates();
      quickfillAPICall(userText);
    }
  }

  async function saveSession() {
    const jsonData = await dataPromise;

    if (jsonData == null) {
      toast.error('Data creation error. Try again');
      console.error('JSON Data not loaded properly.');
      return;
    }
    const STORAGE_KEY = 'workoutPayload';
    sessionStorage.setItem(STORAGE_KEY, jsonData);
    console.log('Storing...');

    goto(`${resolve('/create/manual')}?quickload=1`);
  }

  const loadingTextOptionsByStage = [
    ['Analyzing your notes', 'Reviewing your input', 'Taking in your notes'],
    ['Understanding the workout', 'Finding exercises and sets', 'Mapping the workout structure'],
    ['Building the session', 'Organizing everything', 'Standardizing the format'],
    ['Cleaning up the details', 'Checking for missing details', 'Refining the session'],
    ['Getting it ready', 'Preparing your session', 'Finishing up'],
  ];

  function getRandomItem<T>(items: T[]): T {
    return items[Math.floor(Math.random() * items.length)];
  }

  function createRandomLoadingSequence(): string[] {
    return loadingTextOptionsByStage.map((options) => getRandomItem(options));
  }

  function wait(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  function createRandomStageDurations(totalMs = 20_000): number[] {
    const stageCount = loadingTextOptionsByStage.length;
    const minMsPerStage = 2_000;
    const maxMsPerStage = 6_000;
    const stepMs = 250;

    const durations = Array(stageCount).fill(minMsPerStage);
    const maxExtraStepsPerStage = Math.floor((maxMsPerStage - minMsPerStage) / stepMs);
    let remainingExtraSteps = Math.floor((totalMs - stageCount * minMsPerStage) / stepMs);

    while (remainingExtraSteps > 0) {
      const availableStageIndexes = durations
        .map((duration, index) => {
          const extraStepsUsed = (duration - minMsPerStage) / stepMs;
          return extraStepsUsed < maxExtraStepsPerStage ? index : -1;
        })
        .filter((index) => index !== -1);

      const randomStageIndex = getRandomItem(availableStageIndexes);
      durations[randomStageIndex] += stepMs;
      remainingExtraSteps -= 1;
    }

    return durations;
  }

  async function runLoadingStates(totalMs = 20_000): Promise<void> {
    if (isLoadingStatesRunning) return;

    isLoadingStatesRunning = true;

    const loadingSequence = createRandomLoadingSequence();
    const stageDurations = createRandomStageDurations(totalMs);

    for (let i = 0; i < loadingSequence.length; i++) {
      loadStateText = loadingSequence[i];
      await wait(stageDurations[i]);
    }

    isLoadingStatesRunning = false;
  }
</script>

<!-- Page Wrapper -->
<main class="page">
  <!-- Head -->
  <header class="head">
    <eyebrow>Quick fill</eyebrow>
    <h2 class="">Turn notes into a session</h2>
    <subtitle class="">
      This feature allows you to paste or upload your workout session notes in any
      format-messy,structured, or somewhere in between, and automatically transforms them into a
      clean, standardized <b>reslince</b> session.
      <br /><br />
      No need to worry about how your notes are written. Whether you list exercises, weights, and reps
      on one line or break them into multiple lines, the AI intelligently detects the details and organizes
      everything into a structured format for you.
      <br /><br />
      After the generation is finished, you will get the chance to readjust and edit the session before
      saving!
    </subtitle>
  </header>

  <div
    style="
    width:100%;
    height:2px;
    background:#ffffff;
    margin:8px 0;
  "
  ></div>
  <!-- Input Area -->

  <section class="input-area" aria-labelledby="input-area-label">
    <label for="primary-textarea" class="field-label"> Paste workout session </label>

    <Textarea
      placeholder={'e.g: \nBench press (70kg): reps (7,7,6) etc.'}
      oninput={(e) => (userText = (e.target as HTMLTextAreaElement).value)}
      id="primary-textarea"
      maxlength={400}
    />

    <p id="primary-help" class="help-text">
      The inputted text does not have to adhere to any sort of format.
    </p>
  </section>

  <!-- Additional Info Area -->
  <section
    class="additional-info"
    aria-labelledby="additional-info-label"
    aria-live="polite"
    data-enabled={isAdditionalInfoEnabled ? 'true' : 'false'}
  >
    <Collapsible.Root bind:open={advancedOpen}>
      <Collapsible.Trigger>
        <p id="additional-info-label" class="section-heading">
          Advanced settings
          {#if advancedOpen}
            <Icon icon="si:expand-less-fill" />
          {:else}
            <Icon icon="si:expand-more-fill" />
          {/if}
        </p>
      </Collapsible.Trigger>
      <Collapsible.Content>
        <label class="flex items-center gap-2">
          <Checkbox bind:checked={cbSpellcheck} />
          <span>Spellcheck exercises</span>
        </label>

        <label class="flex items-center gap-2">
          <Checkbox bind:checked={cbRestart} />
          <span>Restart with lighter load</span>
        </label>

        <label class="flex items-center gap-2">
          <Checkbox bind:checked={cbAutoChoose} />
          <span>Auto-choose reps and weights</span>
        </label>

        <label class="flex items-center gap-2">
          <Checkbox bind:checked={cbStrength} />
          <span>Prioritize strength for compound lifts</span>
        </label>
      </Collapsible.Content>
    </Collapsible.Root>

    <div class="additional-info-content"></div>
  </section>

  <!-- Continue -->
  <footer class="footer">
    <Dialog.Root open={collapsibleOpen}>
      <button type="button" class="continue-button" onclick={handleContinue}>
        Continue
        <Icon icon="fluent:sparkle-16-filled" width={20} class="ml-2" />
      </button>
      <Dialog.Content class="border-border bg-card box-shadow-shadow">
        <Dialog.Header>
          <Dialog.Title>
            {#await dataPromise}
              <div class="loading-text-wrap">
                {#key loadStateText}
                  <p
                    class="loading-text"
                    in:fly={{ y: 6, duration: 380, opacity: 0 }}
                    out:fly={{ y: -6, duration: 340, opacity: 0 }}
                  >
                    {loadStateText}...
                  </p>
                {/key}
              </div>
            {:then paragraphs}
              {#if paragraphs == 'limit'}
                AI usage limit reached!
              {:else if paragraphs == 'fail'}
                Generation failed!
              {:else}
                Session created!
              {/if}
            {/await}
          </Dialog.Title>
          <Dialog.Description>
            {#await dataPromise}
              <Icon icon="svg-spinners:ring-resize" height={50} class="m-4"></Icon>
              Hang on as our AI tool analyzes your session...
            {:then result}
              {#if result == 'fail'}
                <ul>
                  <p>{'Try again later.'}</p>
                </ul>
              {:else if result == 'limit'}
                <ul>
                  <p>{'Wait for token count to replenish.'}</p>
                </ul>
              {:else}
                <div transition:fade={{ delay: 200, duration: 200 }}>
                  <p>Your session is ready to review and edit before you save it.</p>
                </div>

                <button
                  type="button"
                  class="continue-button mt-6"
                  transition:fade={{ delay: 400, duration: 200 }}
                  onclick={saveSession}
                >
                  Review generated session!
                </button>
              {/if}
            {/await}
          </Dialog.Description>
        </Dialog.Header>
      </Dialog.Content>
    </Dialog.Root>
  </footer>
</main>

<style>
  /* Layout — vertical, comfortable spacing for phone-sized screens */
  :root {
    --page-max-width: 720px;
    --accent-contrast: #ffffff;
    --panel: #f9fafb;
    --panel-strong: #f3f4f6;
  }

  *,
  *::before,
  *::after {
    box-sizing: border-box;
  }

  .page {
    margin-top: 3rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: var(--spacing-md);
    gap: var(--spacing-lg);
    overflow: scroll;
  }

  .head,
  .input-area,
  .additional-info,
  .footer {
    width: 100%;
    max-width: var(--page-max-width);
  }

  .loading-text-wrap {
    position: relative;
    height: 1.5rem; /* match one line of text */
    overflow: hidden;
  }

  .loading-text {
    position: absolute;
    inset: 0;
    margin: 0;
  }

  .section-heading {
    display: inline-flex;
    align-items: center;
    justify-content: space-between;
    width: 10rem;
    margin: 0 0 var(--spacing-sm) 0;
    font-weight: 600;
  }

  .field-label {
    display: block;
    margin-bottom: var(--spacing-xs);
    margin-top: var(--spacing-sm);
    font-weight: 600;
  }

  .help-text {
    margin: var(--spacing-sm) 0 0 0;
    color: var(--muted-foreground);
    font-size: 0.9rem;
  }

  .additional-info {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: var(--spacing-md);
    display: grid;
  }

  .additional-info-content {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: var(--spacing-sm);
  }

  .footer {
    display: flex;
    justify-content: flex-end;
  }

  .continue-button {
    display: flex;
    flex-direction: row;
    align-items: center;
    padding: 0.7rem 1.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--color-secondary);
    color: var(--accent-foreground);
    font-weight: 600;
    cursor: pointer;
    box-shadow: var(--shadow-dark);
  }

  .continue-button:focus-visible {
    outline: 4px solid color-mix(in oklab, var(--accent) 25%, transparent);
    outline-offset: 2px;
  }
</style>
