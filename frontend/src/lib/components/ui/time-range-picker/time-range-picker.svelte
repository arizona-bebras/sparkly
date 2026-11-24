<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import type { DateRange } from 'bits-ui';
  import { today, getLocalTimeZone } from '@internationalized/date';
  import { Calendar, ChevronDown, Clock } from '@lucide/svelte';
  import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import * as Popover from '$lib/components/ui/popover/index.js';
  import { RangeCalendar } from '$lib/components/ui/range-calendar/index.js';
  import { cn } from '$lib/utils.js';
  import {
    PRESETS,
    calendarDateToDate,
    dateToCalendarDate,
    formatDisplayDates,
    formatDisplayTimes,
    getTimeString,
    type PresetKey,
    type TimeRange,
  } from './types.js';

  let {
    value = $bindable(),
    syncUrl = true,
    class: className,
  }: {
    value?: TimeRange;
    syncUrl?: boolean;
    class?: string;
  } = $props();

  let isOpen = $state(false);

  const defaultPreset = PRESETS.find((p) => p.id === 'today')!;

  function initFromUrlOrState(): TimeRange {
    if (syncUrl) {
      const urlRange = page.url.searchParams.get('range') as PresetKey | null;
      if (urlRange) {
        const found = PRESETS.find((p) => p.id === urlRange);
        if (found) {
          return { ...found.getRange(), preset: found.id };
        }
      }

      const urlFrom = page.url.searchParams.get('from');
      const urlTo = page.url.searchParams.get('to');
      if (urlFrom && urlTo) {
        const f = new Date(urlFrom);
        const t = new Date(urlTo);
        if (!isNaN(f.getTime()) && !isNaN(t.getTime())) {
          return { from: f, to: t };
        }
      }
    }

    if (value) return value;
    return { ...defaultPreset.getRange(), preset: defaultPreset.id };
  }

  const initialRange = initFromUrlOrState();
  let internalValue = $state<TimeRange>(initialRange);

  $effect(() => {
    if (!value) {
      value = internalValue;
    }
  });

  let calendarRange = $state<DateRange>({
    start: dateToCalendarDate(initialRange.from),
    end: dateToCalendarDate(initialRange.to),
  });
  let fromTimeString = $state(getTimeString(initialRange.from));
  let toTimeString = $state(getTimeString(initialRange.to));
  let draftPreset = $state<PresetKey | undefined>(initialRange.preset);

  $effect(() => {
    if (isOpen) {
      calendarRange = {
        start: dateToCalendarDate(internalValue.from),
        end: dateToCalendarDate(internalValue.to),
      };
      fromTimeString = getTimeString(internalValue.from);
      toTimeString = getTimeString(internalValue.to);
      draftPreset = internalValue.preset;
    }
  });

  function updateUrl(range: TimeRange) {
    if (!syncUrl) return;
    const url = new URL(page.url);
    if (range.preset) {
      url.searchParams.set('range', range.preset);
      url.searchParams.delete('from');
      url.searchParams.delete('to');
    } else {
      url.searchParams.delete('range');
      url.searchParams.set('from', range.from.toISOString());
      url.searchParams.set('to', range.to.toISOString());
    }

    if (url.search !== page.url.search) {
      goto(url.toString(), {
        replaceState: true,
        noScroll: true,
        keepFocus: true,
      });
    }
  }

  function selectPreset(presetId: PresetKey) {
    const preset = PRESETS.find((p) => p.id === presetId);
    if (!preset) return;
    const range = preset.getRange();

    draftPreset = preset.id;
    calendarRange = {
      start: dateToCalendarDate(range.from),
      end: dateToCalendarDate(range.to),
    };
    fromTimeString = getTimeString(range.from);
    toTimeString = getTimeString(range.to);
  }

  function parseTime(str: string, defaultH: number, defaultM: number): [number, number] {
    const parts = str.split(':').map((p) => parseInt(p, 10));
    const h = !isNaN(parts[0]) && parts[0] >= 0 && parts[0] <= 23 ? parts[0] : defaultH;
    const m = !isNaN(parts[1]) && parts[1] >= 0 && parts[1] <= 59 ? parts[1] : defaultM;
    return [h, m];
  }

  function handleTimeInput(e: Event, setter: (val: string) => void) {
    draftPreset = undefined;
    const target = e.currentTarget as HTMLInputElement;
    let val = target.value.replace(/[^\d:]/g, '');
    if (val.length === 2 && !val.includes(':')) {
      val = val + ':';
    }
    if (val.length > 5) {
      val = val.slice(0, 5);
    }
    target.value = val;
    setter(val);
  }

  function applyCustomRange() {
    if (!calendarRange.start) return;

    const startDate = calendarDateToDate(calendarRange.start);
    const endDate = calendarRange.end ? calendarDateToDate(calendarRange.end) : new Date(startDate);

    const [fromH, fromM] = parseTime(fromTimeString, 0, 0);
    const [toH, toM] = parseTime(toTimeString, 23, 59);

    startDate.setHours(fromH, fromM, 0, 0);
    endDate.setHours(toH, toM, 59, 999);

    const newRange: TimeRange = {
      from: startDate,
      to: endDate,
      preset: draftPreset,
    };

    internalValue = newRange;
    value = newRange;
    updateUrl(newRange);
    isOpen = false;
  }

  function setToNow() {
    draftPreset = undefined;
    calendarRange = {
      ...calendarRange,
      end: today(getLocalTimeZone()),
    };
    toTimeString = getTimeString(new Date());
  }

  const displayDates = $derived(formatDisplayDates(internalValue.from, internalValue.to));
  const displayTimes = $derived(formatDisplayTimes(internalValue.from, internalValue.to));

  onMount(() => {
    if (syncUrl && !page.url.searchParams.get('range') && !page.url.searchParams.get('from')) {
      updateUrl(internalValue);
    }
  });
</script>

<Popover.Root bind:open={isOpen}>
  <Popover.Trigger
    class={buttonVariants({
      variant: 'outline',
      class: cn(
        'h-9 px-2.5 py-1 gap-2 cursor-pointer text-left bg-background hover:bg-muted/50 border-border/80 rounded-lg transition-colors shrink-0',
        className,
      ),
    })}
  >
    <Calendar class="size-3.5 text-muted-foreground shrink-0 self-center" />

    <div class="flex flex-col justify-center min-w-0 leading-none gap-0.5">
      <span class="text-xs font-semibold tracking-tight text-foreground select-none">
        {displayDates}
      </span>
      <span class="text-[10px] font-mono text-muted-foreground/80 tracking-tight select-none">
        {displayTimes}
      </span>
    </div>

    <ChevronDown
      class={cn(
        'size-3 text-muted-foreground/70 shrink-0 ml-0.5 self-center transition-transform duration-200',
        isOpen && 'rotate-180',
      )}
    />
  </Popover.Trigger>

  <Popover.Content
    class="w-auto p-0 shadow-xl overflow-hidden rounded-xl border border-border"
    align="end"
  >
    <div class="flex flex-col sm:flex-row divide-y sm:divide-y-0 sm:divide-x divide-border">
      <div class="flex flex-col p-3.5 w-[295px] gap-2.5">
        <div class="flex items-center justify-between">
          <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Выбор даты
          </span>
          <Calendar class="size-3.5 text-muted-foreground" />
        </div>

        <div class="flex justify-center">
          <RangeCalendar
            bind:value={calendarRange}
            numberOfMonths={1}
            locale="ru-RU"
            maxValue={today(getLocalTimeZone())}
          />
        </div>
      </div>

      <div class="flex flex-col p-3.5 w-[245px] gap-2.5 bg-muted/20">
        <span
          class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground px-0.5"
        >
          Быстрые диапазоны
        </span>

        <div class="flex flex-col gap-0.5 flex-1 max-h-[300px] overflow-y-auto pr-0.5">
          {#each PRESETS as preset (preset.id)}
            {@const isActive = draftPreset === preset.id}
            <button
              type="button"
              class={cn(
                'flex items-center justify-between px-2.5 py-1.5 text-xs rounded-md text-left transition-colors cursor-pointer',
                isActive
                  ? 'bg-primary/10 text-primary font-medium border-l-2 border-primary pl-2'
                  : 'text-foreground/80 hover:bg-muted hover:text-foreground',
              )}
              onclick={() => selectPreset(preset.id)}
            >
              <span>{preset.label}</span>
              {#if isActive}
                <span class="size-1.5 rounded-full bg-primary shrink-0"></span>
              {/if}
            </button>
          {/each}
        </div>
      </div>
    </div>

    <div
      class="border-t border-border bg-muted/40 p-3 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3"
    >
      <div class="flex items-center gap-2 text-xs">
        <span class="text-muted-foreground font-medium">От:</span>
        <Input
          type="text"
          inputmode="numeric"
          placeholder="00:00"
          maxlength={5}
          bind:value={fromTimeString}
          oninput={(e) => handleTimeInput(e, (v) => (fromTimeString = v))}
          class="h-8 w-16 text-xs font-mono text-center px-1 bg-background"
        />

        <span class="text-muted-foreground font-medium ml-1">До:</span>
        <Input
          type="text"
          inputmode="numeric"
          placeholder="23:59"
          maxlength={5}
          bind:value={toTimeString}
          oninput={(e) => handleTimeInput(e, (v) => (toTimeString = v))}
          class="h-8 w-16 text-xs font-mono text-center px-1 bg-background"
        />

        <Button
          variant="outline"
          size="sm"
          class="h-8 px-2 text-xs text-muted-foreground hover:text-foreground shrink-0 cursor-pointer bg-background"
          onclick={setToNow}
          title="Установить текущее время"
        >
          Сейчас
        </Button>
      </div>

      <Button size="sm" class="h-8 gap-1.5 cursor-pointer shrink-0" onclick={applyCustomRange}>
        Применить диапазон
      </Button>
    </div>
  </Popover.Content>
</Popover.Root>
