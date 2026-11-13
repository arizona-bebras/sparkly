<script lang="ts">
  import CalendarIcon from '@lucide/svelte/icons/calendar';
  import CheckIcon from '@lucide/svelte/icons/check';
  import CopyIcon from '@lucide/svelte/icons/copy';
  import GitCommitIcon from '@lucide/svelte/icons/git-commit';
  import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw';
  import SearchIcon from '@lucide/svelte/icons/search';
  import SparklesIcon from '@lucide/svelte/icons/sparkles';
  import TagIcon from '@lucide/svelte/icons/tag';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { cn } from '$lib/utils.js';
  import { getMockReleases } from './mock.js';

  let { parkId }: { parkId: string } = $props();

  let searchQuery = $state('');
  let isRefreshing = $state(false);
  let copiedCommit = $state<string | null>(null);

  const allReleases = $derived(getMockReleases(parkId));

  const filteredReleases = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return allReleases;
    return allReleases.filter(
      (r) => r.version.toLowerCase().includes(q) || r.commit.toLowerCase().includes(q),
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

  async function copyCommit(commit: string) {
    try {
      await navigator.clipboard.writeText(commit);
      copiedCommit = commit;
      setTimeout(() => {
        if (copiedCommit === commit) copiedCommit = null;
      }, 1500);
    } catch {}
  }

  async function refresh() {
    isRefreshing = true;
    await new Promise((r) => setTimeout(r, 400));
    isRefreshing = false;
  }
</script>

<div class="flex flex-col gap-6">
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="relative w-full sm:w-72">
      <SearchIcon
        class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground pointer-events-none"
      />
      <Input
        placeholder="Поиск по версии или коммиту..."
        bind:value={searchQuery}
        class="pl-9 h-9"
      />
    </div>

    <div class="flex items-center gap-2">
      <Badge variant="secondary" class="font-normal text-xs px-2.5 py-1">
        Всего релизов: {allReleases.length}
      </Badge>
      <Button variant="outline" size="sm" class="gap-1.5" disabled={isRefreshing} onclick={refresh}>
        <RefreshCwIcon class={cn('size-3.5', isRefreshing && 'animate-spin')} />
        <span>Обновить</span>
      </Button>
    </div>
  </div>

  {#if filteredReleases.length === 0}
    <div class="rounded-xl border p-12 flex flex-col items-center justify-center gap-2 text-center">
      <TagIcon class="size-8 text-muted-foreground" />
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
      class="relative pl-12 before:absolute before:left-6 before:-translate-x-1/2 before:top-4 before:bottom-4 before:w-0.5 before:bg-border/80 flex flex-col gap-6"
    >
      {#each filteredReleases as release, index (release.id)}
        {@const isLatest = index === 0 && !searchQuery}
        <div class="relative flex items-start">
          <div class="absolute -left-12 top-3 z-10 flex w-12 items-center justify-center">
            {#if isLatest}
              <div
                class="size-7 rounded-full bg-primary text-primary-foreground flex items-center justify-center ring-4 ring-background shadow-xs"
              >
                <SparklesIcon class="size-3.5" />
              </div>
            {:else}
              <div
                class="size-7 rounded-full bg-muted border border-border text-muted-foreground flex items-center justify-center ring-4 ring-background"
              >
                <GitCommitIcon class="size-3.5" />
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

                <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <CalendarIcon class="size-3.5 shrink-0" />
                  <span>{formatDate(release.createdAt)}</span>
                </div>
              </div>
            </Card.Header>

            <Card.Content class="pt-0">
              <div class="flex items-center gap-2">
                <div
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted/70 font-mono text-xs text-foreground border border-border/60"
                  title="ID коммита: {release.commit}"
                >
                  <GitCommitIcon class="size-3.5 text-muted-foreground" />
                  <span>{release.commit.slice(0, 7)}</span>
                  <button
                    type="button"
                    class="ml-1 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                    onclick={() => copyCommit(release.commit)}
                    title="Скопировать полный хеш коммита"
                    aria-label="Скопировать хеш коммита"
                  >
                    {#if copiedCommit === release.commit}
                      <CheckIcon class="size-3.5 text-emerald-500" />
                    {:else}
                      <CopyIcon class="size-3.5" />
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
