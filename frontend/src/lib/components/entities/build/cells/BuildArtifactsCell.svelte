<script lang="ts">
  import { FileCode, Layers } from '@lucide/svelte';

  let {
    elfObjectKey,
    dwarfObjectKey,
  }: {
    elfObjectKey?: string;
    dwarfObjectKey?: string;
  } = $props();

  const hasElf = $derived(!!elfObjectKey && elfObjectKey.trim().length > 0);
  const hasDwarf = $derived(!!dwarfObjectKey && dwarfObjectKey.trim().length > 0);
</script>

<div class="flex flex-wrap items-center gap-1.5">
  {#if hasElf}
    <span
      class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border border-emerald-500/20 font-mono"
      title={`ELF: ${elfObjectKey}`}
    >
      <FileCode class="size-3 shrink-0" />
      ELF
    </span>
  {:else}
    <span
      class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium bg-muted/60 text-muted-foreground border border-border/40 font-mono"
      title="ELF не загружен"
    >
      Нет ELF
    </span>
  {/if}

  {#if hasDwarf}
    <span
      class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium bg-indigo-500/10 text-indigo-700 dark:text-indigo-300 border border-indigo-500/20 font-mono"
      title={`DWARF: ${dwarfObjectKey}`}
    >
      <Layers class="size-3 shrink-0" />
      DWARF
    </span>
  {:else}
    <span
      class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium bg-muted/60 text-muted-foreground border border-border/40 font-mono"
      title="DWARF не загружен"
    >
      Нет DWARF
    </span>
  {/if}
</div>
