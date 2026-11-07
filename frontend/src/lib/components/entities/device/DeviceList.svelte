<script lang="ts">
	import { goto } from "$app/navigation";
	import { page } from "$app/state";
	import { client } from "$proto";
	import { createQuery } from "@tanstack/svelte-query";
	import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import DataTable from "$lib/components/ui/data-table/data-table.svelte";
	import { columns } from "./columns.js";
	import { getMockDevices } from "./mock.js";
	import DeviceFilters from "./DeviceFilters.svelte";

	let { parkId }: { parkId: string } = $props();

	function getUrlGroups(): string[] {
		const groups = page.url.searchParams.getAll("group");
		if (groups.length > 0) return groups;
		const single =
			page.url.searchParams.get("group_id") ??
			page.url.searchParams.get("group");
		return single ? [single] : [];
	}

	let selectedGroupIds = $state<string[]>(getUrlGroups());
	let selectedBuildIds = $state<string[]>([]);
	let searchQuery = $state("");
	let lastUrlKey = $state(getUrlGroups().slice().sort().join(","));

	$effect(() => {
		const fromUrl = getUrlGroups();
		const currentUrlKey = fromUrl.slice().sort().join(",");

		if (currentUrlKey !== lastUrlKey) {
			lastUrlKey = currentUrlKey;
			selectedGroupIds = fromUrl;
			return;
		}

		const stateKey = selectedGroupIds.slice().sort().join(",");
		if (stateKey !== currentUrlKey) {
			lastUrlKey = stateKey;
			const url = new URL(page.url);
			url.searchParams.delete("group");
			url.searchParams.delete("group_id");
			for (const id of selectedGroupIds) {
				url.searchParams.append("group", id);
			}
			goto(url.toString(), {
				replaceState: true,
				noScroll: true,
				keepFocus: true,
			});
		}
	});

	const defaultGroups = [
		{ id: "grp-1", name: "Кассы" },
		{ id: "grp-2", name: "Турникеты" },
		{ id: "grp-3", name: "Инфокиоски" },
		{ id: "grp-4", name: "Шлагбаумы" },
	];

	const defaultBuilds = [
		{ id: "bld-01", name: "v1.4.2 (Latest)" },
		{ id: "bld-02", name: "v1.4.0 (Stable)" },
		{ id: "bld-03", name: "v1.3.8 (LTS)" },
	];

	const groupsQuery = createQuery(() => ({
		queryKey: ["groups", parkId],
		queryFn: async () => {
			try {
				const response = await client.park.getGroups({ parkId });
				if (response.groups && response.groups.length > 0) {
					return response;
				}
			} catch {}
			return {
				groups: defaultGroups.map((g) => ({
					id: g.id,
					name: g.name,
					description: "",
					deviceCount: 0,
					parkId,
				})),
			};
		},
		enabled: !!parkId,
	}));

	const availableGroups = $derived.by(() => {
		const raw = groupsQuery.data;
		const list = Array.isArray(raw)
			? raw
			: Array.isArray(raw?.groups)
				? raw.groups
				: [];
		if (list.length > 0) {
			return list.map((g) => ({
				id: g.id,
				name: g.name,
			}));
		}
		return defaultGroups;
	});

	const devicesQuery = createQuery(() => ({
		queryKey: ["devices", parkId],
		queryFn: async () => {
			try {
				const response = await client.device.getAllDevices({
					parkId,
				});
				if (response.devices.length > 0) return response;
			} catch {}
			return {
				devices: getMockDevices(parkId),
				pagination: {
					totalItems: getMockDevices(parkId).length,
					totalPages: 1,
					currentPage: 1,
				},
			};
		},
		enabled: !!parkId,
	}));

	const filteredDevices = $derived.by(() => {
		const all = devicesQuery.data?.devices ?? [];
		const query = searchQuery.trim().toLowerCase();

		return all.filter((device) => {
			if (query.length > 0) {
				const matchName = device.name.toLowerCase().includes(query);
				const matchId = device.id.toLowerCase().includes(query);
				if (!matchName && !matchId) return false;
			}

			if (selectedGroupIds.length > 0) {
				const matchesGroup = device.groupIds.some((id) =>
					selectedGroupIds.includes(id)
				);
				if (!matchesGroup) return false;
			}

			return true;
		});
	});
</script>

{#snippet filterToolbar()}
	<DeviceFilters
		bind:selectedGroupIds
		bind:selectedBuildIds
		bind:searchQuery
		groups={availableGroups}
		builds={defaultBuilds}
	/>
{/snippet}

<div class="flex flex-col gap-4">
	<div class="flex items-center justify-end">
		<Button
			variant="outline"
			size="sm"
			class="gap-2"
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
		<DataTable
			data={filteredDevices}
			{columns}
			searchColumn="name"
			searchPlaceholder="Поиск по имени или ID..."
			emptyMessage="Устройства не найдены."
			bind:searchValue={searchQuery}
			toolbar={filterToolbar}
		/>
	{/if}
</div>
