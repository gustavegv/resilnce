<script lang="ts">
  import { goto } from '$app/navigation';
  import { getAllSessionMeta } from '$lib/firebaseDataHandler';
  import { onMount } from 'svelte';

  type SessionMetaData = {
    id: string;
    name: string;
  };

  // TODO: Put in db or smt
  let slugs: SessionMetaData[];

  onMount(async () => {
    slugs = await getAllSessionMeta('user1');
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

  function startSession(id: string) {
    goto(`/tracker/${id}`);
  }
</script>

<div class="main">
  <h2>Sessions:</h2>
  <hr />
  <div class="btn-container">
    {#each slugs as slug}
      <button class="base-btn sesh" onclick={() => startSession(slug.id)}>
        <p>{slug.id}</p>
        <p class="date">{timeAgo(grd(7))}</p>
      </button>
    {/each}
  </div>
</div>

<style>
  .main {
    box-sizing: border-box;
    padding: 5rem 1rem;
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
    align-items: center;
    box-sizing: border-box;
    width: 100%;
    text-align: left;
  }

  .base-btn {
    text-align: left;
    width: 90%;
    text-transform: capitalize;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
  }

  .date {
    color: gray;
    text-transform: none;
  }
</style>
