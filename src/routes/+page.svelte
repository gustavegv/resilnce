<script lang="ts">
  import '../app.css';
  import { goto } from '$app/navigation';
  import { checkActiveSession } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import DbIcon from '../components/icons/DBIcon.svelte';
  import AddIcon from '../components/icons/AddIcon.svelte';
  import { user } from './account/user';
  import Icon from '@iconify/svelte';
  import { blur, crossfade, fade, scale, slide } from 'svelte/transition';
  import { get } from 'svelte/store';
  import { resolve } from '$app/paths';
  import { toast } from 'svelte-sonner';
  import { afterNavigate } from '$app/navigation';
  const linkBase = import.meta.env.BASE_URL;

  afterNavigate(({ to }) => {
    if (to == null) return;
    const sonner = to.url.searchParams.get('sonner');
    if (!sonner) return;
    switch (sonner) {
      case 'saved':
        toast.success(`Session saved successfully!`, { duration: 3000 });
        break;

      default:
        toast.warning(`${sonner}`, { duration: 4000 });
        break;
    }
  });

  const greetings = {
    morning: ['Good morning.', 'Ready to move?', 'Are you ready?', 'Starting strong today!'],
    day: ['Good day.', "Hope your day's going well!", 'Stay focused.', 'Stay strong.'],
    evening: ['Good evening.', 'Hope your day was good.', 'Late night lift?'],
    general: [
      'Welcome back!',
      'Hello.',
      "Let's get moving!",
      'Great to see you.',
      "Glad you're here!",
      'Ready to train?',
      'Welcome!',
      'Hi there.',
      "Let's begin.",
      "Let's get started.",
    ],
    continue: ['Keep going', "Let's go", 'Good work'],
  };

  let existingSession = $state(false);
  let existingID = $state('');
  let greetMessage = $state('');
  let punctuation = $state('');
  let titleSize = $state('2.5rem');

  function greet(): void {
    if (existingSession) {
      let pool = greetings['continue'];
      greetMessage = pool[Math.floor(Math.random() * pool.length)];
      punctuation = '!';
      return;
    }
    const hour = new Date().getHours();
    let timeOfDay: 'morning' | 'day' | 'evening' | 'general' = 'general';

    if (hour >= 5 && hour < 12) {
      timeOfDay = 'morning';
    } else if (hour >= 12 && hour < 18) {
      timeOfDay = 'day';
    } else {
      timeOfDay = 'evening';
    }

    const pool = greetings[timeOfDay];
    const odds = 1 / pool.length;
    const chosenPool = Math.random() > odds ? greetings[timeOfDay] : greetings['general'];
    const indexPicked = Math.floor(Math.random() * chosenPool.length);
    const message = chosenPool[indexPicked];

    greetMessage = message.slice(0, -1);
    const len = message.length;
    punctuation = message[len - 1];

    console.log(`greet -> ${timeOfDay} (hour ${hour}): ${message}`);
  }

  onMount(async () => {
    const userID = get(user);
    if (userID) {
      const prevSession = await checkActiveSession(userID);

      if (prevSession != null) {
        existingSession = prevSession.active;
        console.log('exists', existingSession);

        if (existingSession) {
          titleSize = '2rem';
          existingID = prevSession.session;
        }
      }
      greet();
    } else {
    }
  });
</script>

<img src="{linkBase}books.png" alt="a" class="books {existingSession}" />
{#if $user}
  <div class="body">
    <h1 in:fade={{ duration: 600, delay: 100 }} class="wid" style="font-size:{titleSize}">
      {greetMessage} <span class="toUpper">{$user}</span>{punctuation}
    </h1>

    <hr />

    <div class="btn-container">
      {#if existingSession}
        <button
          in:slide|global={{ duration: 600 }}
          class="base-btn alt buttonClass"
          onclick={() => goto(resolve(`/tracker/${existingID}`))}
        >
          <g>Continue session:</g>
          <i class="font-light">{existingID}</i>
        </button>
      {/if}

      <button class="base-btn sesh buttonClass" onclick={() => goto(resolve(`/tracker`))}>
        <g>Begin a workout</g>
        <DbIcon />
      </button>

      <button class="base-btn sesh buttonClass" onclick={() => goto(resolve(`/create`))}>
        <g>Add new session</g>
        <Icon icon="gridicons:create" width="32" />
      </button>
    </div>
  </div>
{:else}
  <div class="body">
    <h1 class="wid">The only gym tracker you need.</h1>
    <hr />
    <button class="base-btn sesh buttonClass" onclick={() => goto(resolve(`/account`))}>
      <g>Log in</g>
      <Icon icon="material-symbols:login-rounded" width="35" />
    </button>
  </div>
{/if}

<style>
  .books {
    position: fixed; /* Stay fixed to the viewport */
    top: 31%;
    left: 50%;
    transform: translate(-50%, -50%); /* Adjust this to where you want it */
    z-index: 0; /* Sit behind interactive content */
    pointer-events: none; /* Let clicks pass through */
    user-select: none;
    width: 16rem; /* Adjust to taste */
    height: auto;
    filter: drop-shadow(0px 19px 38px rgba(79, 89, 111, 0.187))
      drop-shadow(0px 15px 12px rgba(78, 78, 78, 0.13));
  }

  .toUpper {
    text-transform: capitalize;
  }

  .books.true {
    top: 27%;
  }

  hr {
    height: 0.1px;
    background-color: #ffffff;
    margin: 20px 0;
    width: 85%;
  }

  .wid {
    width: 80%;
    font-size: 2.5rem;
    margin: 0.6rem 0;
    line-height: 3.2rem;
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
      width: 12rem; /* Adjust to taste */
      height: auto;
    }
  }

  h1 {
    font-weight: 600;
    color: var(--color-contrast);
  }

  .btn-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 90%;
  }

  .base-btn {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    box-shadow: var(--shadow-dark);
    width: 95%;
  }

  .base-btn.sesh {
    -webkit-text-stroke: 0.6px rgb(255, 255, 255);
  }

  .base-btn.alt {
    background-color: var(--color-alt);
    color: var(--color-contrast);
    box-shadow: var(--color-alt) 0px 0px 20px 1px;
    height: 5rem;
  }
</style>
