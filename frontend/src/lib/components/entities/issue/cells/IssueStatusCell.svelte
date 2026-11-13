<script lang="ts">
  let { isResolved }: { isResolved: boolean } = $props();
</script>

{#if isResolved}
  <span
    class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
  >
    <span class="size-1.5 rounded-full bg-emerald-500"></span>
    Решено
  </span>
{:else}
  <span
    class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-destructive/15 text-destructive border border-destructive/20"
  >
    <span class="size-1.5 rounded-full bg-destructive"></span>
    Открыта
  </span>
{/if}
