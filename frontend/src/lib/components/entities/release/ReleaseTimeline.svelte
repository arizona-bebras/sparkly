<script lang="ts">
  import {
    Calendar,
    Check,
    CircleAlert,
    Copy,
    Pencil,
    Plus,
    RefreshCw,
    Search,
    Sparkles,
    Tag,
    Trash2,
  } from '@lucide/svelte';
  import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { DebouncedInput } from '$lib/components/ui/debounced-input/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import ReleaseForm from '$lib/components/forms/release/Release.svelte';
  import { cn } from '$lib/utils.js';
  import { client, type Release } from '$proto';

  let { parkId }: { parkId: string } = $props();

  const queryClient = useQueryClient();

  let searchQuery = $state('');
  let copiedId = $state<string | null>(null);
  let isCreateOpen = $state(false);
  let isEditOpen = $state(false);
  let selectedRelease = $state<Release | undefined>(undefined);
  let deletingReleaseId = $state<string | null>(null);

  const releasesQuery = createQuery(() => ({
    queryKey: ['releases', parkId],
    queryFn: async () => {
      const response = await client.release.getReleases({ parkId });
      return response.releases ?? [];
    },
    enabled: !!parkId,
    staleTime: 1000 * 60,
  }));

  const deleteMutation = createMutation(() => ({
    mutationKey: ['deleteRelease', parkId],
    mutationFn: async (id: string) => {
      return await client.release.deleteRelease({ id });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['releases', parkId] });
    },
  }));

  const sortedReleases = $derived.by(() => {
    const list = (releasesQuery.data ?? []).slice();
    list.sort((a, b) => {
      const aSec = Number(a.createdAt?.seconds ?? 0n);
      const bSec = Number(b.createdAt?.seconds ?? 0n);
      if (bSec !== aSec) return bSec - aSec;
      return (b.createdAt?.nanos ?? 0) - (a.createdAt?.nanos ?? 0);
    });
    return list;
  });

  const filteredReleases = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return sortedReleases;
    return sortedReleases.filter(
      (r) => r.version.toLowerCase().includes(q) || r.name.toLowerCase().includes(q),
    );
  });

  function formatDate(ts?: { seconds: bigint; nanos: number }): string {
    if (!ts || Number(ts.seconds) === 0) return '—';
    const ms = Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000);
    const date = new Date(ms);
    return new Intl.DateTimeFormat('ru-RU', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(date);
  }

  async function copyId(id: string) {
    try {
      await navigator.clipboard.writeText(id);
      copiedId = id;
      setTimeout(() => {
        if (copiedId === id) copiedId = null;
      }, 1500);
    } catch {}
  }
</script>

<div class="flex flex-col gap-6">
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="relative w-full sm:w-72">
      <Search
        class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground pointer-events-none"
      />
      <DebouncedInput
        placeholder="Поиск по версии или названию..."
        bind:value={searchQuery}
        class="pl-9 h-9"
      />
    </div>

    <div class="flex items-center gap-2">
      <Badge variant="secondary" class="font-normal text-xs px-2.5 py-1">
        Всего релизов: {sortedReleases.length}
      </Badge>
      <Button
        variant="outline"
        size="sm"
        class="gap-1.5"
        disabled={releasesQuery.isFetching}
        onclick={() => releasesQuery.refetch()}
      >
        <RefreshCw class={cn('size-3.5', releasesQuery.isFetching && 'animate-spin')} />
        <span>Обновить</span>
      </Button>
      <Button size="sm" class="gap-1.5" onclick={() => (isCreateOpen = true)}>
        <Plus class="size-4" />
        <span>Создать релиз</span>
      </Button>
    </div>
  </div>

  {#if releasesQuery.isPending}
    <div
      class="relative pl-12 before:absolute before:left-6 before:-translate-x-1/2 before:top-6 before:bottom-4 before:w-0.5 before:bg-border/60 flex flex-col gap-6"
    >
      {#each Array(3) as _, i (i)}
        <div class="relative flex items-start">
          <div class="absolute -left-12 top-6 z-10 flex w-12 items-center justify-center">
            <Skeleton class="size-7 rounded-full" />
          </div>
          <div class="w-full rounded-xl border p-5 flex flex-col gap-3">
            <div class="flex items-center justify-between">
              <Skeleton class="h-6 w-28" />
              <Skeleton class="h-4 w-36" />
            </div>
            <Skeleton class="h-4 w-48" />
            <Skeleton class="h-6 w-32" />
          </div>
        </div>
      {/each}
    </div>
  {:else if releasesQuery.isError}
    <div
      class="rounded-xl border p-8 flex flex-col items-center justify-center gap-3 text-destructive"
    >
      <CircleAlert class="size-8" />
      <p class="font-medium">Ошибка загрузки релизов</p>
      <p class="text-xs text-muted-foreground">{releasesQuery.error?.message}</p>
      <Button variant="outline" size="sm" onclick={() => releasesQuery.refetch()}>Повторить</Button>
    </div>
  {:else if sortedReleases.length === 0}
    <div class="rounded-xl border p-12 flex flex-col items-center justify-center gap-2 text-center">
      <Tag class="size-8 text-muted-foreground" />
      <p class="font-medium text-foreground">Релизов пока нет</p>
      <p class="text-xs text-muted-foreground">Создайте первый релиз для этого парка</p>
      <Button size="sm" class="mt-2 gap-1.5" onclick={() => (isCreateOpen = true)}>
        <Plus class="size-4" />
        <span>Создать релиз</span>
      </Button>
    </div>
  {:else if filteredReleases.length === 0}
    <div class="rounded-xl border p-12 flex flex-col items-center justify-center gap-2 text-center">
      <Tag class="size-8 text-muted-foreground" />
      <p class="font-medium text-foreground">Релизы не найдены</p>
      <p class="text-xs text-muted-foreground">Попробуйте изменить параметры поиска</p>
      {#if searchQuery}
        <Button variant="outline" size="sm" class="mt-2" onclick={() => (searchQuery = '')}>
          Сбросить поиск
        </Button>
      {/if}
    </div>
  {:else}
    <div
      class="relative pl-12 before:absolute before:left-6 before:-translate-x-1/2 before:top-6 before:bottom-4 before:w-0.5 before:bg-border/80 flex flex-col gap-6"
    >
      {#each filteredReleases as release, index (release.id)}
        {@const isLatest = index === 0 && !searchQuery}
        <div class="relative flex items-start">
          <div class="absolute -left-12 top-6 z-10 flex w-12 items-center justify-center">
            {#if isLatest}
              <div
                class="size-7 rounded-full bg-primary text-primary-foreground flex items-center justify-center ring-4 ring-background shadow-xs"
              >
                <Sparkles class="size-3.5" />
              </div>
            {:else}
              <div
                class="size-7 rounded-full bg-muted border border-border text-muted-foreground flex items-center justify-center ring-4 ring-background"
              >
                <Tag class="size-3.5" />
              </div>
            {/if}
          </div>

          <Card.Root
            class={cn(
              'w-full transition-all hover:border-foreground/30',
              isLatest && 'border-primary/40 shadow-xs',
            )}
          >
            <Card.Header class="pb-3">
              <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <div class="flex items-center gap-2.5">
                  <span class="font-mono text-lg font-bold tracking-tight text-foreground">
                    {release.version}
                  </span>
                  {#if isLatest}
                    <Badge
                      variant="secondary"
                      class="gap-1.5 px-2 py-0.5 text-[11px] font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
                    >
                      <span class="size-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                      <span>Последний релиз</span>
                    </Badge>
                  {/if}
                </div>

                <div class="flex items-center gap-3">
                  <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Calendar class="size-3.5 shrink-0" />
                    <span>{formatDate(release.createdAt)}</span>
                  </div>
                  <div class="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      class="size-7 text-muted-foreground hover:text-foreground"
                      title="Редактировать"
                      onclick={() => {
                        selectedRelease = release;
                        isEditOpen = true;
                      }}
                    >
                      <Pencil class="size-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      class="size-7 text-muted-foreground hover:text-destructive"
                      title="Удалить"
                      onclick={() => (deletingReleaseId = release.id)}
                    >
                      <Trash2 class="size-3.5" />
                    </Button>
                  </div>
                </div>
              </div>
            </Card.Header>

            <Card.Content class="pt-0 flex flex-col gap-3">
              {#if release.name}
                <p class="text-sm text-muted-foreground">
                  {release.name}
                </p>
              {/if}

              <div class="flex items-center gap-2">
                <div
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted/70 font-mono text-xs text-foreground border border-border/60"
                  title="ID релиза: {release.id}"
                >
                  <Tag class="size-3.5 text-muted-foreground" />
                  <span>{release.id.slice(0, 8)}</span>
                  <button
                    type="button"
                    class="ml-1 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                    onclick={() => copyId(release.id)}
                    title="Скопировать ID релиза"
                    aria-label="Скопировать ID релиза"
                  >
                    {#if copiedId === release.id}
                      <Check class="size-3.5 text-emerald-500" />
                    {:else}
                      <Copy class="size-3.5" />
                    {/if}
                  </button>
                </div>
              </div>
            </Card.Content>
          </Card.Root>
        </div>
      {/each}
    </div>
  {/if}
</div>

<Dialog.Root bind:open={isCreateOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Создать релиз</Dialog.Title>
      <Dialog.Description>
        Укажите версию и название для создания нового релиза устройств парка.
      </Dialog.Description>
    </Dialog.Header>

    <ReleaseForm
      {parkId}
      onSuccess={() => {
        isCreateOpen = false;
        queryClient.invalidateQueries({ queryKey: ['releases', parkId] });
      }}
    />
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root
  open={isEditOpen}
  onOpenChange={(open) => {
    isEditOpen = open;
    if (!open) selectedRelease = undefined;
  }}
>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Редактировать релиз</Dialog.Title>
      <Dialog.Description>Измените информацию о релизе.</Dialog.Description>
    </Dialog.Header>

    {#if selectedRelease}
      <ReleaseForm
        {parkId}
        release={selectedRelease}
        onSuccess={() => {
          isEditOpen = false;
          selectedRelease = undefined;
          queryClient.invalidateQueries({ queryKey: ['releases', parkId] });
        }}
      />
    {/if}
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root
  open={!!deletingReleaseId}
  onOpenChange={(open) => {
    if (!open) deletingReleaseId = null;
  }}
>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Удалить релиз</Dialog.Title>
      <Dialog.Description>
        Вы уверены, что хотите удалить этот релиз? Это действие нельзя отменить.
      </Dialog.Description>
    </Dialog.Header>

    <Dialog.Footer class="gap-2 sm:gap-0">
      <Button variant="outline" onclick={() => (deletingReleaseId = null)}>Отмена</Button>
      <Button
        variant="destructive"
        disabled={deleteMutation.isPending}
        onclick={async () => {
          if (deletingReleaseId) {
            await deleteMutation.mutateAsync(deletingReleaseId);
            deletingReleaseId = null;
          }
        }}
      >
        Удалить
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
