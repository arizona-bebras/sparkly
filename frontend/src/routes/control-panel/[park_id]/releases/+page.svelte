<script lang="ts">
	import { page } from "$app/state";

	const parkId = $derived(page.params.park_id);
</script>

<svelte:head>
	<title>Релизы | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-2">
	<h1 class="text-2xl font-bold tracking-tight">Релизы</h1>
	<p class="text-sm text-muted-foreground">
		Управление релизами и версиями прошивок
	</p>
</div>

