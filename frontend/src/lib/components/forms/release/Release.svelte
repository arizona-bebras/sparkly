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
  import { client, type Release } from '$proto';
  import { formSchema } from './schema.js';

  let {
    class: className,
    parkId,
    release,
    onSuccess,
    ...restProps
  }: HTMLAttributes<HTMLFormElement> & {
    parkId: string;
    release?: Release;
    onSuccess?: (release?: Release) => void;
  } = $props();

  const createReleaseMutation = createMutation(() => ({
    mutationKey: ['createRelease', parkId],
    mutationFn: async (data: { version: string; name: string }) => {
      return await client.release.createRelease({
        parkId,
        version: data.version,
        name: data.name,
      });
    },
    onSuccess: (response) => {
      onSuccess?.(response.release);
    },
  }));

  const updateReleaseMutation = createMutation(() => ({
    mutationKey: ['updateRelease', release?.id],
    mutationFn: async (data: { version: string; name: string }) => {
      return await client.release.updateRelease({
        id: release?.id,
        version: data.version,
        name: data.name,
      });
    },
    onSuccess: (response) => {
      onSuccess?.(response.release);
    },
  }));

  let currentAction = $derived(release ? updateReleaseMutation : createReleaseMutation);

  const { form, formData, isValid, enhance } = spaForm(formSchema, {
    async onUpdated({ form }) {
      if (form.valid) {
        await currentAction.mutateAsync({
          version: form.data.version,
          name: form.data.name,
        });
      }
    },
  });

  onMount(() => {
    if (release) {
      $formData.version = release.version;
      $formData.name = release.name;
    }
  });
</script>

<form method="POST" use:enhance class={cn('flex flex-col gap-4', className)} {...restProps}>
  <FieldGroup>
    <Form.Field {form} name="version">
      <Form.Control>
        {#snippet children({ props })}
          <Input
            type="magic"
            magicProps={{ type: 'text', label: 'Версия', placeholder: 'Например, v1.0.0' }}
            {...props}
            bind:value={$formData.version}
          />
        {/snippet}
      </Form.Control>
      <Form.Description>Версия релиза (от 1 до 30 символов).</Form.Description>
      <Form.FieldErrors />
    </Form.Field>

    <Form.Field {form} name="name">
      <Form.Control>
        {#snippet children({ props })}
          <Input
            type="magic"
            magicProps={{
              type: 'text',
              label: 'Название',
              placeholder: 'Например, Стабильная версия с новыми датчиками',
            }}
            {...props}
            bind:value={$formData.name}
          />
        {/snippet}
      </Form.Control>
      <Form.Description>Краткое описание или название релиза.</Form.Description>
      <Form.FieldErrors />
    </Form.Field>

    <SubmitBtn
      {isValid}
      label={currentAction === createReleaseMutation ? 'Создать релиз' : 'Обновить релиз'}
      isAuthError={currentAction.isError}
      isLoading={currentAction.isPending}
      errorMessage="Не удалось сохранить релиз. Попробуйте снова."
    />
  </FieldGroup>
</form>
