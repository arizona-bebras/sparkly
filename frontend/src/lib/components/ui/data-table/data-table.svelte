<script lang="ts" generics="TData extends RowData">
	import {
		type ColumnDef,
		type RowData,
		createTable,
		FlexRender,
	} from "@tanstack/svelte-table";
	import type { Snippet } from "svelte";
	import * as Table from "$lib/components/ui/table/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import SearchIcon from "@lucide/svelte/icons/search";
	import ChevronLeftIcon from "@lucide/svelte/icons/chevron-left";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import { features, type DataTableFeatures } from "./data-table-features.js";

	type DataTableProps<TData extends RowData> = {
		columns: ColumnDef<DataTableFeatures, TData>[];
		data: TData[];
		searchColumn?: string;
		searchPlaceholder?: string;
		emptyMessage?: string;
		searchValue?: string;
		toolbar?: Snippet;
	};

	let {
		data,
		columns,
		searchColumn = "name",
		searchPlaceholder = "Поиск по названию...",
		emptyMessage = "Устройства не найдены.",
		searchValue = $bindable(""),
		toolbar,
	}: DataTableProps<TData> = $props();

	const table = createTable({
		features,
		get data() {
			return data;
		},
		get columns() {
			return columns;
		},
	});

	$effect(() => {
		table.getColumn(searchColumn)?.setFilterValue(searchValue);
	});

	const pagination = $derived(table.atoms.pagination.get());
</script>

<div class="flex flex-col gap-4">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="flex flex-1 flex-wrap items-center gap-2">
			<div class="relative w-full sm:w-64">
				<SearchIcon class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground pointer-events-none" />
				<Input
					placeholder={searchPlaceholder}
					bind:value={searchValue}
					class="pl-9 h-9"
				/>
			</div>

			{@render toolbar?.()}
		</div>

		<div class="text-xs text-muted-foreground whitespace-nowrap">
			Всего строк: <span class="font-medium text-foreground">{table.getFilteredRowModel().rows.length}</span>
		</div>
	</div>

	<div class="rounded-xl border bg-card text-card-foreground shadow-xs overflow-hidden">
		<Table.Root>
			<Table.Header>
				{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
					<Table.Row class="bg-muted/40 hover:bg-muted/40">
						{#each headerGroup.headers as header (header.id)}
							<Table.Head colspan={header.colSpan}>
								{#if !header.isPlaceholder}
									<FlexRender {header} />
								{/if}
							</Table.Head>
						{/each}
					</Table.Row>
				{/each}
			</Table.Header>
			<Table.Body>
				{#each table.getRowModel().rows as row (row.id)}
					<Table.Row data-state={row.getIsSelected() && "selected"} class="hover:bg-muted/50">
						{#each row.getVisibleCells() as cell (cell.id)}
							<Table.Cell>
								<FlexRender {cell} />
							</Table.Cell>
						{/each}
					</Table.Row>
				{:else}
					<Table.Row>
						<Table.Cell colspan={columns.length} class="h-24 text-center text-muted-foreground">
							{emptyMessage}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</div>

	<div class="flex items-center justify-between text-xs text-muted-foreground">
		<div>
			Страница <span class="font-medium text-foreground">{pagination.pageIndex + 1}</span> из <span class="font-medium text-foreground">{table.getPageCount() || 1}</span>
		</div>
		<div class="flex items-center gap-2">
			<Button
				variant="outline"
				size="sm"
				class="gap-1"
				onclick={() => table.previousPage()}
				disabled={!table.getCanPreviousPage()}
			>
				<ChevronLeftIcon class="size-4" />
				<span>Назад</span>
			</Button>
			<Button
				variant="outline"
				size="sm"
				class="gap-1"
				onclick={() => table.nextPage()}
				disabled={!table.getCanNextPage()}
			>
				<span>Вперед</span>
				<ChevronRightIcon class="size-4" />
			</Button>
		</div>
	</div>
</div>