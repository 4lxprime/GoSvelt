<script>
    import { writable } from 'svelte/store';	

    let messages = writable([]);

    let sse = new EventSource("/sse")
        .onmessage = (e)=>{
            console.log(`message: ${e.data}`);
            messages.update(cm => [...cm, e.data])
        }

    sse.onerror = (e)=>{
        console.log(`error ${e}`)
    }

</script>

<main>
    {#each $messages as msg}
        <p>{msg}</p>
    {/each}
</main>