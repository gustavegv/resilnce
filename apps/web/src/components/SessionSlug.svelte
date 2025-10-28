<script lang="ts">
  import Icon from '@iconify/svelte';
  import type { SessionMetaData } from '$lib/firebaseDataHandler';
  import { fade, fly, slide } from 'svelte/transition';

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

  function prs() {
    if (onPress) onPress();
  }
  function edt() {
    if (onEdit) onEdit();
  }
  function del() {
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
    date as Date;
    console.log('timeag', date);
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

  let sty = $state('calc(100vw - 2rem);');

  function sideway() {
    sty = '60vw';

    setTimeout(() => {
      sty = 'calc(100vw - 2rem);';
    }, 3000);
  }
</script>

<div class="inner-cont alt">
  <div class="base-btn alt" style="width:{sty}">
    <button class="start-side buttonClass" onclick={prs}>
      <p>{slug.id}</p>
      <p class="date">{timeAgo(slug.date ?? grd(3))}</p>
    </button>

    <button class="dots-cont buttonClass" onclick={sideway}>
      <Icon icon={'uiw:more'} fill={'#fff'} width={'20px'}></Icon>
    </button>
  </div>

  <div class="under-cont">
    <div class="icon-under edit">
      <Icon onclick={edt} icon="material-symbols:edit-outline" width="32" fill={'#FFC107'} />
    </div>
    <div class="icon-under del">
      <Icon
        onclick={del}
        icon="material-symbols:delete-outline-rounded"
        width="32"
        fill={'#dc3545'}
      />
    </div>
  </div>
</div>

<style>
  .start-side {
    margin: 0;
    padding: 0;
    border-radius: 0;
    box-shadow: none;
    text-align: left;
    height: 5rem;
    background-color: none;
    padding-left: 1rem;
    width: 80%;
  }

  .dots-cont {
    margin: 0;
    padding: 0;
    border-radius: 0;
    background-color: none;

    box-shadow: none;
    display: flex;
    justify-content: center;
    align-items: center;
    touch-action: none;
    width: 20%;
    height: 5rem;
  }

  .icon-under {
    padding: 0 0.8rem;
  }

  .under-cont {
    position: absolute;
    display: flex;
    flex-direction: row;
    width: 90vw;
    height: 5rem;
    align-items: center;
    justify-content: end;
    touch-action: pan-y;
  }

  .inner-cont {
    position: relative;
    display: flex;
    flex-direction: row;
    width: 100%;
    align-items: center;
    justify-content: cen;
    border-radius: 10px;
    overflow: hidden;
  }

  .inner-cont.alt {
    height: 5rem;
    background-color: black;
    box-shadow: var(--shadow-dark);
  }

  .base-btn {
    text-transform: capitalize;
    box-shadow: var(--shadow-dark);
    display: flex;
    flex-direction: row;

    margin: 0;
    overflow: hidden;
  }

  .base-btn.alt {
    background-color: var(--color-secondary);
    position: absolute;
    z-index: 4;
    height: 5rem;
    border-radius: 10px;
    align-items: center;
    opacity: 1;
    justify-content: space-between;
    padding: 0;
    transition: all 0.3s;
    opacity: 1;
  }

  p {
    padding: 0;
    margin: 0.5rem 0;
    font-size: 16px;
  }

  .date {
    color: var(--color-alt);
    text-transform: none;
  }
</style>
