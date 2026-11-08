<script lang="ts">
  import { page } from '$app/state';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { useQueryClient } from '@tanstack/svelte-query';
  import GroupForm from '$lib/components/forms/group/Group.svelte';
  import GroupList from '$lib/components/entities/group/GroupList.svelte';

  const parkId = $derived(page.params.park_id ?? '');
  const queryClient = useQueryClient();
  let isCreateOpen = $state(false);
</script>

<svelte:head>
  <title>Группы | Sparkly</title>
</svelte:head>

<div class="flex flex-col gap-6">
  <div class="flex flex-col gap-1">
    <h1 class="text-2xl font-bold tracking-tight">Группы</h1>
    <p class="text-sm text-muted-foreground">Управление группами устройств парка</p>
  </div>

  <GroupList {parkId} bind:isCreateOpen />

  <Dialog.Root bind:open={isCreateOpen}>
    <Dialog.Content class="sm:max-w-md">
      <Dialog.Header>
        <Dialog.Title>Создать группу</Dialog.Title>
        <Dialog.Description>
          Заполните информацию для создания новой группы устройств.
        </Dialog.Description>
      </Dialog.Header>

      <GroupForm
        {parkId}
        onSuccess={() => {
          isCreateOpen = false;
          queryClient.invalidateQueries({ queryKey: ['groups', parkId] });
        }}
      />
    </Dialog.Content>
  </Dialog.Root>
</div>
