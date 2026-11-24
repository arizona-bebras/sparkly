<script lang="ts">
  import { createQuery } from '@tanstack/svelte-query';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import { client } from '$proto';

  let { groupId }: { groupId: string } = $props();

  const groupQuery = createQuery(() => ({
    queryKey: ['group', groupId],
    queryFn: async () => {
      const response = await client.park.getGroup({ id: groupId });
      return response.group;
    },
    enabled: Boolean(groupId),
    staleTime: 1000 * 60 * 5,
  }));
</script>

{#if groupQuery.data}
  <Badge variant="secondary" class="rounded-sm px-1.5 font-normal text-[11px]">
    {groupQuery.data.name}
  </Badge>
{:else if groupQuery.isPending}
  <Skeleton class="h-5 w-12 rounded-sm" />
{:else}
  <span class="font-mono text-[11px] text-muted-foreground">{groupId.slice(0, 8)}</span>
{/if}
