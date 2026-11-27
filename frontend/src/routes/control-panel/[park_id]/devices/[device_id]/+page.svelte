<script lang="ts">
  import { page } from '$app/state';
  import { ArrowLeft, CircleAlert, RefreshCw } from '@lucide/svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import { TimeRangePicker, type TimeRange } from '$lib/components/ui/time-range-picker/index.js';
  import DeviceLogsTimeline from '$lib/components/entities/device/DeviceLogsTimeline.svelte';
  import DeviceLogsConsole from '$lib/components/entities/device/DeviceLogsConsole.svelte';
  import { client, type AnalyzeDeviceResult } from '$proto';
  import { createQuery } from '@tanstack/svelte-query';
  import { timestampFromDate } from '@bufbuild/protobuf/wkt';
  import { cn } from '$lib/utils.js';

  const parkId = $derived(page.params.park_id ?? '');
  const deviceId = $derived(page.params.device_id ?? '');

  let selectedRange = $state<TimeRange>();
  let activeLogId = $state<string | null>(null);
  let selectedLogLevels = $state<number[]>([]);

  const deviceQuery = createQuery(() => ({
    queryKey: ['device', deviceId],
    queryFn: async () => {
      const res = await client.device.getDevice({ id: deviceId });
      return res.device;
    },
    enabled: !!deviceId,
  }));

  const analysisQuery = createQuery(() => ({
    queryKey: [
      'device-analysis',
      deviceId,
      selectedRange?.from?.toISOString(),
      selectedRange?.to?.toISOString(),
    ],
    queryFn: async () => {
      const res = await client.analytics.analyzeDevice({
        deviceId,
        timerange: selectedRange
          ? {
              fromTime: timestampFromDate(selectedRange.from),
              toTime: timestampFromDate(selectedRange.to),
            }
          : undefined,
      });
      return res as AnalyzeDeviceResult;
    },
    enabled: !!deviceId,
  }));

  const latestBoot = $derived(
    analysisQuery.data?.boots && analysisQuery.data.boots.length > 0
      ? analysisQuery.data.boots[analysisQuery.data.boots.length - 1]
      : null,
  );

  const deviceName = $derived(deviceQuery.data?.name || deviceId);
  const isFetching = $derived(analysisQuery.isFetching || deviceQuery.isFetching);
</script>

<svelte:head>
  <title>{deviceName} | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-between flex-wrap gap-3">
    <div class="flex items-center gap-2.5 flex-wrap">
      <Button
        variant="ghost"
        size="sm"
        class="h-8 gap-1.5 px-2 text-muted-foreground hover:text-foreground cursor-pointer"
        href={`/control-panel/${parkId}/devices`}
      >
        <ArrowLeft class="size-3.5" />
        <span class="text-xs font-medium">Устройства</span>
      </Button>

      <span class="text-muted-foreground/30 text-xs">/</span>

      {#if deviceQuery.isLoading && !deviceQuery.data}
        <Skeleton class="h-6 w-32 rounded" />
      {:else}
        <span
          class="text-xs font-semibold tracking-tight text-foreground bg-muted/50 px-2 py-0.5 rounded border border-border/50 select-all"
          title={`ID: ${deviceId}`}
        >
          {deviceName}
        </span>
      {/if}

      {#if latestBoot?.buildId}
        <Badge
          variant="outline"
          class="font-mono text-[10px] px-1.5 py-0 h-5 font-normal text-muted-foreground"
        >
          {latestBoot.buildId}
        </Badge>
      {/if}
    </div>

    <div class="flex items-center gap-2">
      <TimeRangePicker bind:value={selectedRange} syncUrl />

      <Button
        variant="outline"
        size="icon-sm"
        class="h-9 w-9 cursor-pointer shrink-0"
        disabled={isFetching}
        onclick={() => {
          deviceQuery.refetch();
          analysisQuery.refetch();
        }}
        title="Обновить данные"
      >
        <RefreshCw class={cn('size-3.5', isFetching && 'animate-spin')} />
        <span class="sr-only">Обновить</span>
      </Button>
    </div>
  </div>

  {#if analysisQuery.isLoading}
    <div class="rounded-xl border border-border/70 bg-card/50 p-5 flex flex-col gap-4">
      <div class="flex items-center justify-between">
        <Skeleton class="h-5 w-48 rounded" />
        <Skeleton class="h-5 w-24 rounded" />
      </div>
      <Skeleton class="h-[220px] w-full rounded-md" />
      <Skeleton class="h-[320px] w-full rounded-md" />
    </div>
  {:else if analysisQuery.isError}
    <div
      class="rounded-xl border border-border/70 bg-card/50 p-8 flex flex-col items-center justify-center gap-3 text-center"
    >
      <div
        class="flex size-10 items-center justify-center rounded-full bg-destructive/10 text-destructive"
      >
        <CircleAlert class="size-5" />
      </div>
      <p class="text-sm font-medium text-destructive">
        Ошибка загрузки аналитики: {analysisQuery.error?.message ?? 'Не удалось получить данные'}
      </p>
      <Button
        variant="outline"
        size="sm"
        onclick={() => {
          deviceQuery.refetch();
          analysisQuery.refetch();
        }}
      >
        Попробовать снова
      </Button>
    </div>
  {:else if analysisQuery.data}
    <div class="rounded-xl border border-border/70 bg-card overflow-hidden">
      <div class="p-4 sm:p-5 pb-3">
        <DeviceLogsTimeline
          logs={analysisQuery.data.logs}
          boots={analysisQuery.data.boots}
          crashes={analysisQuery.data.crashes}
          from={selectedRange?.from}
          to={selectedRange?.to}
          bind:activeItemId={activeLogId}
          bind:selectedLevels={selectedLogLevels}
        />
      </div>

      <DeviceLogsConsole
        logs={analysisQuery.data.logs}
        bind:activeLogId
        bind:selectedLevels={selectedLogLevels}
      />
    </div>
  {/if}
</div>
