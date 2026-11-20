import { createColumnHelper, renderComponent } from '@tanstack/svelte-table';
import type { Build } from '$proto';
import type { DataTableFeatures } from '$lib/components/ui/data-table/data-table-features.js';
import BuildActionsCell from './cells/BuildActionsCell.svelte';
import BuildCommitCell from './cells/BuildCommitCell.svelte';
import BuildDateCell from './cells/BuildDateCell.svelte';
import BuildVersionCell from './cells/BuildVersionCell.svelte';

const columnHelper = createColumnHelper<DataTableFeatures, Build>();

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

export interface BuildColumnCallbacks {
  onViewDetails?: (build: Build) => void;
  onDelete?: (build: Build) => void;
}

export function createColumns(callbacks: BuildColumnCallbacks = {}) {
  return columnHelper.columns([
    columnHelper.accessor('version', {
      header: 'Версия',
      filterFn: (row, _columnId, filterValue) => {
        const q = String(filterValue ?? '')
          .toLowerCase()
          .trim();
        if (!q) return true;
        return (
          row.original.version.toLowerCase().includes(q) ||
          row.original.commit.toLowerCase().includes(q) ||
          row.original.id.toLowerCase().includes(q)
        );
      },
      cell: ({ row }) =>
        renderComponent(BuildVersionCell, {
          version: row.original.version,
          id: row.original.id,
        }),
    }),
    columnHelper.accessor('commit', {
      header: 'Git коммит',
      cell: ({ row }) =>
        renderComponent(BuildCommitCell, {
          commit: row.original.commit,
        }),
    }),
    columnHelper.accessor('createdAt', {
      header: 'Дата создания',
      cell: ({ row }) =>
        renderComponent(BuildDateCell, {
          time: formatTimestamp(row.original.createdAt),
        }),
    }),
    columnHelper.display({
      id: 'actions',
      header: '',
      cell: ({ row }) =>
        renderComponent(BuildActionsCell, {
          build: row.original,
          onViewDetails: () => callbacks.onViewDetails?.(row.original),
          onDelete: () => callbacks.onDelete?.(row.original),
        }),
    }),
  ]);
}

export const columns = createColumns();
