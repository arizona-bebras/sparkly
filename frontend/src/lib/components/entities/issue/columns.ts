import { createColumnHelper, renderSnippet } from '@tanstack/svelte-table';
import { createRawSnippet } from 'svelte';
import { type Issue, IssueSeverity } from '$proto';
import type { DataTableFeatures } from '$lib/components/ui/data-table/data-table-features.js';

const columnHelper = createColumnHelper<DataTableFeatures, Issue>();

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
  }).format(date);
}

function getSeverityBadge(severity: IssueSeverity): string {
  switch (severity) {
    case IssueSeverity.CRITICAL:
      return '<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-destructive/15 text-destructive border border-destructive/20"><span class="size-1.5 rounded-full bg-destructive"></span>Критическая</span>';
    case IssueSeverity.ERROR:
      return '<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/20"><span class="size-1.5 rounded-full bg-rose-500"></span>Ошибка</span>';
    case IssueSeverity.WARNING:
      return '<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/20"><span class="size-1.5 rounded-full bg-amber-500"></span>Предупреждение</span>';
    case IssueSeverity.INFO:
      return '<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-500/15 text-blue-600 dark:text-blue-400 border border-blue-500/20"><span class="size-1.5 rounded-full bg-blue-500"></span>Инфо</span>';
    default:
      return '<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-secondary text-secondary-foreground"><span class="size-1.5 rounded-full bg-zinc-400"></span>Не указана</span>';
  }
}

export const columns = columnHelper.columns([
  columnHelper.accessor('title', {
    header: 'Проблема',
    cell: ({ row }) => {
      const snippet = createRawSnippet<
        [{ title: string; id: string; parkId: string; description: string }]
      >((getData) => {
        const { title, id, parkId, description } = getData();
        const descHtml = description
          ? `<span class="line-clamp-1 text-xs text-muted-foreground">${description}</span>`
          : '';
        return {
          render: () =>
            `<div class="flex flex-col gap-0.5"><a href="/control-panel/${parkId}/issues/${id}" class="font-medium text-foreground hover:underline hover:text-primary transition-colors">${title}</a>${descHtml}<span class="font-mono text-[11px] text-muted-foreground">${id}</span></div>`,
        };
      });
      return renderSnippet(snippet, {
        title: row.original.title,
        id: row.original.id,
        parkId: row.original.parkId,
        description: row.original.description,
      });
    },
  }),
  columnHelper.accessor('severity', {
    header: 'Важность',
    cell: ({ row }) => {
      const severity = row.original.severity;
      const snippet = createRawSnippet<[{ severity: IssueSeverity }]>((getData) => {
        const { severity } = getData();
        return {
          render: () => getSeverityBadge(severity),
        };
      });
      return renderSnippet(snippet, { severity });
    },
  }),
  columnHelper.display({
    id: 'status',
    header: 'Статус',
    cell: ({ row }) => {
      const isResolved = !!row.original.resolvedAt && Number(row.original.resolvedAt.seconds) > 0;
      const snippet = createRawSnippet<[{ isResolved: boolean }]>((getData) => {
        const { isResolved } = getData();
        return {
          render: () =>
            isResolved
              ? `<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"><span class="size-1.5 rounded-full bg-emerald-500"></span>Решено</span>`
              : `<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-destructive/15 text-destructive border border-destructive/20"><span class="size-1.5 rounded-full bg-destructive"></span>Открыта</span>`,
        };
      });
      return renderSnippet(snippet, { isResolved });
    },
  }),
  columnHelper.accessor('firstAt', {
    header: 'Первое появление',
    cell: ({ row }) => {
      const formatted = formatTimestamp(row.original.firstAt);
      const snippet = createRawSnippet<[{ time: string }]>((getData) => {
        const { time } = getData();
        return {
          render: () => `<span class="text-xs text-muted-foreground">${time}</span>`,
        };
      });
      return renderSnippet(snippet, { time: formatted });
    },
  }),
  columnHelper.accessor('lastAt', {
    header: 'Последняя активность',
    cell: ({ row }) => {
      const formatted = formatTimestamp(row.original.lastAt);
      const snippet = createRawSnippet<[{ time: string }]>((getData) => {
        const { time } = getData();
        return {
          render: () => `<span class="text-xs text-muted-foreground">${time}</span>`,
        };
      });
      return renderSnippet(snippet, { time: formatted });
    },
  }),
  columnHelper.display({
    id: 'actions',
    header: '',
    cell: ({ row }) => {
      const snippet = createRawSnippet<[{ parkId: string; id: string }]>((getData) => {
        const { parkId, id } = getData();
        return {
          render: () =>
            `<div class="flex justify-end"><a href="/control-panel/${parkId}/issues/${id}" class="inline-flex items-center justify-center size-8 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors" title="Подробнее"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="size-4"><path d="m9 18 6-6-6-6"/></svg></a></div>`,
        };
      });
      return renderSnippet(snippet, {
        parkId: row.original.parkId,
        id: row.original.id,
      });
    },
  }),
]);
