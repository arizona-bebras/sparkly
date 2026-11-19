<script lang="ts">
  import type { DateRange } from 'bits-ui';
  import { GitCommit, X } from '@lucide/svelte';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
  import { DebouncedInput } from '$lib/components/ui/debounced-input/index.js';
  import * as Popover from '$lib/components/ui/popover/index.js';
  import { Separator } from '$lib/components/ui/separator/index.js';
  import { DateRangePicker } from '$lib/components/ui/date-range-picker/index.js';

  let {
    dateRange = $bindable(undefined),
    commitQuery = $bindable(''),
    searchQuery = $bindable(''),
  }: {
    dateRange?: DateRange | undefined;
    commitQuery?: string;
    searchQuery?: string;
  } = $props();

  function resetFilters() {
    dateRange = undefined;
    commitQuery = '';
    searchQuery = '';
  }

  const hasActiveFilters = $derived(
    !!dateRange?.start || commitQuery.trim().length > 0 || searchQuery.trim().length > 0,
  );
</script>

<Popover.Root>
  <Popover.Trigger
    class={buttonVariants({
      variant: 'outline',
      size: 'sm',
      class: 'h-9 gap-1.5 border-dashed cursor-pointer',
    })}
  >
    <GitCommit class="size-3.5 text-muted-foreground" />
    <span>Коммит</span>
    {#if commitQuery.trim().length > 0}
      <Separator orientation="vertical" class="mx-1 h-4" />
      <Badge variant="secondary" class="rounded-sm px-1.5 font-normal text-[11px] font-mono">
        {commitQuery.trim().slice(0, 7)}
      </Badge>
    {/if}
  </Popover.Trigger>
  <Popover.Content align="start" class="w-64 p-3 flex flex-col gap-2">
    <span class="text-xs font-semibold text-foreground">Фильтр по Git коммиту</span>
    <DebouncedInput
      placeholder="Хэш коммита..."
      bind:value={commitQuery}
      class="h-8 text-xs font-mono"
    />
    {#if commitQuery}
      <Button
        variant="ghost"
        size="sm"
        class="h-7 text-xs self-end"
        onclick={() => (commitQuery = '')}
      >
        Очистить
      </Button>
    {/if}
  </Popover.Content>
</Popover.Root>

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
