<script lang="ts">
import { LoaderCircle, Settings, Trash2, Trees } from "@lucide/svelte";
import {
	createMutation,
	createQuery,
	useQueryClient,
} from "@tanstack/svelte-query";
import { Button } from "$lib/components/ui/button/index.js";
import * as Card from "$lib/components/ui/card/index.js";
import { cn } from "$lib/utils.js";
import { client } from "$proto";
import type { Park } from "../../../../../proto/gen/backend/proto/service_pb";
import CreateParkForm from "./ParkDialog.svelte";

let {
	class: className,
}: {
	class?: string;
} = $props();

const queryClient = useQueryClient();

let isFormOpen = $state(false);
let selectedPark = $state<Park>();
let deletingParkId = $state<string | null>(null);

const parks = createQuery(() => ({
	queryKey: ["parks"],
	queryFn: () => client.park.getAllParks({}),
}));

const deleteMutation = createMutation(() => ({
	mutationKey: ["deletePark"],
	mutationFn: async (id: string) => {
		return await client.park.deletePark({ id });
	},
	onSuccess: () => {
		queryClient.refetchQueries({ queryKey: ["parks"] });
	},
}));
</script>

<div class={cn("w-full flex flex-col gap-4", className)}>
	{#if parks.isFetching}
		<div class="flex items-center justify-center py-12 text-muted-foreground">
			<LoaderCircle class="size-6 animate-spin" />
		</div>
	{:else if parks.isError}
		<div class="rounded-2xl border border-destructive/20 bg-destructive/10 p-4 text-center text-sm text-destructive">
			Не удалось загрузить список парков. Попробуйте обновить страницу.
		</div>
	{:else if parks.isSuccess}
		{#if parks.data.parks.length === 0}
			<div class="flex flex-col items-center justify-center gap-2 rounded-3xl border border-dashed p-8 text-center text-muted-foreground">
				<Trees class="size-10 text-muted-foreground/60" />
				<p class="font-medium">Парки пока не созданы</p>
				<p class="text-xs">Нажмите «Создать парк», чтобы добавить свою первую площадку.</p>
			</div>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each parks.data.parks as park (park.id)}
					<Card.Root size="sm" class="relative group/card transition-all hover:ring-foreground/20">
						<Card.Header>
							<Card.Title class="truncate text-base font-semibold">
								{park.name}
							</Card.Title>
							{#if park.description}
								<Card.Description class="line-clamp-2 text-xs">
									{park.description}
								</Card.Description>
							{/if}
							<Card.Action class="flex items-center gap-1">
								<Button
									variant="ghost"
									size="icon-sm"
									class="text-muted-foreground hover:text-foreground"
									onclick={() => {
										selectedPark = park;
										isFormOpen = true;
									}}
									aria-label="Настройки парка {park.name}"
								>
									<Settings class="size-4" />
								</Button>

								<Button
									variant="ghost"
									size="icon-sm"
									class="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
									onclick={async () => await deleteMutation.mutateAsync(park.id)}
									disabled={deletingParkId === park.id}
									aria-label="Удалить парк {park.name}"
								>
									{#if deletingParkId === park.id}
										<LoaderCircle class="size-4 animate-spin text-destructive" />
									{:else}
										<Trash2 class="size-4" />
									{/if}
								</Button>
							</Card.Action>
						</Card.Header>

						<Card.Content class="pt-0 text-xs text-muted-foreground">
							<span>Устройств: {park.deviceCount}</span>
						</Card.Content>
					</Card.Root>
				{/each}
			</div>
		{/if}
	{/if}

	{#if isFormOpen}
		<CreateParkForm bind:open={isFormOpen} park={selectedPark} />
	{/if}
</div>