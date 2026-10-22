<script lang="ts">
	import { page } from "$app/state";

	const parkId = $derived(page.params.park_id);
</script>

<svelte:head>
	<title>Устройства | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-2">
	<h1 class="text-2xl font-bold tracking-tight">Устройства</h1>
	<p class="text-sm text-muted-foreground">
		Список устройств парка #{parkId}
	</p>
</div>

