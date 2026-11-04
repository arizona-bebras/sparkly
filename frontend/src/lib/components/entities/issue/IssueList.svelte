<script lang="ts">
	import type { DateRange } from "bits-ui";
	import { getLocalTimeZone } from "@internationalized/date";
	import { client, type IssueSeverity } from "$proto";
	import { createQuery } from "@tanstack/svelte-query";
	import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import DataTable from "$lib/components/ui/data-table/data-table.svelte";
	import { columns } from "./columns.js";
	import { getMockIssues } from "./mock.js";
	import IssueFilters from "./IssueFilters.svelte";

	let { parkId }: { parkId: string } = $props();

	let selectedStatus = $state<"all" | "open" | "resolved">("all");
	let selectedSeverities = $state<IssueSeverity[]>([]);
	let dateRange = $state<DateRange | undefined>(undefined);
	let searchQuery = $state("");

	const issuesQuery = createQuery(() => ({
		queryKey: ["issues", parkId],
		queryFn: async () => {
			try {
				const response = await client.issue.getAllIssues({
					parkId,
				});
				if (response.issues.length > 0) return response;
			} catch {}
			return {
				issues: getMockIssues(parkId),
				pagination: {
					totalItems: getMockIssues(parkId).length,
					totalPages: 1,
					currentPage: 1,
				},
			};
		},
		enabled: !!parkId,
	}));

	const filteredIssues = $derived.by(() => {
		const all = issuesQuery.data?.issues ?? [];
		return all.filter((issue) => {
			if (selectedStatus === "open") {
				const isResolved =
					!!issue.resolvedAt && Number(issue.resolvedAt.seconds) > 0;
				if (isResolved) return false;
			} else if (selectedStatus === "resolved") {
				const isResolved =
					!!issue.resolvedAt && Number(issue.resolvedAt.seconds) > 0;
				if (!isResolved) return false;
			}

			if (selectedSeverities.length > 0) {
				if (!selectedSeverities.includes(issue.severity)) return false;
			}

			if (dateRange?.start) {
				const startMs = dateRange.start
					.toDate(getLocalTimeZone())
					.setHours(0, 0, 0, 0);
				const endMs = dateRange.end
					? dateRange.end
							.toDate(getLocalTimeZone())
							.setHours(23, 59, 59, 999)
					: dateRange.start
							.toDate(getLocalTimeZone())
							.setHours(23, 59, 59, 999);
				const firstMs =
					issue.firstAt && Number(issue.firstAt.seconds) > 0
						? Number(issue.firstAt.seconds) * 1000
						: 0;
				const lastMs =
					issue.lastAt && Number(issue.lastAt.seconds) > 0
						? Number(issue.lastAt.seconds) * 1000
						: firstMs;
				if (lastMs < startMs || firstMs > endMs) return false;
			}

			return true;
		});
	});
</script>

{#snippet filterToolbar()}
	<IssueFilters
		bind:selectedStatus
		bind:selectedSeverities
		bind:dateRange
		bind:searchQuery
	/>
{/snippet}

<div class="flex flex-col gap-4">
	<div class="flex items-center justify-end">
		<Button
			variant="outline"
			size="sm"
			class="gap-2"
			disabled={issuesQuery.isFetching}
			onclick={() => issuesQuery.refetch()}
		>
			<RefreshCwIcon
				class="size-4 {issuesQuery.isFetching ? 'animate-spin' : ''}"
			/>
			<span>Обновить</span>
		</Button>
	</div>

	{#if issuesQuery.isPending}
		<div class="flex flex-col gap-3">
			<Skeleton class="h-10 w-64 rounded-md" />
			<div class="rounded-xl border p-4 flex flex-col gap-3">
				{#each Array(5) as _}
					<Skeleton class="h-10 w-full" />
				{/each}
			</div>
		</div>
	{:else if issuesQuery.isError}
		<div
			class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive"
		>
			<CircleAlertIcon class="size-8" />
			<p class="font-medium">Ошибка загрузки данных</p>
			<p class="text-xs text-muted-foreground">
				{issuesQuery.error?.message}
			</p>
			<Button
				variant="outline"
				size="sm"
				onclick={() => issuesQuery.refetch()}
			>
				Повторить
			</Button>
		</div>
	{:else}
		<DataTable
			data={filteredIssues}
			{columns}
			searchColumn="title"
			searchPlaceholder="Поиск по проблемам..."
			emptyMessage="Проблемы не найдены."
			bind:searchValue={searchQuery}
			toolbar={filterToolbar}
		/>
	{/if}
</div>
