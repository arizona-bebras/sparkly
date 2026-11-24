<script lang="ts">
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Package } from '@lucide/svelte';

  let { version, id }: { version: string; id: string } = $props();
</script>

<div class="flex flex-col gap-1 items-start">
  <div class="flex items-center gap-1.5">
    <Package class="size-3.5 text-muted-foreground" />
    <Badge variant="secondary" class="font-mono font-medium text-xs">
      {version}
    </Badge>
  </div>
  <span class="font-mono text-[11px] text-muted-foreground truncate max-w-[220px]" title={id}>
    {id}
  </span>
</div>
