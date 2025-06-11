<script>
  let { children } = $props();
</script>

<div class="head">
  <h1>resilnce</h1>
  <nav>
    <a href="/">Home</a>
    <a href="/tracker">Tracker</a>
    <a href="/settings">Settings</a>
  </nav>
</div>

{@render children()}

<style>
  .head {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    box-shadow: rgba(100, 100, 111, 0.2) 0px 7px 29px 0px;
  }

  nav {
    margin: 0 10px;
    color: black;
  }

  a {
    color: inherit;
  }

  h1 {
    color: var(--color-secondary);
    margin: 10px 10px;
  }
</style>
