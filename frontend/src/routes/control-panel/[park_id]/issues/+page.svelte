<script lang="ts">
	import { page } from "$app/state";
	import IssueList from "$lib/components/entities/issue/IssueList.svelte";

	const parkId = $derived(page.params.park_id ?? "");
</script>

<svelte:head>
	<title>Проблемы | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-1">
		<h1 class="text-2xl font-bold tracking-tight">Проблемы (Issues)</h1>
		<p class="text-sm text-muted-foreground">
			Реестр ошибок и инцидентов устройств парка
		</p>
	</div>

	<IssueList {parkId} />
</div>