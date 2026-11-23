<script lang="ts">
  import { page } from '$app/state';
  import { ArrowLeft, Cpu } from '@lucide/svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { TimeRangePicker, type TimeRange } from '$lib/components/ui/time-range-picker/index.js';

  const parkId = $derived(page.params.park_id ?? '');
  const deviceId = $derived(page.params.device_id ?? '');

  let selectedRange = $state<TimeRange>();
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
      <ArrowLeft class="size-4" />
      <span>Назад к списку устройств</span>
    </Button>

    <TimeRangePicker bind:value={selectedRange} syncUrl />
  </div>

  <Card.Root class="p-8">
    <div class="flex flex-col items-center justify-center text-center gap-3">
      <div class="flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
        <Cpu class="size-6" />
      </div>
      <h2 class="text-2xl font-bold tracking-tight">Добро пожаловать на страницу устройства!</h2>
      <p class="text-sm text-muted-foreground max-w-md">
        Здесь скоро появится подробная информация, мониторинг и управление устройством.
      </p>
      {#if deviceId}
        <div
          class="mt-2 inline-flex items-center gap-1.5 px-3 py-1 rounded-md bg-muted font-mono text-xs text-muted-foreground border border-border"
        >
          <span>ID: {deviceId}</span>
        </div>
      {/if}
    </div>
  </Card.Root>
</div>
