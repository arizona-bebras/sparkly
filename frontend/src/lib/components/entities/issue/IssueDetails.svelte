<script lang="ts">
  import { client, IssueSeverity } from '$proto';
  import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
  import {
    ArrowLeft,
    CheckCircle,
    CircleAlert,
    Clock,
    RefreshCw,
    Calendar,
    ShieldAlert,
    Info,
  } from '@lucide/svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Separator } from '$lib/components/ui/separator/index.js';

  let {
    parkId,
    issueId,
  }: {
    parkId: string;
    issueId: string;
  } = $props();

  const queryClient = useQueryClient();

  const issueQuery = createQuery(() => ({
    queryKey: ['issue', issueId],
    queryFn: async () => {
      const response = await client.issue.getIssue({ id: issueId });
      return response.issue;
    },
    enabled: !!issueId,
  }));

  const resolveMutation = createMutation(() => ({
    mutationKey: ['resolveIssue', issueId],
    mutationFn: (id: string) => client.issue.resolveIssue({ id }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['issue', issueId] });
      queryClient.invalidateQueries({ queryKey: ['issues', parkId] });
    },
  }));

  function formatTimestamp(ts?: { seconds: bigint; nanos: number }): string {
    if (!ts || Number(ts.seconds) === 0) return '—';
    const ms = Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000);
    const date = new Date(ms);
    return new Intl.DateTimeFormat('ru-RU', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(date);
  }

  const issue = $derived(issueQuery.data);
  const isResolved = $derived(
    !!issue?.isResolved || (!!issue?.resolvedAt && Number(issue.resolvedAt.seconds) > 0),
  );
</script>

<div class="flex flex-col gap-6">
  <div class="flex items-center justify-between">
    <Button
      variant="ghost"
      size="sm"
      class="gap-2 cursor-pointer"
      href={`/control-panel/${parkId}/issues`}
    >
      <ArrowLeft class="size-4" />
      <span>Назад к списку проблем</span>
    </Button>

    <div class="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        class="gap-2"
        disabled={issueQuery.isFetching}
        onclick={() => issueQuery.refetch()}
      >
        <RefreshCw class="size-4 {issueQuery.isFetching ? 'animate-spin' : ''}" />
        <span>Обновить</span>
      </Button>

      {#if !isResolved && issue}
        <Button
          size="sm"
          class="gap-2"
          disabled={resolveMutation.isPending}
          onclick={() => resolveMutation.mutate(issue.id)}
        >
          <CheckCircle class="size-4" />
          <span>Отметить как решённую</span>
        </Button>
      {/if}
    </div>
  </div>

  {#if issueQuery.isPending}
    <div class="flex flex-col gap-4">
      <Skeleton class="h-8 w-96 rounded-md" />
      <Skeleton class="h-4 w-64 rounded-md" />
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mt-2">
        <Skeleton class="h-28 rounded-xl" />
        <Skeleton class="h-28 rounded-xl" />
        <Skeleton class="h-28 rounded-xl" />
      </div>
      <Skeleton class="h-64 rounded-xl mt-2" />
    </div>
  {:else if issueQuery.isError || !issue}
    <div
      class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive"
    >
      <CircleAlert class="size-8" />
      <p class="font-medium">Ошибка загрузки проблемы</p>
      <p class="text-xs text-muted-foreground">
        {issueQuery.error?.message ?? 'Проблема не найдена'}
      </p>
      <Button variant="outline" size="sm" onclick={() => issueQuery.refetch()}>Повторить</Button>
    </div>
  {:else}
    <div class="flex flex-col gap-6">
      <div class="flex flex-col gap-2">
        <div class="flex flex-wrap items-center gap-3">
          <h1 class="text-2xl font-bold tracking-tight">{issue.title}</h1>
          {#if isResolved}
            <span
              class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
            >
              <CheckCircle class="size-3.5" />
              Решена
            </span>
          {:else}
            <span
              class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/20"
            >
              <Clock class="size-3.5" />
              Открыта
            </span>
          {/if}
        </div>
        <p class="font-mono text-xs text-muted-foreground">{issue.id}</p>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card.Root>
          <Card.Content class="pt-6 flex flex-col gap-2">
            <span class="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
              <ShieldAlert class="size-3.5" />
              Уровень критичности
            </span>
            <div>
              {#if issue.severity === IssueSeverity.CRITICAL}
                <span
                  class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-destructive/15 text-destructive border border-destructive/20"
                >
                  <span class="size-2 rounded-full bg-destructive"></span>
                  Критическая важность
                </span>
              {:else if issue.severity === IssueSeverity.HIGH}
                <span
                  class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/20"
                >
                  <span class="size-2 rounded-full bg-rose-500"></span>
                  Высокая важность
                </span>
              {:else if issue.severity === IssueSeverity.MEDIUM}
                <span
                  class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/20"
                >
                  <span class="size-2 rounded-full bg-amber-500"></span>
                  Средняя важность
                </span>
              {:else if issue.severity === IssueSeverity.LOW}
                <span
                  class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-yellow-500/15 text-yellow-600 dark:text-yellow-400 border border-yellow-500/20"
                >
                  <span class="size-2 rounded-full bg-yellow-500"></span>
                  Низкая важность
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
          </Card.Content>
        </Card.Root>

        <Card.Root>
          <Card.Content class="pt-6 flex flex-col gap-1">
            <span class="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
              <Calendar class="size-3.5" />
              Первая фиксация
            </span>
            <span class="text-sm font-semibold mt-1">
              {formatTimestamp(issue.firstAt)}
            </span>
          </Card.Content>
        </Card.Root>

        <Card.Root>
          <Card.Content class="pt-6 flex flex-col gap-1">
            <span class="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
              <Clock class="size-3.5" />
              Последняя активность
            </span>
            <span class="text-sm font-semibold mt-1">
              {formatTimestamp(issue.lastAt)}
            </span>
          </Card.Content>
        </Card.Root>
      </div>

      <Card.Root>
        <Card.Header>
          <Card.Title class="text-base flex items-center gap-2">
            <Info class="size-4 text-muted-foreground" />
            <span>Описание инцидента</span>
          </Card.Title>
        </Card.Header>
        <Card.Content>
          <p class="text-sm leading-relaxed text-foreground whitespace-pre-wrap">
            {issue.description || 'Подробное описание проблемы отсутствует.'}
          </p>
        </Card.Content>
        {#if isResolved && issue.resolvedAt}
          <Separator />
          <Card.Footer class="py-3 px-6 bg-muted/30">
            <span class="text-xs text-muted-foreground">
              Проблема была решена {formatTimestamp(issue.resolvedAt)}
            </span>
          </Card.Footer>
        {/if}
      </Card.Root>
    </div>
  {/if}
</div>
