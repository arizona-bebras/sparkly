<script lang="ts">
  import { page } from '$app/state';

  const parkId = $derived(page.params.park_id);
</script>

<svelte:head>
  <title>Группы и релизы | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-2">
  <h1 class="text-2xl font-bold tracking-tight">Группы и релизы</h1>
  <p class="text-sm text-muted-foreground">
    Управление группами и версиями парка #{parkId}
  </p>
</div>
