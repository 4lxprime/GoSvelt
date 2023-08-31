<script>
    import { writable } from 'svelte/store';	

    let messages = writable([])

    let sse = new EventSource("/sse")

    sse.onmessage = (e) => messages.update(cm => [...cm, e.data])
    sse.onerror = () => sse.close()

</script>

<main>
    {#each $messages as msg}
        <p>{msg}</p>
    {/each}
</main>