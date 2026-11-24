<script lang="ts">
  import { ChevronRight } from '@lucide/svelte';

  let { parkId, id }: { parkId: string; id: string } = $props();
</script>

<div class="flex justify-end">
  <a
    href="/control-panel/{parkId}/devices/{id}"
    class="inline-flex items-center justify-center size-8 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
    title="Перейти к устройству"
    aria-label="Перейти к устройству"
  >
    <ChevronRight class="size-4" />
  </a>
</div>
