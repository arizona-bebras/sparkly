<script lang="ts">
	import CreateParkDialog from "@/components/entities/park/ParkDialog.svelte";
	import ParkList from "@/components/entities/park/ParkList.svelte";
	import { Button } from "$lib/components/ui/button/index.js";

	let isCreateParkOpen = $state(false);
</script>

<svelte:head>
	<title>Sparkly</title>
</svelte:head>

<main
	class="container mx-auto flex min-h-screen flex-col items-center justify-center gap-6 p-4"
>
	<h1 class="text-3xl font-bold tracking-tight">Sparkly</h1>

	<CreateParkDialog bind:open={isCreateParkOpen}>
		<p>Создать парк</p>
	</CreateParkDialog>

	<ParkList />
</main>
