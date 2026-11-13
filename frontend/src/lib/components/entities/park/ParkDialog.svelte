<script lang="ts">
  import { useQueryClient } from '@tanstack/svelte-query';
  import type { Snippet } from 'svelte';
  import { goto } from '$app/navigation';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import type { Park } from '../../../../../proto/gen/backend/proto/service_pb';
  import ParkForm from '../../forms/park/Park.svelte';

  let client = useQueryClient();

  let {
    open = $bindable(false),
    children,
    park,
  }: {
    open: boolean;
    children?: Snippet;
    park?: Park;
  } = $props();
</script>

<Dialog.Root bind:open>
  <Dialog.Trigger>
    {#if children}
      {@render children()}
    {:else}
      <p>Создать парк</p>
    {/if}
  </Dialog.Trigger>

  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>{park ? 'Редактировать парк' : 'Создать парк'}</Dialog.Title>
      <Dialog.Description>
        {park
          ? 'Отредактируйте информацию о парке.'
          : 'Заполните информацию для создания нового парка.'}
      </Dialog.Description>
    </Dialog.Header>

    <ParkForm
      onSuccess={async (savedPark) => {
        open = false;
        if (savedPark?.id) {
          client.setQueryData(['park', savedPark.id], {
            park: savedPark,
          });
          await client.invalidateQueries({
            queryKey: ['park', savedPark.id],
          });
        }
        if (!park && savedPark?.id) {
          await goto(`/control-panel/${savedPark.id}/settings?onboarding=open`);
        }
        await client.invalidateQueries({ queryKey: ['parks'] });
      }}
      {park}
    />
  </Dialog.Content>
</Dialog.Root>
