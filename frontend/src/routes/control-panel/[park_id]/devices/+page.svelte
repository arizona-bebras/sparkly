<script lang="ts">
	import { page } from "$app/state";
	import { client } from "$proto";
	import { createQuery } from "@tanstack/svelte-query";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
	import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
	import DataTable from "$lib/components/ui/data-table/data-table.svelte";
	import { columns } from "../../../../lib/components/entities/device/columns.js";

	const parkId = $derived(page.params.park_id ?? "");

	const devicesQuery = createQuery(() => ({
		queryKey: ["devices", parkId],
		queryFn: () =>
			client.device.getAllDevices({
				parkId: parkId,
			}),
	}));
</script>

<svelte:head>
	<title>Устройства | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-6">
	<div
		class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between"
	>
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Устройства</h1>
			<p class="text-sm text-muted-foreground">
				Список и мониторинг подключенных устройств парка
			</p>
		</div>
		<Button
			variant="outline"
			size="sm"
			class="gap-2 self-start sm:self-auto"
			disabled={devicesQuery.isFetching}
			onclick={() => devicesQuery.refetch()}
		>
			<RefreshCwIcon
				class="size-4 {devicesQuery.isFetching ? 'animate-spin' : ''}"
			/>
			<span>Обновить</span>
		</Button>
	</div>

	{#if devicesQuery.isPending}
		<div class="flex flex-col gap-3">
			<Skeleton class="h-10 w-64 rounded-md" />
			<div class="rounded-xl border p-4 flex flex-col gap-3">
				{#each Array(5) as _}
					<Skeleton class="h-10 w-full" />
				{/each}
			</div>
		</div>
	{:else if devicesQuery.isError}
		<div
			class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive"
		>
			<CircleAlertIcon class="size-8" />
			<p class="font-medium">Ошибка загрузки данных</p>
			<p class="text-xs text-muted-foreground">
				{devicesQuery.error?.message}
			</p>
			<Button
				variant="outline"
				size="sm"
				onclick={() => devicesQuery.refetch()}
			>
				Повторить
			</Button>
		</div>
	{:else}
		<DataTable data={devicesQuery.data?.devices ?? []} {columns} />
	{/if}
</div>
