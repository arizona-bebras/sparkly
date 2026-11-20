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

const deleteEntityMap: Record<string, EntityTable<any>> = {
  DeleteBuild: entities.builds,
  DeleteGroup: entities.groups,
  DeleteIssue: entities.issues,
  DeletePark: entities.parks,
  DeleteRelease: entities.releases,
};

const entityTableMap: Record<string, { table: EntityTable<any>; isList?: boolean }> = {
  builds: { table: entities.builds, isList: true },
  build: { table: entities.builds },
  devices: { table: entities.devices, isList: true },
  device: { table: entities.devices },
  issues: { table: entities.issues, isList: true },
  issue: { table: entities.issues },
  groups: { table: entities.groups, isList: true },
  group: { table: entities.groups },
  parks: { table: entities.parks, isList: true },
  park: { table: entities.parks },
  releases: { table: entities.releases, isList: true },
  release: { table: entities.releases },
};

const entitiesInterceptor: Interceptor = (next) => async (req) => {
  const res = await next(req);

  if (!res.stream && res.message) {
    const msg = res.message as any;
    const deleteTable = deleteEntityMap[req.method.name];
    const reqId = (req.message as { id?: string })?.id;
    if (deleteTable && reqId && msg?.success) {
      deleteTable.delete(reqId);
      return res;
    }

    for (const key of Object.keys(msg)) {
      const config = entityTableMap[key];
      if (!config) continue;

      const data = msg[key];
      if (config.isList && Array.isArray(data)) {
        config.table.setMany(data);
      } else if (!config.isList && data?.id) {
        config.table.set(data.id, data);
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
