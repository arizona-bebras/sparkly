import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
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

const transport = createConnectTransport({
  baseUrl: '/',
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
