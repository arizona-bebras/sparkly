<script lang="ts">
  import { Chart, Dodge, Layer, Line, Text, Tooltip } from 'layerchart';
  import * as ChartUI from '$lib/components/ui/chart/index.js';
  import { LogLevel, type Log, type DeviceBoot } from '$proto';
  import { timestampDate } from '@bufbuild/protobuf/wkt';
  import { cn } from '$lib/utils.js';

  interface Props {
    logs?: Log[];
    boots?: DeviceBoot[];
    class?: string;
    from?: Date;
    to?: Date;
    activeItemId?: string | null;
  }

  let {
    logs = [],
    boots = [],
    class: className,
    from,
    to,
    activeItemId = $bindable(null),
  }: Props = $props();

  type SeriesKey = 'logs' | 'boots';
  type Shape = 'circle' | 'triangle' | 'diamond' | 'square';

  interface TimelineItem {
    id: string;
    date: Date;
    category: SeriesKey;
    shape: Shape;
    label: string;
    shortLabel: string;
    hasLabel: boolean;
    color: string;
    rawLog?: Log;
    rawBoot?: DeviceBoot;
  }

  function truncate(str: string, maxLen: number): string {
    if (str.length <= maxLen) return str;
    return str.slice(0, maxLen).trimEnd() + '…';
  }

  const data = $derived<TimelineItem[]>(
    [
      ...logs
        .filter((l) => l.timestamp != null)
        .map((l) => {
          const cleanMsg = l.message.replace(/\n/g, ' ');
          const isError = l.level === LogLevel.FATAL || l.level === LogLevel.ERROR;
          const isWarn = l.level === LogLevel.WARN;
          const shape: Shape = isError ? 'diamond' : isWarn ? 'triangle' : 'circle';
          const prefix = isError ? '[ERR] ' : isWarn ? '[WARN] ' : '';
          const hasLabel = isError || isWarn;
          return {
            id: l.id,
            date: timestampDate(l.timestamp!),
            category: 'logs' as const,
            shape,
            label: cleanMsg,
            shortLabel: hasLabel ? prefix + truncate(cleanMsg, isError ? 15 : 14) : '',
            hasLabel,
            color: 'var(--color-info, #3b82f6)',
            rawLog: l,
          };
        }),
      ...boots
        .filter((b) => b.timestamp != null)
        .map((b) => {
          const full = `${b.buildId} (${b.resetReason})`;
          return {
            id: b.id,
            date: timestampDate(b.timestamp),
            category: 'boots' as const,
            shape: 'square' as const,
            label: full,
            shortLabel: truncate(full, 24),
            hasLabel: true,
            color: '#10b981',
            rawBoot: b,
          };
        }),
    ].sort((a, b) => a.date.getTime() - b.date.getTime()),
  );

  function labelHalfWidth(label: string) {
    return (label.length * 6.5) / 2;
  }

  const series = [
    { key: 'logs', label: 'Логи', color: 'var(--color-info, #3b82f6)' },
    { key: 'boots', label: 'Запуски', color: '#10b981' },
  ];

  const minTime = $derived(from?.getTime() ?? data[0]?.date.getTime() ?? Date.now());
  const maxTime = $derived(to?.getTime() ?? data[data.length - 1]?.date.getTime() ?? Date.now());
  const isMultiDay = $derived(maxTime - minTime > 26 * 60 * 60 * 1000);

  const dayBoundaries = $derived.by(() => {
    if (data.length === 0 || !isMultiDay) return [];
    const boundaries: Date[] = [];
    const cur = new Date(minTime);
    cur.setHours(0, 0, 0, 0);
    if (cur.getTime() <= minTime) {
      cur.setDate(cur.getDate() + 1);
    }
    while (cur.getTime() < maxTime) {
      boundaries.push(new Date(cur));
      cur.setDate(cur.getDate() + 1);
    }
    return boundaries;
  });

  function formatAxisTick(d: unknown, index?: number): string {
    if (d instanceof Date) {
      const hours = d.getHours();
      const minutes = d.getMinutes();
      const isMidnight = hours === 0 && minutes === 0;

      if (isMultiDay) {
        if (isMidnight || index === 0) {
          const day = d.getDate();
          const month = d.toLocaleDateString('ru-RU', { month: 'short' }).replace('.', '');
          return `${day} ${month}`;
        }
      }

      const hStr = hours.toString().padStart(2, '0');
      const mStr = minutes.toString().padStart(2, '0');
      return `${hStr}:${mStr}`;
    }
    return String(d ?? '');
  }

  const chartConfig = {
    logs: { label: 'Логи', color: '#3b82f6' },
    boots: { label: 'Запуски', color: '#10b981' },
  } satisfies ChartUI.ChartConfig;
</script>

{#snippet marker(
  item: TimelineItem,
  cx: number,
  cy: number,
  opacity: number,
  fill: string,
  isFocused: boolean,
  onpointermove: (e: PointerEvent) => void,
  onpointerleave: (e: PointerEvent) => void,
)}
  {#if item.shape === 'diamond'}
    <polygon
      role="img"
      aria-label={item.label}
      points={isFocused
        ? `${cx},${cy - 6} ${cx + 6},${cy} ${cx},${cy + 6} ${cx - 6},${cy}`
        : `${cx},${cy - 5} ${cx + 5},${cy} ${cx},${cy + 5} ${cx - 5},${cy}`}
      {fill}
      {opacity}
      class="stroke-background cursor-pointer transition-all duration-150"
      stroke-width="1.5"
      {onpointermove}
      {onpointerleave}
    />
  {:else if item.shape === 'triangle'}
    <polygon
      role="img"
      aria-label={item.label}
      points={isFocused
        ? `${cx},${cy - 6} ${cx + 6},${cy + 5} ${cx - 6},${cy + 5}`
        : `${cx},${cy - 5} ${cx + 5},${cy + 4} ${cx - 5},${cy + 4}`}
      {fill}
      {opacity}
      class="stroke-background cursor-pointer transition-all duration-150"
      stroke-width="1.5"
      {onpointermove}
      {onpointerleave}
    />
  {:else if item.shape === 'square'}
    <rect
      role="img"
      aria-label={item.label}
      x={isFocused ? cx - 4 : cx - 3.5}
      y={isFocused ? cy - 4 : cy - 3.5}
      width={isFocused ? 8 : 7}
      height={isFocused ? 8 : 7}
      rx={1.5}
      {fill}
      {opacity}
      class="stroke-background cursor-pointer transition-all duration-150"
      stroke-width="1.5"
      {onpointermove}
      {onpointerleave}
    />
  {:else}
    <circle
      role="img"
      aria-label={item.label}
      {cx}
      {cy}
      r={isFocused ? 4 : 3}
      {fill}
      {opacity}
      class="stroke-background cursor-pointer transition-all duration-150"
      stroke-width="1.5"
      {onpointermove}
      {onpointerleave}
    />
  {/if}
{/snippet}

<div class={cn('relative w-full', className)}>
  {#if data.length === 0}
    <div
      class="flex h-56 w-full flex-col items-center justify-center rounded-lg border border-dashed border-border p-6 text-center text-muted-foreground"
    >
      <p class="text-sm font-medium">Нет данных за выбранный интервал времени</p>
      <p class="mt-1 text-xs">Попробуйте расширить диапазон в фильтре дат выше</p>
    </div>
  {:else}
    <ChartUI.Container
      config={chartConfig}
      class="min-h-[260px] w-full aspect-auto overflow-visible"
    >
      <Chart
        {data}
        x="date"
        {series}
        padding={{ top: 28, bottom: 24, left: 32, right: 32 }}
        xPadding={[50, 50]}
        height={240}
        transform={{
          mode: 'domain',
          axis: 'x',
          scaleExtent: [1, 50],
          domainExtent: { x: { min: 'data', max: 'data' } },
        }}
        motion={{ type: 'spring' }}
        clip
        axis="x"
        props={{
          xAxis: {
            format: formatAxisTick,
          },
        }}
        legend={{ placement: 'top', variant: 'swatches' }}
      >
        {#snippet aboveContext({ context })}
          {@const visibleSeries = context.series.visibleSeries}
          {@const visibleKeys = new Set(visibleSeries.map((s) => s.key))}
          {@const visibleItems = data.filter((d) => visibleKeys.has(d.category))}
          {@const baselineY = context.height}
          {@const labeledItems = visibleItems.filter((d) => d.hasLabel)}
          {@const dotOnlyItems = visibleItems.filter((d) => !d.hasLabel)}
          {@const isHovered = activeItemId != null}

          <Layer>
            {#each dayBoundaries as day (day.getTime())}
              {@const x = context.xScale(day)}
              {#if x >= 0 && x <= context.width}
                <line
                  x1={x}
                  x2={x}
                  y1={0}
                  y2={baselineY}
                  stroke="currentColor"
                  stroke-dasharray="4 4"
                  class="stroke-muted-foreground/30 dark:stroke-muted-foreground/40"
                  stroke-width="1.2"
                />
              {/if}
            {/each}

            {#each dotOnlyItems as item (item.id)}
              {@const isSeriesHighlighted = context.series.isHighlighted(item.category, true)}
              {@const isFocused = activeItemId === item.id}
              {@const opacity = isHovered ? (isFocused ? 1 : 0.15) : isSeriesHighlighted ? 1 : 0.2}
              {@const x = context.xGet(item)}

              {@render marker(
                item,
                x,
                baselineY,
                opacity,
                item.color,
                isFocused,
                (e) => {
                  activeItemId = item.id;
                  context.tooltip.show(e, item);
                },
                () => {
                  activeItemId = null;
                  context.tooltip.hide();
                },
              )}
            {/each}

            <Dodge
              data={labeledItems}
              axis="y"
              anchor="bottom"
              baseline={baselineY - 36}
              padding={6}
              rx={(d) => labelHalfWidth(d.shortLabel)}
              ry={8}
            >
              {#snippet children({ items: dodged })}
                {#each dodged as { data: item, x, y, index } (index)}
                  {@const isSeriesHighlighted = context.series.isHighlighted(item.category, true)}
                  {@const isFocused = activeItemId === item.id}
                  {@const opacity = isHovered
                    ? isFocused
                      ? 1
                      : 0.15
                    : isSeriesHighlighted
                      ? 1
                      : 0.2}
                  {@const labelY = y - 6}

                  <Line
                    x1={x}
                    x2={x}
                    y1={baselineY - 4}
                    y2={labelY + 6}
                    strokeWidth={isFocused ? 1.5 : 1}
                    {opacity}
                    class={cn(
                      '!stroke-muted-foreground/80 transition-all duration-150',
                      isFocused && '!stroke-foreground/90',
                    )}
                  />
                  {@render marker(
                    item,
                    x,
                    baselineY,
                    opacity,
                    item.color,
                    isFocused,
                    (e) => {
                      activeItemId = item.id;
                      context.tooltip.show(e, item);
                    },
                    () => {
                      activeItemId = null;
                      context.tooltip.hide();
                    },
                  )}
                  <Text
                    {x}
                    y={labelY}
                    value={item.shortLabel}
                    textAnchor="middle"
                    verticalAnchor="middle"
                    fill={item.color}
                    {opacity}
                    class={cn(
                      'text-[11px] font-medium cursor-pointer transition-all duration-150',
                      isFocused && 'font-semibold',
                    )}
                    onpointermove={(e) => {
                      activeItemId = item.id;
                      context.tooltip.show(e, item);
                    }}
                    onpointerleave={() => {
                      activeItemId = null;
                      context.tooltip.hide();
                    }}
                  />
                {/each}
              {/snippet}
            </Dodge>
          </Layer>
        {/snippet}

        {#snippet tooltip({ context })}
          <Tooltip.Root {context}>
            {#snippet children({ data: item })}
              {#if item}
                {#if item.category === 'boots' && item.rawBoot}
                  <Tooltip.Header>Запуск устройства</Tooltip.Header>
                  <Tooltip.List>
                    <Tooltip.Item label="Время" value={item.date.toLocaleString()} />
                    <Tooltip.Item label="Версия" value={item.rawBoot.buildId} />
                    <Tooltip.Item label="Причина" value={item.rawBoot.resetReason} />
                  </Tooltip.List>
                {:else if item.rawLog}
                  <Tooltip.Header>{item.label}</Tooltip.Header>
                  <Tooltip.List>
                    <Tooltip.Item label="Время" value={item.date.toLocaleString()} />
                    <Tooltip.Item label="Уровень" value={LogLevel[item.rawLog.level] ?? 'LOG'} />
                    {#if item.rawLog.stack && item.rawLog.stack.length > 0}
                      <Tooltip.Item
                        label="Стек"
                        value={`${item.rawLog.stack[0].file}:${item.rawLog.stack[0].line}`}
                      />
                    {/if}
                    {#if Object.keys(item.rawLog.attributes || {}).length > 0}
                      {#each Object.entries(item.rawLog.attributes) as [k, v]}
                        <Tooltip.Item label={k} value={v} />
                      {/each}
                    {/if}
                  </Tooltip.List>
                {/if}
              {/if}
            {/snippet}
          </Tooltip.Root>
        {/snippet}
      </Chart>
    </ChartUI.Container>

    <div
      class="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-border/40 pt-2 text-xs text-muted-foreground"
    >
      <div class="flex flex-wrap items-center gap-4">
        <span class="font-medium text-foreground/80">Обозначения:</span>
        <span class="inline-flex items-center gap-1.5">
          <svg class="size-2.5 text-[#10b981]" viewBox="0 0 10 10" fill="currentColor">
            <rect width="10" height="10" rx="2" />
          </svg>
          Запуск
        </span>
        <span class="inline-flex items-center gap-1.5">
          <svg class="size-2.5 text-[#3b82f6]" viewBox="0 0 10 10" fill="currentColor">
            <polygon points="5,0 10,5 5,10 0,5" />
          </svg>
          Ошибка
        </span>
        <span class="inline-flex items-center gap-1.5">
          <svg class="size-2.5 text-[#3b82f6]" viewBox="0 0 10 10" fill="currentColor">
            <polygon points="5,0 10,9 0,9" />
          </svg>
          Предупреждение
        </span>
        <span class="inline-flex items-center gap-1.5">
          <svg class="size-2.5 text-[#3b82f6]" viewBox="0 0 10 10" fill="currentColor">
            <circle cx="5" cy="5" r="4.5" />
          </svg>
          Инфо / Отладка
        </span>
      </div>
      <div>
        Всего событий: {data.length}
      </div>
    </div>
  {/if}
</div>
