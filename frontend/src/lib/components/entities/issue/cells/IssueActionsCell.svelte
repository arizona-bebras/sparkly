<script lang="ts">
  import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';

  let { parkId, id }: { parkId: string; id: string } = $props();
</script>

<div class="flex justify-end">
  <a
    href="/control-panel/{parkId}/issues/{id}"
    class="inline-flex items-center justify-center size-8 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
    title="Подробнее"
    aria-label="Подробнее о проблеме"
  >
    <ChevronRightIcon class="size-4" />
  </a>
</div>
