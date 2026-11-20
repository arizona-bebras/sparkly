<script setup lang="ts">
const devices = [
  { name: 'Промышленный контроллер', signal: 'Циклы и ошибки' },
  { name: 'Датчик среды', signal: 'Показания и заряд' },
  { name: 'Шлюз связи', signal: 'Обрывы соединения' },
  { name: 'Контроллер освещения', signal: 'Перезапуски и отказы' },
  { name: 'Платёжный терминал', signal: 'Ошибки периферии' },
]
</script>

<template>
  <ol class="sp-device-gallery" aria-label="Примеры устройств">
    <li v-for="(device, index) in devices" :key="device.name">
      <span class="sp-mono sp-device-index">{{ String(index + 1).padStart(2, '0') }}</span>
      <h2>{{ device.name }}</h2>
      <p>{{ device.signal }}</p>
    </li>
  </ol>
</template>
