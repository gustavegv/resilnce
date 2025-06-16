<script lang="ts">
  let { size }: { size?: number } = $props();

  let sz = size ?? 0;
</script>

<svg
  class="svg-color"
  xmlns="http://www.w3.org/2000/svg"
  width="1080"
  height={400 - sz * 50}
  viewBox="0 {sz * 10} 1080 {400 - sz * 50}"
  xml:space="preserve"
>
  <desc>Created with Fabric.js 5.2.4</desc>
  <defs></defs>
  <g transform="matrix(17.06 0 0 2.86 549.75 95.62)">
    <rect
      style="stroke: currentColor; stroke-width: 0; fill: currentColor;"
      x="-33.085"
      y="-33.085"
      width="66.17"
      height="55"
    ></rect>
  </g>
  <g transform="matrix(5.2 0 0 {4.46 - sz * 0.5} 540 150)">
    <circle
      style="stroke: currentColor; stroke-width: 0; fill: currentColor;"
      cx="0"
      cy="0"
      r={-2 * sz + 30}
    ></circle>
  </g>
</svg>

<style>
  .svg-color {
    color: var(--color-background);
    width: inherit;
    height: inherit;
    filter: drop-shadow(4px 4px 10px rgba(0, 0, 0, 0.4));
  }
</style>
