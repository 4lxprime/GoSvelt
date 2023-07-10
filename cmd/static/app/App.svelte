<script>
    import Gosvelt from "../gosvelt/gosvelt.svelte";

    let name = "world";
</script>

<main class="bg-zinc-50 justify-center items-center">
    <h1 class="p-2">Hello {name}</h1>
    <Gosvelt/>
</main>

<style>
    @tailwind base;
	@tailwind components;
	@tailwind utilities;

    main {
        align-items: center;
        justify-content: center;
    }

    h1 {
        text-transform: uppercase;
        color: aqua;
    }
</style>