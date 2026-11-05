<script lang="ts">
  import { page } from '$app/state';

  const parkId = $derived(page.params.park_id);
</script>

<svelte:head>
  <title>Проблемы | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-2">
  <h1 class="text-2xl font-bold tracking-tight">Проблемы (Issues)</h1>
  <p class="text-sm text-muted-foreground">Реестр ошибок и инцидентов парка</p>
</div>
