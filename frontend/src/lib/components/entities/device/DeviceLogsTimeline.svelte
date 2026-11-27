<script lang="ts">
  import { Chart, Dodge, Line, Tooltip } from 'layerchart';
  import { ChevronDown } from '@lucide/svelte';
  import * as ChartUI from '$lib/components/ui/chart/index.js';
  import * as Popover from '$lib/components/ui/popover/index.js';
  import { LogLevel, type Log, type DeviceBoot, type DeviceCrash } from '$proto';
  import { timestampDate } from '@bufbuild/protobuf/wkt';
  import { cn } from '$lib/utils.js';

  interface Props {
    logs?: Log[];
    boots?: DeviceBoot[];
    crashes?: DeviceCrash[];
    class?: string;
    from?: Date;
    to?: Date;
    activeItemId?: string | null;
    selectedLevels?: number[];
  }

  let {
    logs = [],
    boots = [],
    crashes = [],
    class: className,
    from,
    to,
    activeItemId = $bindable(null),
    selectedLevels = $bindable([]),
  }: Props = $props();

  let timelineWidth = $state(0);
  const chartPadding = { top: 28, bottom: 28, left: 16, right: 16 };
  const timePadding: [number, number] = [84, 84];
  const labelRowRadius = 8;
  const labelPadding = 4;
  const labelOffset = 26;

  const chartConfig = {
    logs: { label: 'Логи', color: 'var(--muted-foreground)' },
    boots: { label: 'Запуски', color: 'var(--muted-foreground)' },
    crashes: { label: 'Сбои', color: 'var(--foreground)' },
  } satisfies ChartUI.ChartConfig;

  type SeriesKey = keyof typeof chartConfig;

  interface LogStyle {
    level: LogLevel;
    label: string;
    color: string;
    prefix?: string;
  }

  function timelineLogColor(token: string) {
    return `color-mix(in oklab, var(${token}) 85%, var(--muted-foreground))`;
  }

  const traceLogStyle: LogStyle = {
    level: LogLevel.TRACE,
    label: 'TRACE',
    color: timelineLogColor('--color-zinc-400'),
  };
  const logStyles: LogStyle[] = [
    traceLogStyle,
    { level: LogLevel.DEBUG, label: 'DEBUG', color: timelineLogColor('--color-sky-400') },
    { level: LogLevel.INFO, label: 'INFO', color: timelineLogColor('--color-emerald-400') },
    {
      level: LogLevel.WARN,
      label: 'WARN',
      color: timelineLogColor('--color-amber-400'),
      prefix: '[WARN]',
    },
    {
      level: LogLevel.ERROR,
      label: 'ERROR',
      color: timelineLogColor('--color-red-400'),
      prefix: '[ERR]',
    },
    {
      level: LogLevel.FATAL,
      label: 'FATAL',
      color: timelineLogColor('--color-rose-500'),
      prefix: '[FATAL]',
    },
  ];

  const branchWidth = 360;
  const branchHeight = 84;
  const branchHeights = [33, 50, 67, 67, 50, 33];
  const levelBranches = logStyles.map((style, index) => ({
    ...style,
    x: ((index + 0.5) * branchWidth) / logStyles.length,
    y: branchHeights[index],
    delay: index * 24,
  }));
  let levelChevron = $state<HTMLSpanElement>();
  let levelMenu = $state<HTMLDivElement | null>(null);
  let branchOriginX = $state(branchWidth / 2);
  let hoveredLogLevel = $state<LogLevel | null>(null);

  function alignLevelBranches() {
    requestAnimationFrame(() => {
      if (!levelChevron || !levelMenu) return;
      const arrow = levelChevron.getBoundingClientRect();
      const menu = levelMenu.getBoundingClientRect();
      if (menu.width === 0) return;

      const x = ((arrow.left + arrow.width / 2 - menu.left) / menu.width) * branchWidth;
      branchOriginX = Math.min(branchWidth, Math.max(0, x));
    });
  }

  $effect(() => {
    if (!levelChevron || !levelMenu) return;
    const frame = requestAnimationFrame(alignLevelBranches);
    return () => cancelAnimationFrame(frame);
  });

  interface TimelineItem {
    id: string;
    date: Date;
    category: SeriesKey;
    label: string;
    shortLabel: string;
    color: string;
    rawLog?: Log;
    deviceEvent?: { buildId: string; reason: string };
  }

  function truncate(str: string, maxLen: number): string {
    if (str.length <= maxLen) return str;
    return str.slice(0, maxLen).trimEnd() + '…';
  }

  function toDeviceEvent(
    event: DeviceBoot | DeviceCrash,
    category: 'boots' | 'crashes',
    reason: string,
  ): TimelineItem {
    const label = `${event.buildId} (${reason})`;
    return {
      id: event.id,
      date: timestampDate(event.timestamp),
      category,
      label,
      shortLabel: truncate(label, 24),
      color: chartConfig[category].color,
      deviceEvent: { buildId: event.buildId, reason },
    };
  }

  const data = $derived<TimelineItem[]>(
    [
      ...logs
        .filter((l) => l.timestamp != null)
        .map((l) => {
          const cleanMsg = l.message.replace(/\n/g, ' ');
          const style = logStyles.find((entry) => entry.level === l.level) ?? traceLogStyle;
          const prefix = style.prefix ?? `[${LogLevel[l.level] ?? 'LOG'}]`;
          return {
            id: l.id,
            date: timestampDate(l.timestamp!),
            category: 'logs' as const,
            label: cleanMsg,
            shortLabel: `${prefix} ${truncate(cleanMsg, 23 - prefix.length)}`,
            color: style.color,
            rawLog: l,
          };
        }),
      ...boots
        .filter((b) => b.timestamp != null)
        .map((b) => toDeviceEvent(b, 'boots', b.resetReason)),
      ...crashes
        .filter((crash) => crash.timestamp != null)
        .map((crash) => toDeviceEvent(crash, 'crashes', crash.reason)),
    ].sort((a, b) => a.date.getTime() - b.date.getTime()),
  );

  const filteredData = $derived(
    selectedLevels.length === 0
      ? data
      : data.filter((item) => !item.rawLog || selectedLevels.includes(item.rawLog.level)),
  );

  const previewItems = $derived(
    hoveredLogLevel !== null &&
      selectedLevels.length > 0 &&
      !selectedLevels.includes(hoveredLogLevel)
      ? data.filter((item) => item.rawLog?.level === hoveredLogLevel)
      : [],
  );

  function logMarkerSize(level?: LogLevel) {
    return level === LogLevel.FATAL ? 10 : 8;
  }

  function toggleLevel(level: LogLevel) {
    selectedLevels = selectedLevels.includes(level)
      ? selectedLevels.filter((selected) => selected !== level)
      : [...selectedLevels, level];
    alignLevelBranches();
  }

  function labelHalfWidth(item: TimelineItem) {
    return (item.shortLabel.length * 6.5) / 2;
  }

  function maxLabelRows(items: TimelineItem[], overlapTime: number) {
    let first = 0;
    let rows = 0;
    for (const [last, item] of items.entries()) {
      while (item.date.getTime() - items[first].date.getTime() > overlapTime) first += 1;
      rows = Math.max(rows, last - first + 1);
    }
    return rows;
  }

  const chartHeight = $derived.by(() => {
    if (!timelineWidth || !data.length) return 400;

    const timeWidth = Math.max(
      1,
      timelineWidth - chartPadding.left - chartPadding.right - timePadding[0] - timePadding[1],
    );
    const start = data[0].date.getTime();
    const duration = data[data.length - 1].date.getTime() - start;
    const maxLabelWidth = data.reduce(
      (width, item) => Math.max(width, 2 * labelHalfWidth(item)),
      0,
    );
    const overlapTime = (duration * (maxLabelWidth + labelPadding)) / timeWidth;
    const rows = Math.max(
      maxLabelRows(
        filteredData.filter((item) => item.category === 'logs'),
        overlapTime,
      ),
      maxLabelRows(
        data.filter((item) => item.category !== 'logs'),
        overlapTime,
      ),
    );
    const sideHeight = labelOffset + 2 * labelRowRadius * rows;
    return Math.max(400, chartPadding.top + chartPadding.bottom + 2 * sideHeight);
  });

  const categories = ['logs', 'boots', 'crashes'] as const satisfies readonly SeriesKey[];
  const series = categories.map((key) => ({ key, ...chartConfig[key] }));

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
</script>

<svelte:window onresize={alignLevelBranches} />

{#snippet eventShape(category: SeriesKey, cx: number, cy: number, size: number)}
  {@const radius = size / 2}
  {#if category === 'logs'}
    {@const thickness = size > 8 ? 3.5 : 2.5}
    <rect
      x={cx - thickness / 2}
      y={cy - radius}
      width={thickness}
      height={size}
      rx={thickness / 2}
      stroke="none"
    />
  {:else if category === 'boots'}
    <rect x={cx - radius} y={cy - radius} width={size} height={size} rx={1} />
  {:else}
    <circle
      {cx}
      {cy}
      r={radius - 1}
      fill="var(--card)"
      stroke="var(--foreground)"
      stroke-width="1.75"
    />
  {/if}
{/snippet}

{#snippet marker(
  item: TimelineItem,
  cx: number,
  cy: number,
  opacity: number,
  isFocused: boolean,
  onpointermove: (e: PointerEvent) => void,
  onpointerleave: (e: PointerEvent) => void,
)}
  {@const size =
    item.category === 'boots'
      ? 9
      : item.category === 'crashes'
        ? 11
        : logMarkerSize(item.rawLog?.level)}
  <g
    role="img"
    aria-label={item.rawLog ? `${LogLevel[item.rawLog.level] ?? 'LOG'}: ${item.label}` : item.label}
    fill={item.color}
    {opacity}
    class="stroke-background cursor-pointer transition-opacity duration-150"
    stroke-width="1"
    {onpointermove}
    {onpointerleave}
  >
    {#if item.category === 'logs'}
      <rect x={cx - 6} y={cy - 7} width={12} height={14} fill="transparent" stroke="none" />
    {/if}
    {@render eventShape(item.category, cx, cy, isFocused ? size + 2 : size)}
  </g>
{/snippet}

<div class={cn('relative w-full', className)} bind:clientWidth={timelineWidth}>
  {#if data.length === 0}
    <div
      class="flex h-56 w-full flex-col items-center justify-center rounded-lg border border-dashed border-border p-6 text-center text-muted-foreground"
    >
      <p class="text-sm font-medium">Нет данных за выбранный интервал времени</p>
      <p class="mt-1 text-xs">Попробуйте расширить диапазон в фильтре дат выше</p>
    </div>
  {:else}
    <ChartUI.Container config={chartConfig} class="w-full aspect-auto overflow-visible">
      <Chart
        {data}
        x="date"
        {series}
        padding={chartPadding}
        xPadding={timePadding}
        height={chartHeight}
        transform={{
          mode: 'domain',
          axis: 'x',
          scaleExtent: [1, 50],
          domainExtent: { x: { min: 'data', max: 'data' } },
        }}
        motion={{ type: 'spring' }}
        clip
        axis={false}
        rule={false}
        grid={false}
      >
        {#snippet legend({ context })}
          <div
            class="absolute inset-x-0 top-0 z-10 flex justify-center gap-3 text-xs"
            aria-label="Серии таймлайна"
            role="group"
          >
            {#each categories as category (category)}
              <div class="inline-flex items-center">
                <button
                  type="button"
                  class={cn(
                    'inline-flex h-7 cursor-pointer items-center gap-1.5 rounded-md px-1.5 hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring',
                    !context.series.isVisible(category) && 'opacity-40',
                  )}
                  aria-pressed={context.series.isVisible(category)}
                  onclick={() => context.series.selectedKeys.toggle(category)}
                  onpointerenter={() => (context.series.highlightKey = category)}
                  onpointerleave={() => (context.series.highlightKey = null)}
                >
                  <svg
                    aria-hidden="true"
                    class="size-3.5"
                    viewBox="0 0 14 14"
                    fill={chartConfig[category].color}
                  >
                    {@render eventShape(category, 7, 7, category === 'logs' ? 8 : 10)}
                  </svg>
                  {chartConfig[category].label}
                </button>

                {#if category === 'logs'}
                  <Popover.Root
                    onOpenChange={(open) => {
                      if (!open) hoveredLogLevel = null;
                    }}
                  >
                    <Popover.Trigger
                      openOnHover
                      openDelay={180}
                      closeDelay={220}
                      aria-label="Фильтр по уровням логов"
                      class={cn(
                        'inline-flex h-7 cursor-pointer items-center gap-1 rounded-md px-1.5 text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring data-open:bg-muted',
                        selectedLevels.length > 0 && 'bg-muted text-foreground',
                      )}
                    >
                      {#if selectedLevels.length === 1}
                        {LogLevel[selectedLevels[0]]}
                      {:else if selectedLevels.length > 1}
                        {selectedLevels.length}
                      {/if}
                      <span bind:this={levelChevron}>
                        <ChevronDown aria-hidden="true" class="size-3.5" />
                      </span>
                    </Popover.Trigger>
                    <Popover.Content
                      bind:ref={levelMenu}
                      side="bottom"
                      sideOffset={5}
                      collisionPadding={8}
                      aria-label="Уровни логов"
                      class="w-[min(19rem,calc(100vw-1rem))] rounded-none bg-transparent p-0 shadow-none ring-0 data-open:animate-none data-closed:animate-none"
                    >
                      <div
                        class="log-level-options"
                        style:height={`${branchHeight}px`}
                        role="group"
                        aria-label="Выберите уровни"
                      >
                        <svg
                          aria-hidden="true"
                          class="log-level-branches"
                          viewBox={`0 0 ${branchWidth} ${branchHeight}`}
                          preserveAspectRatio="none"
                          fill="none"
                        >
                          {#each levelBranches as branch (branch.level)}
                            <line
                              x1={branchOriginX}
                              y1={0}
                              x2={branch.x}
                              y2={branch.y - 12}
                              pathLength="1"
                              stroke="var(--muted-foreground)"
                              stroke-width="1"
                              stroke-linecap="round"
                              vector-effect="non-scaling-stroke"
                              opacity={hoveredLogLevel === branch.level ||
                              selectedLevels.includes(branch.level)
                                ? 1
                                : 0.6}
                              style:--log-branch-delay={`${branch.delay}ms`}
                            />
                          {/each}
                        </svg>
                        {#each levelBranches as branch (branch.level)}
                          <button
                            type="button"
                            class="log-level-option cursor-pointer rounded-sm px-1 text-[11px] font-semibold focus-visible:outline-2 focus-visible:outline-ring"
                            style:--log-level-color={branch.color}
                            style:--log-branch-x={`${(branch.x * 100) / branchWidth}%`}
                            style:--log-branch-y={`${branch.y}px`}
                            style:--log-branch-delay={`${branch.delay}ms`}
                            aria-pressed={selectedLevels.includes(branch.level)}
                            onpointerenter={(event) => {
                              if (event.pointerType !== 'touch') hoveredLogLevel = branch.level;
                            }}
                            onpointerleave={() => (hoveredLogLevel = null)}
                            onclick={() => toggleLevel(branch.level)}
                          >
                            {branch.label}
                          </button>
                        {/each}
                      </div>
                    </Popover.Content>
                  </Popover.Root>
                {/if}
              </div>
            {/each}
          </div>
        {/snippet}

        {#snippet marks({ context })}
          {@const visibleSeries = context.series.visibleSeries}
          {@const visibleKeys = new Set(visibleSeries.map((s) => s.key))}
          {@const visibleItems = filteredData.filter((d) => visibleKeys.has(d.category))}
          {@const baselineY = context.height / 2}
          {@const deviceEvents = visibleItems.filter((d) => d.category !== 'logs')}
          {@const logItems = visibleItems.filter((d) => d.category === 'logs')}
          {@const isHovered = activeItemId != null}

          {#snippet labeledEvents(items: TimelineItem[], direction: -1 | 1)}
            <Dodge
              data={items}
              axis="y"
              anchor={direction < 0 ? 'bottom' : 'top'}
              baseline={baselineY}
              padding={labelPadding}
              rx={labelHalfWidth}
              ry={labelRowRadius}
            >
              {#snippet children({ items: dodged })}
                {#each dodged as { data: item, x, y } (item.id)}
                  {@const isSeriesHighlighted = context.series.isHighlighted(item.category, true)}
                  {@const isFocused = hoveredLogLevel === null && activeItemId === item.id}
                  {@const highlighted =
                    hoveredLogLevel !== null
                      ? item.rawLog?.level === hoveredLogLevel
                      : isHovered
                        ? isFocused
                        : isSeriesHighlighted}
                  {@const opacity = highlighted
                    ? 1
                    : hoveredLogLevel !== null || isHovered
                      ? 0.15
                      : 0.2}
                  {@const labelY = y + direction * labelOffset}

                  <Line
                    x1={x}
                    x2={x}
                    y1={baselineY + direction * 4}
                    y2={labelY - direction * 8}
                    strokeWidth={isFocused ? 1.5 : 1}
                    opacity={isFocused ? 1 : 0.35 * opacity}
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
                  <text
                    role="img"
                    aria-label={item.rawLog
                      ? `${LogLevel[item.rawLog.level] ?? 'LOG'}: ${item.label}`
                      : item.label}
                    {x}
                    y={labelY}
                    text-anchor="middle"
                    dominant-baseline="central"
                    fill={item.rawLog ? item.color : 'var(--foreground)'}
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
                    }}>{item.shortLabel}</text
                  >
                {/each}
              {/snippet}
            </Dodge>
          {/snippet}

          {#each dayBoundaries as day (day.getTime())}
            {@const x = context.xScale(day)}
            {#if x >= 0 && x <= context.width}
              <line
                x1={x}
                x2={x}
                y1={0}
                y2={context.height}
                stroke="currentColor"
                stroke-dasharray="4 4"
                class="stroke-muted-foreground/30 dark:stroke-muted-foreground/40"
                stroke-width="1.2"
              />
            {/if}
          {/each}

          <Line x1={0} x2={context.width} y1={baselineY} y2={baselineY} class="!stroke-border" />

          {@render labeledEvents(deviceEvents, -1)}
          {@render labeledEvents(logItems, 1)}
          {#if visibleKeys.has('logs') && previewItems.length > 0}
            <g aria-hidden="true" pointer-events="none">
              {#each previewItems as item (item.id)}
                <g fill={item.color}>
                  {@render eventShape(
                    'logs',
                    context.xGet(item),
                    baselineY,
                    logMarkerSize(item.rawLog?.level),
                  )}
                </g>
              {/each}
            </g>
          {/if}
        {/snippet}

        {#snippet tooltip({ context })}
          <Tooltip.Root {context}>
            {#snippet children({ data: item })}
              {#if item}
                {#if item.deviceEvent}
                  <Tooltip.Header>
                    {item.category === 'crashes' ? 'Сбой устройства' : 'Запуск устройства'}
                  </Tooltip.Header>
                  <Tooltip.List>
                    <Tooltip.Item label="Время" value={item.date.toLocaleString()} />
                    <Tooltip.Item label="Версия" value={item.deviceEvent.buildId} />
                    <Tooltip.Item label="Причина" value={item.deviceEvent.reason} />
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
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
        <span class="font-medium text-foreground/80">Уровни логов</span>
        {#each logStyles as style (style.label)}
          <span class="inline-flex items-center gap-1.5">
            <svg aria-hidden="true" class="size-3.5" viewBox="0 0 14 14" fill={style.color}>
              {@render eventShape('logs', 7, 7, logMarkerSize(style.level))}
            </svg>
            {style.label}
          </span>
        {/each}
      </div>
      <div>
        Всего событий: {filteredData.length}
      </div>
    </div>
  {/if}
</div>

<style>
  .log-level-options {
    position: relative;
    --log-branch-enter-y: -5px;
  }

  .log-level-branches {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    overflow: visible;
    pointer-events: none;
  }

  .log-level-branches line {
    stroke-dasharray: 1;
    animation: draw-log-branch 280ms ease-out var(--log-branch-delay) both;
  }

  .log-level-option {
    position: absolute;
    top: var(--log-branch-y);
    left: var(--log-branch-x);
    min-height: 1.75rem;
    transform: translate(-50%, -50%);
    color: color-mix(in oklab, var(--log-level-color) 68%, var(--foreground));
    animation: reveal-log-label 180ms ease-out calc(135ms + var(--log-branch-delay)) both;
  }

  .log-level-option:hover,
  .log-level-option[aria-pressed='true'] {
    text-decoration: underline;
    text-underline-offset: 0.2em;
  }

  :global([data-side='top']) .log-level-options {
    --log-branch-enter-y: 5px;
  }

  :global([data-side='top']) .log-level-branches {
    transform: scaleY(-1);
  }

  :global([data-side='top']) .log-level-option {
    top: calc(100% - var(--log-branch-y));
  }

  @keyframes draw-log-branch {
    from {
      stroke-dashoffset: 1;
    }
    to {
      stroke-dashoffset: 0;
    }
  }

  @keyframes reveal-log-label {
    from {
      opacity: 0;
      translate: 0 var(--log-branch-enter-y);
    }
    to {
      opacity: 1;
      translate: 0 0;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .log-level-branches line,
    .log-level-option {
      animation: none;
    }
  }
</style>
