<script lang="ts">
	import { IssueSeverity } from "$proto";

	let { severity }: { severity: IssueSeverity } = $props();
</script>

{#if severity === IssueSeverity.CRITICAL}
	<span
		class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-destructive/15 text-destructive border border-destructive/20"
	>
		<span class="size-1.5 rounded-full bg-destructive"></span>
		Критическая
	</span>
{:else if severity === IssueSeverity.ERROR}
	<span
		class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/20"
	>
		<span class="size-1.5 rounded-full bg-rose-500"></span>
		Ошибка
	</span>
{:else if severity === IssueSeverity.WARNING}
	<span
		class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/20"
	>
		<span class="size-1.5 rounded-full bg-amber-500"></span>
		Предупреждение
	</span>
{:else if severity === IssueSeverity.INFO}
	<span
		class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-500/15 text-blue-600 dark:text-blue-400 border border-blue-500/20"
	>
		<span class="size-1.5 rounded-full bg-blue-500"></span>
		Инфо
	</span>
{:else}
	<span
		class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-secondary text-secondary-foreground"
	>
		<span class="size-1.5 rounded-full bg-zinc-400"></span>
		Не указана
	</span>
{/if}

