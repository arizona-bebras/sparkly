<script lang="ts">
  import { GitCommit, Copy, Check } from '@lucide/svelte';

  let { commit }: { commit: string } = $props();

  let copied = $state(false);

  async function copyCommit() {
    if (!commit) return;
    try {
      await navigator.clipboard.writeText(commit);
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 1500);
    } catch {}
  }
</script>

{#if commit}
  <div class="inline-flex items-center gap-1.5 font-mono text-xs">
    <GitCommit class="size-3.5 text-muted-foreground shrink-0" />
    <span class="text-foreground" title={commit}>
      {commit.slice(0, 7)}
    </span>
    <button
      type="button"
      onclick={copyCommit}
      class="inline-flex items-center justify-center size-5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
      title="Скопировать хэш коммита"
      aria-label="Скопировать коммит"
    >
      {#if copied}
        <Check class="size-3 text-emerald-500" />
      {:else}
        <Copy class="size-3" />
      {/if}
    </button>
  </div>
{:else}
  <span class="text-xs text-muted-foreground">—</span>
{/if}
