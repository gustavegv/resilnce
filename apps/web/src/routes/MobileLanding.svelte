<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';

  import Icon from '@iconify/svelte';
  import { blur, fade, slide } from 'svelte/transition';
  import { resolve } from '$app/paths';
  import { toast } from 'svelte-sonner';
  import { afterNavigate } from '$app/navigation';
  import { greet } from './greeting';
  import { CheckActiveSession } from './tracker/dbFetches';
  import { getMe } from './me';

  const linkBase = import.meta.env.BASE_URL;

  afterNavigate(({ to }) => {
    if (to == null) return;
    const sonner: string = to.url.searchParams.get('sonner') || '';
    toastPopup(sonner);
  });

  function toastPopup(sonner: string) {
    if (!sonner) return;
    switch (sonner) {
      case 'saved':
        toast.success(`Session saved successfully!`, { duration: 3000 });
        break;
      case 'loggedin':
        toast.success('Log in successul!', { duration: 3000 });
        break;

      case 'auth':
        toast.warning('Please log in first.', { duration: 4000 });
        break;
      default:
        toast.warning(`${sonner}`, { duration: 4000 });
        break;
    }
  }

  let existingSession = $state(false);
  let existingID = $state('');
  let existingName = $state('');
  let greetMessage = $state('');
  let loading = $state(true);

  onMount(async () => {
    const value = get(user);

    if (value?.name || (await getMe())) {
      loadUserHomepage(value);
    } else {
      greetMessage = 'resilnce.';
    }

    loading = false;
  });

  async function loadUserHomepage(uData: { name: string; id: string } | null) {
    var name: string = uData?.name ?? '';
    var id: string = uData?.id ?? '';
    if (uData == null) {
      name = get(user)?.name ?? '';
      id = get(user)?.id ?? '';
    }
    if (!name || !id) {
      console.error('No username/ID');
      window.location.reload();
      return;
    }

    const [activeID, activeName] = await CheckActiveSession();
    console.log('Active session:', activeID, activeName);

    var actv = activeID != '';

    if (actv != null) {
      existingSession = actv;
      console.log('exists', existingSession);

      if (existingSession) {
        existingID = activeID;
        existingName = activeName;
      }
    }
    greetMessage = greet(name, existingSession);
  }
</script>

{#if loading}
  <p></p>
{:else}
  <div in:fade={{ duration: 400, delay: 50 }} class="body">
    <div class="flex w-full justify-end p-6">
      {#if existingSession}
        <button
          in:slide|global={{ duration: 600 }}
          class="base-btn alt flex w-full items-center justify-between"
          onclick={() => goto(resolve(`/tracker/${existingID}`))}
        >
          <div class="flex min-w-0 flex-1 flex-col text-left">
            <g class="font-bold">Resume session</g>
            <i class="truncate">{existingName}</i>
          </div>

          <Icon icon="material-symbols:resume-rounded" width={40} />
        </button>
      {/if}
    </div>

    <cont class="cont">
      <h1 in:fade={{ duration: 400, delay: 100 }} class="wid">
        {greetMessage}
      </h1>

      <div
        in:fade={{ duration: 400, delay: 150 }}
        class="mt-3 mb-4 w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-white/70 backdrop-blur-md"
      >
        {#if !$user?.name}
          The only gym tracker you need.
        {:else if existingSession}
          Pick up where you left off or start something new.
        {:else}
          What are we doing today?
        {/if}
      </div>

      <div class="btn-container">
        {#if $user?.name}
          <button
            in:fade={{ duration: 400, delay: 200 }}
            class="base-btn sesh"
            onclick={() => goto(resolve(`/tracker`))}
          >
            <g>Begin a workout</g>
            <Icon icon="mdi:arm-flex" width="32" />
          </button>

          <button
            in:fade={{ duration: 400, delay: 300 }}
            class="base-btn sesh"
            onclick={() => goto(resolve(`/create`))}
          >
            <g>Add new session</g>
            <Icon icon="gridicons:create" width="32" />
          </button>
        {:else}
          <button
            in:fade={{ duration: 400, delay: 150 }}
            class="base-btn sesh"
            onclick={() => goto(resolve(`/account`))}
          >
            <g>Log in</g>
            <Icon icon="material-symbols:login-rounded" width="28" />
          </button>
        {/if}
      </div>
    </cont>
  </div>

  <img
    in:fade={{ duration: 400, delay: 0 }}
    src="{linkBase}books.png"
    alt="a"
    class="books {existingSession} bob"
  />
{/if}

<style>
  @keyframes tilt {
    0%,
    100% {
      transform: translate(-50%, -50%) rotateX(0deg) rotateY(0deg);
    }
    25% {
      transform: translate(-50%, -50%) rotateX(2deg) rotateY(-4deg);
    }
    50% {
      transform: translate(-50%, -50%) rotateX(-1deg) rotateY(2deg);
    }
    75% {
      transform: translate(-50%, -50%) rotateX(1deg) rotateY(-1deg);
    }
  }

  .bob {
    transform-style: preserve-3d;
    animation: tilt 7s cubic-bezier(0.37, 0, 0.63, 1) infinite;
  }

  .books {
    position: fixed;
    top: 32%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 0;
    pointer-events: none;
    user-select: none;
    width: 55vw;
    height: auto;
    filter: drop-shadow(0px 19px 38px rgba(79, 89, 111, 0.187))
      drop-shadow(0px 15px 12px rgba(78, 78, 78, 0.13));
    opacity: 1;
  }

  .body {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-end;
    box-sizing: border-box;
    padding: 5rem 0;
    min-height: 100vh;
    overflow: hidden;
    background: #0b0f0b;
  }

  .body::before {
    content: '';
    position: absolute;
    inset: 0;
    z-index: 0;

    background-image:
      linear-gradient(rgba(45, 45, 45, 0.45), rgba(13, 13, 13, 0.65)),
      url('https://images.pexels.com/photos/30004058/pexels-photo-30004058.jpeg');
    background-size: cover;
    background-position: center;
    background-repeat: no-repeat;

    filter: blur(12px) saturate(0) brightness(0.55);
    transform: scale(1.08);
  }

  .body > * {
    position: relative;
    z-index: 1;
  }

  .books.true {
    top: 30%;
  }

  .wid {
    width: 100%;
    font-size: 2rem;
    margin-left: 0.5rem;
    line-height: 2.5rem;
    letter-spacing: -0.02em;
  }

  @media screen and (max-device-height: 667px) and (orientation: portrait) {
    .body {
      padding: 1rem 0;
    }
    .wid {
      margin: 0;
    }
    .books {
      top: 29%;
      left: 50%;
      width: 12rem;
      height: auto;
    }
  }

  h1 {
    font-weight: 600;
    color: var(--color-text);
  }

  .cont {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 90%;
  }

  .btn-container {
    display: flex;
    width: 100%;
    flex-direction: column;
    align-items: center;
    gap: 0.8rem;
  }

  .base-btn {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    width: 100%;
    min-height: 3.5rem;
    margin: 0;
    padding: 0.875rem 1rem;

    border: 1px solid rgb(255 255 255 / 0.08);
    border-radius: 12px;
    background: var(--color-secondary);
    color: white;

    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.22),
      inset 0 -1px 0 rgb(0 0 0 / 0.08),
      0 10px 24px rgb(0 0 0 / 0.18);

    transition:
      transform 120ms ease,
      box-shadow 180ms ease,
      filter 180ms ease,
      background-color 180ms ease;

    background:
      linear-gradient(to bottom, rgb(255 255 255 / 0.08), rgb(255 255 255 / 0)),
      var(--color-secondary);
  }

  .base-btn:hover {
    filter: brightness(1.03) saturate(1.02);
    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.24),
      inset 0 -1px 0 rgb(0 0 0 / 0.08),
      0 12px 28px rgb(0 0 0 / 0.2);
  }

  .base-btn:active {
    transform: scale(0.985);
    filter: brightness(0.98);
    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.16),
      inset 0 -1px 0 rgb(0 0 0 / 0.12),
      0 6px 14px rgb(0 0 0 / 0.16);
  }

  .base-btn:focus-visible {
    outline: 2px solid rgb(255 255 255 / 0.35);
    outline-offset: 2px;
  }

  .base-btn.sesh {
    font-weight: 500;
  }

  .base-btn.secondary {
    color: var(--color-text);
    border: 1px solid rgb(255 255 255 / 0.08);
    background:
      linear-gradient(to bottom, rgb(255 255 255 / 0.08), rgb(255 255 255 / 0.03)),
      rgb(255 255 255 / 0.04);
    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.06),
      0 8px 24px rgb(0 0 0 / 0.12);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
  }

  .base-btn.secondary:hover {
    background:
      linear-gradient(to bottom, rgb(255 255 255 / 0.1), rgb(255 255 255 / 0.04)),
      rgb(255 255 255 / 0.05);
    border-color: rgb(255 255 255 / 0.12);
  }

  .base-btn.secondary:active {
    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.05),
      0 4px 14px rgb(0 0 0 / 0.1);
  }

  .base-btn.alt {
    color: rgb(255 255 255 / 0.7);
    border: 1px solid rgb(255 255 255 / 0.1);
    background: rgb(255 255 255 / 0.05);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);

    width: 50%;
    font-size: 12px;
    border-radius: 16px;
    min-height: 4rem;

    box-shadow: none;
  }

  .base-btn.alt:hover {
    background: rgb(255 255 255 / 0.04);
    border-color: rgb(255 255 255 / 0.09);
  }

  .base-btn.alt:active {
    box-shadow: inset 0 1px 0 rgb(255 255 255 / 0.03);
  }
</style>
