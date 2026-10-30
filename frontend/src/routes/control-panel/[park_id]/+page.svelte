<script lang="ts">
	import { page } from "$app/state";

	const parkId = $derived(page.params.park_id);
</script>

<svelte:head>
	<title>Панель управления | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-2">
	<h1 class="text-2xl font-bold tracking-tight">Панель управления</h1>
	<p class="text-sm text-muted-foreground">
		Сводка по парку
	</p>
</div>
