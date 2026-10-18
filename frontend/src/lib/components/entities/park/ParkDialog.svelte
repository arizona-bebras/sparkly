<script lang="ts">
import { useQueryClient } from "@tanstack/svelte-query";
import type { Snippet } from "svelte";
import * as Dialog from "$lib/components/ui/dialog/index.js";
import type { Park } from "../../../../../proto/gen/backend/proto/service_pb";
import ParkForm from "../../forms/park/Park.svelte";

let client = useQueryClient();

let {
	open = $bindable(false),
	children,
	park,
}: {
	open: boolean;
	children?: Snippet;
	park?: Park;
} = $props();
</script>

<Dialog.Root bind:open>
	<Dialog.Trigger>
	{#if children}
		{@render children()}
	{:else}
		<p>создать паркFSDAFLSJFSJDOKFS</p>
	{/if} 
	</Dialog.Trigger>

	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Создать парк</Dialog.Title>
			<Dialog.Description>
				Заполните информацию для создания нового парка.
			</Dialog.Description>
		</Dialog.Header>

		<ParkForm onSuccess={() => {
			open = false
			client.refetchQueries({queryKey: ['parks']})
		}} {park} />
	</Dialog.Content>
</Dialog.Root>
