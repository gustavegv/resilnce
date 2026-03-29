<script lang="ts">
  import Icon from '@iconify/svelte';
  import type { SessionMetaData } from '../routes/tracker/dbFetches';
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
    return date;
  }

  function timeAgo(date: Date): string {
    date as Date;
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
    sty = '65vw';

    setTimeout(() => {
      sty = 'calc(100vw - 2rem);';
    }, 3000);
  }
</script>

<div class="inner-cont alt">
  <div class="base-btn alt" style="width:{sty}">
    <button class="start-side" onclick={prs}>
      <p>{slug.name}</p>
      <p class="date">{timeAgo(slug.date ?? grd(3))}</p>
    </button>

    <button class="dots-cont" onclick={sideway}>
      <Icon icon={'uiw:more'} fill={'#fff'} width={16}></Icon>
    </button>
  </div>

  <div class="under-cont">
    <button class="icon-under edit" onclick={edt}>
      <Icon icon="material-symbols:edit-outline" width={24} fill={'#FFC107'} />
    </button>
    <button class="icon-under del" onclick={del}>
      <Icon icon="material-symbols:delete-outline-rounded" width={24} fill={'#dc3545'} />
    </button>
  </div>
</div>

<style>
  :root {
    --slug-color: var(--surface-low);
    --edit-under-color: var(--surface-middle);
    --slug-height: 4rem;
  }
  .start-side {
    margin: 0;
    padding: 0;
    border-radius: 0;
    box-shadow: none;
    text-align: left;
    height: var(--slug-height);
    display: flex;
    flex-direction: column;
    justify-content: space-evenly;
    background-color: var(--slug-color);
    padding-left: 1rem;
    width: 85%;
  }

  .dots-cont {
    margin: 0;
    padding: 0;
    border-radius: 0;
    background-color: var(--slug-color);

    box-shadow: none;
    display: flex;
    justify-content: center;
    align-items: center;
    touch-action: none;
    width: 15%;
    height: var(--slug-height);
  }

  .icon-under {
    padding: 0 0.8rem;
  }

  .under-cont {
    position: absolute;
    display: flex;
    flex-direction: row;
    width: 90vw;
    height: var(--slug-height);

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
    height: var(--slug-height);

    background-color: var(--edit-under-color);

    box-shadow: var(--shadow-dark);
  }

  .base-btn {
    text-transform: capitalize;
    box-shadow: var(--shadow-dark);
    display: flex;
    flex-direction: row;
    margin: 0;
    overflow: hidden;
    border: 1px solid var(--border);
  }

  .base-btn.alt {
    background-color: var(--slug-color);

    position: absolute;
    z-index: 4;
    height: var(--slug-height);
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
    font-size: 14px;
  }

  .date {
    color: var(--color-alt);
    text-transform: none;
    font-size: 12px;
  }
</style>
