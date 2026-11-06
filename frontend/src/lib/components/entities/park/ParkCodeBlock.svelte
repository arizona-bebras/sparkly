<script lang="ts">
  import * as CodeBlock from '$lib/components/ui/code-block/index.js';
  import { useHighlightedCode } from '$lib/hooks/use-highlighted-code.svelte.js';
  import { RustIcon, CIcon } from '$lib/components/icons/index.js';
  import { cn } from '$lib/utils.js';

  let {
    number,
    title,
    file,
    code,
    lang,
    isLast = false,
    class: className,
  }: {
    number?: number | string;
    title?: string;
    file: string;
    code: string;
    lang: 'rust' | 'c';
    isLast?: boolean;
    class?: string;
  } = $props();

  const lines = useHighlightedCode(
    () => code,
    () => lang,
  );
</script>

{#snippet content()}
  <div class="flex flex-col gap-2.5">
    {#if title}
      <div class="flex min-h-8 items-center">
        <h3 class="text-sm font-semibold leading-tight text-foreground">
          {title}
        </h3>
      </div>
    {/if}

    <CodeBlock.Root
      class="dark bg-neutral-950 text-neutral-100 border border-neutral-800 rounded-xl shadow-xs overflow-hidden"
    >
      <CodeBlock.Header
        class="flex items-center justify-between border-b border-neutral-800/80 px-4 py-2 bg-neutral-900/60"
      >
        <CodeBlock.Title class="text-xs text-neutral-400 font-mono flex items-center gap-1.5">
          {#if lang === 'rust'}
            <RustIcon class="size-3.5 text-[#dea584]" />
          {:else}
            <CIcon class="size-3.5" />
          {/if}
          <span>{file}</span>
        </CodeBlock.Title>
        <CodeBlock.Actions>
          <CodeBlock.CopyButton value={code} class="text-neutral-400 hover:text-white" />
        </CodeBlock.Actions>
      </CodeBlock.Header>

      <CodeBlock.Content
        lineNumbers
        wrap
        class="px-5 py-3.5 text-xs font-mono leading-relaxed overflow-hidden text-neutral-200"
      >
        {#each lines.current as line, lineIndex (lineIndex)}
          <CodeBlock.Line code={line} />
        {/each}
      </CodeBlock.Content>
    </CodeBlock.Root>
  </div>
{/snippet}

{#if number !== undefined}
  <div class={cn('relative flex gap-4 sm:gap-5', className)}>
    <div class="flex flex-col items-center">
      <div class="flex h-8 items-center justify-center">
        <div
          class="flex size-7 shrink-0 items-center justify-center rounded-full font-semibold text-xs shadow-xs ring-4 ring-background z-10 bg-neutral-900 text-white"
        >
          {number}
        </div>
      </div>

      {#if !isLast}
        <div
          class="w-0 flex-1 border-l-2 border-dashed my-1 border-border"
          aria-hidden="true"
        ></div>
      {/if}
    </div>

    <div class="flex-1 min-w-0 {!isLast ? 'pb-7' : 'pb-2'}">
      {@render content()}
    </div>
  </div>
{:else}
  <div class={className}>
    {@render content()}
  </div>
{/if}
