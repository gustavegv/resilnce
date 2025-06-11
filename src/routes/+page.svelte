<script lang="ts">
  import '../app.css';
import { goto } from '$app/navigation';

function route(destination: string) {
    switch(destination){
        case "start" : {
            goto('/tracker')
            break;
        }
        case "new" : {
            break;
        }

        case "cont" : {
            break;
        }
    }
}

</script>

<div class="btn-container">
    <button class="base-btn sesh" on:click={() => route("start")}>Start new session</button>
    <button class="base-btn sesh" on:click={() => route("new")}>Create new session</button>
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