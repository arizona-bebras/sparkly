<script lang="ts">
  import type { DateRange } from 'bits-ui';
  import { getLocalTimeZone } from '@internationalized/date';
  import { timestampFromDate } from '@bufbuild/protobuf/wkt';
  import { client, type Build } from '$proto';
  import {
    createQuery,
    createMutation,
    keepPreviousData,
    useQueryClient,
  } from '@tanstack/svelte-query';
  import { CircleAlert, FileCode, Layers, LoaderCircle, Package, RefreshCw } from '@lucide/svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import DataTable from '$lib/components/ui/data-table/data-table.svelte';
  import BuildDetailsDialog from './BuildDetailsDialog.svelte';
  import BuildFilters from './BuildFilters.svelte';
  import { createColumns } from './columns.js';

  let {
    parkId,
    class: className = '',
  }: {
    parkId: string;
    class?: string;
  } = $props();

  const queryClient = useQueryClient();

  let searchQuery = $state('');
  let commitQuery = $state('');
  let dateRange = $state<DateRange | undefined>(undefined);
  let selectedBuild = $state<Build | null>(null);
  let isDetailsOpen = $state(false);
  let deletingBuild = $state<Build | null>(null);

  const versionParam = $derived.by(() => {
    const trimmed = searchQuery.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  });

  const commitParam = $derived.by(() => {
    const trimmed = commitQuery.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  });

  const buildsQuery = createQuery(() => ({
    queryKey: [
      'builds',
      parkId,
      versionParam,
      commitParam,
      dateRange?.start?.toString(),
      dateRange?.end?.toString(),
    ],
    queryFn: async () => {
      const currentKey = [
        'builds',
        parkId,
        versionParam,
        commitParam,
        dateRange?.start?.toString(),
        dateRange?.end?.toString(),
      ];
      const hasCached = !!queryClient.getQueryData(currentKey);

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
        client.build.getBuilds({
          parkId,
          version: versionParam,
          commit: commitParam,
          fromTime,
          toTime,
        }),
        !hasCached ? new Promise((resolve) => setTimeout(resolve, 350)) : Promise.resolve(),
      ]);
      return response;
    },
    placeholderData: keepPreviousData,
    staleTime: 1000 * 60,
    enabled: !!parkId,
  }));

  const builds = $derived(buildsQuery.data?.builds ?? []);
  const totalCount = $derived(builds.length);
  const elfCount = $derived(
    builds.filter((b) => !!b.elfObjectKey && b.elfObjectKey.trim().length > 0).length,
  );
  const dwarfCount = $derived(
    builds.filter((b) => !!b.dwarfObjectKey && b.dwarfObjectKey.trim().length > 0).length,
  );

  const deleteMutation = createMutation(() => ({
    mutationKey: ['deleteBuild', parkId],
    mutationFn: async (id: string) => {
      return await client.build.deleteBuild({ id });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['builds', parkId] });
      deletingBuild = null;
    },
  }));

  function handleViewDetails(build: Build) {
    selectedBuild = build;
    isDetailsOpen = true;
  }

  function handleDelete(build: Build) {
    deletingBuild = build;
  }

  async function confirmDelete() {
    if (!deletingBuild) return;
    deleteMutation.mutate(deletingBuild.id);
  }

  const tableColumns = $derived(
    createColumns({
      onViewDetails: handleViewDetails,
      onDelete: handleDelete,
    }),
  );
</script>

{#snippet filterToolbar()}
  <BuildFilters bind:dateRange bind:commitQuery bind:searchQuery />
  <div class="hidden lg:flex items-center gap-2 ml-auto">
    <div
      class="inline-flex items-center gap-1.5 rounded-lg border bg-muted/40 px-2.5 py-1 text-xs text-muted-foreground"
      title="Сборок с бинарным образом прошивки ELF"
    >
      <FileCode class="size-3.5 text-emerald-600 dark:text-emerald-400" />
      <span>С ELF:</span>
      <span class="font-semibold text-foreground">{elfCount}</span>
    </div>
    <div
      class="inline-flex items-center gap-1.5 rounded-lg border bg-muted/40 px-2.5 py-1 text-xs text-muted-foreground"
      title="Сборок с отладочной таблицей DWARF"
    >
      <Layers class="size-3.5 text-indigo-600 dark:text-indigo-400" />
      <span>С DWARF:</span>
      <span class="font-semibold text-foreground">{dwarfCount}</span>
    </div>
  </div>
{/snippet}

<div class="flex flex-col gap-4 {className}">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div class="flex items-center gap-2 text-sm text-muted-foreground">
      <Package class="size-4" />
      <span>Найдено сборок: <strong class="text-foreground">{totalCount}</strong></span>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        class="gap-2"
        disabled={buildsQuery.isFetching}
        onclick={() => buildsQuery.refetch()}
      >
        <RefreshCw class="size-4 {buildsQuery.isFetching ? 'animate-spin' : ''}" />
        <span>Обновить</span>
      </Button>
    </div>
  </div>

  {#if buildsQuery.isPending && !buildsQuery.data}
    <div class="flex flex-col gap-3">
      <Skeleton class="h-10 w-64 rounded-md" />
      <div class="rounded-xl border p-4 flex flex-col gap-3">
        {#each Array(5) as _, i (i)}
          <Skeleton class="h-10 w-full" />
        {/each}
      </div>
    </div>
  {:else if buildsQuery.isError && !buildsQuery.data}
    <div
      class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive"
    >
      <CircleAlert class="size-8" />
      <p class="font-medium">Ошибка загрузки списка сборок</p>
      <p class="text-xs text-muted-foreground">
        {buildsQuery.error?.message}
      </p>
      <Button variant="outline" size="sm" onclick={() => buildsQuery.refetch()}>Повторить</Button>
    </div>
  {:else}
    <div class="relative">
      {#if buildsQuery.isPlaceholderData}
        <div
          class="absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-background/40 backdrop-blur-[1px] transition-all duration-200"
        >
          <div
            class="flex items-center gap-2 rounded-full border bg-background/95 px-3 py-1.5 text-xs font-medium text-foreground shadow-sm"
          >
            <RefreshCw class="size-3.5 animate-spin text-primary" />
            <span>Загрузка сборок...</span>
          </div>
        </div>
      {/if}
      <div
        class="transition-opacity duration-200 {buildsQuery.isPlaceholderData
          ? 'opacity-60 pointer-events-none'
          : 'opacity-100'}"
      >
        <DataTable
          data={builds}
          columns={tableColumns}
          searchColumn=""
          searchPlaceholder="Поиск по версии..."
          emptyMessage="Сборки не найдены."
          bind:searchValue={searchQuery}
          toolbar={filterToolbar}
        />
      </div>
    </div>
  {/if}
</div>

<BuildDetailsDialog bind:open={isDetailsOpen} build={selectedBuild} />

<Dialog.Root
  open={!!deletingBuild}
  onOpenChange={(open) => {
    if (!open) deletingBuild = null;
  }}
>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Удалить сборку {deletingBuild?.version}?</Dialog.Title>
      <Dialog.Description>
        Вы уверены, что хотите удалить эту сборку? Запись о сборке будет удалена из базы данных. Это
        действие нельзя отменить.
      </Dialog.Description>
    </Dialog.Header>

    {#if deletingBuild}
      <div class="flex flex-col gap-1.5 rounded-lg border bg-muted/30 p-3 text-xs font-mono">
        <div class="flex justify-between">
          <span class="text-muted-foreground">Версия:</span>
          <span class="text-foreground font-semibold">{deletingBuild.version}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">ID:</span>
          <span class="text-muted-foreground truncate max-w-[240px]">{deletingBuild.id}</span>
        </div>
      </div>
    {/if}

    <Dialog.Footer class="gap-2 sm:gap-0">
      <Button
        variant="outline"
        onclick={() => (deletingBuild = null)}
        disabled={deleteMutation.isPending}
      >
        Отмена
      </Button>
      <Button
        variant="destructive"
        onclick={confirmDelete}
        disabled={deleteMutation.isPending}
        class="gap-2"
      >
        {#if deleteMutation.isPending}
          <LoaderCircle class="size-4 animate-spin" />
          <span>Удаление...</span>
        {:else}
          <span>Удалить сборку</span>
        {/if}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
