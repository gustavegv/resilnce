<script>
  // no changes needed here
</script>

<div class="spacer"></div>
<!-- HEADER SKELETON -->
<div class="skeleton-header">
  <div class="skeleton skeleton-title"></div>
  <div class="skeleton skeleton-subtitle"></div>
</div>

<!-- REPS BLOCKS SKELETON -->
{#each Array(3) as _, index}
  <div class="skeleton skeleton-block"></div>
{/each}

<!-- BUTTON SKELETON -->

<style>
  /* SHIMMER ANIMATION */

  :root {
    --main: #343434;
    --alt: #383838;
  }
  @keyframes shimmer {
    0% {
      background-position: -1000px 0;
    }
    100% {
      background-position: 1000px 0;
    }
  }

  .skeleton {
    position: relative;
    overflow: hidden;
    background-image: linear-gradient(135deg, var(--main) 25%, var(--alt) 50%, var(--main) 75%);
    background-size: 200% 200%;
    animation: shimmer 5.5s infinite linear;
    animation-delay: var(--delay, 0s);
    border-radius: 10px;
  }

  /* HEADER SKELETON */
  .skeleton-header {
    width: 80%;
    margin-top: 2.5rem;
    text-align: center;
    padding: 1rem 0;
  }
  .skeleton-title {
    width: 100%;
    height: 3em;
    margin: 1em auto;
    border-radius: inherit;
    background-color: var(--main);
  }
  .skeleton-subtitle {
    width: 50%;
    height: 2em;
    margin: 0.25em auto;
    border-radius: inherit;
    background-color: var(--main);
  }

  /* REPS BLOCK SKELETON */
  .skeleton-block {
    width: 80%;
    height: 6em;
    margin: 0.5rem auto;
  }
</style>
