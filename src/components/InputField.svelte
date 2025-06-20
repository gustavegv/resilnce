<script lang="ts">
  let {
    label = 'Label',
    value = $bindable(''),
    type = '',
  } = $props<{ label: string; value: string; type: string }>();

  let typet = type;
</script>

<label class="inp">
  <input type={typet} min={1} placeholder=" " bind:value />
  <span class="label">{label}</span>
  <span class="focus-bg"></span>
</label>

<style>
  :root {
    --small: #b1ff4a;
    --big: var(--color-secondary);
    --action: #535353;
  }

  .inp {
    position: relative;
    width: 100%;
    max-width: 280px;
    border-radius: 3px;
    overflow: hidden;
  }

  .inp .label {
    position: absolute;
    top: 20px;
    left: 12px;
    font-size: 16px;
    color: var(--dark);
    font-weight: 500;
    transform-origin: 0 0;
    transform: translate3d(0, 0, 0);
    transition: all 0.2s ease;
    pointer-events: none;
  }

  .inp .focus-bg {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: var(--action);
    z-index: -1;
    transform: scaleX(0);
    transform-origin: left;
  }

  input {
    -webkit-appearance: none;
    appearance: none;
    width: 100%;
    border: 0;
    font-family: inherit;
    padding: 16px 12px 0 12px;
    height: 56px;
    font-size: 16px;
    font-weight: 400;
    background: var(--big);
    box-shadow: inset 0 -1px 0 rgba(51, 51, 51, 0.3);
    color: var(--dark);
    transition: all 0.15s ease;
  }

  input:hover {
    background: var(--action);
    box-shadow: inset 0 -1px 0 rgba(51, 51, 51, 0.5);
  }

  input:not(:placeholder-shown) + .label {
    color: var(--dark);
    transform: translate3d(0, -12px, 0) scale(0.75);
  }

  input:focus {
    background: rgba(51, 51, 51, 0.05);
    outline: none;
    box-shadow: inset 0 -2px 0 var(--primary);
  }

  input:focus + .label {
    color: var(--dark);
    transform: translate3d(0, -12px, 0) scale(0.75);
  }

  input:focus + .label + .focus-bg {
    transform: scaleX(1);
    transition: all 0.1s ease;
  }
</style>
