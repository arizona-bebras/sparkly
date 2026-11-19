import { createClient, type Interceptor } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { entities, type EntityTable } from '../src/lib/stores/entities.svelte.js';
import {
  BuildSchema,
  BuildService,
  DeviceSchema,
  DeviceService,
  IssueSchema,
  IssueService,
  ParkService,
  ReleaseSchema,
  ReleaseService,
  UserService,
  type Build,
  type Device,
  type DeviceGroup,
  type Issue,
  type Park,
  type Release,
  IssueSeverity,
} from './gen/backend/proto/service_pb';

const singleEntityMap: Record<string, { table: EntityTable<any>; key: string }> = {
  GetBuild: { table: entities.builds, key: 'build' },
  GetDevice: { table: entities.devices, key: 'device' },
  GetIssue: { table: entities.issues, key: 'issue' },
  GetPark: { table: entities.parks, key: 'park' },
  GetGroup: { table: entities.groups, key: 'group' },
  GetRelease: { table: entities.releases, key: 'release' },
};

const entityCollections: Array<{ key: string; table: EntityTable<any>; isList?: boolean }> = [
  { key: 'builds', table: entities.builds, isList: true },
  { key: 'build', table: entities.builds },
  { key: 'devices', table: entities.devices, isList: true },
  { key: 'device', table: entities.devices },
  { key: 'issues', table: entities.issues, isList: true },
  { key: 'issue', table: entities.issues },
  { key: 'groups', table: entities.groups, isList: true },
  { key: 'group', table: entities.groups },
  { key: 'parks', table: entities.parks, isList: true },
  { key: 'park', table: entities.parks },
  { key: 'releases', table: entities.releases, isList: true },
  { key: 'release', table: entities.releases },
];

const entitiesInterceptor: Interceptor = (next) => async (req) => {
  if (!req.stream) {
    const id = (req.message as { id?: string })?.id;
    const entityConfig = singleEntityMap[req.method.name];
    if (id && entityConfig && entityConfig.table.has(id)) {
      return {
        stream: false,
        service: req.service,
        method: req.method,
        header: new Headers(),
        trailer: new Headers(),
        message: { [entityConfig.key]: entityConfig.table.get(id)! } as any,
      };
    }
  }

  const res = await next(req);

  if (!res.stream && res.message) {
    const msg = res.message as any;

    for (const { key, table, isList } of entityCollections) {
      const data = msg[key];
      if (!data) continue;

      if (isList && Array.isArray(data)) {
        table.setMany(data);
      } else if (!isList && data.id) {
        table.set(data.id, data);
      }
    }
  }

  return res;
};

const transport = createConnectTransport({
  baseUrl: '/',
  interceptors: [entitiesInterceptor],
});

export const client = {
  user: createClient(UserService, transport),
  park: createClient(ParkService, transport),
  device: createClient(DeviceService, transport),
  issue: createClient(IssueService, transport),
  build: createClient(BuildService, transport),
  release: createClient(ReleaseService, transport),
};

export type { Build, Device, DeviceGroup, Issue, Park, Release };
export {
  IssueSeverity,
  BuildSchema,
  BuildService,
  DeviceSchema,
  IssueSchema,
  ReleaseSchema,
  ReleaseService,
};
