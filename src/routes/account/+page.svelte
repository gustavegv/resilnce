<script lang="ts">
import '../../app.css';

import { Button } from '$lib/components/ui/button'
  import { Label }  from '$lib/components/ui/label'
  import { Input }  from '$lib/components/ui/input'
  import * as Card   from '$lib/components/ui/card'

  let email = '';
  let password = '';

  function handleSubmit(event: Event) {
    console.log("d")
    // Handle form logic here
  }

let loggedIn = false


</script>


{#if loggedIn}
<p>logged in</p>
{:else}

<main class="main">
    <Card.Root class="w-full max-w-sm">
      <Card.Header>
        <Card.Title>Login to your account</Card.Title>
        <Card.Description>
          Enter your email below to login to your account
        </Card.Description>
        <Card.Action>
          <Button variant="link">Sign Up</Button>
        </Card.Action>
      </Card.Header>
    
      <form on:submit|preventDefault={handleSubmit}>
        <Card.Content>
          <div class="flex flex-col gap-6">
            <div class="grid gap-2">
              <Label for="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="m@example.com"
                bind:value={email}
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
    
        <Card.Footer class="flex-col gap-2">
          <Button type="submit" class="w-full">Login</Button>
          <Button variant="outline" class="w-full">
            Login with Google
          </Button>
        </Card.Footer>
      </form>
    </Card.Root>
</main>
{/if}

<style>
    .main{
        margin-top: 8rem;
        display: flex;
        justify-content: center;
        align-items: baseline;
        height: 100vh;
    }
</style>