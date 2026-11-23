<script lang="ts">
  let {
    name,
    id,
    parkId,
  }: {
    name: string;
    id: string;
    parkId: string;
  } = $props();
</script>

<div class="flex flex-col gap-0.5">
  <a
    href="/control-panel/{parkId}/devices/{id}"
    class="font-medium text-foreground hover:underline hover:text-primary transition-colors"
  >
    {name}
  </a>
  <span class="font-mono text-xs text-muted-foreground">{id}</span>
</div>
