<script lang="ts">
	import { client, IssueSeverity } from "$proto";
	import {
		createQuery,
		createMutation,
		useQueryClient,
	} from "@tanstack/svelte-query";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import CheckCircleIcon from "@lucide/svelte/icons/check-circle";
	import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
	import ClockIcon from "@lucide/svelte/icons/clock";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
	import CalendarIcon from "@lucide/svelte/icons/calendar";
	import ShieldAlertIcon from "@lucide/svelte/icons/shield-alert";
	import InfoIcon from "@lucide/svelte/icons/info";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import { getMockIssue } from "./mock.js";

	let {
		parkId,
		issueId,
	}: {
		parkId: string;
		issueId: string;
	} = $props();

	const queryClient = useQueryClient();

	const issueQuery = createQuery(() => ({
		queryKey: ["issue", issueId],
		queryFn: async () => {
			try {
				const response = await client.issue.getIssue({ id: issueId });
				if (response.issue) return response.issue;
			} catch {}
			return getMockIssue(issueId, parkId);
		},
		enabled: !!issueId,
	}));

	const resolveMutation = createMutation(() => ({
		mutationKey: ["resolveIssue", issueId],
		mutationFn: async () => {
			try {
				return await client.issue.resolveIssue({ id: issueId });
			} catch {}
			return { issue: undefined };
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["issue", issueId] });
			queryClient.invalidateQueries({ queryKey: ["issues", parkId] });
		},
	}));

	function formatTimestamp(ts?: {
		seconds: bigint;
		nanos: number;
	}): string {
		if (!ts || Number(ts.seconds) === 0) return "—";
		const ms = Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000);
		const date = new Date(ms);
		return new Intl.DateTimeFormat("ru-RU", {
			day: "2-digit",
			month: "long",
			year: "numeric",
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
		}).format(date);
	}

	const isResolved = $derived(
		!!issueQuery.data?.resolvedAt &&
			Number(issueQuery.data.resolvedAt.seconds) > 0
	);
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center justify-between">
		<Button
			variant="ghost"
			size="sm"
			href="/control-panel/{parkId}/issues"
			class="gap-1.5 -ml-2 text-muted-foreground hover:text-foreground"
		>
			<ArrowLeftIcon class="size-4" />
			<span>Назад к проблемам</span>
		</Button>

		<div class="flex items-center gap-2">
			<Button
				variant="outline"
				size="sm"
				class="gap-2"
				disabled={issueQuery.isFetching}
				onclick={() => issueQuery.refetch()}
			>
				<RefreshCwIcon
					class="size-4 {issueQuery.isFetching ? 'animate-spin' : ''}"
				/>
				<span>Обновить</span>
			</Button>

			{#if issueQuery.data && !isResolved}
				<Button
					size="sm"
					class="gap-1.5"
					disabled={resolveMutation.isPending}
					onclick={() => resolveMutation.mutate()}
				>
					<CheckCircleIcon class="size-4" />
					<span>Отметить как решённую</span>
				</Button>
			{/if}
		</div>
	</div>

	{#if issueQuery.isPending}
		<div class="flex flex-col gap-6">
			<div class="flex flex-col gap-2">
				<Skeleton class="h-8 w-96 rounded-md" />
				<Skeleton class="h-4 w-64 rounded-md" />
			</div>
			<div class="grid gap-6 md:grid-cols-3">
				<Skeleton class="h-48 md:col-span-2 rounded-xl" />
				<Skeleton class="h-48 rounded-xl" />
			</div>
		</div>
	{:else if issueQuery.isError || !issueQuery.data}
		<div
			class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive"
		>
			<CircleAlertIcon class="size-8" />
			<p class="font-medium">Проблема не найдена или произошла ошибка</p>
			<Button
				variant="outline"
				size="sm"
				href="/control-panel/{parkId}/issues"
			>
				Вернуться к списку
			</Button>
		</div>
	{:else}
		{@const issue = issueQuery.data}
		<div class="flex flex-col gap-6">
			<div
				class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between"
			>
				<div class="flex flex-col gap-1.5">
					<div class="flex flex-wrap items-center gap-2.5">
						<h1 class="text-2xl font-bold tracking-tight">
							{issue.title}
						</h1>
						{#if isResolved}
							<span
								class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
							>
								<span
									class="size-1.5 rounded-full bg-emerald-500"
								></span>
								Решено
							</span>
						{:else}
							<span
								class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-destructive/15 text-destructive border border-destructive/20"
							>
								<span
									class="size-1.5 rounded-full bg-destructive"
								></span>
								Открыта
							</span>
						{/if}
					</div>
					<div class="flex items-center gap-2 text-xs text-muted-foreground">
						<span class="font-mono">{issue.id}</span>
					</div>
				</div>

				<div>
					{#if issue.severity === IssueSeverity.CRITICAL}
						<span
							class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-destructive/15 text-destructive border border-destructive/20"
						>
							<span class="size-2 rounded-full bg-destructive"
							></span>
							Критическая важность
						</span>
					{:else if issue.severity === IssueSeverity.ERROR}
						<span
							class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/20"
						>
							<span class="size-2 rounded-full bg-rose-500"></span>
							Ошибка
						</span>
					{:else if issue.severity === IssueSeverity.WARNING}
						<span
							class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/20"
						>
							<span class="size-2 rounded-full bg-amber-500"
							></span>
							Предупреждение
						</span>
					{:else if issue.severity === IssueSeverity.INFO}
						<span
							class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-blue-500/15 text-blue-600 dark:text-blue-400 border border-blue-500/20"
						>
							<span class="size-2 rounded-full bg-blue-500"></span>
							Информационное сообщение
						</span>
					{:else}
						<span
							class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-secondary text-secondary-foreground"
						>
							Не указана
						</span>
					{/if}
				</div>
			</div>

			<div class="grid gap-6 md:grid-cols-3">
				<Card.Root class="md:col-span-2">
					<Card.Header>
						<Card.Title class="text-base flex items-center gap-2">
							<InfoIcon class="size-4 text-muted-foreground" />
							<span>Описание проблемы</span>
						</Card.Title>
					</Card.Header>
					<Card.Content>
						{#if issue.description}
							<p
								class="text-sm leading-relaxed text-foreground whitespace-pre-wrap"
							>
								{issue.description}
							</p>
						{:else}
							<p class="text-sm text-muted-foreground italic">
								Описание отсутствует
							</p>
						{/if}
					</Card.Content>
				</Card.Root>

				<Card.Root>
					<Card.Header>
						<Card.Title class="text-base flex items-center gap-2">
							<ClockIcon class="size-4 text-muted-foreground" />
							<span>Хронология событий</span>
						</Card.Title>
					</Card.Header>
					<Card.Content class="flex flex-col gap-4 text-sm">
						<div class="flex flex-col gap-1">
							<span class="text-xs text-muted-foreground">
								Первое появление
							</span>
							<span class="font-medium">
								{formatTimestamp(issue.firstAt)}
							</span>
						</div>

						<Separator />

						<div class="flex flex-col gap-1">
							<span class="text-xs text-muted-foreground">
								Последняя активность
							</span>
							<span class="font-medium">
								{formatTimestamp(issue.lastAt)}
							</span>
						</div>

						{#if isResolved && issue.resolvedAt}
							<Separator />
							<div class="flex flex-col gap-1">
								<span class="text-xs text-muted-foreground">
									Время решения
								</span>
								<span class="font-medium text-emerald-600">
									{formatTimestamp(issue.resolvedAt)}
								</span>
							</div>
						{/if}
					</Card.Content>
				</Card.Root>
			</div>

			<Card.Root>
				<Card.Header>
					<Card.Title class="text-base flex items-center gap-2">
						<ShieldAlertIcon
							class="size-4 text-muted-foreground"
						/>
						<span>Системная информация</span>
					</Card.Title>
				</Card.Header>
				<Card.Content class="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
					<div class="flex flex-col gap-1">
						<span class="text-xs text-muted-foreground"
							>Идентификатор парка</span
						>
						<span class="font-mono text-xs">{issue.parkId}</span>
					</div>

					<div class="flex flex-col gap-1">
						<span class="text-xs text-muted-foreground"
							>Идентификатор проблемы</span
						>
						<span class="font-mono text-xs">{issue.id}</span>
					</div>
				</Card.Content>
			</Card.Root>
		</div>
	{/if}
</div>

