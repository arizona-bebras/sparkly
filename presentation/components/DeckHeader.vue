<script setup lang="ts">
withDefaults(defineProps<{ section: string; context?: string }>(), {
  context: 'Sparkly / Мониторинг парка IoT',
})
</script>

<template>
  <header class="sp-header sp-mono">
    <span class="sp-section">{{ section }}</span>
    <span>{{ context }}</span>
  </header>
</template>
