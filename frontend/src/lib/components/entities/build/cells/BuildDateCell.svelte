<script lang="ts">
  import { Calendar } from '@lucide/svelte';

  let { time }: { time: string } = $props();
</script>

<div class="inline-flex items-center gap-1.5 text-xs text-muted-foreground whitespace-nowrap">
  <Calendar class="size-3.5 shrink-0" />
  <span>{time}</span>
</div>
