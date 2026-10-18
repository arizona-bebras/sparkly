import { Code, ConnectError, createClient, createRouterTransport } from '@connectrpc/connect';
// import { createConnectTransport } from "@connectrpc/connect-web";
// import { PUBLIC_API_BASE_URL } from "$env/static/public";
import { ParkService, UserService } from './gen/backend/proto/service_pb';

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

const mockTransport = createRouterTransport(({ service }) => {
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

    // READ (filtered)
    async getParksByFilters(req) {
      console.log('⚡ [MOCK] getParksByFilters:', req.name);
      const filtered = mockParks.filter((p) =>
        p.name.toLowerCase().includes((req.name || '').toLowerCase()),
      );

      return {
        parks: filtered,
        pagination: {
          totalItems: filtered.length,
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
});

// const transport = createConnectTransport({
// 	baseUrl: PUBLIC_API_BASE_URL || "https://demo.connectrpc.com",
// });

export const client = {
  user: createClient(UserService, mockTransport),
  park: createClient(ParkService, mockTransport),
};
