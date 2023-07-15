<script>
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';

	let canvas;

	onMount(() => {
		const ctx = canvas.getContext('2d');
		let frame;

		(function loop() {
			frame = requestAnimationFrame(loop);

			const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);

			for (let p = 0; p < imageData.data.length; p += 4) {
				const i = p / 4;
				const x = i % canvas.width;
				const y = (i / canvas.height) >>> 0;

				const t = window.performance.now();

				const r = 64 + (128 * x) / canvas.width + 64 * Math.sin(t / 1000);
				const g = 64 + (128 * y) / canvas.height + 64 * Math.cos(t / 1400);
				const b = 128;

				imageData.data[p + 0] = r;
				imageData.data[p + 1] = g;
				imageData.data[p + 2] = b;
				imageData.data[p + 3] = 255;
			}

			ctx.putImageData(imageData, 0, 0);
		})();

		return () => {
			cancelAnimationFrame(frame);
		};
	});
</script>

<main class="bg-zinc-800 h-screen items-center justify-center text-center flex">
	<header class="absolute top-0 left-0 w-full h-8 bg-zinc-700 flex-grow">
		<div class="">
			<h1 class="text-xl text-white">GoSvelt</h1>
		</div>
	</header>
	<canvas transition:fly={{ y:200, duration: 800 }} class="w-60" bind:this={canvas} width={32} height={32} />
</main>

<style>
    @tailwind base;
	@tailwind components;
	@tailwind utilities;
    
	canvas {
		-webkit-mask: url(/svelte_logo) 50% 50% no-repeat;
		mask: url(/svelte_logo) 50% 50% no-repeat;
	}
</style>
