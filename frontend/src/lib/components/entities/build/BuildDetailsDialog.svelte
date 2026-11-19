<script lang="ts">
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Package, GitCommit, FileCode, Layers, Calendar, Copy, Check } from '@lucide/svelte';
  import type { Build } from '$proto';

  let {
    open = $bindable(false),
    build = null,
  }: {
    open: boolean;
    build?: Build | null;
  } = $props();

  let copiedField = $state<string | null>(null);

  async function copyValue(field: string, text: string) {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      copiedField = field;
      setTimeout(() => {
        if (copiedField === field) {
          copiedField = null;
        }
      }, 1500);
    } catch {}
  }

  function formatTimestamp(ts?: { seconds: bigint; nanos: number }): string {
    if (!ts || Number(ts.seconds) === 0) return '—';
    const ms = Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000);
    const date = new Date(ms);
    return new Intl.DateTimeFormat('ru-RU', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(date);
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <div class="flex items-center gap-2">
        <Package class="size-5 text-primary" />
        <Dialog.Title>Сборка {build?.version ?? ''}</Dialog.Title>
      </div>
      <Dialog.Description>
        Сведения о бинарных артефактах и метаданных сборки прошивки.
      </Dialog.Description>
    </Dialog.Header>

    {#if build}
      <div class="flex flex-col gap-4 py-2">
        <div class="flex flex-col gap-1.5 rounded-lg border bg-muted/20 p-3">
          <span class="text-xs font-medium text-muted-foreground">Идентификатор (UUID)</span>
          <div class="flex items-center justify-between gap-2">
            <code class="font-mono text-xs break-all text-foreground select-all">
              {build.id}
            </code>
            <Button
              variant="ghost"
              size="icon"
              class="size-7 shrink-0"
              onclick={() => copyValue('id', build?.id ?? '')}
              title="Скопировать ID"
            >
              {#if copiedField === 'id'}
                <Check class="size-3.5 text-emerald-500" />
              {:else}
                <Copy class="size-3.5" />
              {/if}
            </Button>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div class="flex flex-col gap-1.5 rounded-lg border bg-muted/20 p-3">
            <span class="text-xs font-medium text-muted-foreground">Версия</span>
            <div>
              <Badge variant="secondary" class="font-mono text-xs">
                {build.version}
              </Badge>
            </div>
          </div>

          <div class="flex flex-col gap-1.5 rounded-lg border bg-muted/20 p-3">
            <span class="text-xs font-medium text-muted-foreground">Git коммит</span>
            {#if build.commit}
              <div class="flex items-center justify-between gap-2">
                <span
                  class="inline-flex items-center gap-1.5 font-mono text-xs text-foreground truncate"
                  title={build.commit}
                >
                  <GitCommit class="size-3.5 text-muted-foreground shrink-0" />
                  {build.commit.slice(0, 10)}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  class="size-6 shrink-0"
                  onclick={() => copyValue('commit', build?.commit ?? '')}
                  title="Скопировать полный хэш"
                >
                  {#if copiedField === 'commit'}
                    <Check class="size-3 text-emerald-500" />
                  {:else}
                    <Copy class="size-3" />
                  {/if}
                </Button>
              </div>
            {:else}
              <span class="text-xs text-muted-foreground">—</span>
            {/if}
          </div>
        </div>

        <div class="flex flex-col gap-1.5 rounded-lg border bg-muted/20 p-3">
          <div class="flex items-center justify-between">
            <span
              class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground"
            >
              <FileCode class="size-3.5 text-emerald-600 dark:text-emerald-400" />
              ELF артефакт (Образ прошивки)
            </span>
            {#if build.elfObjectKey}
              <span
                class="rounded bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-medium text-emerald-700 dark:text-emerald-300"
              >
                Загружен
              </span>
            {:else}
              <span class="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
                Отсутствует
              </span>
            {/if}
          </div>
          {#if build.elfObjectKey}
            <div class="flex items-center justify-between gap-2 mt-1">
              <code class="font-mono text-xs break-all text-foreground select-all">
                {build.elfObjectKey}
              </code>
              <Button
                variant="ghost"
                size="icon"
                class="size-7 shrink-0"
                onclick={() => copyValue('elf', build?.elfObjectKey ?? '')}
                title="Скопировать S3 ключ"
              >
                {#if copiedField === 'elf'}
                  <Check class="size-3.5 text-emerald-500" />
                {:else}
                  <Copy class="size-3.5" />
                {/if}
              </Button>
            </div>
          {:else}
            <span class="text-xs text-muted-foreground">Файл прошивки не был прикреплен</span>
          {/if}
        </div>

        <div class="flex flex-col gap-1.5 rounded-lg border bg-muted/20 p-3">
          <div class="flex items-center justify-between">
            <span
              class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground"
            >
              <Layers class="size-3.5 text-indigo-600 dark:text-indigo-400" />
              DWARF артефакт (Таблица отладочных символов)
            </span>
            {#if build.dwarfObjectKey}
              <span
                class="rounded bg-indigo-500/10 px-1.5 py-0.5 text-[10px] font-medium text-indigo-700 dark:text-indigo-300"
              >
                Загружен
              </span>
            {:else}
              <span class="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
                Отсутствует
              </span>
            {/if}
          </div>
          {#if build.dwarfObjectKey}
            <div class="flex items-center justify-between gap-2 mt-1">
              <code class="font-mono text-xs break-all text-foreground select-all">
                {build.dwarfObjectKey}
              </code>
              <Button
                variant="ghost"
                size="icon"
                class="size-7 shrink-0"
                onclick={() => copyValue('dwarf', build?.dwarfObjectKey ?? '')}
                title="Скопировать S3 ключ"
              >
                {#if copiedField === 'dwarf'}
                  <Check class="size-3.5 text-emerald-500" />
                {:else}
                  <Copy class="size-3.5" />
                {/if}
              </Button>
            </div>
          {:else}
            <span class="text-xs text-muted-foreground"
              >Таблица символов DWARF не была прикреплена</span
            >
          {/if}
        </div>

        <div class="flex items-center gap-2 text-xs text-muted-foreground px-1">
          <Calendar class="size-3.5 shrink-0" />
          <span>Создана: {formatTimestamp(build.createdAt)}</span>
        </div>
      </div>
    {/if}

    <Dialog.Footer>
      <Button variant="outline" onclick={() => (open = false)}>Закрыть</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
