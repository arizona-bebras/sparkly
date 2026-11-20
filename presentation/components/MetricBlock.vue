<script setup lang="ts">
defineProps<{ value: string; label: string; detail: string }>()
</script>

<template>
  <article class="sp-card sp-metric">
    <div class="sp-metric-value sp-accent sp-mono">{{ value }}</div>
    <h2>{{ label }}</h2>
    <p>{{ detail }}</p>
  </article>
</template>
