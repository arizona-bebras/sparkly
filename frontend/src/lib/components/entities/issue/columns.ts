import { createColumnHelper, renderComponent } from '@tanstack/svelte-table';
import { type Issue } from '$proto';
import type { DataTableFeatures } from '$lib/components/ui/data-table/data-table-features.js';
import IssueActionsCell from './cells/IssueActionsCell.svelte';
import IssueDateCell from './cells/IssueDateCell.svelte';
import IssueSeverityCell from './cells/IssueSeverityCell.svelte';
import IssueStatusCell from './cells/IssueStatusCell.svelte';
import IssueTitleCell from './cells/IssueTitleCell.svelte';

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

export const columns = columnHelper.columns([
  columnHelper.accessor('title', {
    header: 'Проблема',
    cell: ({ row }) =>
      renderComponent(IssueTitleCell, {
        title: row.original.title,
        id: row.original.id,
        parkId: row.original.parkId,
        description: row.original.description,
      }),
  }),
  columnHelper.accessor('severity', {
    header: 'Важность',
    cell: ({ row }) =>
      renderComponent(IssueSeverityCell, {
        severity: row.original.severity,
      }),
  }),
  columnHelper.display({
    id: 'status',
    header: 'Статус',
    cell: ({ row }) => {
      const isResolved =
        !!row.original.isResolved ||
        (!!row.original.resolvedAt && Number(row.original.resolvedAt.seconds) > 0);
      return renderComponent(IssueStatusCell, { isResolved });
    },
  }),
  columnHelper.accessor('firstAt', {
    header: 'Первое появление',
    cell: ({ row }) =>
      renderComponent(IssueDateCell, {
        time: formatTimestamp(row.original.firstAt),
      }),
  }),
  columnHelper.accessor('lastAt', {
    header: 'Последняя активность',
    cell: ({ row }) =>
      renderComponent(IssueDateCell, {
        time: formatTimestamp(row.original.lastAt),
      }),
  }),
  columnHelper.display({
    id: 'actions',
    header: '',
    cell: ({ row }) =>
      renderComponent(IssueActionsCell, {
        parkId: row.original.parkId,
        id: row.original.id,
      }),
  }),
]);
