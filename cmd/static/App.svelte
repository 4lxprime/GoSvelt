<script>
    import Gosvelt from "./gosvelt.svelte";

    let name = "world";
</script>

<main>
    <h1>Hello {name}</h1>
    <Gosvelt/>
</main>

<style>
    main {
        align-items: center;
        justify-content: center;
    }

    h1 {
        text-transform: uppercase;
        color: aqua;
    }
</style>