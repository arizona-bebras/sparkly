<script setup lang="ts">
withDefaults(defineProps<{ label: string; tone?: 'neutral' | 'accent' | 'success' }>(), {
  tone: 'neutral',
})
</script>

<template>
  <span class="sp-badge sp-mono" :class="`sp-badge--${tone}`">
    <span class="sp-badge-dot" aria-hidden="true"></span>{{ label }}
  </span>
</template>
