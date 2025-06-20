<script lang="ts">
import '../../app.css';

import { Button } from '$lib/components/ui/button'
  import { Label }  from '$lib/components/ui/label'
  import { Input }  from '$lib/components/ui/input'
  import { Checkbox }  from '$lib/components/ui/checkbox'

  import * as Card   from '$lib/components/ui/card'

  import { user } from './user';

  let username = '';
  let password = '';

  export function tempLogin(user:string, pass:string):boolean{
    return true
  }

  function handleSubmit(event: Event) {
    console.log("d")
    if (tempLogin(username, password)){
      handleLogin()
    }
  }

  function handleLogin() {
    if (!username) return;
    user.set(username);
    username = '';
  }

  function handleLogout() {
    user.set(null);
  }

let loggedIn = false


</script>


{#if $user}
<main class="main">
<Card.Root class="w-full max-w-sm bg-neutral-900 rounded-2xl p-6 space-y-4 shadow">
    <Card.Header class="border-b border-neutral-700">
      <Card.Title class="text-lg font-semibold text-white">
        Logged in as <strong>{$user}</strong>
      </Card.Title>
    </Card.Header>

    <Card.Footer class="flex justify-start">
      <Button variant="outline" onclick={handleLogout}>
        Logout
      </Button>
    </Card.Footer>
  </Card.Root>

  <!-- Settings Card -->
  <Card.Root class="w-full max-w-sm bg-neutral-900 rounded-2xl p-6 space-y-4 shadow">
    <Card.Header class="border-b border-neutral-700 pb-3">
      <Card.Title class="text-lg font-semibold text-white">
        Settings
      </Card.Title>
    </Card.Header>

    <Card.Content class="space-y-4">
      <div class="flex items-center space-x-3">
        <Checkbox id="cb1" />
        <label for="cb1" class="text-neutral-300 select-none">
          Autoincrease weight when rep-goal is hit
        </label>
      </div>
      <div class="flex items-center space-x-3">
        <Checkbox id="cb2" />
        <label for="cb2" class="text-neutral-300 select-none">
          Checkbox 2
        </label>
      </div>
    </Card.Content>
  </Card.Root>
  </main>

{:else}

<main class="main">
    <Card.Root class="w-full max-w-sm">
      <Card.Header>
        <Card.Title>Login to your account</Card.Title>
        <Card.Description>
          Enter your email below to login to your account
        </Card.Description>

      </Card.Header>
    
      <form onsubmit={handleSubmit}>
        <Card.Content>
          <div class="flex flex-col gap-6">
            <div class="grid gap-2">
              <Label for="text">Email</Label>
              <Input
                id="text"
                type="text"
                placeholder='"user1"'
                bind:value={username}
                required
              />
            </div>
            <div class="grid gap-2">
              <Label for="password">Password</Label>
              <Input
                id="password"
                type="password"
                bind:value={password}
                required
              />
            </div>
          </div>
        </Card.Content>
    
        <Card.Footer class="flex-col gap-2 mar">
          <Button type="submit" class="w-full gap-4">Login</Button>
        </Card.Footer>
      </form>
    </Card.Root>
</main>
{/if}

<style>
    .main{
        margin-top: 8rem;
        display: flex;
        justify-content: baseline;
        align-items: center;
        height: 100vh;
        flex-direction: column;
    }

    :global(.mar){
      margin-top: 1rem;
    }
</style>