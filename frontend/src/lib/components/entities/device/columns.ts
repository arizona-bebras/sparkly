import { createColumnHelper, renderSnippet } from '@tanstack/svelte-table';
import { createRawSnippet } from 'svelte';
import type { Device } from '$proto';
import type { DataTableFeatures } from '$lib/components/ui/data-table/data-table-features.js';

const columnHelper = createColumnHelper<DataTableFeatures, Device>();

function formatTimestamp(ts?: { seconds: bigint; nanos: number }): string {
  if (!ts) return '—';
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

export const columns = columnHelper.columns([
  columnHelper.accessor('name', {
    header: 'Устройство',
    filterFn: (row, _columnId, filterValue) => {
      const q = String(filterValue ?? '')
        .toLowerCase()
        .trim();
      if (!q) return true;
      return (
        row.original.name.toLowerCase().includes(q) || row.original.id.toLowerCase().includes(q)
      );
    },
    cell: ({ row }) => {
      const snippet = createRawSnippet<[{ name: string; id: string }]>((getData) => {
        const { name, id } = getData();
        return {
          render: () =>
            `<div class="flex flex-col"><span class="font-medium text-foreground">${name}</span><span class="font-mono text-xs text-muted-foreground">${id}</span></div>`,
        };
      });
      return renderSnippet(snippet, {
        name: row.original.name,
        id: row.original.id,
      });
    },
  }),
  columnHelper.accessor('ipAddress', {
    header: 'IP-адрес',
    cell: ({ row }) => {
      const snippet = createRawSnippet<[{ ip: string }]>((getData) => {
        const { ip } = getData();
        return {
          render: () => `<span class="font-mono text-xs">${ip || '—'}</span>`,
        };
      });
      return renderSnippet(snippet, { ip: row.original.ipAddress });
    },
  }),
  columnHelper.accessor('lastSeenAt', {
    header: 'Активность',
    cell: ({ row }) => {
      const formatted = formatTimestamp(row.original.lastSeenAt);
      const snippet = createRawSnippet<[{ time: string }]>((getData) => {
        const { time } = getData();
        return {
          render: () => `<span class="text-xs text-muted-foreground">${time}</span>`,
        };
      });
      return renderSnippet(snippet, { time: formatted });
    },
  }),
  columnHelper.accessor('joinedAt', {
    header: 'Подключено',
    cell: ({ row }) => {
      const formatted = formatTimestamp(row.original.joinedAt);
      const snippet = createRawSnippet<[{ time: string }]>((getData) => {
        const { time } = getData();
        return {
          render: () => `<span class="text-xs text-muted-foreground">${time}</span>`,
        };
      });
      return renderSnippet(snippet, { time: formatted });
    },
  }),
]);
