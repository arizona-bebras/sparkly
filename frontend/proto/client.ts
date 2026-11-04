import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { Code, ConnectError, createClient, createRouterTransport } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
// import { PUBLIC_API_BASE_URL } from "$env/static/public";
import {
  DeviceSchema,
  DeviceService,
  IssueSchema,
  IssueService,
  ParkService,
  ReleaseSchema,
  UserService,
  type Device,
  type Issue,
  type Release,
  IssueSeverity,
} from './gen/backend/proto/service_pb';

// Тип для элементов мока парка
interface MockPark {
  id: string;
  name: string;
  description: string;
  deviceCount: number;
  createdAt?: undefined;
  updatedAt?: undefined;
}

// In-memory хранилище для CRUD операций над парками
const mockParks: MockPark[] = [
  {
    id: 'park-1',
    name: 'Центральный парк',
    description: 'Главная площадка мониторинга',
    deviceCount: 5,
  },
  {
    id: 'park-2',
    name: 'Северный терминал',
    description: 'Терминал в северной промзоне',
    deviceCount: 2,
  },
];

// Тип для элементов мока устройства
interface MockDevice {
  id: string;
  name: string;
  parkId: string;
  groupIds: string[];
  ipAddress: string;
  joinedAt?: ReturnType<typeof timestampFromDate>;
  lastSeenAt?: ReturnType<typeof timestampFromDate>;
  createdAt?: ReturnType<typeof timestampFromDate>;
  updatedAt?: ReturnType<typeof timestampFromDate>;
}

// In-memory хранилище для устройств
const mockDevices: MockDevice[] = [
  {
    id: 'dev-1',
    name: 'Касса #1 (Главный вход)',
    parkId: 'park-1',
    groupIds: ['grp-1'],
    ipAddress: '192.168.1.101',
    joinedAt: timestampFromDate(new Date('2026-09-01T10:00:00Z')),
    lastSeenAt: timestampFromDate(new Date()),
    createdAt: timestampFromDate(new Date('2026-09-01T10:00:00Z')),
    updatedAt: timestampFromDate(new Date()),
  },
  {
    id: 'dev-2',
    name: 'Турникет Восток',
    parkId: 'park-1',
    groupIds: ['grp-2'],
    ipAddress: '192.168.1.102',
    joinedAt: timestampFromDate(new Date('2026-09-05T12:30:00Z')),
    lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 2)),
    createdAt: timestampFromDate(new Date('2026-09-05T12:30:00Z')),
    updatedAt: timestampFromDate(new Date()),
  },
  {
    id: 'dev-3',
    name: 'Турникет Запад',
    parkId: 'park-1',
    groupIds: ['grp-2'],
    ipAddress: '192.168.1.103',
    joinedAt: timestampFromDate(new Date('2026-09-05T12:35:00Z')),
    lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 15)),
    createdAt: timestampFromDate(new Date('2026-09-05T12:35:00Z')),
    updatedAt: timestampFromDate(new Date()),
  },
  {
    id: 'dev-4',
    name: 'Инфокиоск Центральный',
    parkId: 'park-1',
    groupIds: ['grp-3'],
    ipAddress: '192.168.1.104',
    joinedAt: timestampFromDate(new Date('2026-09-10T08:00:00Z')),
    lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 3)),
    createdAt: timestampFromDate(new Date('2026-09-10T08:00:00Z')),
    updatedAt: timestampFromDate(new Date()),
  },
  {
    id: 'dev-5',
    name: 'Шлагбаум Въезд',
    parkId: 'park-1',
    groupIds: ['grp-4'],
    ipAddress: '192.168.1.105',
    joinedAt: timestampFromDate(new Date('2026-09-12T09:20:00Z')),
    lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24)),
    createdAt: timestampFromDate(new Date('2026-09-12T09:20:00Z')),
    updatedAt: timestampFromDate(new Date()),
  },
  {
    id: 'dev-6',
    name: 'Терминал погрузки #1',
    parkId: 'park-2',
    groupIds: ['grp-5'],
    ipAddress: '10.0.1.50',
    joinedAt: timestampFromDate(new Date('2026-09-15T11:00:00Z')),
    lastSeenAt: timestampFromDate(new Date()),
    createdAt: timestampFromDate(new Date('2026-09-15T11:00:00Z')),
    updatedAt: timestampFromDate(new Date()),
  },
  {
    id: 'dev-7',
    name: 'Контроллер ворот склад',
    parkId: 'park-2',
    groupIds: ['grp-5'],
    ipAddress: '10.0.1.51',
    joinedAt: timestampFromDate(new Date('2026-09-18T14:10:00Z')),
    lastSeenAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 5)),
    createdAt: timestampFromDate(new Date('2026-09-18T14:10:00Z')),
    updatedAt: timestampFromDate(new Date()),
  },
];

const _mockTransport = createRouterTransport(({ service }) => {
  // ==================== USER SERVICE MOCK ====================
  service(UserService, {
    async checkAuth() {
      console.log('⚡ [MOCK] checkAuth вызван');
      return { auth: true };
    },
    async getToken(req) {
      console.log('⚡ [MOCK] getToken для:', req.email);
      await new Promise((r) => setTimeout(r, 1200));

      if (req.email !== 'admin@mail.ru' || req.password !== 'admin') {
        throw new ConnectError('Неверный логин или пароль!', Code.PermissionDenied);
      }

      return {
        email: req.email,
        token: 'mock-jwt-token-xyz',
        refreshToken: 'mock-refresh-token-xyz',
        expiresIn: 3600n,
      };
    },
    async reg(req) {
      console.log('⚡ [MOCK] reg для:', req.email);
      await new Promise((r) => setTimeout(r, 1200));

      if (req.email === 'admin@mail.ru') {
        throw new ConnectError('Пользователь с таким email уже существует', Code.AlreadyExists);
      }

      return {
        email: req.email,
        token: 'mock-jwt-token-xyz',
        refreshToken: 'mock-refresh-token-xyz',
        expiresIn: 3600n,
      };
    },
    async updateUser(req) {
      return {
        user: {
          id: req.id,
          email: req.email || 'user@example.com',
          createdAt: undefined,
          updatedAt: undefined,
        },
      };
    },
    async deleteUser() {
      return { success: true };
    },
  });

  // ==================== PARK SERVICE MOCK (CRUD) ====================
  service(ParkService, {
    // CREATE
    async createPark(req) {
      console.log('⚡ [MOCK] createPark:', req.name);
      await new Promise((r) => setTimeout(r, 600));

      if (!req.name || req.name.trim().length === 0) {
        throw new ConnectError('Название парка обязательно', Code.InvalidArgument);
      }

      const newPark: MockPark = {
        id: `park-${Date.now()}-${Math.random().toString(36).substring(2, 6)}`,
        name: req.name,
        description: req.description || '',
        deviceCount: 0,
      };

      mockParks.unshift(newPark);
      console.log(mockParks);
      return { park: newPark };
    },

    // READ (one)
    async getPark(req) {
      console.log('⚡ [MOCK] getPark:', req.id);
      await new Promise((r) => setTimeout(r, 300));

      const park = mockParks.find((p) => p.id === req.id);
      if (!park) {
        throw new ConnectError('Парк не найден', Code.NotFound);
      }
      return { park };
    },

    // READ (all)
    async getAllParks() {
      console.log('⚡ [MOCK] getAllParks, всего парков:', mockParks.length);
      await new Promise((r) => setTimeout(r, 400));

      return {
        parks: mockParks,
        pagination: {
          totalItems: mockParks.length,
          totalPages: 1,
          currentPage: 1,
        },
      };
    },

    // UPDATE
    async updatePark(req) {
      console.log('⚡ [MOCK] updatePark:', req.id);
      await new Promise((r) => setTimeout(r, 500));

      const park = mockParks.find((p) => p.id === req.id);
      if (!park) {
        throw new ConnectError('Парк не найден', Code.NotFound);
      }

      if (req.name !== undefined) park.name = req.name;
      if (req.description !== undefined) park.description = req.description;

      return { park };
    },

    // DELETE
    async deletePark(req) {
      console.log('⚡ [MOCK] deletePark:', req.id);
      await new Promise((r) => setTimeout(r, 400));

      const index = mockParks.findIndex((p) => p.id === req.id);
      if (index === -1) {
        throw new ConnectError('Парк не найден', Code.NotFound);
      }

      mockParks.splice(index, 1);
      return { success: true };
    },

    // Группы устройств (заглушки)
    async createGroup(req) {
      return {
        group: {
          id: `grp-${Date.now()}`,
          parkId: req.parkId,
          name: req.name,
          description: req.description,
          deviceCount: 0,
        },
      };
    },
    async getGroup() {
      return { group: undefined };
    },
    async getGroups() {
      return { groups: [], pagination: undefined };
    },
    async updateGroup() {
      return { group: undefined };
    },
    async deleteGroup() {
      return { success: true };
    },
    async addDeviceToGroup() {
      return { success: true };
    },
    async removeDeviceFromGroup() {
      return { success: true };
    },
  });

  // ==================== DEVICE SERVICE MOCK ====================
  service(DeviceService, {
    // Получить устройство по id
    async getDevice(req) {
      console.log('⚡ [MOCK] getDevice:', req.id);
      await new Promise((r) => setTimeout(r, 200));

      const device = mockDevices.find((d) => d.id === req.id);
      if (!device) {
        throw new ConnectError('Устройство не найдено', Code.NotFound);
      }
      return { device };
    },

    // Получить все устройства парка с пагинацией
    async getAllDevices(req) {
      console.log('⚡ [MOCK] getAllDevices для parkId:', req.parkId);
      await new Promise((r) => setTimeout(r, 300));

      const parkDevices = mockDevices.filter((d) => !req.parkId || d.parkId === req.parkId);
      const page = req.pagination?.page || 1;
      const pageSize = req.pagination?.pageSize || 10;
      const startIndex = (page - 1) * pageSize;
      const paged = parkDevices.slice(startIndex, startIndex + pageSize);

      return {
        devices: paged,
        pagination: {
          totalItems: parkDevices.length,
          totalPages: Math.ceil(parkDevices.length / pageSize) || 1,
          currentPage: page,
        },
      };
    },

    // Получить устройства по фильтрам (по имени, ID устройств, группам)
    async getDevicesByFilters(req) {
      console.log('⚡ [MOCK] getDevicesByFilters:', req);
      await new Promise((r) => setTimeout(r, 300));

      let filtered = mockDevices.filter((d) => !req.parkId || d.parkId === req.parkId);

      if (req.deviceIds && req.deviceIds.length > 0) {
        filtered = filtered.filter((d) => req.deviceIds.includes(d.id));
      }

      if (req.groupIds && req.groupIds.length > 0) {
        filtered = filtered.filter((d) => d.groupIds.some((gId) => req.groupIds.includes(gId)));
      }

      if (req.name && req.name.trim().length > 0) {
        const search = req.name.toLowerCase();
        filtered = filtered.filter((d) => d.name.toLowerCase().includes(search));
      }

      const page = req.pagination?.page || 1;
      const pageSize = req.pagination?.pageSize || 10;
      const startIndex = (page - 1) * pageSize;
      const paged = filtered.slice(startIndex, startIndex + pageSize);

      return {
        devices: paged,
        pagination: {
          totalItems: filtered.length,
          totalPages: Math.ceil(filtered.length / pageSize) || 1,
          currentPage: page,
        },
      };
    },
  });
});

const transport = createConnectTransport({
  baseUrl: '/',
});

export const client = {
  user: createClient(UserService, transport),
  park: createClient(ParkService, transport),
  device: createClient(DeviceService, transport),
  issue: createClient(IssueService, transport),
};

export type { Device, Issue, Release };
export { IssueSeverity, DeviceSchema, IssueSchema, ReleaseSchema };
