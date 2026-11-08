<script lang="ts">
  let { ip }: { ip?: string } = $props();
</script>

<span class="font-mono text-xs">{ip || '—'}</span>
