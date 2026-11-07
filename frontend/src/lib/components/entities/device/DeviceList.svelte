<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { client } from '$proto';
  import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
  import { CircleAlert, RefreshCw } from '@lucide/svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import DataTable from '$lib/components/ui/data-table/data-table.svelte';
  import { columns } from './columns.js';
  import DeviceFilters from './DeviceFilters.svelte';

  let { parkId }: { parkId: string } = $props();

  function getUrlGroups(): string[] {
    const groups = page.url.searchParams.getAll('group');
    if (groups.length > 0) return groups;
    const single = page.url.searchParams.get('group_id') ?? page.url.searchParams.get('group');
    return single ? [single] : [];
  }

  let selectedGroupIds = $state<string[]>(getUrlGroups());
  let selectedBuildIds = $state<string[]>([]);
  let searchQuery = $state('');
  let debouncedSearch = $state('');
  let lastUrlKey = $state(getUrlGroups().slice().sort().join(','));

  $effect(() => {
    const query = searchQuery;
    const timer = setTimeout(() => {
      debouncedSearch = query;
    }, 350);
    return () => clearTimeout(timer);
  });

  $effect(() => {
    const fromUrl = getUrlGroups();
    const currentUrlKey = fromUrl.slice().sort().join(',');

    if (currentUrlKey !== lastUrlKey) {
      lastUrlKey = currentUrlKey;
      selectedGroupIds = fromUrl;
      return;
    }

    const stateKey = selectedGroupIds.slice().sort().join(',');
    if (stateKey !== currentUrlKey) {
      lastUrlKey = stateKey;
      const url = new URL(page.url);
      url.searchParams.delete('group');
      url.searchParams.delete('group_id');
      for (const id of selectedGroupIds) {
        url.searchParams.append('group', id);
      }
      goto(url.toString(), {
        replaceState: true,
        noScroll: true,
        keepFocus: true,
      });
    }
  });

  const groupsQuery = createQuery(() => ({
    queryKey: ['groups', parkId],
    queryFn: () => client.park.getGroups({ parkId }),
    enabled: !!parkId,
  }));

  const availableGroups = $derived.by(() => {
    const list = groupsQuery.data?.groups ?? [];
    return list.map((g) => ({
      id: g.id,
      name: g.name,
    }));
  });

  const searchParam = $derived.by(() => {
    const trimmed = debouncedSearch.trim();
    return trimmed.length > 0 ? trimmed.slice(0, 30) : undefined;
  });

  const devicesQuery = createQuery(() => ({
    queryKey: ['devices', parkId, selectedGroupIds, searchParam],
    queryFn: async () => {
      const [response] = await Promise.all([
        client.device.getDevices({
          parkId,
          groupIds: selectedGroupIds,
          name: searchParam,
        }),
        new Promise((resolve) => setTimeout(resolve, 350)),
      ]);
      return response;
    },
    placeholderData: keepPreviousData,
    enabled: !!parkId,
  }));

  const devices = $derived(devicesQuery.data?.devices ?? []);
</script>

{#snippet filterToolbar()}
  <DeviceFilters
    bind:selectedGroupIds
    bind:selectedBuildIds
    bind:searchQuery
    groups={availableGroups}
    builds={[]}
  />
{/snippet}

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-end">
    <Button
      variant="outline"
      size="sm"
      class="gap-2"
      disabled={devicesQuery.isFetching}
      onclick={() => devicesQuery.refetch()}
    >
      <RefreshCw class="size-4 {devicesQuery.isFetching ? 'animate-spin' : ''}" />
      <span>Обновить</span>
    </Button>
  </div>

  {#if devicesQuery.isPending && !devicesQuery.data}
    <div class="flex flex-col gap-3">
      <Skeleton class="h-10 w-64 rounded-md" />
      <div class="rounded-xl border p-4 flex flex-col gap-3">
        {#each Array(5) as _, i (i)}
          <Skeleton class="h-10 w-full" />
        {/each}
      </div>
    </div>
  {:else if devicesQuery.isError && !devicesQuery.data}
    <div
      class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive"
    >
      <CircleAlert class="size-8" />
      <p class="font-medium">Ошибка загрузки данных</p>
      <p class="text-xs text-muted-foreground">
        {devicesQuery.error?.message}
      </p>
      <Button variant="outline" size="sm" onclick={() => devicesQuery.refetch()}>Повторить</Button>
    </div>
  {:else}
    <div class="relative">
      {#if devicesQuery.isFetching}
        <div
          class="absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-background/40 backdrop-blur-[1px] transition-all duration-200"
        >
          <div
            class="flex items-center gap-2 rounded-full border bg-background/95 px-3 py-1.5 text-xs font-medium text-foreground shadow-sm"
          >
            <RefreshCw class="size-3.5 animate-spin text-primary" />
            <span>Загрузка устройств...</span>
          </div>
        </div>
      {/if}
      <div
        class="transition-opacity duration-200 {devicesQuery.isFetching
          ? 'opacity-60 pointer-events-none'
          : 'opacity-100'}"
      >
        <DataTable
          data={devices}
          {columns}
          searchColumn=""
          searchPlaceholder="Поиск по названию..."
          emptyMessage="Устройства не найдены."
          bind:searchValue={searchQuery}
          toolbar={filterToolbar}
        />
      </div>
    </div>
  {/if}
</div>
