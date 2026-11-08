<script lang="ts">
  let { name, id }: { name: string; id: string } = $props();
</script>

<div class="flex flex-col">
  <span class="font-medium text-foreground">{name}</span>
  <span class="font-mono text-xs text-muted-foreground">{id}</span>
</div>
