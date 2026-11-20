<script setup lang="ts">
const entries = [
  { time: '12:41:02', level: 'INFO', message: 'Пакет телеметрии отправлен' },
  { time: '12:41:08', level: 'WARN', message: 'Низкое напряжение питания' },
  { time: '12:41:09', level: 'ERROR', message: 'Внеплановая перезагрузка' },
]
</script>

<template>
  <section class="sp-log" aria-label="Журнал ESP32-017">
    <div class="sp-log-heading sp-mono sp-meta"><span>ЖУРНАЛ СОБЫТИЙ</span><span>ESP32-017</span></div>
    <ol class="sp-log-entries">
      <li v-for="entry in entries" :key="entry.time">
        <lucide:send v-if="entry.level === 'INFO'" class="sp-log-icon" aria-hidden="true" />
        <lucide:battery-low v-else-if="entry.level === 'WARN'" class="sp-log-icon sp-level-warn" aria-hidden="true" />
        <lucide:rotate-ccw v-else class="sp-log-icon sp-level-error" aria-hidden="true" />
        <div>
          <div class="sp-log-stamp sp-mono">
            <time :datetime="entry.time">{{ entry.time }}</time>
            <strong :class="`sp-level-${entry.level.toLowerCase()}`">{{ entry.level }}</strong>
          </div>
          <p>{{ entry.message }}</p>
        </div>
      </li>
    </ol>
  </section>
</template>
