<script lang="ts">
  import type { DateRange } from 'bits-ui';
  import { IssueSeverity } from '$proto';
  import { CircleDot, SlidersHorizontal, X } from '@lucide/svelte';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
  import { Separator } from '$lib/components/ui/separator/index.js';
  import { DateRangePicker } from '$lib/components/ui/date-range-picker/index.js';

  let {
    selectedStatus = $bindable('all'),
    selectedSeverities = $bindable([]),
    dateRange = $bindable(undefined),
    searchQuery = $bindable(''),
  }: {
    selectedStatus?: 'all' | 'open' | 'resolved';
    selectedSeverities?: IssueSeverity[];
    dateRange?: DateRange | undefined;
    searchQuery?: string;
  } = $props();

  function toggleSeverity(sev: IssueSeverity, checked: boolean) {
    if (checked) {
      selectedSeverities = [...selectedSeverities, sev];
    } else {
      selectedSeverities = selectedSeverities.filter((s) => s !== sev);
    }
  }

  function resetFilters() {
    selectedStatus = 'all';
    selectedSeverities = [];
    dateRange = undefined;
    searchQuery = '';
  }

  const hasActiveFilters = $derived(
    selectedStatus !== 'all' ||
      selectedSeverities.length > 0 ||
      !!dateRange?.start ||
      searchQuery.trim().length > 0,
  );

  const statusLabel = $derived(
    selectedStatus === 'open' ? 'Открытые' : selectedStatus === 'resolved' ? 'Решённые' : 'Все',
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
    <CircleDot class="size-3.5 text-muted-foreground" />
    <span>Статус:</span>
    <span class="font-medium text-foreground">{statusLabel}</span>
  </DropdownMenu.Trigger>
  <DropdownMenu.Content align="start" class="w-40">
    <DropdownMenu.Label class="text-xs font-semibold">Фильтр по статусу</DropdownMenu.Label>
    <DropdownMenu.Separator />
    <DropdownMenu.RadioGroup
      value={selectedStatus}
      onValueChange={(v) => (selectedStatus = v as 'all' | 'open' | 'resolved')}
    >
      <DropdownMenu.RadioItem value="all">Все</DropdownMenu.RadioItem>
      <DropdownMenu.RadioItem value="open">Открытые</DropdownMenu.RadioItem>
      <DropdownMenu.RadioItem value="resolved">Решённые</DropdownMenu.RadioItem>
    </DropdownMenu.RadioGroup>
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
    <SlidersHorizontal class="size-3.5 text-muted-foreground" />
    <span>Важность</span>
    {#if selectedSeverities.length > 0}
      <Separator orientation="vertical" class="mx-1 h-4" />
      <Badge variant="secondary" class="rounded-sm px-1.5 font-normal text-[11px]">
        {selectedSeverities.length}
      </Badge>
    {/if}
  </DropdownMenu.Trigger>
  <DropdownMenu.Content align="start" class="w-48">
    <DropdownMenu.Label class="text-xs font-semibold">Уровень важности</DropdownMenu.Label>
    <DropdownMenu.Separator />
    <DropdownMenu.CheckboxItem
      checked={selectedSeverities.includes(IssueSeverity.CRITICAL)}
      onCheckedChange={(c) => toggleSeverity(IssueSeverity.CRITICAL, !!c)}
    >
      <span class="flex items-center gap-2">
        <span class="size-2 rounded-full bg-destructive"></span>
        <span>Критическая</span>
      </span>
    </DropdownMenu.CheckboxItem>
    <DropdownMenu.CheckboxItem
      checked={selectedSeverities.includes(IssueSeverity.HIGH)}
      onCheckedChange={(c) => toggleSeverity(IssueSeverity.HIGH, !!c)}
    >
      <span class="flex items-center gap-2">
        <span class="size-2 rounded-full bg-rose-500"></span>
        <span>Высокая</span>
      </span>
    </DropdownMenu.CheckboxItem>
    <DropdownMenu.CheckboxItem
      checked={selectedSeverities.includes(IssueSeverity.MEDIUM)}
      onCheckedChange={(c) => toggleSeverity(IssueSeverity.MEDIUM, !!c)}
    >
      <span class="flex items-center gap-2">
        <span class="size-2 rounded-full bg-amber-500"></span>
        <span>Средняя</span>
      </span>
    </DropdownMenu.CheckboxItem>
    <DropdownMenu.CheckboxItem
      checked={selectedSeverities.includes(IssueSeverity.LOW)}
      onCheckedChange={(c) => toggleSeverity(IssueSeverity.LOW, !!c)}
    >
      <span class="flex items-center gap-2">
        <span class="size-2 rounded-full bg-yellow-500"></span>
        <span>Низкая</span>
      </span>
    </DropdownMenu.CheckboxItem>
    <DropdownMenu.CheckboxItem
      checked={selectedSeverities.includes(IssueSeverity.INFO)}
      onCheckedChange={(c) => toggleSeverity(IssueSeverity.INFO, !!c)}
    >
      <span class="flex items-center gap-2">
        <span class="size-2 rounded-full bg-blue-500"></span>
        <span>Инфо</span>
      </span>
    </DropdownMenu.CheckboxItem>
  </DropdownMenu.Content>
</DropdownMenu.Root>

<DateRangePicker bind:value={dateRange} placeholder="Период" />

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
