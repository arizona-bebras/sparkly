import { createColumnHelper, renderComponent } from '@tanstack/svelte-table';
import type { Device } from '$proto';
import type { DataTableFeatures } from '$lib/components/ui/data-table/data-table-features.js';
import DeviceDateCell from './cells/DeviceDateCell.svelte';
import DeviceIpCell from './cells/DeviceIpCell.svelte';
import DeviceNameCell from './cells/DeviceNameCell.svelte';

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
    cell: ({ row }) =>
      renderComponent(DeviceNameCell, {
        name: row.original.name,
        id: row.original.id,
      }),
  }),
  columnHelper.accessor('ipAddress', {
    header: 'IP-адрес',
    cell: ({ row }) =>
      renderComponent(DeviceIpCell, {
        ip: row.original.ipAddress,
      }),
  }),
  columnHelper.accessor('lastSeenAt', {
    header: 'Активность',
    cell: ({ row }) =>
      renderComponent(DeviceDateCell, {
        time: formatTimestamp(row.original.lastSeenAt),
      }),
  }),
  columnHelper.accessor('joinedAt', {
    header: 'Подключено',
    cell: ({ row }) =>
      renderComponent(DeviceDateCell, {
        time: formatTimestamp(row.original.joinedAt),
      }),
  }),
]);
