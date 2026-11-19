<script lang="ts">
  import { Eye, Trash2 } from '@lucide/svelte';
  import type { Build } from '$proto';

  let {
    build,
    onViewDetails,
    onDelete,
  }: {
    build: Build;
    onViewDetails?: () => void;
    onDelete?: () => void;
  } = $props();
</script>

<div class="flex items-center justify-end gap-1">
  <button
    type="button"
    onclick={onViewDetails}
    class="inline-flex items-center justify-center size-8 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
    title="Сведения о сборке"
    aria-label="Сведения о сборке"
  >
    <Eye class="size-4" />
  </button>
  <button
    type="button"
    onclick={onDelete}
    class="inline-flex items-center justify-center size-8 rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
    title="Удалить сборку"
    aria-label="Удалить сборку"
  >
    <Trash2 class="size-4" />
  </button>
</div>
