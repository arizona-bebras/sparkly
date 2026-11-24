<script lang="ts">
  import { page } from '$app/state';
  import BuildList from '$lib/components/entities/build/BuildList.svelte';

  const parkId = $derived(page.params.park_id ?? '');
</script>

<svelte:head>
  <title>Сборки | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-6">
  <div class="flex flex-col gap-1">
    <h1 class="text-2xl font-bold tracking-tight">Сборки</h1>
    <p class="text-sm text-muted-foreground">Управление сборками и артефактами прошивок</p>
  </div>

  <BuildList {parkId} />
</div>
