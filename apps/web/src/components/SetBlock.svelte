<script lang="ts">
  import { onMount } from 'svelte';
  import '../app.css';
  import Icon from '@iconify/svelte';

  let {
    id,
    reps,
    finished,
    onCountChange,
  }: {
    id: number;
    reps: number;
    finished?: boolean;
    onCountChange?: (detail: { id: number; count: number }) => void;
  } = $props();

  let curCount = $state(reps);
  const ogCount = reps;

  onMount(() => {
    onCountChange?.({ id, count: curCount });
  });

  function decrement() {
    popScaleWeight();
    curCount = Math.max(0, curCount - 1);
    checksRep();
    onCountChange?.({ id, count: curCount });
  }

  function increment() {
    popScaleWeight();
    curCount += 1;
    checksRep();
    onCountChange?.({ id, count: curCount });
  }

  function checksRep() {
    console.log('Checked rep');
    if (!checked) {
      checked = true;
    }
  }

  let checked: boolean = $state(false);

  function getActive(check: boolean): string {
    if (check) {
      return 'active';
    }
    return 'inactive';
  }

  let weightPopped = $state(false);
  function popScaleWeight() {
    weightPopped = true;
    setTimeout(() => {
      weightPopped = false;
    }, 150);
  }

  function statusText(): string {
    if (checked) {
      return 'Completed';
    }
    return 'Set goal';
  }
</script>

<div class="counter-container set--{getActive(checked)} finished-{finished}">
  <div class="check {checked}"><Icon icon="gg:check-o" /></div>
  <div class="set-card-left" class:complete={checked}>
    <div class="set-count">{id}</div>
    <div class="set-description">
      <p class="set-status">{statusText()}</p>
      <p class="set-goal">{ogCount} <span> reps</span></p>
    </div>
  </div>

  <div class="controls">
    <button class="stepper" onclick={decrement}>
      <Icon icon="ic:sharp-minus"></Icon>
    </button>
    <button class="stepper-count" class:stepper-scale={weightPopped} onclick={checksRep}
      >{curCount}</button
    >
    <button class="stepper" onclick={increment}>
      <Icon icon="ic:sharp-plus"></Icon>
    </button>
  </div>
</div>

<style>
  .check {
    position: absolute;
    right: 2.5rem;
    transition: all 0.3s;
    opacity: 0;
  }

  .check.false {
    opacity: 0;
  }

  .counter-container {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: var(--spacing-md);
    background: var(--surface-low);
    width: 100%;
    border-radius: 10px;
    box-shadow: var(--shadow-dark);
  }

  .counter-container.set--active {
  }

  .finished-true {
    pointer-events: none;
    opacity: 0.5;
  }

  .set-count {
    background: var(--surface-middle);
    padding: 1rem;
    width: 2.5rem;
    height: 2.5rem;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    transition: all ease 300ms;
  }

  .set--active .set-count {
    background: #3c513b;
    border: 1px solid rgba(133, 255, 103, 0.441);
    color: rgb(117, 255, 117);
  }

  .finished-true .set-count {
    background: #2b322b;
    border: 1px solid rgba(133, 255, 103, 0.441);
    color: rgb(160, 255, 160);
  }

  .set-card-left {
    display: flex;
    align-items: center;
    gap: 1rem;
    min-width: 0;
    color: var(--color-text);
  }

  .set-card-left.complete {
    color: #838782;
  }

  .set-description {
    display: flex;
    flex-direction: column;
    justify-content: baseline;
  }

  .set-status {
    margin: 0 0 0.125rem;
    font-size: 0.625rem;
    letter-spacing: -0.01em;
    text-transform: uppercase;
    text-align: left;
    color: inherit;
  }

  .set-goal {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 700;
    white-space: nowrap;
    text-align: left;
  }

  .set-goal span {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-muted);
  }

  .complete .set-goal span {
    font-size: 0.875rem;
    font-weight: 500;
    color: inherit;
  }

  .stepper {
    height: 2.5rem;
    width: 2.5rem;

    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--surface-middle);
    color: var(--color-text);
    transition:
      scale 80ms cubic-bezier(0.175, 0.885, 0.32, 1.275),
      filter ease 100ms;
    filter: brightness(1);
  }

  .stepper:active {
    filter: brightness(0.9);
    transform: scale(0.9);
  }

  .stepper-count {
    transition: scale ease-out 250ms;
    width: 3rem;
    text-align: center;
    font-size: 1.25rem;
    font-weight: 700;
    scale: 1;
  }

  .stepper-scale {
    scale: 1.1;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex: 0 0 auto;
  }

  .counter-container.mini {
    background-color: #111111;
    text-align: left;
    padding: 0.5rem 0.5rem;
    border-radius: 0;
    margin: 0;

    font-size: 16px;
    font-weight: 500;
    pointer-events: none;
    box-shadow: none;
    margin: auto;

    width: 100%;
    max-width: 280px;
  }

  .controls.mini {
    font-size: 16px;
  }

  .mini-count {
    font-size: 1rem;
  }

  button {
    padding: 0.5rem 1rem;
  }

  span {
    font-size: 2rem;
    width: 3rem;
    text-align: center;
  }

  .disabled {
    pointer-events: none;
    opacity: 0.5;
  }
</style>
