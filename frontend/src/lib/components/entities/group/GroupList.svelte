<script lang="ts">
import { goto } from "$app/navigation";
import {
	createMutation,
	createQuery,
	useQueryClient,
} from "@tanstack/svelte-query";
import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
import CpuIcon from "@lucide/svelte/icons/cpu";
import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
import PlusIcon from "@lucide/svelte/icons/plus";
import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
import SettingsIcon from "@lucide/svelte/icons/settings";
import Trash2Icon from "@lucide/svelte/icons/trash-2";
import GroupForm from "$lib/components/forms/group/Group.svelte";
import { Button } from "$lib/components/ui/button/index.js";
import * as Card from "$lib/components/ui/card/index.js";
import * as Dialog from "$lib/components/ui/dialog/index.js";
import { Skeleton } from "$lib/components/ui/skeleton/index.js";
import { cn } from "$lib/utils.js";
import { client } from "$proto";
import type { DeviceGroup } from "../../../../../proto/gen/backend/proto/service_pb.js";

let {
	parkId,
	class: className,
	onCreateClick,
	isCreateOpen = $bindable(false),
}: {
	parkId: string;
	class?: string;
	onCreateClick?: () => void;
	isCreateOpen?: boolean;
} = $props();

const queryClient = useQueryClient();

let isEditOpen = $state(false);
let selectedGroup = $state<DeviceGroup | undefined>(undefined);
let deletingGroupId = $state<string | null>(null);

const groupsQuery = createQuery(() => ({
	queryKey: ["groups", parkId],
	queryFn: () => client.park.getGroups({ parkId }),
	enabled: !!parkId,
}));

const deleteMutation = createMutation(() => ({
	mutationKey: ["deleteGroup", parkId],
	mutationFn: async (id: string) => {
		return await client.park.deleteGroup({ id });
	},
	onSuccess: () => {
		queryClient.invalidateQueries({ queryKey: ["groups", parkId] });
	},
}));
</script>

<div class={cn("flex flex-col gap-6", className)}>
	<div class="flex items-center justify-between">
		<div class="text-sm text-muted-foreground">
			Всего групп: {groupsQuery.data?.groups.length ?? 0}
		</div>
		<Button
			variant="outline"
			size="sm"
			class="gap-2"
			disabled={groupsQuery.isFetching}
			onclick={() => groupsQuery.refetch()}
		>
			<RefreshCwIcon class="size-4 {groupsQuery.isFetching ? 'animate-spin' : ''}" />
			<span>Обновить</span>
		</Button>
	</div>

	{#if groupsQuery.isPending}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(4) as _}
				<div class="rounded-2xl border p-5 flex flex-col gap-3">
					<Skeleton class="h-5 w-32" />
					<Skeleton class="h-4 w-48" />
					<div class="mt-auto pt-3">
						<Skeleton class="h-4 w-24" />
					</div>
				</div>
			{/each}
		</div>
	{:else if groupsQuery.isError}
		<div class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive">
			<CircleAlertIcon class="size-8" />
			<p class="font-medium">Ошибка загрузки групп</p>
			<p class="text-xs text-muted-foreground">{groupsQuery.error?.message}</p>
			<Button variant="outline" size="sm" onclick={() => groupsQuery.refetch()}>Повторить</Button>
		</div>
	{:else if groupsQuery.isSuccess}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			<button
				type="button"
				onclick={() => {
					isCreateOpen = true;
					onCreateClick?.();
				}}
				class="group flex min-h-[160px] w-full flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed border-border/80 bg-background/50 p-6 text-muted-foreground transition-all hover:border-primary/50 hover:bg-muted/50 hover:text-foreground cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
			>
				<div
					class="flex size-11 items-center justify-center rounded-full bg-muted transition-colors group-hover:bg-primary group-hover:text-primary-foreground"
				>
					<PlusIcon class="size-5" />
				</div>
				<div class="flex flex-col items-center gap-0.5 text-center">
					<span class="text-sm font-semibold">Добавить группу</span>
					<span class="text-xs text-muted-foreground">Создать новую группу устройств</span>
				</div>
			</button>

			{#each groupsQuery.data.groups as group (group.id)}
				<Card.Root
					size="sm"
					class="relative group/card transition-all hover:ring-foreground/20 cursor-pointer"
					onclick={() => goto(`/control-panel/${parkId}/devices?group=${group.id}`)}
				>
					<Card.Header>
						<Card.Title class="truncate text-base font-semibold" title={group.name}>
							{group.name}
						</Card.Title>
						{#if group.description}
							<Card.Description class="line-clamp-2 text-xs" title={group.description}>
								{group.description}
							</Card.Description>
						{/if}
						<Card.Action class="flex items-center gap-1" onclick={(e) => e.stopPropagation()}>
							<Button
								variant="ghost"
								size="icon-sm"
								class="text-muted-foreground hover:text-foreground"
								onclick={(e) => {
									e.stopPropagation();
									selectedGroup = group;
									isEditOpen = true;
								}}
								aria-label="Настройки группы {group.name}"
							>
								<SettingsIcon class="size-4" />
							</Button>

							<Button
								variant="ghost"
								size="icon-sm"
								class="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
								onclick={async (e) => {
									e.stopPropagation();
									deletingGroupId = group.id;
									try {
										await deleteMutation.mutateAsync(group.id);
									} finally {
										deletingGroupId = null;
									}
								}}
								disabled={deletingGroupId === group.id}
								aria-label="Удалить группу {group.name}"
							>
								{#if deletingGroupId === group.id}
									<LoaderCircleIcon class="size-4 animate-spin text-destructive" />
								{:else}
									<Trash2Icon class="size-4" />
								{/if}
							</Button>
						</Card.Action>
					</Card.Header>

					<Card.Content class="pt-0 text-xs text-muted-foreground flex items-center gap-1.5">
						<CpuIcon class="size-3.5" />
						<span>Устройств: {group.deviceCount}</span>
					</Card.Content>
				</Card.Root>
			{/each}
		</div>
	{/if}

	{#if isEditOpen}
		<Dialog.Root bind:open={isEditOpen}>
			<Dialog.Content class="sm:max-w-md">
				<Dialog.Header>
					<Dialog.Title>Редактировать группу</Dialog.Title>
					<Dialog.Description>
						Измените информацию о группе устройств.
					</Dialog.Description>
				</Dialog.Header>

				<GroupForm
					{parkId}
					group={selectedGroup}
					onSuccess={() => {
						isEditOpen = false;
						selectedGroup = undefined;
						queryClient.invalidateQueries({ queryKey: ["groups", parkId] });
					}}
				/>
			</Dialog.Content>
		</Dialog.Root>
	{/if}
</div>
