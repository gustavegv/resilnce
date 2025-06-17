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

  let sty = $state('90%');

  function sideway() {
    sty = '60%';

    setTimeout(() => {
      sty = '90%';
    }, 3000);
  }
</script>

<!-- <div class="inner-cont">
  <button class="base-btn extended sesh" onclick={prs}>
    <p>{slug.id}</p>
    <p class="date">{timeAgo(slug.date ?? grd(3))}</p>
  </button>

  <Icon
    style="margin:0 2rem; position:fixed; right:0;"
    onclick={edt}
    icon="material-symbols:edit-outline"
    width="24"
    height="24"
  />
</div> -->

<div class="inner-cont alt">
  <button class="base-btn alt" style="width:{sty}" onclick={prs}>
    <div>
      <p>{slug.id}</p>
      <p class="date">{timeAgo(slug.date ?? grd(3))}</p>
    </div>

    <div class="dots-cont">
      <Icon onclick={sideway} icon={'uiw:more'} fill={'#fff'} width={'20px'}></Icon>
    </div>
  </button>

  <div class="under-cont"></div>

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
  .dots-cont {
    padding: 1rem 0.5rem 1rem 1rem;
    touch-action: none;
  }

  .icon-under {
    padding: 2rem 1rem;
  }

  .under-cont {
    display: flex;
    flex-direction: row;
    width: 33%;
    height: 5rem;
    align-items: center;
    justify-content: space-between;
  }

  .inner-cont {
    background-color: var(--color-sec-dark);
    display: flex;
    flex-direction: row;
    width: 97%;
    align-items: center;
    justify-content: space-between;
    border-radius: 10px;
    margin: 0.5rem;
    overflow: hidden;
  }

  .base-btn {
    text-align: left;
    text-transform: capitalize;
    box-shadow: var(--shadow-dark);

    position: relative;
    display: flex;
    flex-direction: row;
    justify-content: space-between;

    padding: 8px 20px;
    margin: 0;
    overflow: hidden;
  }

  .inner-cont.alt {
    height: 5rem;
    background-color: black;
    border-radius: 10px;
  }

  .base-btn.alt {
    background-color: var(--color-secondary);
    width: var(--thing);
    position: absolute;
    width: 90%;
    z-index: 4;
    height: 5rem;
    border-radius: 10px;
    align-items: center;
    opacity: 1;
  }

  p {
    padding: 0;
    margin: 0.5rem 0;
    font-size: 16px;
  }

  .date {
    color: gray;
    text-transform: none;
  }
</style>
