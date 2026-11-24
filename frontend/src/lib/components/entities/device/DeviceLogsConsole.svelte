<script lang="ts">
  import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
  import { Separator } from '$lib/components/ui/separator/index.js';
  import { LogLevel, type Log } from '$proto';
  import { timestampDate } from '@bufbuild/protobuf/wkt';
  import { cn } from '$lib/utils.js';
  import {
    Search,
    ChevronDown,
    ChevronRight,
    MoreVertical,
    Copy,
    Check,
    X,
    Filter,
    ArrowDown,
  } from '@lucide/svelte';

  interface Props {
    logs?: Log[];
    class?: string;
    activeLogId?: string | null;
  }

  let { logs = [], class: className, activeLogId = $bindable(null) }: Props = $props();

  let searchQuery = $state('');
  let selectedLevels = $state<number[]>([]);
  let expandedIds = $state<Set<string>>(new Set());
  let copiedLogId = $state<string | null>(null);
  let scrollContainer = $state<HTMLDivElement | null>(null);

  type LevelMeta = {
    level: number;
    label: string;
    textClass: string;
    dotClass: string;
  };

  const ALL_LEVELS: LevelMeta[] = [
    { level: LogLevel.FATAL, label: 'FATAL', dotClass: 'bg-rose-500', textClass: 'text-rose-500' },
    { level: LogLevel.ERROR, label: 'ERROR', dotClass: 'bg-red-400', textClass: 'text-red-400' },
    { level: LogLevel.WARN, label: 'WARN', dotClass: 'bg-amber-400', textClass: 'text-amber-400' },
    {
      level: LogLevel.INFO,
      label: 'INFO',
      dotClass: 'bg-emerald-400',
      textClass: 'text-emerald-400',
    },
    { level: LogLevel.DEBUG, label: 'DEBUG', dotClass: 'bg-sky-400', textClass: 'text-sky-400' },
    { level: LogLevel.TRACE, label: 'TRACE', dotClass: 'bg-zinc-400', textClass: 'text-zinc-400' },
  ];

  const levelMetaMap = new Map<number, LevelMeta>(ALL_LEVELS.map((item) => [item.level, item]));

  const levelCounts = $derived.by(() => {
    const counts: Record<number, number> = {};
    for (const l of logs) {
      counts[l.level] = (counts[l.level] ?? 0) + 1;
    }
    return counts;
  });

  const availableLevels = $derived(ALL_LEVELS.filter((lvl) => (levelCounts[lvl.level] ?? 0) > 0));

  function toggleLevel(level: number, checked: boolean) {
    if (checked) {
      if (!selectedLevels.includes(level)) {
        selectedLevels = [...selectedLevels, level];
      }
    } else {
      selectedLevels = selectedLevels.filter((l) => l !== level);
    }
  }

  function clearFilters() {
    selectedLevels = [];
    searchQuery = '';
  }

  function formatLogTimestamp(ts: NonNullable<Log['timestamp']>): string {
    const d = timestampDate(ts);
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const hours = String(d.getHours()).padStart(2, '0');
    const minutes = String(d.getMinutes()).padStart(2, '0');
    const seconds = String(d.getSeconds()).padStart(2, '0');
    const ms = String(d.getMilliseconds()).padStart(3, '0');
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}.${ms}`;
  }

  const filteredLogs = $derived.by(() => {
    const query = searchQuery.trim().toLowerCase();
    return logs.filter((log) => {
      if (selectedLevels.length > 0 && !selectedLevels.includes(log.level)) {
        return false;
      }
      if (query) {
        const matchMsg = log.message.toLowerCase().includes(query);
        const matchLevel = (LogLevel[log.level] ?? '').toLowerCase().includes(query);
        const matchAttr = Object.values(log.attributes || {}).some((v) =>
          v.toLowerCase().includes(query),
        );
        if (!matchMsg && !matchLevel && !matchAttr) return false;
      }
      return true;
    });
  });

  function toggleExpand(id: string) {
    const next = new Set(expandedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    expandedIds = next;
  }

  function copySingleLog(log: Log) {
    const ts = log.timestamp ? formatLogTimestamp(log.timestamp) : '';
    const lvl = (LogLevel[log.level] ?? 'LOG').padEnd(5, ' ');
    const line = `${ts}  ${lvl}  ${log.message}`;
    navigator.clipboard.writeText(line).then(() => {
      copiedLogId = log.id;
      setTimeout(() => {
        if (copiedLogId === log.id) copiedLogId = null;
      }, 1500);
    });
  }

  function scrollToBottom() {
    if (scrollContainer) {
      scrollContainer.scrollTop = scrollContainer.scrollHeight;
    }
  }

  const hasActiveFilters = $derived(selectedLevels.length > 0 || searchQuery.trim().length > 0);
</script>

<div class={cn('flex flex-col w-full border-t border-border/70', className)}>
  <div
    class="flex flex-wrap items-center justify-between gap-3 px-4 sm:px-5 py-2 bg-muted/20 border-b border-border/60"
  >
    <div class="relative flex-1 min-w-[240px]">
      <Search
        class="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground pointer-events-none"
      />
      <Input
        type="text"
        placeholder="Поиск по тексту лога..."
        class="pl-8 pr-8 h-8 text-xs bg-background/80"
        bind:value={searchQuery}
      />
      {#if searchQuery}
        <button
          type="button"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground cursor-pointer"
          onclick={() => (searchQuery = '')}
          aria-label="Очистить поиск"
        >
          <X class="size-3" />
        </button>
      {/if}
    </div>

    <div class="flex items-center gap-2">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger
          class={buttonVariants({
            variant: 'outline',
            size: 'sm',
            class: 'h-8 text-xs gap-1.5 border-dashed cursor-pointer',
          })}
        >
          <Filter data-icon="inline-start" class="size-3 text-muted-foreground" />
          <span>Уровень</span>
          {#if selectedLevels.length > 0}
            <Separator orientation="vertical" class="mx-1 h-3.5" />
            <Badge variant="secondary" class="rounded-sm px-1.5 font-normal text-[10px]">
              {selectedLevels.length}
            </Badge>
          {/if}
          <ChevronDown data-icon="inline-end" class="size-3 text-muted-foreground" />
        </DropdownMenu.Trigger>
        <DropdownMenu.Content align="end" class="w-56">
          <DropdownMenu.Group>
            <DropdownMenu.GroupHeading>Уровень логирования</DropdownMenu.GroupHeading>
            <DropdownMenu.Separator />
            {#each availableLevels as lvl (lvl.level)}
              <DropdownMenu.CheckboxItem
                checked={selectedLevels.includes(lvl.level)}
                onCheckedChange={(checked) => toggleLevel(lvl.level, checked)}
                class="cursor-pointer"
              >
                <div class="flex items-center justify-between w-full pr-2">
                  <div class="flex items-center gap-2">
                    <span class={cn('size-2 rounded-full', lvl.dotClass)}></span>
                    <span class="font-medium text-xs">{lvl.label}</span>
                  </div>
                  <span class="text-[11px] text-muted-foreground">
                    {levelCounts[lvl.level] ?? 0}
                  </span>
                </div>
              </DropdownMenu.CheckboxItem>
            {/each}
            {#if selectedLevels.length > 0}
              <DropdownMenu.Separator />
              <DropdownMenu.Item
                class="justify-center text-xs text-muted-foreground cursor-pointer"
                onSelect={() => (selectedLevels = [])}
              >
                Сбросить фильтр
              </DropdownMenu.Item>
            {/if}
          </DropdownMenu.Group>
        </DropdownMenu.Content>
      </DropdownMenu.Root>

      {#if hasActiveFilters}
        <Button
          variant="ghost"
          size="sm"
          class="h-8 text-xs gap-1 px-2 text-muted-foreground hover:text-foreground cursor-pointer"
          onclick={clearFilters}
        >
          <X data-icon="inline-start" class="size-3" />
          <span>Сброс</span>
        </Button>
      {/if}
    </div>
  </div>

  <div
    bind:this={scrollContainer}
    class="custom-scrollbar min-h-[260px] max-h-[500px] overflow-y-auto overflow-x-hidden bg-zinc-950 text-zinc-100 font-mono text-xs"
  >
    {#if filteredLogs.length === 0}
      <div class="flex flex-col items-center justify-center p-8 text-center text-zinc-500">
        <p class="text-sm font-medium">Нет логов, соответствующих критериям</p>
        {#if hasActiveFilters}
          <Button
            variant="link"
            size="sm"
            class="mt-1 text-xs text-zinc-400 hover:text-zinc-200 cursor-pointer"
            onclick={clearFilters}
          >
            Сбросить фильтры
          </Button>
        {/if}
      </div>
    {:else}
      {#each filteredLogs as log (log.id)}
        {@const levelCfg = levelMetaMap.get(log.level) ?? {
          level: log.level,
          label: LogLevel[log.level] ?? 'LOG',
          textClass: 'text-zinc-400',
          dotClass: 'bg-zinc-400',
        }}
        {@const hasDetails =
          (log.stack && log.stack.length > 0) ||
          Object.keys(log.attributes || {}).length > 0 ||
          Boolean(log.buildId)}
        {@const isExpanded = expandedIds.has(log.id)}
        {@const isFocused = activeLogId === log.id}

        <div
          role="none"
          class={cn(
            'group relative flex items-start gap-2.5 px-4 sm:px-5 py-1.5 transition-colors hover:bg-zinc-900/80',
            isExpanded && 'bg-zinc-900/60',
            isFocused && 'bg-primary/10 ring-1 ring-inset ring-primary/40',
          )}
          onpointerenter={() => (activeLogId = log.id)}
          onpointerleave={() => {
            if (activeLogId === log.id) activeLogId = null;
          }}
        >
          <button
            type="button"
            class={cn(
              'mt-0.5 inline-flex size-4 shrink-0 items-center justify-center rounded text-zinc-500 hover:text-zinc-200 transition-colors',
              !hasDetails && 'opacity-30 cursor-default',
            )}
            disabled={!hasDetails}
            onclick={(e) => {
              e.stopPropagation();
              if (hasDetails) toggleExpand(log.id);
            }}
            aria-label={hasDetails ? 'Подробнее' : undefined}
          >
            {#if hasDetails}
              <ChevronRight
                class={cn(
                  'size-3.5 transition-transform duration-150',
                  isExpanded && 'rotate-90 text-zinc-200',
                )}
              />
            {:else}
              <MoreVertical class="size-3.5" />
            {/if}
          </button>

          <span class="text-zinc-500 shrink-0 select-all font-mono">
            {log.timestamp ? formatLogTimestamp(log.timestamp) : ''}
          </span>

          <span class={cn('w-14 shrink-0 font-semibold select-none', levelCfg.textClass)}>
            {levelCfg.label}
          </span>

          <span class="text-zinc-200 select-text break-words flex-1 min-w-0 font-mono">
            {log.message}
          </span>

          <div class="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
            <button
              type="button"
              class="size-5 inline-flex items-center justify-center rounded text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
              onclick={(e) => {
                e.stopPropagation();
                copySingleLog(log);
              }}
              title="Копировать строку"
              aria-label="Копировать строку лога"
            >
              {#if copiedLogId === log.id}
                <Check class="size-3 text-emerald-400" />
              {:else}
                <Copy class="size-3" />
              {/if}
            </button>
          </div>
        </div>

        {#if isExpanded && hasDetails}
          <div
            class="flex flex-col gap-2 bg-zinc-900/90 border-t border-b border-zinc-800/80 px-4 sm:px-5 py-2.5 pl-10 sm:pl-11 text-xs"
          >
            {#if log.stack && log.stack.length > 0}
              <div class="flex flex-col gap-1">
                <span class="text-zinc-400 font-semibold text-[11px]">Стек вызовов:</span>
                <div
                  class="flex flex-col gap-1 rounded-md bg-zinc-950/80 border border-zinc-800/80 p-2 font-mono text-[11px] text-zinc-300"
                >
                  {#each log.stack as frame, i (i)}
                    <div class="flex items-center gap-2">
                      <span class="text-zinc-500">#{i}</span>
                      <span class="text-zinc-200">{frame.file}:{frame.line}</span>
                      {#if frame.pc}
                        <span class="text-zinc-500 font-mono text-[10px]">
                          ({frame.pc.toString(16)})
                        </span>
                      {/if}
                    </div>
                  {/each}
                </div>
              </div>
            {/if}

            {#if Object.keys(log.attributes || {}).length > 0}
              <div class="flex flex-col gap-1">
                <span class="text-zinc-400 font-semibold text-[11px]">Атрибуты:</span>
                <div class="flex flex-wrap gap-1.5">
                  {#each Object.entries(log.attributes) as [key, val]}
                    <span
                      class="rounded bg-zinc-800 px-2 py-0.5 text-[11px] font-mono text-zinc-300 border border-zinc-700/50"
                    >
                      <span class="text-zinc-400">{key}:</span>
                      {val}
                    </span>
                  {/each}
                </div>
              </div>
            {/if}

            {#if log.buildId}
              <div class="flex items-center gap-2 text-[11px] text-zinc-400">
                <span>Build ID:</span>
                <span class="font-mono text-zinc-300">{log.buildId}</span>
              </div>
            {/if}
          </div>
        {/if}
      {/each}
    {/if}
  </div>

  <div
    class="flex items-center justify-between border-t border-zinc-800 bg-zinc-950 px-4 sm:px-5 py-2 text-[11px] text-zinc-400 font-mono"
  >
    <div class="flex items-center gap-3">
      <span>Показано: {filteredLogs.length} из {logs.length} логов</span>
      {#if hasActiveFilters}
        <span class="text-amber-400 font-medium">Фильтры активны</span>
      {/if}
    </div>

    <button
      type="button"
      class="inline-flex items-center gap-1 text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
      onclick={scrollToBottom}
      title="Прокрутить в конец"
    >
      <ArrowDown class="size-3" />
      <span>В конец</span>
    </button>
  </div>
</div>

<style>
  .custom-scrollbar {
    scrollbar-width: thin;
    scrollbar-color: rgba(255, 255, 255, 0.22) transparent;
  }

  .custom-scrollbar::-webkit-scrollbar {
    width: 5px;
    height: 5px;
  }

  .custom-scrollbar::-webkit-scrollbar-track {
    background: transparent;
  }

  .custom-scrollbar::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.22);
    border-radius: 9999px;
  }

  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.38);
  }

  .custom-scrollbar::-webkit-scrollbar-button {
    display: none;
    width: 0;
    height: 0;
  }
</style>
