<script lang="ts">
    import { base } from "$app/paths";

  let { size }: { size?: number } = $props();

  let sz: number = size ?? 5;
</script>

<a class="cont" href="{base}/">
  <h1 style="font-size:{sz * 5}px">resilnce</h1>
  <img style="width:{sz}rem" class="ba" src="/icons/logbottom.png" alt="log" />
</a>

<style>
  .ba {
    width: 5rem;
  }
  .cont {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
  }
  a {
    color: inherit;
    text-decoration: none;
  }

  h1 {
    color: var(--color-white);
    margin: 0 10px;
    font-weight: 700;
    -webkit-text-stroke: 0.6px rgb(255, 255, 255);
  }
</style>
