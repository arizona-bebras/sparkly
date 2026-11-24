<script lang="ts">
  import DeviceGroupBadge from './DeviceGroupBadge.svelte';

  let { groupIds = [] }: { groupIds?: string[] } = $props();
</script>

{#if groupIds.length === 0}
  <span class="text-xs text-muted-foreground">—</span>
{:else}
  <div class="flex flex-wrap gap-1">
    {#each groupIds as groupId (groupId)}
      <DeviceGroupBadge {groupId} />
    {/each}
  </div>
{/if}
