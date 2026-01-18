<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { checkActiveSession } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import { user } from '$lib/stores/appState';
  import { get } from 'svelte/store';

  import Icon from '@iconify/svelte';
  import { blur, fade, slide } from 'svelte/transition';
  import { resolve } from '$app/paths';
  import { toast } from 'svelte-sonner';
  import { afterNavigate } from '$app/navigation';
  import { PUBLIC_BACKEND_BASE_URL } from '$env/static/public';
  import { greet } from './greeting';
  import { CheckActiveSession } from './tracker/dbFetches';

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
      default:
        toast.warning(`${sonner}`, { duration: 4000 });
        break;
    }
  }

  let existingSession = $state(false);
  let existingID = $state('');
  let greetMessage = $state('');
  let loading = $state(true);

  function backendAdress(dir: string): string {
    const baseURL: string = PUBLIC_BACKEND_BASE_URL;
    return baseURL + dir;
  }

  async function getMe(): Promise<boolean> {
    const res = await fetch(backendAdress('/api/me'), {
      method: 'GET',
      credentials: 'include',
    });
    if (res.ok) {
      const data = await res.json();
      console.log(data);
      user.set(data);
      return true;
    } else {
      user.set(null);
      console.error('User not logged in.');
      return false;
    }
  }

  onMount(async () => {
    const value = get(user);

    if (value?.name || (await getMe())) {
      loadUserHomepage(value);
    } else {
      greetMessage = 'The only gym tracker you need.';
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

    const res = await CheckActiveSession();
    console.log('Active session:', res);
    var actv = res != '';

    if (actv != null) {
      existingSession = actv;
      console.log('exists', existingSession);

      if (existingSession) {
        existingID = res;
      }
    }
    greetMessage = greet(name, existingSession);
  }
</script>

{#if loading}
  <p>loading</p>
{:else}
  <div in:fade={{ duration: 400, delay: 50 }} class="body">
    <cont class="cont">
      <h1 in:fade={{ duration: 400, delay: 100 }} class="wid">
        {greetMessage}
      </h1>

      <hr in:fade={{ duration: 400, delay: 150 }} />

      <div class="btn-container">
        {#if $user?.name}
          {#if existingSession}
            <button
              in:slide|global={{ duration: 600 }}
              class="base-btn alt buttonClass"
              onclick={() => goto(resolve(`/tracker/${existingID}`))}
            >
              <g>Continue session:</g>
              <i class="font-regular">{existingID}</i>
            </button>
          {/if}

          <button
            in:fade={{ duration: 400, delay: 200 }}
            class="base-btn sesh buttonClass"
            onclick={() => goto(resolve(`/tracker`))}
          >
            <g>Begin a workout</g>
            <Icon icon="mdi:arm-flex" width="32" />
          </button>

          <button
            in:fade={{ duration: 400, delay: 300 }}
            class="base-btn sesh buttonClass"
            onclick={() => goto(resolve(`/create`))}
          >
            <g>Add new session</g>
            <Icon icon="gridicons:create" width="32" />
          </button>
        {:else}
          <button
            in:fade={{ duration: 400, delay: 150 }}
            class="base-btn sesh buttonClass"
            onclick={() => goto(resolve(`/account`))}
          >
            <g>Log in</g>
            <Icon icon="material-symbols:login-rounded" width="32" />
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
    width: 15rem;
    height: auto;
    filter: drop-shadow(0px 19px 38px rgba(79, 89, 111, 0.187))
      drop-shadow(0px 15px 12px rgba(78, 78, 78, 0.13));
  }

  .toUpper {
    text-transform: capitalize;
  }

  .books.true {
    top: 30%;
  }

  hr {
    height: 0.1px;
    background-color: #ffffff;
    margin: 1rem 0 1rem 0;
    width: 100%;
  }

  .wid {
    width: 100%;
    font-size: 2rem;
    margin-left: 0.5rem;
    line-height: 2.5rem;
    letter-spacing: -0.02em;
  }

  .body {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-end;
    box-sizing: border-box;
    padding: 5rem 0;
    height: 100vh;
    background: var(--gradient-prim-o);
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
    color: var(--color-contrast);
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
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow-dark);
    width: 100%;
    margin: 0;
    padding: 0.8rem 1.2rem;
    border-radius: 6px;
    height: 4rem;
  }

  .base-btn.sesh {
    font-weight: 600;
  }

  .base-btn.alt {
    background-color: var(--color-alt);
    color: var(--color-contrast);
    box-shadow: var(--color-alt) 0px 0px 20px 1px;
    margin-bottom: 1rem;
  }
</style>
