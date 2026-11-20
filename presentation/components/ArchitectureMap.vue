<script setup lang="ts">
const layers = ['Устройство', 'Приём', 'Хранилища', 'Бизнес-логика', 'Интерфейс']
const nodes = [
  { x: 0, y: 100, name: 'ESP32', detail: 'Rust-агент · C FFI' },
  { x: 240, y: 100, name: 'Ingress', detail: 'Валидация', stack: 'Go или Rust' },
  { x: 480, y: 36, name: 'VictoriaLogs', detail: 'Текстовые логи' },
  { x: 480, y: 164, name: 'ClickHouse', detail: 'Метрики' },
  { x: 720, y: 100, name: 'Сервис', detail: 'Статусы, инциденты', stack: 'Go' },
  { x: 720, y: 228, name: 'PostgreSQL', detail: 'Устройства и связи' },
  { x: 960, y: 100, name: 'Веб-панель', detail: 'Парк и диагностика' },
]
</script>

<template>
  <div class="sp-architecture">
    <div class="sp-architecture-top sp-mono sp-meta">
      <span class="sp-architecture-key">Поток телеметрии</span>
      <span>Изолированный контур</span>
    </div>
    <div class="sp-map" role="group" aria-label="ESP32 отправляет телеметрию в Ingress. Логи идут в VictoriaLogs, метрики — в ClickHouse. Сервис читает данные, ведёт реестр в PostgreSQL и обслуживает веб-панель через gRPC.">
      <div class="sp-map-layers sp-mono sp-meta">
        <span v-for="(layer, index) in layers" :key="layer"><b>0{{ index + 1 }}</b> {{ layer }}</span>
      </div>
      <svg class="sp-map-lines" viewBox="0 0 1152 332" aria-hidden="true" fill="none">
        <path class="sp-map-data" d="M192 152H238m-6-5 6 5-6 5M432 152H456V88H478m-6-5 6 5-6 5M456 152V216H478m-6-5 6 5-6 5M672 88H696V216H672M696 152H718m-6-5 6 5-6 5" />
        <circle cx="456" cy="152" r="3" fill="var(--sp-signal)" />
        <circle cx="696" cy="152" r="3" fill="var(--sp-signal)" />
        <path class="sp-map-api" d="M912 152H958m-6-5 6 5-6 5M816 204V228" />
      </svg>
      <article v-for="node in nodes" :key="node.name" class="sp-card sp-map-node" :style="{ left: `${node.x}px`, top: `${node.y}px` }">
        <h2>{{ node.name }}</h2>
        <p>{{ node.detail }}</p>
        <span v-if="node.stack" class="sp-mono sp-meta">{{ node.stack }}</span>
      </article>
      <span class="sp-map-grpc sp-mono">gRPC</span>
    </div>
    <p class="sp-architecture-registration"><span class="sp-accent sp-mono">РЕГИСТРАЦИЯ</span> Устройство подключается по токену группы.</p>
  </div>
</template>
