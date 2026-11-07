<script lang="ts">
  import { Folder, Package, X } from '@lucide/svelte';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
  import { Separator } from '$lib/components/ui/separator/index.js';

  let {
    selectedGroupIds = $bindable([]),
    selectedBuildIds = $bindable([]),
    searchQuery = $bindable(''),
    groups = [],
    builds = [],
  }: {
    selectedGroupIds?: string[];
    selectedBuildIds?: string[];
    searchQuery?: string;
    groups?: { id: string; name: string }[];
    builds?: { id: string; name: string }[];
  } = $props();

  function toggleItem(
    list: string[],
    id: string,
    checked: boolean,
    setter: (updated: string[]) => void,
  ) {
    if (checked) {
      setter([...list, id]);
    } else {
      setter(list.filter((item) => item !== id));
    }
  }

  function resetFilters() {
    selectedGroupIds = [];
    selectedBuildIds = [];
    searchQuery = '';
  }

  const hasActiveFilters = $derived(
    selectedGroupIds.length > 0 || selectedBuildIds.length > 0 || searchQuery.trim().length > 0,
  );
</script>

<DropdownMenu.Root>
  <DropdownMenu.Trigger
    class={buttonVariants({
      variant: 'outline',
      size: 'sm',
      class: 'h-9 gap-1.5 border-dashed cursor-pointer',
    })}
  >
    <Folder class="size-3.5 text-muted-foreground" />
    <span>Группа</span>
    {#if selectedGroupIds.length > 0}
      <Separator orientation="vertical" class="mx-1 h-4" />
      <Badge variant="secondary" class="rounded-sm px-1.5 font-normal text-[11px]">
        {selectedGroupIds.length}
      </Badge>
    {/if}
  </DropdownMenu.Trigger>
  <DropdownMenu.Content align="start" class="w-48">
    <DropdownMenu.Label class="text-xs font-semibold">Группы</DropdownMenu.Label>
    <DropdownMenu.Separator />
    {#each groups as group (group.id)}
      <DropdownMenu.CheckboxItem
        checked={selectedGroupIds.includes(group.id)}
        onCheckedChange={(c) =>
          toggleItem(selectedGroupIds, group.id, !!c, (v) => (selectedGroupIds = v))}
      >
        <span>{group.name}</span>
      </DropdownMenu.CheckboxItem>
    {/each}
  </DropdownMenu.Content>
</DropdownMenu.Root>

<DropdownMenu.Root>
  <DropdownMenu.Trigger
    class={buttonVariants({
      variant: 'outline',
      size: 'sm',
      class: 'h-9 gap-1.5 border-dashed cursor-pointer',
    })}
  >
    <Package class="size-3.5 text-muted-foreground" />
    <span>Сборка</span>
    {#if selectedBuildIds.length > 0}
      <Separator orientation="vertical" class="mx-1 h-4" />
      <Badge variant="secondary" class="rounded-sm px-1.5 font-normal text-[11px]">
        {selectedBuildIds.length}
      </Badge>
    {/if}
  </DropdownMenu.Trigger>
  <DropdownMenu.Content align="start" class="w-48">
    <DropdownMenu.Label class="text-xs font-semibold">Сборки (Builds)</DropdownMenu.Label>
    <DropdownMenu.Separator />
    {#each builds as build (build.id)}
      <DropdownMenu.CheckboxItem
        checked={selectedBuildIds.includes(build.id)}
        onCheckedChange={(c) =>
          toggleItem(selectedBuildIds, build.id, !!c, (v) => (selectedBuildIds = v))}
      >
        <span>{build.name}</span>
      </DropdownMenu.CheckboxItem>
    {/each}
  </DropdownMenu.Content>
</DropdownMenu.Root>

{#if hasActiveFilters}
  <Button
    variant="ghost"
    size="sm"
    class="h-9 px-2 text-xs text-muted-foreground hover:text-foreground cursor-pointer"
    onclick={resetFilters}
  >
    <span>Сбросить</span>
    <X class="size-3.5 ml-1" />
  </Button>
{/if}
