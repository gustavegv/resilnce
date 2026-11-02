<script lang="ts">
  import { tick } from 'svelte';
  import InputField from '../../components/InputField.svelte';
  import type { ExerciseSelection } from './types';

  let {
    value = $bindable(''),
    expanded = $bindable(false),
    personal = [],
    catalog = [],
    searching = false,
    disabled = false,
    onquery,
    onselect,
  }: {
    value?: string;
    expanded?: boolean;
    personal?: ExerciseSelection[];
    catalog?: string[];
    searching?: boolean;
    disabled?: boolean;
    onquery: (composing: boolean) => void;
    onselect: (selection: ExerciseSelection | string) => void;
  } = $props();

  type Option = { key: string; name: string; personal?: ExerciseSelection; custom?: boolean };
  const id = $props.id();
  let focused = $state(false);
  let atEnd = $state(true);
  let composing = $state(false);
  let active = $state(-1);
  const normalize = (name: string) => name.trim().toLocaleLowerCase();
  let query = $derived(normalize(value));
  let options = $derived.by(() => {
    const unique = new Map<string, Option>();
    for (const selection of personal) {
      const name = selection.exercise.name;
      if (normalize(name).includes(query))
        unique.set(normalize(name), {
          key: `personal-${selection.exercise.id}`,
          name,
          personal: selection,
        });
    }
    for (const name of catalog) {
      const normalized = normalize(name);
      if (normalized.includes(query) && !unique.has(normalized))
        unique.set(normalized, { key: `suggestion-${encodeURIComponent(normalized)}`, name });
    }
    const rank = (option: Option) =>
      normalize(option.name) === query ? 0 : normalize(option.name).startsWith(query) ? 1 : 2;
    const matching = [...unique.values()].sort(
      (a, b) => rank(a) - rank(b) || Number(!a.personal) - Number(!b.personal)
    );
    if (
      query &&
      !searching &&
      !personal.some((selection) => normalize(selection.exercise.name) === query)
    )
      matching.push({ key: 'custom', name: value.trim(), custom: true });
    return matching;
  });
  let visible = $derived(expanded && !disabled && !composing && options.length > 0);
  let candidate = $derived(
    active >= 0 ? options[active] : options.find((option) => !option.custom)
  );
  let completion = $derived(
    focused &&
      visible &&
      atEnd &&
      !searching &&
      query &&
      candidate &&
      !candidate.custom &&
      value === value.trim() &&
      normalize(candidate.name).startsWith(query)
      ? candidate.name.slice(value.length)
      : ''
  );
  function caret(input: HTMLInputElement) {
    atEnd =
      input.selectionStart === value.length &&
      input.selectionEnd === value.length &&
      input.scrollLeft === 0;
  }
  function choose(option: Option) {
    if (disabled || searching || composing) return;
    value = option.name;
    expanded = false;
    active = -1;
    onselect(option.personal ?? option.name);
  }
  function keydown(event: KeyboardEvent & { currentTarget: HTMLInputElement }) {
    if (
      disabled ||
      composing ||
      event.isComposing ||
      event.altKey ||
      event.ctrlKey ||
      event.metaKey
    )
      return;
    caret(event.currentTarget);
    if (event.key === 'Escape') {
      expanded = false;
      active = -1;
      return;
    }
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      if (!options.length) return;
      event.preventDefault();
      expanded = true;
      active =
        event.key === 'ArrowDown'
          ? (active + 1) % options.length
          : active <= 0
            ? options.length - 1
            : active - 1;
      const input = event.currentTarget;
      void tick().then(() => {
        input
          .closest('.autocomplete')
          ?.querySelector<HTMLElement>('[aria-selected="true"]')
          ?.scrollIntoView?.({ block: 'nearest' });
      });
    } else if (event.key === 'Enter' && visible) {
      event.preventDefault();
      if (options[active >= 0 ? active : 0]) choose(options[active >= 0 ? active : 0]);
    } else if (
      ((event.key === 'Tab' && !event.shiftKey) || event.key === 'ArrowRight') &&
      completion &&
      candidate
    ) {
      event.preventDefault();
      choose(candidate);
      event.currentTarget.setSelectionRange(value.length, value.length);
    }
  }
</script>

<div class="autocomplete">
  <InputField
    label="Exercise name"
    aria-label="Exercise name"
    type="text"
    maxlength={100}
    bind:value
    {completion}
    {disabled}
    role="combobox"
    aria-autocomplete="both"
    aria-expanded={visible}
    aria-controls={`${id}-options`}
    aria-activedescendant={visible && active >= 0 && options[active]
      ? `${id}-${options[active].key}`
      : undefined}
    aria-busy={searching}
    autocomplete="off"
    spellcheck="false"
    oninput={(event) => {
      value = event.currentTarget.value;
      active = -1;
      expanded = true;
      caret(event.currentTarget);
      onquery(composing || (event instanceof InputEvent && event.isComposing));
    }}
    onfocus={(event) => {
      focused = true;
      expanded = true;
      caret(event.currentTarget);
      if (!value.trim()) onquery(false);
    }}
    onblur={() => {
      focused = false;
      expanded = false;
      active = -1;
    }}
    onclick={(event) => {
      expanded = true;
      caret(event.currentTarget);
    }}
    onselect={(event) => caret(event.currentTarget)}
    onkeyup={(event) => caret(event.currentTarget)}
    onscroll={(event) => caret(event.currentTarget)}
    onkeydown={keydown}
    oncompositionstart={() => {
      composing = true;
      active = -1;
      onquery(true);
    }}
    oncompositionend={(event) => {
      composing = false;
      value = event.currentTarget.value;
      caret(event.currentTarget);
      onquery(false);
    }}
  />
  {#if visible}
    <ul class="options" id={`${id}-options`} role="listbox" aria-label="Exercise suggestions">
      {#each options as option, index (option.key)}
        <li role="presentation">
          <button
            type="button"
            role="option"
            id={`${id}-${option.key}`}
            data-name={option.custom ? undefined : option.name}
            aria-label={option.custom
              ? `Create new exercise “${option.name}” with Standard`
              : `${option.name}, ${option.personal ? 'your library' : 'suggested name'}`}
            aria-selected={index === active}
            class={['option', { active: index === active, custom: option.custom }]}
            disabled={searching}
            tabindex="-1"
            onpointerdown={(event) => event.preventDefault()}
            onclick={() => choose(option)}
          >
            {#if option.custom}Create new exercise “{option.name}” with Standard
            {:else}<span>{option.name}</span><span
                class="source"
                aria-hidden="true"
                data-source={option.personal ? 'Your library' : 'Suggested name'}
              ></span>{/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .autocomplete {
    position: relative;
    width: 100%;
    max-width: 280px;
  }
  .options {
    position: absolute;
    inset: 100% 0 auto;
    z-index: 50;
    max-height: 16rem;
    overflow-y: auto;
    margin: 0.25rem 0 0;
    padding: 0.25rem;
    list-style: none;
    border: 1px solid var(--border);
    border-radius: var(--border-middle, 8px);
    background: var(--surface-middle);
    box-shadow: var(--shadow-dark);
  }
  .option {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    width: 100%;
    margin: 0;
    padding: 0.75rem;
    background: transparent;
    box-shadow: none;
    border-radius: 4px;
    text-align: left;
    font-size: 14px;
    font-weight: 400;
    color: var(--text);
  }
  .option:hover,
  .option.active {
    background: var(--surface-low);
  }
  .option:disabled {
    opacity: 0.5;
  }
  .source {
    font-size: 10px;
    line-height: 1.3;
    color: var(--text-muted, #888);
    text-align: right;
  }
  .source::after {
    content: attr(data-source);
  }
  .custom {
    color: var(--text-muted, #aaa);
    border-top: 1px solid var(--border);
    font-size: 12px;
  }
</style>
