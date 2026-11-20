<script setup lang="ts">
const entries = [
  { time: '12:41:02', level: 'INFO', message: 'telemetry batch sent' },
  { time: '12:41:08', level: 'WARN', message: 'supply voltage low' },
  { time: '12:41:09', level: 'ERROR', message: 'unexpected reboot' },
]
</script>

<template>
  <section class="sp-log" aria-label="Журнал ESP32-017">
    <div class="sp-log-heading"><h2>Журнал событий</h2><span class="sp-mono sp-meta">ESP32-017</span></div>
    <ol class="sp-log-entries sp-mono">
      <li v-for="entry in entries" :key="entry.time">
        <time>{{ entry.time }}</time>
        <strong :class="`sp-level-${entry.level.toLowerCase()}`">{{ entry.level }}</strong>
        <span>{{ entry.message }}</span>
      </li>
    </ol>
    <p class="sp-log-note">Перед перезагрузкой агент зафиксировал низкое напряжение питания.</p>
  </section>
</template>
