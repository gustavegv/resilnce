<script lang="ts">
  import { asset } from '$app/paths';
  import '../app.css';
  import { onMount } from 'svelte';
  import { previewLabelConfig, config, carouselSources } from './landscapePreviewConfig';
    import Icon from '@iconify/svelte';

  const currentYear = new Date().getFullYear();
  const githubUrl = 'https://github.com/gustavegv/resilnce';


  const shellLayers = Array.from({ length: config.shellSteps }, (_, step) => {
    const t = step / (config.shellSteps - 1);
    return -9 + t * 18;
  });

  const previewCards = Array.from({ length: config.cardCount }, (_, index) => ({
    index,
    angle: (360 / config.cardCount) * index,
    src: carouselSources[index % carouselSources.length],
    alt: `Carousel card ${index + 1}`,
  }));

  let section!: HTMLElement;
  let landingRoot!: HTMLDivElement;
  let rig!: HTMLDivElement;
  let wheel!: HTMLDivElement;
  let progressBar!: HTMLSpanElement;
  let reduceMotion = false;
  let theme: 'light' | 'dark' = 'dark';
  let scrollContainer: Window | HTMLElement = window;
  let resizeObserver: ResizeObserver | null = null;
  let visiblePreviewLabelCount = previewLabelConfig.length > 0 ? 1 : 0;

  const state = {
    sectionRange: 1,
    targetRotation: 0,
    currentRotation: 0,
    progress: 0,
    rafId: 0,
  };

  function clamp(value: number, min: number, max: number) {
    return Math.max(min, Math.min(max, value));
  }

  function isScrollable(element: HTMLElement) {
    const styles = getComputedStyle(element);
    const overflowY = `${styles.overflow} ${styles.overflowY}`;
    return /(auto|scroll|overlay)/.test(overflowY);
  }

  function getScrollContainer(element: HTMLElement) {
    let parent = element.parentElement;

    while (parent) {
      if (isScrollable(parent) && parent.scrollHeight > parent.clientHeight) {
        return parent;
      }

      parent = parent.parentElement;
    }

    return window;
  }

  function getViewportHeight() {
    return scrollContainer instanceof Window ? window.innerHeight : scrollContainer.clientHeight;
  }

  function getSectionTop() {
    const sectionTop = section.getBoundingClientRect().top;

    if (scrollContainer instanceof Window) {
      return sectionTop;
    }

    return sectionTop - scrollContainer.getBoundingClientRect().top;
  }

  function setRadius() {
    const baseRadius = Math.max(210, Math.min(window.innerWidth * 0.25, 380));
    const radius = Math.min(baseRadius, 400);



    landingRoot.style.setProperty('--radius', `${radius}px`);
    if (window.innerWidth < 1150) {
      const subt = (1 / window.innerWidth) * 10000 * 4
      landingRoot.style.setProperty('--phone-width', `${config.phoneWidth - subt}px`);
      console.log("W:",window.innerWidth,"SB:",subt)
      return
    }
    landingRoot.style.setProperty('--phone-width', `${config.phoneWidth}px`);
  }

  function setTheme(nextTheme: 'light' | 'dark') {
    theme = nextTheme;
    localStorage.setItem('landscape-landing-theme', nextTheme);
  }

  function toggleTheme() {
    setTheme(theme === 'light' ? 'dark' : 'light');
  }

  function startRender() {
    if (!state.rafId) {
      state.rafId = requestAnimationFrame(render);
    }
  }

  function updateTarget() {
    const raw = -getSectionTop() / state.sectionRange;
    state.progress = clamp(raw, 0, 1);
    state.targetRotation = state.progress * 360 * config.scrollTurns;
    visiblePreviewLabelCount = previewLabelConfig.filter(
      ({ appearAtPercent }) => state.progress * 100 >= appearAtPercent
    ).length;

    progressBar?.style.setProperty('--progress', state.progress.toFixed(4));
    startRender();
  }

  function measure() {
    state.sectionRange = Math.max(1, section.offsetHeight - getViewportHeight());

    setRadius();
    updateTarget();
  }

  function render() {
    const delta = state.targetRotation - state.currentRotation;
    const ease = reduceMotion ? 1 : config.smoothing;

    state.currentRotation += delta * ease;

    wheel.style.transform = `rotateY(${state.currentRotation.toFixed(3)}deg)`;

    const orbitDrift = (state.progress - 0.5) * 10;
    rig.style.transform = `
      rotateX(${config.orbitX}deg)
      rotateY(${(config.orbitY + orbitDrift).toFixed(3)}deg)
      rotateZ(${config.orbitZ}deg)
    `;

    if (Math.abs(delta) > 0.02) {
      state.rafId = requestAnimationFrame(render);
      return;
    }

    state.currentRotation = state.targetRotation;
    wheel.style.transform = `rotateY(${state.currentRotation.toFixed(3)}deg)`;
    state.rafId = 0;
  }

  onMount(() => {
    const savedTheme = localStorage.getItem('landscape-landing-theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    theme = "dark";
    reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    scrollContainer = getScrollContainer(section);

    const onScroll = () => updateTarget();

    measure();
    render();

    if (scrollContainer instanceof Window) {
      window.addEventListener('scroll', onScroll, { passive: true });
    } else {
      scrollContainer.addEventListener('scroll', onScroll, { passive: true });
    }

    window.addEventListener('resize', measure);
    resizeObserver = new ResizeObserver(() => measure());
    resizeObserver.observe(section);

    if (!(scrollContainer instanceof Window)) {
      resizeObserver.observe(scrollContainer);
    }

    return () => {
      if (scrollContainer instanceof Window) {
        window.removeEventListener('scroll', onScroll);
      } else {
        scrollContainer.removeEventListener('scroll', onScroll);
      }

      window.removeEventListener('resize', measure);
      resizeObserver?.disconnect();
      resizeObserver = null;

      if (state.rafId) {
        cancelAnimationFrame(state.rafId);
      }
    };
  });
</script>

<svelte:head>
  <title>resilnce</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />

  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
  <link
    href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;700;800;900&display=swap"
    rel="stylesheet"
  />
  <link
    href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:wght,FILL@100..700,0..1&display=swap"
    rel="stylesheet"
  />
</svelte:head>

<div bind:this={landingRoot} class="landscape-landing" data-theme={theme}>
  <header class="site-header">
    <nav class="site-nav container">
      <div class="brand">
        <img src={asset('/FriendsWhite.svg')} alt="Friends holding hands logo" draggable="false" />
      </div>

      <div class="nav-links">
        <a href="#overview">Overview</a>
        <a href="#carousel-section">Preview</a>
        <a href="#usage">Usage</a>
      </div>

      <div class="header-actions">
        <button
          class="theme-toggle"
          type="button"
          aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
          aria-pressed={theme === 'dark'}
          onclick={toggleTheme}
        >
          <span class="material-symbols-outlined" aria-hidden="true">
            {theme === 'light' ? 'dark_mode' : 'light_mode'}
          </span>
        </button>

        <a
          class="button-primary"
          href={githubUrl}
          target="_blank"
          rel="noreferrer"
          aria-label="Open resilnce on GitHub"
        >
          <img src="/GitHub_Invertocat_Black.svg" alt="" />
        </a>
      </div>
    </nav>
  </header>

  <main class="page" id="overview">
    <section class="hero container">
      <div class="hero-layout">
        <div class="hero-copy">
          <h1 class="hero-title">resilnce</h1>
          <p class="hero-subtitle">The only gym tracker you need.</p>

          <div class="hero-note">
            <span class="material-symbols-outlined hero-note-icon">stay_primary_portrait</span>
            <p class="hero-note-text">
              To enter the app, open this site in portrait mode on your mobile device to start
              tracking.
            </p>
          </div>
        </div>

        <div class="hero-visual">
          <div class="mockup-stack" aria-label="App preview mockups">
            <div class="phone-frame phone-right" data-parallax data-speed="0.18" data-tilt="left">
              <img
                src="app_screenshots/preview_workout_screen.png"
                alt="Strength progress charts and analytics dashboard interface"
              />
            </div>

            <div class="phone-frame phone-left" data-parallax data-speed="0.25" data-tilt="right">
              <img
                src="app_screenshots/preview_create_screen.png"
                alt="Quick fill workout plan interface with exercise list"
              />
            </div>

            <div class="phone-frame phone-front" data-parallax data-speed="0.32" data-tilt="left">
              <img
                src="app_screenshots/preview_main_screen.png"
                alt="Active workout tracking screen with weight increase controls"
              />
            </div>
          </div>
        </div>
      </div>
    </section>

    <section bind:this={section} class="carousel-section" id="carousel-section">
      <div class="carousel-pin">
        <div class="scene">
          <div bind:this={rig} class="rig">
            <div class="ground"></div>
            <div class="track"></div>

            <div bind:this={wheel} class="wheel">
              {#each previewCards as card (card.index)}
                <div class="card-mount" style={`--angle:${card.angle}deg`}>
                  <div class="card-anchor">
                    <div class="iphone-frame">
                      {#each shellLayers as z, step (step)}
                        <div class="iphone-shell-layer" style={`--z:${z}px`}></div>
                      {/each}

                      <div class="iphone-back"></div>

                      <div class="iphone-front">
                        <div class="dynamic-island"></div>
                        <div class="top-bar-phone">
                          <div class="status-bar">
                            <div class="flex w-[37%] justify-center items-center h-screen flex-row">
                              <p>09:40</p>

                            </div>
                            <div class="flex w-[40%] justify-end items-center h-screen flex-row gap-1 pr-4">
                              <Icon icon="material-symbols:wifi" width={9}/>
                              <Icon icon="fluent:battery-9-20-regular" width={12}/>
                            </div>

                          </div>
                        </div>
                        <div class="screen-holder">
                          <img
                            class="screen no-top"
                            src={card.src}
                            alt={card.alt}
                            loading="eager"
                            decoding="async"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <div class="hud">
            <div class="hud-eyebrow">See what resilnce can offer</div>
            <strong>Tracking. Programs. Progress.</strong>
            <div class="progress"><span bind:this={progressBar}></span></div>
            <aside class="preview-labels" aria-label="Preview highlights">
              {#each previewLabelConfig as item, index (item.label)}
                <div
                  class="preview-label"
                  class:preview-label--visible={index < visiblePreviewLabelCount}
                >
                  <span class="preview-label__index">{String(index + 1).padStart(2, '0')}</span>
                  <span class="preview-label__text">{item.label}</span>
                </div>
              {/each}
            </aside>
          </div>
        </div>
      </div>
    </section>

    <section id="usage" class="usage">
      <div class="container">
        <div class="usage-header">
          <h2 class="usage-title">Usage</h2>
          <div class="usage-title-line"></div>
        </div>

        <div class="usage-grid">
          <div>
            <div class="usage-panel-heading">
              <span class="material-symbols-outlined usage-panel-icon">smartphone</span>
              <h3 class="usage-panel-title">Mobile 
                <span>(iOS)</span>
              </h3>
            </div>

            <div class="usage-steps">
              <div class="usage-step">
                <div class="usage-step-icon-box">
                  <span class="material-symbols-outlined">explore</span>
                </div>
                <div>
                  <p class="usage-step-title">1. Visit the site on Safari</p>
                  <p class="usage-step-copy">
                    Open this site in your mobile browser to get started.
                  </p>
                </div>
              </div>

              <div class="usage-step">
                <div class="usage-step-icon-box">
                  <span class="material-symbols-outlined">ios_share</span>
                </div>
                <div>
                  <p class="usage-step-title">2. Click the Share button</p>
                  <p class="usage-step-copy">
                    Locate the share icon at the bottom center of your screen.
                  </p>
                </div>
              </div>

              <div class="usage-step">
                <div class="usage-step-icon-box">
                  <span class="material-symbols-outlined">add_box</span>
                </div>
                <div>
                  <p class="usage-step-title">3. Select "Add to Home Screen"</p>
                  <p class="usage-step-copy">
                    Scroll down and select the option with the plus icon.
                  </p>
                </div>
              </div>

              <div class="usage-step is-complete">
                <div class="usage-step-icon-box">
                  <span class="material-symbols-outlined">check_circle</span>
                </div>
                <div>
                  <p class="usage-step-title">4. Finished</p>
                  <p class="usage-step-copy">
                    The app is now on your home screen, ready for your next session.
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div class="usage-secondary-card">
            <div class="usage-panel-heading">
              <span class="material-symbols-outlined usage-panel-icon">code</span>
              <h3 class="usage-panel-title">
                Desktop Preview 
                <span>(DevTools)</span>
              </h3>
            </div>

            <div class="usage-steps">
              <div class="usage-step">
                <div class="usage-step-icon-box">
                  <span class="material-symbols-outlined">search</span>
                </div>
                <div>
                  <p class="usage-step-title">1. Inspect element</p>
                  <p class="usage-step-copy">
                    Right-click anywhere on the page and select "Inspect" to open your browser's
                    developer tools.
                  </p>
                </div>
              </div>

              <div class="usage-step">
                <div class="usage-step-icon-box">
                  <span class="material-symbols-outlined">devices</span>
                </div>
                <div>
                  <p class="usage-step-title">2. Toggle device toolbar</p>
                  <p class="usage-step-copy">
                    Switch to the responsive device view so the page renders inside a mobile-sized
                    viewport.
                  </p>
                </div>
              </div>

              <div class="usage-step is-complete">
                <div class="usage-step-icon-box">
                  <span class="material-symbols-outlined">check_circle</span>
                </div>
                <div>
                  <p class="usage-step-title">3. Start testing</p>
                  <p class="usage-step-copy">
                    You can now interact with the app in a simulated mobile layout directly from
                    your desktop browser.
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div class="usage-simulator-column">
            <div class="usage-simulator-card">
              <div class="usage-simulator-content">
                <div class="usage-panel-heading">
                  <span class="material-symbols-outlined usage-panel-icon">desktop_windows</span>
                  <h3 class="usage-panel-title">Desktop Version</h3>
                </div>

                <p class="usage-simulator-copy">
                  Experience the interface directly from your browser. Plan, create and view
                  previous session statistics.
                </p>

                <a class="usage-simulator-link is-disabled"  aria-disabled="true">
                  Coming soon
                  <span class="material-symbols-outlined">open_in_new</span>
                </a>

              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="vision" id="vision">
      <h2 class="vision-title">
        Focus on the lift. <br />
        We'll handle the math.
      </h2>

      <p class="vision-copy">
        Let progressive overload be easy.
        <strong>resilnce</strong> is the quiet partner in your pursuit of strength.
      </p>

      <div class="vision-line"></div>
    </section>
  </main>

  <footer class="site-footer">
    <div class="footer-row container">
      <div class="footer-brand">resilnce</div>
      <div class="footer-copy">{currentYear} resilnce.</div>
    </div>
  </footer>
</div>

<style>
  .landscape-landing[data-theme='light'] {
    --landing-page-background: #f7fbf1;
    --landing-header-background: rgba(247, 251, 241, 0.8);
    --landing-text-primary: #182018;
    --landing-text-secondary: #4b5a4d;
    --landing-brand-color: #4a984b;
    --landing-brand-color-strong: #36843a;
    --landing-note-background: #eff5e8;
    --landing-footer-background: #edf3e5;
    --landing-border-subtle: rgba(24, 32, 24, 0.08);
    --landing-button-text: #ffffff;
    --landing-button-shadow: 0 14px 32px -16px rgba(25, 54, 28, 0.45);
    --landing-button-shadow-hover: 0 22px 42px -18px rgba(25, 54, 28, 0.5);
    --landing-vision-divider: rgba(54, 132, 58, 0.25);
    --landing-toggle-background: rgba(255, 255, 255, 0.72);
    --landing-toggle-background-hover: rgba(255, 255, 255, 0.9);
    --landing-toggle-border: rgba(24, 32, 24, 0.08);
    --landing-toggle-text: #243126;
    --landing-toggle-shadow: 0 10px 30px -18px rgba(24, 32, 24, 0.28);
    --landing-preview-label-background: rgba(255, 255, 255, 0.78);
    --landing-preview-label-background-active: rgba(255, 255, 255, 0.96);
    --landing-preview-label-border: rgba(24, 32, 24, 0.08);
    --landing-preview-label-index: rgba(74, 152, 75, 0.72);
    --landing-preview-label-shadow: 0 18px 36px -24px rgba(24, 32, 24, 0.24);
    --landing-usage-border: rgba(112, 122, 108, 0.2);
    --landing-usage-divider: rgba(25, 106, 35, 0.3);
    --landing-usage-icon-background: #ebefe5;
    --landing-usage-icon-border: rgba(112, 122, 108, 0.3);
    --landing-usage-icon-text: #182018;
    --landing-usage-icon-hover-text: #ffffff;
    --landing-usage-complete-background: rgba(25, 106, 35, 0.1);
    --landing-usage-complete-border: rgba(25, 106, 35, 0.2);
    --landing-usage-card-background: #f1f5eb;
    --landing-usage-card-border: rgba(112, 122, 108, 0.3);
    --landing-usage-card-glow: rgba(25, 106, 35, 0.05);
    --landing-usage-card-glow-hover: rgba(25, 106, 35, 0.1);
    --landing-usage-link-shadow: 0 20px 25px -5px rgba(25, 106, 35, 0.2);
    --landing-page-glow: radial-gradient(
      circle at top center,
      rgba(130, 183, 117, 0.12),
      transparent 36%
    );
  }

  .landscape-landing {
    --shadow-phone: 0 10px 30px -10px rgba(0, 0, 0, 0.1), 0 1px 4px rgba(0, 0, 0, 0.05);
    --shadow-phone-front: 0 25px 50px -12px rgba(24, 29, 23, 0.15);

    --radius-card: 1.5rem;
    --radius-chart: 1rem;
    --radius-pill: 9999px;
    --radius-phone: 3rem;

    --container-width: 80rem;
    --page-gutter: 2rem;

    --radius: 340px;
    --phone-width: 200px;
  }

  .landscape-landing[data-theme='dark'] {
    --landing-page-background: var(--color-background);
    --landing-header-background: rgba(35, 35, 35, 0.78);
    --landing-text-primary: #f3f7ef;
    --landing-text-secondary: #b9c6b5;
    --landing-brand-color: #4a984b;
    --landing-brand-color-strong: #71d772;
    --landing-note-background: rgba(255, 255, 255, 0.05);
    --landing-footer-background: rgba(255, 255, 255, 0.035);
    --landing-border-subtle: rgba(255, 255, 255, 0.1);
    --landing-button-text: #081008;
    --landing-button-shadow: 0 18px 40px -18px rgba(6, 18, 8, 0.65);
    --landing-button-shadow-hover: 0 26px 50px -18px rgba(6, 18, 8, 0.72);
    --landing-vision-divider: rgba(113, 215, 114, 0.38);
    --landing-toggle-background: rgba(255, 255, 255, 0.06);
    --landing-toggle-background-hover: rgba(255, 255, 255, 0.1);
    --landing-toggle-border: rgba(255, 255, 255, 0.1);
    --landing-toggle-text: #eef6eb;
    --landing-toggle-shadow: 0 16px 34px -22px rgba(0, 0, 0, 0.7);
    --landing-preview-label-background: rgba(255, 255, 255, 0.04);
    --landing-preview-label-background-active: rgba(255, 255, 255, 0.08);
    --landing-preview-label-border: rgba(255, 255, 255, 0.12);
    --landing-preview-label-index: rgba(113, 215, 114, 0.82);
    --landing-preview-label-shadow: 0 22px 40px -28px rgba(0, 0, 0, 0.72);
    --landing-usage-border: rgba(255, 255, 255, 0.1);
    --landing-usage-divider: rgba(113, 215, 114, 0.35);
    --landing-usage-icon-background: rgba(255, 255, 255, 0.06);
    --landing-usage-icon-border: rgba(255, 255, 255, 0.14);
    --landing-usage-icon-text: #f3f7ef;
    --landing-usage-icon-hover-text: #081008;
    --landing-usage-complete-background: rgba(113, 215, 114, 0.14);
    --landing-usage-complete-border: rgba(113, 215, 114, 0.24);
    --landing-usage-card-background: rgba(255, 255, 255, 0.04);
    --landing-usage-card-border: rgba(255, 255, 255, 0.12);
    --landing-usage-card-glow: rgba(113, 215, 114, 0.02);
    --landing-usage-card-glow-hover: rgba(113, 215, 114, 0.08);
    --landing-usage-link-shadow: 0 20px 28px -8px rgba(8, 16, 8, 0.55);
    --landing-page-glows: radial-gradient(
      circle at top center,
      rgba(113, 215, 114, 0.12),
      transparent 34%
    );
  }

  .landscape-landing {
    position: relative;
    min-height: 100vh;
    /* Keep horizontal overflow clipped without creating a scroll container that breaks sticky pinning. */
    overflow-x: clip;
    overflow-y: visible;
    font-family:
      Inter,
      ui-sans-serif,
      system-ui,
      -apple-system,
      BlinkMacSystemFont,
      'Segoe UI',
      sans-serif;
    background: var(--landing-page-background);
    color: var(--landing-text-primary);
    isolation: isolate;
  }

  .landscape-landing::before {
    content: '';
    position: absolute;
    inset: 0 0 auto 0;
    height: 32rem;
    background: var(--landing-page-glow);
    pointer-events: none;
    z-index: 0;
  }

  .landscape-landing > * {
    position: relative;
    z-index: 1;
  }

  * {
    box-sizing: border-box;
  }

  img {
    display: block;
    max-width: 100%;
  }

  a {
    color: inherit;
    text-decoration: none;
  }

  .material-symbols-outlined {
    font-variation-settings:
      'FILL' 0,
      'wght' 400,
      'GRAD' 0,
      'opsz' 24;
  }

  .container {
    width: min(var(--container-width), calc(100% - (var(--page-gutter) * 2)));
    margin: 0 auto;
  }

  .site-header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    z-index: 50;
    background: var(--landing-header-background);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    transition: all 300ms ease;
  }

  .site-nav {
    height: 5rem;
    padding: 0 1.5rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1.5rem;
  }

  .brand {
    font-size: 1.5rem;
    font-weight: 900;
    letter-spacing: -0.05em;
    color: var(--landing-brand-color);
    width: 4rem;
  }

  .nav-links {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    gap: 3rem;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .nav-links a {
    font-weight: 700;
    letter-spacing: -0.025em;
    color: var(--landing-text-secondary);
    transition:
      color 300ms ease,
      opacity 300ms ease;
  }

  .nav-links a:hover {
    color: var(--landing-brand-color);
    opacity: 0.8;
  }

  .button-primary {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 0;
    border-radius: var(--radius-pill);
    padding: 0.5rem 2rem;
    font: inherit;
    font-weight: 700;
    color: var(--landing-button-text);
    background: linear-gradient(
      135deg,
      var(--landing-brand-color)
    );
    box-shadow: var(--landing-button-shadow);
    cursor: pointer;
    transition:
      transform 300ms ease,
      box-shadow 300ms ease;
  }

  .button-primary img {
    width: 1.25rem;
    height: 1.25rem;
  }

  .button-primary:hover {
    transform: translateY(-2px);
    box-shadow: var(--landing-button-shadow-hover);
  }

  .button-primary:active {
    transform: scale(0.95);
  }

  .theme-toggle {
    display: inline-flex;
    align-items: center;
    gap: 0.55rem;
    border: 1px solid var(--landing-toggle-border);
    border-radius: var(--radius-pill);
    padding: 0.7rem 1rem;
    background: var(--landing-toggle-background);
    color: var(--landing-toggle-text);
    box-shadow: var(--landing-toggle-shadow);
    backdrop-filter: blur(18px);
    -webkit-backdrop-filter: blur(18px);
    font: inherit;
    font-weight: 700;
    letter-spacing: -0.02em;
    cursor: pointer;
    transition:
      transform 200ms ease,
      background 200ms ease,
      border-color 200ms ease,
      box-shadow 200ms ease;
      display: none;
  }

  .theme-toggle:hover {
    transform: translateY(-1px);
    background: var(--landing-toggle-background-hover);
  }

  .theme-toggle:active {
    transform: scale(0.98);
  }

  .theme-toggle .material-symbols-outlined {
    font-size: 1.15rem;
  }

  .page {
    padding-top: 6rem;
  }

  .hero {
    margin-bottom: 8rem;
    padding: 0 var(--page-gutter);
  }

  .hero-layout {
    display: flex;
    align-items: center;
    gap: 4rem;
  }

  .hero-copy,
  .hero-visual {
    width: 50%;
    min-width: 0;
  }

  .hero-title {
    margin: 0 0 1.5rem;
    font-size: 6rem;
    line-height: 1;
    font-weight: 900;
    letter-spacing: -0.05em;
    color: var(--landing-brand-color);
  }

  .hero-subtitle {
    margin: 0;
    font-size: 1.875rem;
    line-height: 1.625;
    font-weight: 500;
    color: var(--landing-text-secondary);
    opacity: 0.9;
  }

  .hero-note {
    display: inline-flex;
    align-items: flex-start;
    gap: 1rem;
    max-width: 28rem;
    margin-top: 3rem;
    padding: 1.5rem;
    background: var(--landing-note-background);
    border-left: 4px solid var(--landing-brand-color);
  }

  .hero-note-icon {
    flex: 0 0 auto;
    font-size: 1.875rem;
    line-height: 1;
    color: var(--landing-brand-color);
  }

  .hero-note-text {
    margin: 0;
    font-size: 0.875rem;
    line-height: 1.625;
    font-weight: 500;
    color: var(--landing-text-secondary);
    opacity: 0.9;
  }

  .mockup-stack {
    position: relative;
    width: 100%;
    max-width: 28rem;
    height: 600px;
    margin: 0 auto;
    overflow: visible;
    scale: 0.9;
  }


  .phone-frame {
    position: absolute;
    overflow: hidden;
    border: 6px solid #1a1a1a;
    border-radius: var(--radius-phone);
    background: #202020;
    will-change: transform;
    transform-origin: 50% 100%;
  }

  .phone-frame img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .phone-frame::after {
    content: '';
    position: absolute;
    inset: 0;
    pointer-events: none;
  }

  .phone-right {
    top: 5rem;
    right: 0;
    z-index: 10;
    width: 16rem;
    aspect-ratio: 9 / 19;
    opacity: 0.8;
    box-shadow: var(--shadow-phone);
    transform: translateY(calc(-1rem - var(--parallax-y, 0px)))
      rotate(calc(6deg + var(--parallax-rotate, 0deg) / 2)) translateX(3rem) scale(0.9);
  }

  .phone-left {
    top: 2.5rem;
    left: 0;
    z-index: 20;
    width: 16rem;
    aspect-ratio: 9 / 19;
    opacity: 0.9;
    box-shadow: var(--shadow-phone);
    transform: translateY(calc(0px - var(--parallax-y, 0px)))
      rotate(calc(-3deg + var(--parallax-rotate, 0deg) / 2)) translateX(-2rem) scale(0.95);
  }

  .phone-front {
    top: 0;
    left: 50%;
    z-index: 30;
    width: 18rem;
    aspect-ratio: 9 / 19;
    box-shadow: var(--shadow-phone-front);
    transform: translateX(-50%) translateY(0px) rotate(0deg);
  }

  .phone-right::after,
  .phone-left::after {
    background: rgba(84, 84, 84, 0.1);
  }


  .carousel-section {
    position: relative;
    height: 320vh;
  }

  .carousel-pin {
    position: sticky;
    top: 0;
    height: 100vh;
    scale: 0.8;
  }

  .scene {
    position: relative;
    width: 100%;
    height: 100%;
    perspective: 1800px;
    perspective-origin: 50% 42%;
    isolation: isolate;
  }

  .preview-labels {
    display: grid;
    margin-top: 0.5rem;
    gap: 0.75rem;
    width: min(22rem, calc(100vw - 4rem));
    pointer-events: none;
  }

  .preview-label {
    display: grid;
    grid-template-columns: auto 1fr;
    align-items: center;
    gap: 0.9rem;
    padding: 0.95rem 1rem;
    border: 1px solid var(--landing-preview-label-border);
    border-radius: 1rem;
    background: var(--landing-preview-label-background);
    box-shadow: var(--landing-preview-label-shadow);
    backdrop-filter: blur(18px);
    -webkit-backdrop-filter: blur(18px);
    opacity: 0;
    transform: translateX(1.5rem) scale(0.96);
    transition:
      opacity 220ms ease,
      transform 220ms ease,
      background 220ms ease;
  }

  .preview-label--visible {
    opacity: 1;
    transform: translateX(0) scale(1);
    background: var(--landing-preview-label-background-active);
  }

  .preview-label__index {
    font-size: 0.72rem;
    font-weight: 900;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    color: var(--landing-preview-label-index);
  }

  .preview-label__text {
    font-size: 0.98rem;
    font-weight: 700;
    letter-spacing: -0.02em;
    color: var(--landing-text-primary);
  }

  .rig {
    position: absolute;
    inset: 0;
    transform-style: preserve-3d;
    will-change: transform;
    margin-left: 30rem;
    margin-top: 10rem;
  }

  .ground {
    position: absolute;
    left: 50%;
    top: 74%;
    width: 1400px;
    height: 1400px;
    transform: translate(-50%, -50%) rotateX(90deg);
    border-radius: 50%;
    background: radial-gradient(
      circle at center,
      rgba(255, 255, 255, 0.055) 0%,
      rgba(255, 255, 255, 0.018) 30%,
      transparent 58%
    );
    pointer-events: none;
  }

  .track {
    position: absolute;
    left: 50%;
    top: 74%;
    width: calc(var(--radius) * 2 + 56px);
    height: calc(var(--radius) * 2 + 56px);
    transform: translate(-50%, -50%) rotateX(90deg);
    border-radius: 50%;
    border: 1px solid rgba(255, 255, 255, 0.08);
    box-shadow:
      0 0 100px rgba(255, 255, 255, 0.05),
      inset 0 0 50px rgba(255, 255, 255, 0.025);
    pointer-events: none;
  }

  .wheel {
    position: absolute;
    left: 50%;
    top: 74%;
    width: 0;
    height: 0;
    transform-style: preserve-3d;
    will-change: transform;
  }

  .card-mount {
    position: absolute;
    left: 0;
    top: 0;
    transform-style: preserve-3d;
    transform: rotateY(var(--angle)) translateZ(var(--radius));
    will-change: transform;
  }

  .card-anchor {
    position: absolute;
    left: 0;
    top: 0;
    transform-style: preserve-3d;
    transform: translate(-50%, -100%);
    will-change: transform;
  }

  .iphone-frame {
    --depth: 18px;
    --frame-radius: 30px;
    --bezel-size: 5px;
    --screen-radius: calc(var(--frame-radius) - var(--bezel-size));
    --iphone-tint: 10%;
    --iphone-color: linear-gradient(
      180deg,
      hsl(0 0% var(--iphone-tint)),
      hsl(0 0% calc(var(--iphone-tint) * 0.5))
    );

    position: relative;
    width: var(--phone-width);
    aspect-ratio: 100 / 212;
    transform-style: preserve-3d;
    backface-visibility: hidden;
  }

  .iphone-shell-layer,
  .iphone-back,
  .iphone-front {
    position: absolute;
    inset: 0;
    border-radius: var(--frame-radius);
    backface-visibility: hidden;
    transform-style: preserve-3d;
  }

  .iphone-shell-layer {
    background: var(--iphone-color);
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.05);
    transform: translateZ(var(--z));
  }

  .iphone-back {
    transform: translateZ(calc(var(--depth) / -2));
    background: var(--iphone-color);
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.04);
  }

  .iphone-front {
    padding: var(--bezel-size);
    overflow: hidden;
    transform: translateZ(calc(var(--depth) / 2));
    box-shadow:
      inset 0 0 0 1px rgba(255, 255, 255, 0.013),
      0 20px 50px rgba(0, 0, 0, 0.04);
  }

  .screen {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: var(--screen-radius);
    backface-visibility: hidden;
    user-select: none;
    -webkit-user-drag: none;
  }

  .screen.no-top{
    border-radius: 0 0 var(--screen-radius) var(--screen-radius);
    height: 100%;
    object-fit:cover;
  }


  .dynamic-island {
    position: absolute;
    top: 11px;
    left: 50%;
    transform: translateX(-50%);
    width: 55px;
    height: 16px;
    border-radius: 999px;
    background: #000000;
    box-shadow:
      inset 0 1px 0 rgba(146, 146, 146, 0.15),
      0 2px 8px rgba(0, 0, 0, 0.15);
    z-index: 2;
    pointer-events: none;
  }

  .top-bar-phone {
    width: 100%;
    height: 25px;
    background: var(--color-background);
    z-index: 0;
    pointer-events: none;
    border-radius: var(--screen-radius) var(--screen-radius) 0 0;
  }

  .status-bar {
    font-size: 8px;
    height: 100%;
    display: flex;
    align-items: center;
    flex-direction: row;
    justify-content: space-between;
    color: rgb(219, 219, 219);
  }

  .hud {
    position: absolute;
    left: 0;
    top: 28px;
    z-index: 4;
    display: grid;
    gap: 8px;
    pointer-events: none;
  }

  .hud-eyebrow {
    font-size: 0.72rem;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    color: rgba(255, 255, 255, 0.5);
  }

  .hud strong {
    font-size: clamp(1.25rem, 2vw, 1.8rem);
    line-height: 1.1;
    letter-spacing: -0.03em;
  }

  .progress {
    width: 347px;
    height: 2px;
    background: rgba(255, 255, 255, 0.12);
    overflow: hidden;
    border-radius: 999px;
  }

  .progress > span {
    display: block;
    width: 100%;
    height: 100%;
    transform-origin: left center;
    transform: scaleX(var(--progress, 0));
    background: rgba(255, 255, 255, 0.92);
  }

  .vision {
    max-width: 64rem;
    margin: 0 auto;
    padding: 10rem var(--page-gutter);
    text-align: center;
  }

  .vision-title {
    margin: 0 0 2rem;
    font-size: 3rem;
    line-height: 1.1;
    font-weight: 900;
    letter-spacing: -0.05em;
    text-align: center;
  }

  .vision-copy {
    max-width: 42rem;
    margin: 0 auto 3rem;
    font-size: 1.25rem;
    line-height: 1.625;
    color: var(--landing-text-secondary);
    opacity: 0.9;
  }

  .vision-line {
    width: 6rem;
    height: 0.25rem;
    margin: 0 auto;
    border-radius: 9999px;
    background: var(--landing-vision-divider);
  }

  .site-footer {
    width: 100%;
    padding: 1rem var(--page-gutter);
  }

  .footer-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1.5rem;
  }

  .footer-brand {
    font-size: 1.125rem;
    font-weight: 700;
    letter-spacing: -0.05em;
    color: var(--landing-brand-color);
  }

  .footer-copy {
    font-size: 0.75rem;
    font-weight: 500;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--landing-text-secondary);
    opacity: 0.7;
  }

  @media (min-width: 1500px) {
    .hero {
      margin-top: 0rem;
    }

    .page{
      margin-top: 6rem;
    }

    .site-footer {
      padding: 3rem;
    }

    .scene {
      perspective-origin: 50% 38%;
    }

    .rig {
      margin-left: 31rem;
      margin-top: 4rem;
      transform-origin: 50% 128%;
      scale: 1.15;
    }

    .hud {
      margin-top: 5rem;
      scale: 1.1;
    }
  }

  @media (max-width: 1150px) {
    .rig {
      margin-left: 25rem;
      margin-top: 0;
    }

    .preview-labels {
      width: min(18rem, calc(100vw - 3rem));
    }

    .mockup-stack {
      scale: 0.7;
      width: 8rem;
    }

    .phone-left {
      opacity: 0;
    }

    .phone-right {
      opacity: 0;
    }

    .dynamic-island {
      width: 45px;
      height: 14px;
    }
  }

  @media (max-width: 720px) {
    .preview-labels {
      width: min(16.5rem, calc(100vw - 2rem));
    }
  }

  .usage {
    padding: 6.5rem var(--page-gutter);
    border-top: 1px solid var(--landing-usage-border);
  }

  .usage-header {
    margin-bottom: 3rem;
  }

  .usage-title {
    margin: 0 0 1rem;
    font-size: 2.5rem;
    line-height: 0.95;
    font-weight: 900;
    letter-spacing: -0.05em;
    color: var(--landing-brand-color);
  }

  .usage-title-line {
    width: 4rem; /* w-16 */
    height: 0.25rem; /* h-1 */
    border-radius: 9999px;
    background: var(--landing-usage-divider);
  }

  .usage-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 3.5rem;
  }

  .usage-panel-heading {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 1.5rem;
  }

  .usage-panel-icon {
    font-size: 1.5rem;
    color: var(--landing-brand-color);
  }

  .usage-panel-title {
    margin: 0;
    font-size: 1.55rem;
    line-height: 1.2;
    font-weight: 600;
    letter-spacing: -0.025em;
  }

  .usage-panel-title span {
    font-weight: 300;
    color: var(--landing-text-secondary);
  }

  .usage-steps {
    display: grid;
    gap: 1.75rem;
  }

  .usage-step {
    display: flex;
    gap: 1rem;
  }

  .usage-step-icon-box {
    flex: 0 0 auto;
    width: 2.5rem;
    height: 2.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 0.85rem;
    background: var(--landing-usage-icon-background);
    border: 1px solid var(--landing-usage-icon-border);
    color: var(--landing-usage-icon-text);
    transition:
      background 300ms ease,
      color 300ms ease,
      border-color 300ms ease;
  }

  .usage-step:hover .usage-step-icon-box {
    background: var(--landing-brand-color);
    color: rgb(201, 255, 201);
    border-color: var(--landing-brand-color);
  }

  .usage-step-title {
    margin: 0 0 0.25rem;
    font-size: 1rem;
    line-height: 1.35;
    font-weight: 600;
    letter-spacing: -0.025em;
  }

  .usage-step-copy {
    margin: 0;
    font-size: 0.95rem;
    line-height: 1.55;
    color: var(--landing-text-secondary);
  }

  .usage-step.is-complete .usage-step-icon-box {
    background: var(--landing-usage-complete-background);
    color: var(--landing-brand-color);
    border-color: var(--landing-usage-complete-border);
  }

  .usage-step.is-complete .usage-step-title {
    color: var(--landing-brand-color);
  }

  .usage-simulator-column {
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .usage-simulator-card {
    position: relative;
    overflow: hidden;
    padding: 1.75rem;
    border-radius: 1rem;
    background: var(--landing-usage-card-background);
    border: 1px solid var(--landing-usage-card-border);
  }

  .usage-simulator-card::after {
    content: '';
    position: absolute;
    top: -4rem;
    right: -4rem;
    width: 16rem; /* w-64 */
    height: 16rem; /* h-64 */
    border-radius: 9999px;
    background: var(--landing-usage-card-glow);
    filter: blur(48px);
    transition: background 500ms ease;
  }

  .usage-simulator-card:hover::after {
    background: var(--landing-usage-card-glow-hover);
  }

  .usage-simulator-content {
    position: relative;
    z-index: 1;
  }

  .usage-simulator-copy {
    margin: 0 0 1.9rem;
    font-size: 1rem;
    line-height: 1.65;
    color: var(--landing-text-secondary);
  }

  .usage-simulator-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem;
    border-radius: 0.5rem;
    background: var(--landing-brand-color);
    color: var(--landing-button-text);
    font-size: 1rem;
    font-weight: 600;
    letter-spacing: -0.025em;
    box-shadow: var(--landing-usage-link-shadow);
    transition: transform 300ms ease;
  }

  .usage-simulator-link:hover {
    transform: scale(1.02);
  }

  .usage-simulator-link:active {
    transform: scale(0.95);
  }



  .usage-simulator-link.is-disabled {
    pointer-events: none;
    cursor: not-allowed;
    opacity: 0.5;
    box-shadow: none;
    transform: none;
  }

  .usage-simulator-link.is-disabled:hover,
  .usage-simulator-link.is-disabled:active {
    transform: none;
  }
</style>
