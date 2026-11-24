<script lang="ts">
  import { page } from '$app/state';
  import { ArrowLeft, Activity, CircleAlert, RefreshCw } from '@lucide/svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import { Separator } from '$lib/components/ui/separator/index.js';
  import { TimeRangePicker, type TimeRange } from '$lib/components/ui/time-range-picker/index.js';
  import DeviceLogsTimeline from '$lib/components/entities/device/DeviceLogsTimeline.svelte';
  import { client, type AnalyzeDeviceResult } from '$proto';
  import { createQuery } from '@tanstack/svelte-query';
  import { timestampFromDate } from '@bufbuild/protobuf/wkt';

  const parkId = $derived(page.params.park_id ?? '');
  const deviceId = $derived(page.params.device_id ?? '');

  let selectedRange = $state<TimeRange>();

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
</script>

<svelte:head>
  <title>Устройство | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-6">
  <div class="flex items-center justify-between flex-wrap gap-3">
    <Button
      variant="ghost"
      size="sm"
      class="gap-2 cursor-pointer"
      href={`/control-panel/${parkId}/devices`}
    >
      <ArrowLeft data-icon="inline-start" />
      <span>Назад к списку устройств</span>
    </Button>

    <TimeRangePicker bind:value={selectedRange} syncUrl />
  </div>

  <Card.Root>
    <Card.Header class="flex flex-row items-center justify-between flex-wrap gap-4 pb-4">
      <div class="flex flex-col gap-1">
        <div class="flex items-center gap-2">
          <div
            class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary"
          >
            <Activity class="size-4" />
          </div>
          <Card.Title class="text-xl">Хронология событий устройства</Card.Title>
        </div>
        <Card.Description>
          График логов и запусков устройства во времени с алгоритмом распределения Dodge
        </Card.Description>
      </div>

      <div class="flex items-center gap-2">
        {#if deviceId}
          <Badge variant="outline" class="font-mono text-xs">
            ID: {deviceId}
          </Badge>
        {/if}
        <Button
          variant="outline"
          size="icon-sm"
          disabled={analysisQuery.isFetching}
          onclick={() => analysisQuery.refetch()}
          title="Обновить данные"
        >
          <RefreshCw
            data-icon="inline-start"
            class={analysisQuery.isFetching ? 'animate-spin' : ''}
          />
          <span class="sr-only">Обновить</span>
        </Button>
      </div>
    </Card.Header>

    <Separator />

    <Card.Content class="pt-6">
      {#if analysisQuery.isLoading}
        <div class="flex flex-col gap-3">
          <Skeleton class="h-[240px] w-full rounded-md" />
        </div>
      {:else if analysisQuery.isError}
        <div class="flex flex-col items-center justify-center gap-3 p-8 text-center">
          <div
            class="flex size-10 items-center justify-center rounded-full bg-destructive/10 text-destructive"
          >
            <CircleAlert class="size-5" />
          </div>
          <p class="text-sm font-medium text-destructive">
            Ошибка загрузки аналитики: {analysisQuery.error?.message ??
              'Не удалось получить данные'}
          </p>
          <Button variant="outline" size="sm" onclick={() => analysisQuery.refetch()}>
            Попробовать снова
          </Button>
        </div>
      {:else if analysisQuery.data}
        <DeviceLogsTimeline
          logs={analysisQuery.data.logs}
          boots={analysisQuery.data.boots}
          from={selectedRange?.from}
          to={selectedRange?.to}
        />
      {/if}
    </Card.Content>
  </Card.Root>
</div>
