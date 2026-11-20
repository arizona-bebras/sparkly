<script setup lang="ts">
defineProps<{ note: string }>()
const format = (number: number) => String(number).padStart(2, '0')
</script>

<template>
  <footer class="sp-footer sp-mono">
    <span>{{ note }}</span>
    <span>{{ format($page) }} <span class="sp-footer-sep">/</span> {{ format($nav.total) }}</span>
  </footer>
</template>
