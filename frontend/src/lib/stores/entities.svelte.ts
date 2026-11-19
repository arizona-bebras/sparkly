import { SvelteMap } from 'svelte/reactivity';
import type { Build, Device, DeviceGroup, Issue, Park, Release } from '$proto';

export class EntityTable<T extends { id: string }> extends SvelteMap<string, T> {
  setMany(items: T[]) {
    for (const item of items) {
      if (item?.id) {
        this.set(item.id, item);
      }
    }
  }
}

class EntitiesDatabase {
  builds = new EntityTable<Build>();
  devices = new EntityTable<Device>();
  releases = new EntityTable<Release>();
  groups = new EntityTable<DeviceGroup>();
  issues = new EntityTable<Issue>();
  parks = new EntityTable<Park>();
}

export const entities = new EntitiesDatabase();
