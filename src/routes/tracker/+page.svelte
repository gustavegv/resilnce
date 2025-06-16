<script lang="ts">
  import { goto } from '$app/navigation';
  import { getAllSessionMeta } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';
  import ErrorPopup from '../../components/ErrorPopup.svelte';
  import Icon from '@iconify/svelte';

  import type { SessionMetaData } from '$lib/firebaseDataHandler';

  import Popup from '../../components/Popup.svelte';
  import AddIcon from '../../components/icons/AddIcon.svelte';

  let slugs: SessionMetaData[] = $state([]);
  let activeSession: boolean = $state(false);

  let showPopup: boolean = $state(false);
  let showError: string = $state('');

  onMount(async () => {
    const data = await getAllSessionMeta('user1');
    slugs = data.slugs;
    activeSession = data.active;
  });

  function grd(intervalDays: number): Date {
    const now = new Date();
    const twoMonthsInMs = intervalDays * 24 * 60 * 60 * 1000; // 60 days in milliseconds
    const nowTime = now.getTime();
    const randomOffset = Math.random() * twoMonthsInMs;
    const date = new Date(nowTime - randomOffset);
    console.log(date);
    return date;
  }

  function timeAgo(date: Date): string {
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    const intervals = [
      { label: 'year', seconds: 31536000 },
      { label: 'month', seconds: 2592000 },
      { label: 'week', seconds: 604800 },
      { label: 'day', seconds: 86400 },
      { label: 'hour', seconds: 3600 },
      { label: 'minute', seconds: 60 },
    ];

    if (seconds < 60) return 'Just now';
    if (seconds < 120) return 'A minute ago';

    for (const interval of intervals) {
      const count = Math.floor(seconds / interval.seconds);
      if (count >= 1) {
        if (interval.label === 'day' && count === 1) return 'Yesterday';
        return `${count} ${interval.label}${count > 1 ? 's' : ''} ago`;
      }
    }

    return 'Just now';
  }

  function closePopup() {
    showPopup = false;
    showError = 'Error broski';
  }

  function openPopup() {
    showPopup = true;
    showError = '';
  }

  function editWorkout(sesID: string) {
    console.log('Edit', sesID + '!');
  }

  function startSession(id: string) {
    if (activeSession) {
      openPopup();
    } else {
    }
    goto(`/tracker/${id}`);
  }
</script>

<div class="main">
  <h2>Sessions:</h2>
  <ErrorPopup message={showError}></ErrorPopup>

  <hr />
  <div class="btn-container">
    {#each slugs as slug}
      <div class="inner-cont">
        <button class="base-btn extended sesh" onclick={() => startSession(slug.id)}>
          <p>{slug.id}</p>
          <p class="date">{slug.date ?? 'No date'}</p>
        </button>

        <Icon
          style="margin:0 2rem; position:fixed; right:0;"
          onclick={() => editWorkout(slug.id)}
          icon="material-symbols:edit-outline"
          width="24"
          height="24"
        />
      </div>
    {/each}
  </div>
</div>

<style>
  .main {
    box-sizing: border-box;
    padding: 5rem 1rem;
  }

  .inner-cont {
    background-color: var(--color-sec-dark);
    display: flex;
    flex-direction: row;
    width: 100%;
    align-items: center;
    justify-content: space-between;
    border-radius: 10px;
    margin: 0.5rem;
  }

  hr {
    height: 0.1px;
    background-color: #ffffff;
    margin: 20px 0;
    width: 100%;
  }

  .btn-container {
    display: flex;
    flex-direction: column;
    align-items: baseline;
    box-sizing: border-box;
    width: 100%;
    text-align: left;
  }

  .base-btn {
    text-align: left;
    width: 80%;
    text-transform: capitalize;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    padding: 8px 20px;
    margin: 0;
    box-shadow: var(--shadow-dark);
    overflow: hidden;
  }

  .base-btn.extended {
    width: 80%;
  }

  .date {
    color: gray;
    text-transform: none;
  }
</style>
