<script lang="ts">
	import { page } from "$app/state";
	import DeviceList from "$lib/components/entities/device/DeviceList.svelte";

	const parkId = $derived(page.params.park_id ?? "");
</script>

<svelte:head>
  <title>Устройства | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-1">
		<h1 class="text-2xl font-bold tracking-tight">Устройства</h1>
		<p class="text-sm text-muted-foreground">
			Список и мониторинг подключенных устройств парка
		</p>
	</div>

	<DeviceList {parkId} />
</div>
