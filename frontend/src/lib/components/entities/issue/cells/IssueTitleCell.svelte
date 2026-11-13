<script lang="ts">
  let {
    title,
    id,
    parkId,
    description,
  }: {
    title: string;
    id: string;
    parkId: string;
    description?: string;
  } = $props();
</script>

<div class="flex flex-col gap-0.5">
  <a
    href="/control-panel/{parkId}/issues/{id}"
    class="font-medium text-foreground hover:underline hover:text-primary transition-colors"
  >
    {title}
  </a>
  {#if description}
    <span class="line-clamp-1 text-xs text-muted-foreground">{description}</span>
  {/if}
  <span class="font-mono text-[11px] text-muted-foreground">{id}</span>
</div>
