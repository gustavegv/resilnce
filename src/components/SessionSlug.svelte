<script lang="ts">
  import Icon from '@iconify/svelte';
  import type { SessionMetaData } from '$lib/firebaseDataHandler';

  let {
    onEdit,
    onPress,
    onDel,
    slug,
  }: {
    onEdit: () => void;
    onPress: () => void;
    onDel: () => void;
    slug: SessionMetaData;
  } = $props();

  function handleClick() {
    if (onEdit) onEdit();
    if (onPress) onPress();
    if (onDel) onDel();
  }


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
</script>

<div class="inner-cont">
  <button class="base-btn extended sesh" onclick={handleClick}>
    <p>{slug.id}</p>
    <p class="date">{timeAgo(slug.date ?? grd(3))}</p>
  </button>

  <Icon
    style="margin:0 2rem; position:fixed; right:0;"
    onclick={handleClick}
    icon="material-symbols:edit-outline"
    width="24"
    height="24"
  />
</div>

<style>
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
