import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { type Device, DeviceSchema } from '$proto';

export function getMockDevices(parkId: string): Device[] {
  return [
    create(DeviceSchema, {
      id: 'dev-01a1a601-e241-7110-8b9a-112233445566',
      name: 'Касса #1 (Главный вход)',
      parkId,
      groupIds: ['grp-1'],
      ipAddress: '192.168.1.101',
      joinedAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 60)),
      lastSeenAt: timestampFromDate(new Date()),
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 60)),
      updatedAt: timestampFromDate(new Date()),
    }),
    create(DeviceSchema, {
      id: 'dev-01a1a602-f352-7221-9c0b-223344556677',
      name: 'Турникет Восток',
      parkId,
      groupIds: ['grp-2'],
      ipAddress: '192.168.1.102',
      joinedAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 45)),
      lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 2)),
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 45)),
      updatedAt: timestampFromDate(new Date()),
    }),
    create(DeviceSchema, {
      id: 'dev-01a1a603-a463-7332-ad1c-334455667788',
      name: 'Турникет Запад',
      parkId,
      groupIds: ['grp-2'],
      ipAddress: '192.168.1.103',
      joinedAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 30)),
      lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 18)),
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 30)),
      updatedAt: timestampFromDate(new Date()),
    }),
    create(DeviceSchema, {
      id: 'dev-01a1a604-b574-7443-be2d-445566778899',
      name: 'Инфокиоск Центральный',
      parkId,
      groupIds: ['grp-3'],
      ipAddress: '192.168.1.104',
      joinedAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 14)),
      lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 4)),
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 14)),
      updatedAt: timestampFromDate(new Date()),
    }),
    create(DeviceSchema, {
      id: 'dev-01a1a605-c685-7554-cf3e-556677889900',
      name: 'Шлагбаум Въезд',
      parkId,
      groupIds: ['grp-4'],
      ipAddress: '192.168.1.105',
      joinedAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 7)),
      lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 26)),
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 7)),
      updatedAt: timestampFromDate(new Date()),
    }),
  ];
}
