<script lang="ts">
  import ChartLineDefault, {
    type ChartData,
  } from '$lib/components/ui/chart/chart-line-default.svelte';
  import * as Card from '$lib/components/ui/card/index.js';
  import { onMount } from 'svelte';
  import Input from '$lib/components/ui/input/input.svelte';
  import * as Tabs from '$lib/components/ui/tabs/index.js';
  import type { ExerciseInfo } from '../tracker/dbFetches';
  import { user } from '../account/user';
  import { get } from 'svelte/store';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';

  let isAuth = $state(false);

  onMount(async () => {
    if (!get(user)) {
      goto(resolve(`/account`));
    } else {
      console.log('User auth');
      isAuth = true;
    }
  });

  let formattedData: ChartData[] = $state([]);
  let sessionInput: string = $state('Link');
  let exDa: ExerciseInfo[] = $state([]);
  let uID = get(user) ?? '';

  /*   async function as(u: string, s: string) {
    exDa = await getOrderedExercises(u, s);
    const cD = await calling(s);
    formattedData = cD;
  } */
</script>

<br />
<br />
<br />
<br />
<br />
{#if isAuth}
  <main class="flex flex-col items-center justify-center gap-4 px-4">
    <div class="grid grid-cols-2 items-center">
      <Input bind:value={sessionInput} class="" placeholder="Session name" />
      <!-- <button class="buttonClass m-4" onclick={() => as(uID, sessionInput)}>Get session</button> -->
    </div>

    <Tabs.Root value="account" class="w-[400px]">
      <Tabs.List>
        {#each exDa as ex}
          <Tabs.Trigger value={ex.name}>{ex.name}</Tabs.Trigger>
        {/each}
      </Tabs.List>
      {#each exDa as ex}
        <Tabs.Content value={ex.name}>
          <Card.Root class="w-full">
            <ChartLineDefault
              title="Latest session"
              desc="One Rep Max Progression layered with the current weight"
              data={formattedData}
            />
          </Card.Root>
        </Tabs.Content>
      {/each}
    </Tabs.Root>
  </main>
{/if}
