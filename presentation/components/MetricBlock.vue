<script setup lang="ts">
defineProps<{ value: string; label: string; detail: string }>()
</script>

<template>
  <article class="sp-metric">
    <div class="sp-metric-value sp-mono">{{ value }}</div>
    <div>
      <h2>{{ label }}</h2>
      <p>{{ detail }}</p>
    </div>
  </article>
</template>
