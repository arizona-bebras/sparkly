<script lang="ts">
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import AppSidebar from "$lib/components/entities/sidebar/app-sidebar.svelte";

	let { children } = $props();
</script>

<Sidebar.Provider>
	<AppSidebar />
	<Sidebar.Inset>
		<header class="flex h-14 shrink-0 items-center gap-2 border-b px-4">
			<Sidebar.Trigger class="-ml-1" />
		</header>
		<div class="flex flex-1 flex-col p-4">
			{@render children?.()}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>

