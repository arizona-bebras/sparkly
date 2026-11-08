<script lang="ts">
  import { page } from '$app/state';
  import ReleaseTimeline from '$lib/components/entities/release/ReleaseTimeline.svelte';

  const parkId = $derived(page.params.park_id ?? '');
</script>

<svelte:head>
  <title>Релизы | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-6">
  <div class="flex flex-col gap-1">
    <h1 class="text-2xl font-bold tracking-tight">Релизы</h1>
    <p class="text-sm text-muted-foreground">Хронология версий и прошивок устройств парка</p>
  </div>

  <ReleaseTimeline {parkId} />
</div>
