<script lang="ts">
	import { RangeCalendar as RangeCalendarPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: RangeCalendarPrimitive.DayProps = $props();
</script>

<RangeCalendarPrimitive.Day
	bind:ref
	class={cn(
		"flex size-(--cell-size) flex-col items-center justify-center gap-1 rounded-(--cell-radius) p-0 leading-none font-normal whitespace-nowrap select-none",
		"not-data-selected:hover:bg-accent not-data-selected:hover:text-accent-foreground",
		"data-[range-middle]:rounded-none data-[range-middle]:bg-accent data-[range-middle]:text-accent-foreground",
		"[&[data-today]:not([data-selected])]:bg-primary/10 [&[data-today]:not([data-selected])]:ring-1.5 [&[data-today]:not([data-selected])]:ring-inset [&[data-today]:not([data-selected])]:ring-primary/40 [&[data-today]:not([data-selected])]:font-bold [&[data-today]:not([data-selected])]:text-foreground",
		"[&[data-today][data-disabled]]:text-muted-foreground",
		"[&[data-today][data-range-middle]]:font-bold [&[data-today][data-range-middle]]:ring-1 [&[data-today][data-range-middle]]:ring-inset [&[data-today][data-range-middle]]:ring-primary/40",
		"data-[range-start]:bg-primary data-[range-start]:text-primary-foreground data-[range-start]:hover:bg-primary data-[range-start]:hover:text-primary-foreground",
		"data-[range-end]:bg-primary data-[range-end]:text-primary-foreground data-[range-end]:hover:bg-primary data-[range-end]:hover:text-primary-foreground",
		"[&[data-outside-month]:not([data-selected])]:text-muted-foreground [&[data-outside-month]:not([data-selected])]:hover:text-accent-foreground",
		"data-[disabled]:pointer-events-none data-[disabled]:text-muted-foreground data-[disabled]:opacity-50",
		"data-[unavailable]:line-through",
		"focus:relative focus:border-ring focus:ring-ring/50",
		"[&>span]:text-xs [&>span]:opacity-70",
		className
	)}
	{...restProps}
/>