<script>
  let { children } = $props();
</script>

<div class="head">
  <a href="/"><h1>resilnce</h1></a>

  <a class="nav">&#9776;</a>
</div>

{@render children()}

<style>
  .head {
    position: relative;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    background-color: var(--color-background);
    box-shadow: rgba(0, 0, 0, 0.35) 0px 5px 15px;
    z-index: 2;
    overflow: visible;
  }

  .invis {
    height: 4rem;
  }

  .nav {
    margin: 0 10px;
    color: rgb(255, 255, 255);
    font-size: 30px;
  }

  a {
    color: inherit;
    text-decoration: none;
  }

  h1 {
    color: var(--color-white);
    margin: 10px 10px;
  }
</style>
