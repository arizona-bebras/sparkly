<script lang="ts">
	import type { DateRange } from "bits-ui";
	import { IssueSeverity } from "$proto";
	import CircleDotIcon from "@lucide/svelte/icons/circle-dot";
	import SlidersHorizontalIcon from "@lucide/svelte/icons/sliders-horizontal";
	import XIcon from "@lucide/svelte/icons/x";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button, buttonVariants } from "$lib/components/ui/button/index.js";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import { DateRangePicker } from "$lib/components/ui/date-range-picker/index.js";

	let {
		selectedStatus = $bindable("all"),
		selectedSeverities = $bindable([]),
		dateRange = $bindable(undefined),
		searchQuery = $bindable(""),
	}: {
		selectedStatus?: "all" | "open" | "resolved";
		selectedSeverities?: IssueSeverity[];
		dateRange?: DateRange;
		searchQuery?: string;
	} = $props();

	function toggleSeverity(severity: IssueSeverity, checked: boolean) {
		if (checked) {
			selectedSeverities = [...selectedSeverities, severity];
		} else {
			selectedSeverities = selectedSeverities.filter((s) => s !== severity);
		}
	}

	function resetFilters() {
		selectedStatus = "all";
		selectedSeverities = [];
		dateRange = undefined;
		searchQuery = "";
	}

	const hasActiveFilters = $derived(
		selectedStatus !== "all" ||
			selectedSeverities.length > 0 ||
			dateRange?.start !== undefined ||
			searchQuery.trim().length > 0
	);
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class={buttonVariants({
			variant: "outline",
			size: "sm",
			class: "h-9 gap-1.5 border-dashed cursor-pointer",
		})}
	>
		<CircleDotIcon class="size-3.5 text-muted-foreground" />
		<span>Статус</span>
		{#if selectedStatus !== "all"}
			<Separator orientation="vertical" class="mx-1 h-4" />
			<Badge
				variant="secondary"
				class="rounded-sm px-1.5 font-normal text-[11px]"
			>
				{selectedStatus === "open" ? "Открытые" : "Решённые"}
			</Badge>
		{/if}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="w-44">
		<DropdownMenu.Label class="text-xs font-semibold">
			Статус
		</DropdownMenu.Label>
		<DropdownMenu.Separator />
		<DropdownMenu.RadioGroup bind:value={selectedStatus}>
			<DropdownMenu.RadioItem value="all">Все</DropdownMenu.RadioItem>
			<DropdownMenu.RadioItem value="open">
				<span class="flex items-center gap-2">
					<span class="size-2 rounded-full bg-destructive"></span>
					<span>Открытые</span>
				</span>
			</DropdownMenu.RadioItem>
			<DropdownMenu.RadioItem value="resolved">
				<span class="flex items-center gap-2">
					<span class="size-2 rounded-full bg-emerald-500"></span>
					<span>Решённые</span>
				</span>
			</DropdownMenu.RadioItem>
		</DropdownMenu.RadioGroup>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class={buttonVariants({
			variant: "outline",
			size: "sm",
			class: "h-9 gap-1.5 border-dashed cursor-pointer",
		})}
	>
		<SlidersHorizontalIcon class="size-3.5 text-muted-foreground" />
		<span>Важность</span>
		{#if selectedSeverities.length > 0}
			<Separator orientation="vertical" class="mx-1 h-4" />
			<Badge
				variant="secondary"
				class="rounded-sm px-1.5 font-normal text-[11px]"
			>
				{selectedSeverities.length}
			</Badge>
		{/if}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="w-48">
		<DropdownMenu.Label class="text-xs font-semibold">
			Уровень важности
		</DropdownMenu.Label>
		<DropdownMenu.Separator />
		<DropdownMenu.CheckboxItem
			checked={selectedSeverities.includes(IssueSeverity.CRITICAL)}
			onCheckedChange={(c) =>
				toggleSeverity(IssueSeverity.CRITICAL, !!c)}
		>
			<span class="flex items-center gap-2">
				<span class="size-2 rounded-full bg-destructive"></span>
				<span>Критическая</span>
			</span>
		</DropdownMenu.CheckboxItem>
		<DropdownMenu.CheckboxItem
			checked={selectedSeverities.includes(IssueSeverity.ERROR)}
			onCheckedChange={(c) => toggleSeverity(IssueSeverity.ERROR, !!c)}
		>
			<span class="flex items-center gap-2">
				<span class="size-2 rounded-full bg-rose-500"></span>
				<span>Ошибка</span>
			</span>
		</DropdownMenu.CheckboxItem>
		<DropdownMenu.CheckboxItem
			checked={selectedSeverities.includes(IssueSeverity.WARNING)}
			onCheckedChange={(c) =>
				toggleSeverity(IssueSeverity.WARNING, !!c)}
		>
			<span class="flex items-center gap-2">
				<span class="size-2 rounded-full bg-amber-500"></span>
				<span>Предупреждение</span>
			</span>
		</DropdownMenu.CheckboxItem>
		<DropdownMenu.CheckboxItem
			checked={selectedSeverities.includes(IssueSeverity.INFO)}
			onCheckedChange={(c) => toggleSeverity(IssueSeverity.INFO, !!c)}
		>
			<span class="flex items-center gap-2">
				<span class="size-2 rounded-full bg-blue-500"></span>
				<span>Инфо</span>
			</span>
		</DropdownMenu.CheckboxItem>
	</DropdownMenu.Content>
</DropdownMenu.Root>

<DateRangePicker bind:value={dateRange} placeholder="Период" />

{#if hasActiveFilters}
	<Button
		variant="ghost"
		size="sm"
		class="h-9 px-2 text-xs text-muted-foreground hover:text-foreground cursor-pointer"
		onclick={resetFilters}
	>
		<span>Сбросить</span>
		<XIcon class="size-3.5 ml-1" />
	</Button>
{/if}
