<script lang="ts">
  import { createMutation } from '@tanstack/svelte-query';
  import { onMount } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';
  import SubmitBtn from '$lib/components/forms/SubmitBtn.svelte';
  import { FieldGroup } from '$lib/components/ui/field/index.js';
  import * as Form from '$lib/components/ui/form/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { spaForm } from '$lib/form-spa-util.svelte';
  import { cn } from '$lib/utils.js';
  import { client } from '$proto';
  import type { DeviceGroup } from '../../../../../proto/gen/backend/proto/service_pb.js';
  import { formSchema } from './schema.js';

  let {
    class: className,
    parkId,
    group,
    onSuccess,
    ...restProps
  }: HTMLAttributes<HTMLFormElement> & {
    parkId: string;
    group?: DeviceGroup;
    onSuccess?: (group?: DeviceGroup) => void;
  } = $props();

  const createGroupMutation = createMutation(() => ({
    mutationKey: ['createGroup', parkId],
    mutationFn: async (data: { name: string; description: string }) => {
      return await client.park.createGroup({
        parkId,
        name: data.name,
        description: data.description,
      });
    },
    onSuccess: (response) => {
      onSuccess?.(response.group);
    },
  }));

  const updateGroupMutation = createMutation(() => ({
    mutationKey: ['updateGroup', group?.id],
    mutationFn: async (data: { name: string; description: string }) => {
      return await client.park.updateGroup({
        id: group?.id,
        name: data.name,
        description: data.description,
      });
    },
    onSuccess: (response) => {
      onSuccess?.(response.group);
    },
  }));

  let currentAction = $derived(group ? updateGroupMutation : createGroupMutation);

  const { form, formData, isValid, enhance } = spaForm(formSchema, {
    async onUpdated({ form }) {
      if (form.valid) {
        await currentAction.mutateAsync({
          name: form.data.name,
          description: form.data.description,
        });
      }
    },
  });

  onMount(() => {
    if (group) {
      $formData.name = group.name;
      $formData.description = group.description;
    }
  });
</script>

<form method="POST" use:enhance class={cn('flex flex-col gap-4', className)} {...restProps}>
  <FieldGroup>
    <Form.Field {form} name="name">
      <Form.Control>
        {#snippet children({ props })}
          <Input
            type="magic"
            magicProps={{ type: 'text', label: 'Название', placeholder: 'Введите название группы' }}
            {...props}
            bind:value={$formData.name}
          />
        {/snippet}
      </Form.Control>
      <Form.Description>Название группы устройств (от 1 до 30 символов).</Form.Description>
      <Form.FieldErrors />
    </Form.Field>

    <Form.Field {form} name="description">
      <Form.Control>
        {#snippet children({ props })}
          <Input
            type="magic"
            magicProps={{ type: 'text', label: 'Описание', placeholder: 'Введите описание группы' }}
            {...props}
            bind:value={$formData.description}
          />
        {/snippet}
      </Form.Control>
      <Form.Description>Описание назначения группы устройств.</Form.Description>
      <Form.FieldErrors />
    </Form.Field>

    <SubmitBtn
      {isValid}
      label={currentAction === createGroupMutation ? 'Создать группу' : 'Обновить группу'}
      isAuthError={currentAction.isError}
      isLoading={currentAction.isPending}
      errorMessage="Не удалось сохранить группу. Попробуйте снова."
    />
  </FieldGroup>
</form>
