<script lang="ts">
  import { LoaderCircle } from '@lucide/svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { cn } from '$lib/utils.js';

  let {
    isValid,
    isAuthError,
    isLoading,
    errorMessage = 'Ошибка авторизации! Проверьте введённые данные',
    onSubmitAction,
    className,
    label = 'Войти',
  }: {
    isValid: () => boolean;
    isAuthError: boolean;
    isLoading: boolean;
    errorMessage?: string;
    onSubmitAction?: () => void;
    className?: string;
    label?: string;
  } = $props();
</script>

<Button
  type="submit"
  disabled={!isValid()}
  class="w-full flex items-center justify-center gap-2"
  onclick={onSubmitAction}
>
  <span>{label}</span>
  {#if isLoading}
    <LoaderCircle class="size-4 animate-spin" />
  {/if}
</Button>

{#if isAuthError}
  <p
    class={cn(
      '-mt-5 text-sm font-medium text-destructive text-center animate-in fade-in slide-in-from-top-1 duration-300 ease-out',
      className,
    )}
    role="alert"
  >
    {errorMessage}
  </p>
{/if}
