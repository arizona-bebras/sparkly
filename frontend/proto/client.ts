import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
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
