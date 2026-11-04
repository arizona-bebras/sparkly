<script lang="ts">
	import { page } from "$app/state";
	import IssueDetails from "$lib/components/entities/issue/IssueDetails.svelte";

	const parkId = $derived(page.params.park_id ?? "");
	const issueId = $derived(page.params.issue_id ?? "");
</script>

<svelte:head>
	<title>Детали проблемы | Sparkly</title>
</svelte:head>

<IssueDetails {parkId} {issueId} />

