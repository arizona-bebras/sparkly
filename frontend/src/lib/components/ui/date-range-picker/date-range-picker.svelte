<script lang="ts">
	import type { DateRange } from "bits-ui";
	import CalendarIcon from "@lucide/svelte/icons/calendar";
	import {
		DateFormatter,
		getLocalTimeZone,
		today,
		type DateValue,
	} from "@internationalized/date";
	import * as Popover from "$lib/components/ui/popover/index.js";
	import { RangeCalendar } from "$lib/components/ui/range-calendar/index.js";
	import { buttonVariants } from "$lib/components/ui/button/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import { cn } from "$lib/utils.js";

	let {
		value = $bindable(),
		placeholder = "Период",
		class: className,
		maxValue = today(getLocalTimeZone()),
	}: {
		value?: DateRange;
		placeholder?: string;
		class?: string;
		maxValue?: DateValue;
	} = $props();

	let calendarPlaceholder = $state<DateValue>(
		value?.start ?? today(getLocalTimeZone()).subtract({ months: 1 })
	);

	const df = new DateFormatter("ru-RU", {
		month: "short",
		day: "numeric",
	});
</script>

<Popover.Root>
	<Popover.Trigger
		class={buttonVariants({
			variant: "outline",
			size: "sm",
			class: cn(
				"h-9 gap-1.5 border-dashed cursor-pointer",
				className
			),
		})}
	>
		<CalendarIcon class="size-3.5 text-muted-foreground" />
		<span>{placeholder}</span>
		{#if value?.start}
			<Separator orientation="vertical" class="mx-1 h-4" />
			<Badge
				variant="secondary"
				class="rounded-sm px-1.5 font-normal text-[11px]"
			>
				{#if value.end}
					{df.format(value.start.toDate(getLocalTimeZone()))} - {df.format(
						value.end.toDate(getLocalTimeZone())
					)}
				{:else}
					{df.format(value.start.toDate(getLocalTimeZone()))}
				{/if}
			</Badge>
		{/if}
	</Popover.Trigger>
	<Popover.Content class="w-auto p-0" align="start">
		<RangeCalendar
			bind:value
			bind:placeholder={calendarPlaceholder}
			numberOfMonths={2}
			locale="ru-RU"
			{maxValue}
		/>
	</Popover.Content>
</Popover.Root>
