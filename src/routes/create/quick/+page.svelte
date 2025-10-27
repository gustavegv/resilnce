<script lang="ts">
  import { Textarea } from '$lib/components/ui/textarea/index.js';
  import * as Collapsible from '$lib/components/ui/collapsible/index.js';
  import { Checkbox } from '$lib/components/ui/checkbox/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { goto } from '$app/navigation';
  import Icon from '@iconify/svelte';
  import { toast } from 'svelte-sonner';
  import { promptAutoChoose, promptRestart, promptSpell, promptStandard } from './prompt';
  import { base } from '$app/paths';

  var sessionTitle = $state<string>('');
  var userText = $state<string>('');
  var isAdditionalInfoEnabled = $state<boolean>(false);

  let cbSpellcheck = $state(false);
  let cbRestart = $state(false);
  let cbAutoChoose = $state(false);

  let collapsibleOpen = $state(false);

  let dataPromise = $state<Promise<any> | null>(null);

  function getCheckboxPrompt(): string {
    if (!cbSpellcheck && !cbRestart && !cbAutoChoose) {
      return '';
    }
    const advancedPrompt = `
        {
        ${cbSpellcheck ? promptSpell : ''} \n
        ${cbRestart ? promptRestart : ''} \n
        ${cbAutoChoose ? promptAutoChoose : ''} \n
        }
        `;
    return advancedPrompt;
  }

  function getDataPrompt(): string {
    const userData = `
      # HERE IS THE DATA YOU WILL PROCESS AND TRANSFORM: 
      { 
      session name: '${sessionTitle}'; 
      unformatted text workout session input: '${userText}';
      }
    `;
    if (userData.length > 500) {
      return '### USER INPUT INVALID. RETURN AN ERROR';
    }
    return userData;
  }

  function delay(ms: number) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  function quickfillAPICall(prompt: string) {
    dataPromise = (async () => {
      await delay(300); // ← simulate slow API
      const res = await fetch('https://baconipsum.com/api/?type=meat-and-filler&paras=3');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.json();
    })();
  }

  async function sendValues() {
    const userInput = getDataPrompt();
    const advancedPrompt = getCheckboxPrompt();
    const finalPrompt = promptStandard + '\n' + advancedPrompt + '\n' + userInput;
    console.log(finalPrompt);

    quickfillAPICall(finalPrompt);
  }

  function validateForm(): true | string {
    if (userText.trim().length < 10) return 'Please enter at least 10 characters of data.';
    if (sessionTitle.trim().length < 2) return 'Title must be at least 2 characters.';
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
      sendValues();
    }
  }

  function saveSession() {
    goto(`${base}/?sonner=saved`);
  }
</script>

<!-- Page Wrapper -->
<main class="page">
  <!-- Head -->
  <header class="head">
    <h1 class="title">Quick Fill</h1>
    <p class="description">
      This feature allows you to paste or upload your workout session notes in any
      format—messy,structured, or somewhere in between—and automatically transforms them into a
      clean, standardized <b>reslince</b> session.
      <br /><br />
      No need to worry about how your notes are written. Whether you list exercises, weights, and reps
      on one line or break them into multiple lines, the AI intelligently detects the details and organizes
      everything into a structured format for you.
    </p>
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
    <label for="title-textarea" class="field-label"> Session title </label>

    <Textarea
      placeholder="e.g Push Day"
      oninput={(e) => (sessionTitle = (e.target as HTMLTextAreaElement).value)}
      id="title-textarea"
      maxlength={50}
    />

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
    <Collapsible.Root>
      <Collapsible.Trigger>
        <p id="additional-info-label" class="section-heading">Click for advanced settings ⚙️</p>
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
      <Dialog.Content>
        <Dialog.Header>
          <Dialog.Title>
            {#await dataPromise}
              Creating session...
            {:then paragraphs}
              Session created!
            {/await}
          </Dialog.Title>
          <Dialog.Description>
            {#await dataPromise}
              <Icon icon="svg-spinners:blocks-shuffle-3" height={50} class="m-4"></Icon>
              Hang on as our AI tool analyzes your session...
            {:then result}
              <ul>
                <p>{result}</p>
              </ul>
              <button type="button" class="continue-button" onclick={saveSession}
                >Save session!</button
              >
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
    margin: 8svh 0 4svh 0;
    padding-top: 400px;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: var(--spacing-md);
    gap: var(--spacing-lg);
  }

  .head,
  .input-area,
  .additional-info,
  .footer {
    width: 100%;
    max-width: var(--page-max-width);
  }

  .title {
    margin: 0 0 var(--spacing-sm) 0;
    font-size: 1.75rem;
    line-height: 1.2;
    font-weight: 700;
  }

  .description {
    margin: 0;
    color: var(--muted-foreground);
    line-height: 1.6;
  }

  .section-heading {
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
    padding: 0.9rem 1.25rem;
    border: none;
    border-radius: var(--radius);
    background: var(--color-secondary);
    color: var(--accent-foreground);
    font-weight: 600;
    cursor: pointer;
  }

  .continue-button:focus-visible {
    outline: 4px solid color-mix(in oklab, var(--accent) 25%, transparent);
    outline-offset: 2px;
  }
</style>
