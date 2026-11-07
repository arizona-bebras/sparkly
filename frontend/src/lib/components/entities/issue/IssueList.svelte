<script lang="ts">
  import type { DateRange } from 'bits-ui';
  import { getLocalTimeZone } from '@internationalized/date';
  import { timestampFromDate } from '@bufbuild/protobuf/wkt';
  import { client, type Issue, type IssueSeverity } from '$proto';
  import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
  import { CircleAlert, RefreshCw } from '@lucide/svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import DataTable from '$lib/components/ui/data-table/data-table.svelte';
  import { columns } from './columns.js';
  import IssueFilters from './IssueFilters.svelte';

  let { parkId }: { parkId: string } = $props();

  let selectedStatus = $state<'all' | 'open' | 'resolved'>('all');
  let selectedSeverities = $state<IssueSeverity[]>([]);
  let dateRange = $state<DateRange | undefined>(undefined);
  let searchQuery = $state('');
  let debouncedSearch = $state('');

  $effect(() => {
    const query = searchQuery;
    const timer = setTimeout(() => {
      debouncedSearch = query;
    }, 350);
    return () => clearTimeout(timer);
  });

  const searchParam = $derived.by(() => {
    const trimmed = debouncedSearch.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  });

  const issuesQuery = createQuery(() => ({
    queryKey: [
      'issues',
      parkId,
      selectedStatus,
      selectedSeverities,
      dateRange?.start?.toString(),
      dateRange?.end?.toString(),
      searchParam,
    ],
    queryFn: async () => {
      const isResolvedParam =
        selectedStatus === 'open' ? false : selectedStatus === 'resolved' ? true : undefined;
      const severityParam = selectedSeverities.length === 1 ? selectedSeverities[0] : undefined;

      let fromTime = undefined;
      let toTime = undefined;
      if (dateRange?.start) {
        const startDate = dateRange.start.toDate(getLocalTimeZone());
        startDate.setHours(0, 0, 0, 0);
        fromTime = timestampFromDate(startDate);

        const endDate = dateRange.end
          ? dateRange.end.toDate(getLocalTimeZone())
          : dateRange.start.toDate(getLocalTimeZone());
        endDate.setHours(23, 59, 59, 999);
        toTime = timestampFromDate(endDate);
      }

      const [response] = await Promise.all([
        client.issue.getIssues({
          parkId,
          isResolved: isResolvedParam,
          severity: severityParam,
          fromTime,
          toTime,
          title: searchParam,
        }),
        new Promise((resolve) => setTimeout(resolve, 350)),
      ]);
      return response;
    },
    placeholderData: keepPreviousData,
    enabled: !!parkId,
  }));

  const filteredIssues = $derived.by(() => {
    const all = issuesQuery.data?.issues ?? [];

    if (selectedSeverities.length > 1) {
      return all.filter((issue: Issue) => selectedSeverities.includes(issue.severity));
    }

    return all;
  });
</script>

{#snippet filterToolbar()}
  <IssueFilters bind:selectedStatus bind:selectedSeverities bind:dateRange bind:searchQuery />
{/snippet}

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-end">
    <Button
      variant="outline"
      size="sm"
      class="gap-2"
      disabled={issuesQuery.isFetching}
      onclick={() => issuesQuery.refetch()}
    >
      <RefreshCw class="size-4 {issuesQuery.isFetching ? 'animate-spin' : ''}" />
      <span>Обновить</span>
    </Button>
  </div>

  {#if issuesQuery.isPending && !issuesQuery.data}
    <div class="flex flex-col gap-3">
      <Skeleton class="h-10 w-64 rounded-md" />
      <div class="rounded-xl border p-4 flex flex-col gap-3">
        {#each Array(5) as _, i (i)}
          <Skeleton class="h-10 w-full" />
        {/each}
      </div>
    </div>
  {:else if issuesQuery.isError && !issuesQuery.data}
    <div
      class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive"
    >
      <CircleAlert class="size-8" />
      <p class="font-medium">Ошибка загрузки данных</p>
      <p class="text-xs text-muted-foreground">
        {issuesQuery.error?.message}
      </p>
      <Button variant="outline" size="sm" onclick={() => issuesQuery.refetch()}>Повторить</Button>
    </div>
  {:else}
    <div class="relative">
      {#if issuesQuery.isFetching}
        <div
          class="absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-background/40 backdrop-blur-[1px] transition-all duration-200"
        >
          <div
            class="flex items-center gap-2 rounded-full border bg-background/95 px-3 py-1.5 text-xs font-medium text-foreground shadow-sm"
          >
            <RefreshCw class="size-3.5 animate-spin text-primary" />
            <span>Загрузка проблем...</span>
          </div>
        </div>
      {/if}
      <div
        class="transition-opacity duration-200 {issuesQuery.isFetching
          ? 'opacity-60 pointer-events-none'
          : 'opacity-100'}"
      >
        <DataTable
          data={filteredIssues}
          {columns}
          searchColumn=""
          searchPlaceholder="Поиск по проблемам..."
          emptyMessage="Проблемы не найдены."
          bind:searchValue={searchQuery}
          toolbar={filterToolbar}
        />
      </div>
    </div>
  {/if}
</div>
