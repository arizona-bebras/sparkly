import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { type Release, ReleaseSchema } from '$proto';

export function getMockReleases(parkId: string): Release[] {
  return [
    create(ReleaseSchema, {
      id: 'rel-01a1a801-b111-7110-8b9a-112233445566',
      parkId,
      version: 'v1.4.2',
      commit: '7f3a8b1c94d0e123456789abcdef0123456789ab',
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 48)),
    }),
    create(ReleaseSchema, {
      id: 'rel-01a1a802-c222-7221-9c0b-223344556677',
      parkId,
      version: 'v1.4.1',
      commit: 'e2d4c6a810f39572481632641282565120124816',
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 9)),
    }),
    create(ReleaseSchema, {
      id: 'rel-01a1a803-d333-7332-ad1c-334455667788',
      parkId,
      version: 'v1.4.0',
      commit: '3b9a1c5d7e2f4680135792468013579246801357',
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 24)),
    }),
    create(ReleaseSchema, {
      id: 'rel-01a1a804-e444-7443-be2d-445566778899',
      parkId,
      version: 'v1.3.8',
      commit: 'c8d2e4f6a0b13579246801357924680135792468',
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 45)),
    }),
    create(ReleaseSchema, {
      id: 'rel-01a1a805-f555-7554-cf3e-556677889900',
      parkId,
      version: 'v1.3.0',
      commit: '1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b',
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 70)),
    }),
    create(ReleaseSchema, {
      id: 'rel-01a1a806-a666-7665-de4f-667788990011',
      parkId,
      version: 'v1.0.0',
      commit: '9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f0e',
      createdAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 24 * 120)),
    }),
  ];
}
