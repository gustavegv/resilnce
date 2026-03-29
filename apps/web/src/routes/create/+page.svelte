<script lang="ts">
  import Icon from '@iconify/svelte';
  import { base } from '$app/paths';

  type Mode = {
    id: string;
    title: string;
    description: string;
    tag: string;
    icon: string;
    href: string;
  };

  type CarouselTuning = {
    swipeThresholdPx: number;
    clickSuppressionThresholdPx: number;
    dragMultiplier: number;
    maxMotionDelta: number;
    spreadPercent: number;
    curveTension: number;
    rotateYDeg: number;
    scaleDrop: number;
    opacityDrop: number;
    blurPx: number;
    transitionTransformMs: number;
    transitionFadeMs: number;
    transitionEase: string;

    // new anti-overlap controls
    activeLiftPx: number;
    sidePushBackPx: number;
    depthFalloff: number;
    sideSpreadBoost: number;
  };

  const modes = [
    {
      id: 'manual',
      title: 'Manual mode',
      description:
        'Start from scratch and enter everything yourself. Full control, perfect for creating new sessions.',
      tag: 'Create',
      icon: 'mdi:pencil-outline',
      href: `${base}/create/manual`,
    },
    {
      id: 'quick',
      title: 'Quick fill mode',
      description:
        'Import a workout from your notes and let our AI tool get you set up in seconds.',
      tag: 'Import',
      icon: 'mdi:wand',
      href: `${base}/create/quick`,
    },
    {
      id: 'program',
      title: 'Start a program',
      description:
        'Start from our collection of our premade programs, everything from powerlifting to bodybuilding.',
      tag: 'Program',
      icon: 'akar-icons:book',
      href: `${base}/create/program`,
    },
  ] satisfies Mode[];

  /**
   * Central tuning panel for swipe feel + resting card stack appearance.
   *
   * Tweak these values to change behavior:
   * - swipeThresholdPx: how far the user must drag to commit to the next/previous card
   * - dragMultiplier: how directly the cards follow the finger
   * - spreadPercent: horizontal fan-out of resting side cards
   * - curveTension: shape of the horizontal curve
   * - rotateYDeg: how angled the side cards are
   * - scaleDrop: how much smaller non-active cards are
   * - opacityDrop: how much dimmer non-active cards are
   * - blurPx: how blurred non-active cards are
   * - maxMotionDelta: cap for how extreme off-center cards can get
   * - transition* values: settle / snap animation after release
   */
  const carouselPresets = {
    default: {
      swipeThresholdPx: 72,
      clickSuppressionThresholdPx: 10,
      dragMultiplier: 1,
      maxMotionDelta: 1.5,
      spreadPercent: 98,
      curveTension: 0.95,
      rotateYDeg: 38,
      scaleDrop: 0.15,
      opacityDrop: 0.42,
      blurPx: 1.7,
      transitionTransformMs: 620,
      transitionFadeMs: 280,
      transitionEase: 'cubic-bezier(.22,1,.36,1)',
      activeLiftPx: 0,
      sidePushBackPx: 0,
      depthFalloff: 0,
      sideSpreadBoost: 8,
    },
  } satisfies Record<string, CarouselTuning>;

  let carouselTuning: CarouselTuning = { ...carouselPresets.default };

  let carousel: HTMLDivElement;
  let activeIndex = $state(0);
  let dragOffset = $state(0);
  let containerWidth = $state(1);
  let isDragging = $state(false);

  let startX = 0;
  let activePointerId: number | null = null;
  let suppressClick = false;

  function clamp(value: number, min: number, max: number): number {
    return Math.min(max, Math.max(min, value));
  }

  function wrapIndex(index: number): number {
    return (index + modes.length) % modes.length;
  }

  function shortestDelta(index: number): number {
    let delta = index - activeIndex;

    if (delta > modes.length / 2) delta -= modes.length;
    if (delta < -modes.length / 2) delta += modes.length;

    return delta;
  }

  function cardStyle(index: number): string {
    const delta = shortestDelta(index);
    const dragProgress = (dragOffset / Math.max(containerWidth, 1)) * carouselTuning.dragMultiplier;

    const motionDelta = delta + dragProgress;
    const limited = clamp(
      motionDelta,
      -carouselTuning.maxMotionDelta,
      carouselTuning.maxMotionDelta
    );
    const abs = Math.abs(limited);

    // Keep side cards farther apart horizontally near crossover
    const spread = carouselTuning.spreadPercent + abs * carouselTuning.sideSpreadBoost;

    const x = Math.sin(limited * carouselTuning.curveTension) * spread;
    const rotateY = limited * -carouselTuning.rotateYDeg;
    const scale = 1 - abs * carouselTuning.scaleDrop;
    const opacity = 1 - abs * carouselTuning.opacityDrop;
    const blur = abs * carouselTuning.blurPx;

    // New: true depth separation
    const depthProgress = Math.pow(1 - Math.min(abs, 1), carouselTuning.depthFalloff);
    const translateZ =
      depthProgress * carouselTuning.activeLiftPx - abs * carouselTuning.sidePushBackPx;

    // Keep cards ordered consistently while still favoring the centered one
    const z = Math.max(1, 1000 - Math.round(abs * 100));

    const transition = isDragging
      ? 'none'
      : [
          `transform ${carouselTuning.transitionTransformMs}ms ${carouselTuning.transitionEase}`,
          `opacity ${carouselTuning.transitionFadeMs}ms ease`,
          `filter ${carouselTuning.transitionFadeMs}ms ease`,
        ].join(', ');

    return `
		--card-translate-x: ${x.toFixed(3)}%;
		--card-translate-z: ${translateZ.toFixed(3)}px;
		--card-scale: ${scale.toFixed(3)};
		--card-rotate-y: ${rotateY.toFixed(3)}deg;
		--card-opacity: ${opacity.toFixed(3)};
		--card-z: ${z};
		--card-blur: ${blur.toFixed(3)}px;
		--card-transition: ${transition};
	`;
  }

  function resetDragState(): void {
    dragOffset = 0;
    isDragging = false;
    activePointerId = null;
    suppressClick = false;
  }

  function goTo(index: number): void {
    activeIndex = wrapIndex(index);
    resetDragState();
  }

  function next(): void {
    goTo(activeIndex + 1);
  }

  function previous(): void {
    goTo(activeIndex - 1);
  }

  function finishDrag(commit: boolean): void {
    if (!isDragging) return;

    if (commit) {
      if (dragOffset <= -carouselTuning.swipeThresholdPx) {
        activeIndex = wrapIndex(activeIndex + 1);
      } else if (dragOffset >= carouselTuning.swipeThresholdPx) {
        activeIndex = wrapIndex(activeIndex - 1);
      }
    }

    resetDragState();
  }

  function releasePointerCapture(pointerId: number): void {
    if (!carousel) return;

    try {
      if (carousel.hasPointerCapture(pointerId)) {
        carousel.releasePointerCapture(pointerId);
      }
    } catch {
      // capture may already be gone
    }
  }

  function handlePointerDown(event: PointerEvent): void {
    if (modes.length < 2 || !event.isPrimary) return;
    if (event.pointerType === 'mouse' && event.button !== 0) return;

    const target = event.target as HTMLElement | null;
    if (target?.closest('.session-carousel__nav, .session-carousel__dot')) return;

    activePointerId = event.pointerId;
    startX = event.clientX;
    dragOffset = 0;
    isDragging = true;
    suppressClick = false;

    carousel?.setPointerCapture(event.pointerId);
  }

  function handlePointerMove(event: PointerEvent): void {
    if (!isDragging || event.pointerId !== activePointerId) return;

    dragOffset = clamp(event.clientX - startX, -containerWidth * 1.25, containerWidth * 1.25);

    if (Math.abs(dragOffset) > carouselTuning.clickSuppressionThresholdPx) {
      suppressClick = true;
    }
  }

  function handlePointerUp(event: PointerEvent): void {
    if (event.pointerId !== activePointerId) return;
    finishDrag(true);
    releasePointerCapture(event.pointerId);
  }

  function handlePointerCancel(event: PointerEvent): void {
    if (event.pointerId !== activePointerId) return;
    finishDrag(false);
    releasePointerCapture(event.pointerId);
  }

  function handleLostPointerCapture(event: PointerEvent): void {
    if (event.pointerId !== activePointerId) return;
    finishDrag(false);
  }

  function handleCardClick(event: MouseEvent, index: number): void {
    if (suppressClick) {
      event.preventDefault();
      suppressClick = false;
      return;
    }

    if (index !== activeIndex) {
      event.preventDefault();
      goTo(index);
    }
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (modes.length < 2) return;

    switch (event.key) {
      case 'ArrowLeft':
        event.preventDefault();
        previous();
        break;
      case 'ArrowRight':
        event.preventDefault();
        next();
        break;
      case 'Home':
        event.preventDefault();
        goTo(0);
        break;
      case 'End':
        event.preventDefault();
        goTo(modes.length - 1);
        break;
    }
  }
</script>

<div class="page">
  <div class="">
    <eyebrow>Create session</eyebrow>
    <h2 class="mt-1">Pick a creation mode</h2>
    <subtitle class="mt-3">Choose how you want to build your session.</subtitle>

    <div class="helper-card">
      <div class="helper-card__row">
        <div class="helper-card__icon">
          <Icon icon="mdi:star-four-points" />
        </div>

        <div>
          <p class="helper-card__title">You can edit everything later</p>
          <p class="helper-card__text">
            Regardless of how you start, you can always edit a session later.
          </p>
        </div>
      </div>
    </div>
  </div>

  <div
    class="session-carousel"
    bind:this={carousel}
    bind:clientWidth={containerWidth}
    role="region"
    aria-roledescription="carousel"
    aria-label="Choose how you want to create a session"
    data-dragging={isDragging ? 'true' : 'false'}
    onpointerdown={handlePointerDown}
    onpointermove={handlePointerMove}
    onpointerup={handlePointerUp}
    onpointercancel={handlePointerCancel}
    onlostpointercapture={handleLostPointerCapture}
    onkeydown={handleKeydown}
  >
    <p class="sr-only" aria-live="polite">
      {activeIndex + 1} of {modes.length}: {modes[activeIndex].title}
    </p>

    <div class="session-carousel__track">
      {#each modes as mode, index (mode.id)}
        <a
          class="session-card"
          class:session-card--active={index === activeIndex}
          href={mode.href}
          style={cardStyle(index)}
          tabindex={index === activeIndex ? 0 : -1}
          aria-hidden={index !== activeIndex ? 'true' : undefined}
          draggable="false"
          onclick={(event) => handleCardClick(event, index)}
          ondragstart={(event) => event.preventDefault()}
        >
          <div class="session-card__top">
            <div class="session-card__icon-wrap" aria-hidden="true">
              <Icon icon={mode.icon} class="session-card__icon" />
            </div>

            <span class="session-card__tag">{mode.tag}</span>
          </div>

          <div class="session-card__content">
            <h2 class="session-card__title">{mode.title}</h2>
            <p class="session-card__description">{mode.description}</p>
          </div>

          <div class="session-card__footer">
            <span class="session-card__cta">Get started</span>
            <span class="session-card__arrow" aria-hidden="true">→</span>
          </div>
        </a>
      {/each}
    </div>

    <button
      class="session-carousel__nav session-carousel__nav--left"
      type="button"
      aria-label="Previous card"
      disabled={modes.length < 2}
      onclick={previous}
    >
      ‹
    </button>

    <button
      class="session-carousel__nav session-carousel__nav--right"
      type="button"
      aria-label="Next card"
      disabled={modes.length < 2}
      onclick={next}
    >
      ›
    </button>

    <div class="session-carousel__dots" aria-label="Carousel position">
      {#each modes as mode, index (mode.id)}
        <button
          class="session-carousel__dot"
          class:session-carousel__dot--active={index === activeIndex}
          type="button"
          aria-label={`Go to ${mode.title}`}
          aria-current={index === activeIndex ? 'true' : undefined}
          onclick={() => goTo(index)}
        ></button>
      {/each}
    </div>
  </div>
</div>

<style>
  :global(:root) {
    --session-white-rgb: 255 255 255;
    --session-black-rgb: 0 0 0;
    --session-accent-rgb: 74 152 75;

    --session-text-primary: var(--color-text);
    --session-text-secondary: var(--text-muted);
    --session-text-strong: var(--color-text);

    --session-surface-top: var(--surface-low);
    --session-surface-bottom: var(--color-background);

    --session-white-strong: rgb(var(--session-white-rgb) / 0.16);
    --session-white-medium: rgb(var(--session-white-rgb) / 0.1);
    --session-white-soft: rgb(var(--session-white-rgb) / 0.08);
    --session-white-faint: rgb(var(--session-white-rgb) / 0.06);
    --session-white-muted: rgb(var(--session-white-rgb) / 0.04);
    --session-white-pill: rgb(var(--session-white-rgb) / 0.9);
    --session-white-dot: rgb(var(--session-white-rgb) / 0.22);

    --session-accent-glow: rgb(var(--session-accent-rgb) / 0.05);
    --session-focus-ring: rgb(var(--session-accent-rgb) / 0.9);

    --session-shadow-soft: rgb(var(--session-black-rgb) / 0.42);
    --session-shadow-strong: rgb(var(--session-black-rgb) / 0.52);
  }

  .page {
    min-height: 100svh;
    background-color: var(--color-background);
    display: flex;
    flex-direction: column;
    padding: 0rem 1rem;
    padding-top: 4rem;
  }

  .helper-card {
    margin-top: 1rem;
    padding: 1rem;
    border-radius: 1.5rem;
    border: 1px solid rgb(74 152 75 / 0.15);
    background: rgb(74 152 75 / 0.1);
  }

  .helper-card__row {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .helper-card__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 2.25rem;
    height: 2.25rem;
    margin-top: 0.125rem;
    border-radius: 1rem;
    background: rgb(74 152 75 / 0.15);
    color: rgb(220 252 231);
  }

  .helper-card__title {
    font-size: 0.875rem;
    font-weight: 600;
    color: rgb(240 253 244);
  }

  .helper-card__text {
    margin-top: 0.25rem;
    font-size: 0.875rem;
    line-height: 1.5rem;
    color: rgb(240 253 244 / 0.75);
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }

  .session-carousel {
    position: relative;
    height: 62svh;
    perspective: 1200px;
    touch-action: pan-y;
    user-select: none;
    -webkit-tap-highlight-color: transparent;
  }

  .session-carousel[data-dragging='true'] {
    cursor: grabbing;
  }

  .session-carousel__track {
    position: relative;
    height: 100%;
    transform-style: preserve-3d;
    isolation: isolate;
  }

  .session-card {
    position: absolute;
    inset: 0;
    margin: auto;
    display: grid;
    border-radius: 1rem;

    width: min(100%, 21.5rem);
    height: min(100%, 30rem);
    padding: 1.15rem;

    grid-template-rows: auto 1fr auto;
    gap: 1.25rem;
    text-decoration: none;
    color: var(--session-text-primary);
    background:
      linear-gradient(180deg, var(--session-white-strong), var(--session-white-muted)),
      linear-gradient(180deg, var(--session-surface-top), var(--session-surface-bottom));
    border: 1px solid var(--session-white-soft);
    box-shadow:
      0 24px 80px var(--session-shadow-soft),
      inset 0 1px 0 var(--session-white-medium);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    transform-origin: center center;
    transform: translateX(var(--card-translate-x, 0%)) translateZ(var(--card-translate-z, 0px))
      scale(var(--card-scale, 1)) rotateY(var(--card-rotate-y, 0deg));
    opacity: var(--card-opacity, 1);
    z-index: var(--card-z, 1);
    filter: blur(var(--card-blur, 0px));
    transition: var(--card-transition, none);
    will-change: transform, opacity, filter;
    pointer-events: none;
    cursor: grab;
    outline: none;
  }

  .session-card::before {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    background:
      radial-gradient(circle at top left, var(--session-white-strong), transparent 32%),
      radial-gradient(circle at bottom right, var(--session-accent-glow), transparent 28%);
    pointer-events: none;
  }

  .session-card--active {
    pointer-events: auto;
    box-shadow:
      0 34px 100px var(--session-shadow-strong),
      0 0 0 1px var(--session-white-faint),
      inset 0 1px 0 var(--session-white-medium);
  }

  .session-card--active:focus-visible,
  .session-carousel__nav:focus-visible,
  .session-carousel__dot:focus-visible {
    outline: 2px solid var(--session-focus-ring);
    outline-offset: 3px;
  }

  .session-card__top,
  .session-card__content,
  .session-card__footer {
    position: relative;
    z-index: 1;
  }

  .session-card__top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.9rem;
  }

  .session-card__icon-wrap {
    width: 3.4rem;
    height: 3.4rem;
    border-radius: 1rem;
    display: grid;
    place-items: center;
    background: var(--session-white-soft);
    border: 1px solid var(--session-white-soft);
    box-shadow: inset 0 1px 0 var(--session-white-soft);
  }

  .session-card__icon {
    width: 1.5rem;
    height: 1.5rem;
    color: var(--session-text-primary);
  }

  .session-card__tag {
    padding: 0.45rem 0.75rem;
    border-radius: 999px;
    font-size: 0.78rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    color: var(--session-white-pill);
    background: var(--session-white-soft);
    border: 1px solid var(--session-white-faint);
  }

  .session-card__content {
    display: grid;
    align-content: center;
    gap: 0.85rem;
    padding: 0.25rem 0.15rem;
  }

  .session-card__title {
    margin: 0;
    font-size: 1.8rem;
    line-height: 1;
    letter-spacing: -0.05em;
  }

  .session-card__description {
    margin: 0;
    max-width: 22rem;
    font-size: 0.95rem;
    line-height: 1.6;
    color: var(--session-text-secondary);
  }

  .session-card__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 0.2rem;
  }

  .session-card__cta {
    font-size: 0.95rem;
    font-weight: 700;
    color: var(--session-text-strong);
  }

  .session-card__arrow {
    width: 2.5rem;
    height: 2.5rem;
    display: grid;
    place-items: center;
    border-radius: 999px;
    background: var(--session-white-soft);
    border: 1px solid var(--session-white-soft);
    font-size: 1.1rem;
  }

  .session-carousel__nav {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    z-index: 60;
    width: 2.8rem;
    height: 2.8rem;
    border: 1px solid var(--session-white-soft);
    border-radius: 999px;
    background: var(--session-white-soft);
    color: var(--session-text-primary);
    font-size: 1.7rem;
    line-height: 1;
    cursor: pointer;
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    transition:
      transform 180ms ease,
      background-color 180ms ease,
      border-color 180ms ease,
      opacity 180ms ease;
    display: none;
  }

  .session-carousel__nav:hover:not(:disabled) {
    background: var(--session-white-medium);
  }

  .session-carousel__nav:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .session-carousel__nav--left {
    left: max(0.15rem, calc(50% - 15rem));
  }

  .session-carousel__nav--right {
    right: max(0.15rem, calc(50% - 15rem));
  }

  .session-carousel__dots {
    display: flex;
    justify-content: center;
    gap: 0.55rem;
  }

  .session-carousel__dot {
    width: 0.6rem;
    height: 0.6rem;
    padding: 0;
    border: 0;
    border-radius: 999px;
    background: var(--session-white-dot);
    cursor: pointer;
    transition:
      transform 180ms ease,
      background-color 180ms ease,
      width 180ms ease;
  }

  .session-carousel__dot--active {
    width: 1.6rem;
    background: var(--session-white-pill);
  }

  @media (prefers-reduced-motion: reduce) {
    .session-card {
      transform: translateX(var(--card-translate-x, 0%)) scale(var(--card-scale, 1));
      filter: none !important;
      transition:
        opacity 160ms ease,
        transform 160ms ease;
    }

    .session-carousel__nav,
    .session-carousel__dot {
      transition: none;
    }
  }

  @media (max-width: 400px) {
    .session-carousel {
      scale: 0.9;
    }

    .page {
      padding-top: 3rem;
    }

    .session-carousel__dots {
      margin-top: 1rem;
    }
  }
</style>
