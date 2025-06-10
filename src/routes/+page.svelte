<script lang="ts">
  import '../app.css';
import { goto } from '$app/navigation';

function route(destination: string) {
    switch(destination){
        case "push" : {
            goto('/tracker')
            break;
        }
        case "pull" : {
            break;
        }
        case "legs" : {
            break;
        }
        case "cont" : {
            break;
        }
    }
}

</script>

<div class="btn-container">
    <button class="base-btn sesh" on:click={() => route("push")}>Push</button>
    <button class="base-btn sesh" on:click={() => route("pull")}>Pull</button>
    <button class="base-btn sesh" on:click={() => route("legs")}>Legs</button>
    <button class="base-btn alt" on:click={() => route("cont")}>Continue workout</button>
    
</div>

<style>

.btn-container{
    display: flex;
    flex-direction: column;
    align-items: center;
}

.base-btn{
    margin: 10px;
    padding: 10px 40px;

}

.base-btn.sesh{
    background-color: var(--color-secondary);
}

.base-btn.alt{
    background-color: var(--color-alt);
    color: var(--color-secondary);


}

</style>