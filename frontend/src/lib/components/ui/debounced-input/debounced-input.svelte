<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import { untrack } from 'svelte';
  import { Search } from '@lucide/svelte';
  import { Input } from '$lib/components/ui/input/index.js';
  import { cn, type WithElementRef } from '$lib/utils.js';

  type Props = WithElementRef<Omit<HTMLInputAttributes, 'type' | 'value' | 'files'>> & {
    value?: string;
    immediateValue?: string;
    debounce?: number;
    showIcon?: boolean;
    type?: 'text' | 'search' | 'email' | 'url' | 'tel' | 'password';
    onDebounce?: (value: string) => void;
  };

  let {
    ref = $bindable(null),
    value = $bindable(''),
    immediateValue = $bindable(''),
    debounce = 350,
    showIcon = false,
    class: className,
    onDebounce,
    type = 'text',
    placeholder,
    disabled,
    ...restProps
  }: Props = $props();

  let innerValue = $state(value ?? '');
  let timer: ReturnType<typeof setTimeout> | undefined;

  $effect(() => {
    const external = value ?? '';
    untrack(() => {
      if (external !== innerValue) {
        innerValue = external;
        immediateValue = external;
      }
    });
  });

  function onInput(e: Event & { currentTarget: HTMLInputElement }) {
    const next = e.currentTarget.value;
    innerValue = next;
    immediateValue = next;

    clearTimeout(timer);
    timer = setTimeout(() => {
      value = next;
      onDebounce?.(next);
    }, debounce);
  }

  $effect(() => {
    return () => clearTimeout(timer);
  });
</script>

{#if showIcon}
  <div class="relative w-full sm:w-64">
    <Search
      class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground pointer-events-none"
    />
    <Input
      bind:ref
      value={innerValue}
      oninput={onInput}
      type="text"
      {placeholder}
      {disabled}
      class={cn('pl-9 h-9', className)}
      {...restProps}
    />
  </div>
{:else}
  <Input
    bind:ref
    value={innerValue}
    oninput={onInput}
    type="text"
    {placeholder}
    {disabled}
    class={className}
    {...restProps}
  />
{/if}
