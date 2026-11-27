<script lang="ts">
  import BackgroundImage from '$assets/background.png';
  import { createMutation } from '@tanstack/svelte-query';
  import type { HTMLAttributes } from 'svelte/elements';
  import { goto } from '$app/navigation';
  import SubmitBtn from '$lib/components/forms/SubmitBtn.svelte';
  import { formSchema } from '$lib/components/forms/signin/schema.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import {
    Field,
    FieldDescription,
    FieldGroup,
    FieldLabel,
    FieldSeparator,
  } from '$lib/components/ui/field/index.js';
  import * as Form from '$lib/components/ui/form/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { spaForm } from '$lib/form-spa-util.svelte';
  import { cn } from '$lib/utils.js';
  import { client } from '$proto';

  let { class: className, ...restProps }: HTMLAttributes<HTMLDivElement> = $props();

  const { form, formData, isValid, enhance } = spaForm(formSchema, {
    async onUpdated({ form }) {
      await loginMutation.mutateAsync({
        email: form.data.email,
        password: form.data.password,
      });
    },
  });

  const loginMutation = createMutation(() => ({
    mutationKey: ['auth'],
    mutationFn: async (form: { email: string; password: string }) =>
      await client.user.getToken({
        email: form.email,
        password: form.password,
      }),
    onSuccess: () => {
      goto('/');
    },
  }));

  $inspect(isValid());
</script>

<div class={cn('flex flex-col gap-6', className)} {...restProps}>
  <Card.Root
    class="animate-in fade-in zoom-in-95 duration-500 ease-out overflow-hidden p-0 shadow-lg"
  >
    <Card.Content class="grid overflow-hidden p-0 md:grid-cols-2">
      <form
        method="POST"
        use:enhance
        class="relative z-10 bg-card animate-in fade-in slide-in-from-bottom-4 md:slide-in-from-bottom-0 md:slide-in-from-right-[50%] duration-700 ease-out p-6 md:p-8"
      >
        <FieldGroup>
          <div class="flex flex-col items-center gap-2 text-center">
            <h1 class="text-2xl font-bold">Добро пожаловать</h1>
            <p class="text-balance text-muted-foreground">Войдите в свой аккаунт Sparkly</p>
          </div>
          <Form.Field {form} name="email">
            <Form.Control>
              {#snippet children({ props })}
                <Input
                  type="magic"
                  magicProps={{
                    type: 'email',
                    label: 'Email',
                    placeholder: 'Enter your email',
                  }}
                  {...props}
                  bind:value={$formData.email}
                />
              {/snippet}
            </Form.Control>
            <Form.Description>Это ваша электронная почта.</Form.Description>
            <Form.FieldErrors />
          </Form.Field>
          <Form.Field {form} name="password">
            <Form.Control>
              {#snippet children({ props })}
                <Input
                  type="magic"
                  magicProps={{
                    type: 'password',
                    label: 'Password',
                    placeholder: 'Create a password',
                  }}
                  {...props}
                  bind:value={$formData.password}
                />
              {/snippet}
            </Form.Control>
            <Form.Description>Должен содержать не менее 6 символов.</Form.Description>
            <Form.FieldErrors />
          </Form.Field>
          <SubmitBtn
            {isValid}
            isAuthError={loginMutation.isError}
            isLoading={loginMutation.isPending}
          />
          <FieldDescription class="text-center">
            Ешё нет аккаунта? <a href="/auth/signup">Зарегестрироваться</a>
          </FieldDescription>
        </FieldGroup>
      </form>
      <div
        class="relative z-0 animate-in fade-in md:slide-in-from-left-[50%] duration-700 ease-out hidden overflow-hidden bg-muted md:block"
      >
        <img
          src={BackgroundImage}
          alt="placeholder"
          class="absolute inset-0 h-full w-full object-fit dark:brightness-[0.2] dark:grayscale"
        />
      </div>
    </Card.Content>
  </Card.Root>
  <!-- <FieldDescription class="px-6 text-center">
    By clicking continue, you agree to our <a href="##">Terms of Service</a> and
    <a href="##">Privacy Policy</a>.
  </FieldDescription>  -->
</div>
