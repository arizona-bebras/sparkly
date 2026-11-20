<script setup lang="ts">
// Synthetic temperatures sampled once a second. ESP32-017 has a gap around the reboot.
const devices = [
  { id: 'ESP32-017', tone: 'accent', samples: [37, 38, 37, 39, 40, 42, 44, 46, null, null, 43, 42, 41] },
  { id: 'ESP32-024', tone: 'signal', samples: [35, 35, 36, 35, 36, 36, 35, 37, 36, 37, 36, 36, 37] },
]
const ticksY = [50, 45, 40, 35, 30]
const ticksX = [0, 3, 6, 9, 12]
const x = (second: number) => 44 + second * 43
const y = (degrees: number) => 34 + (50 - degrees) * 6.6

function path(samples: (number | null)[]) {
  return samples.map((value, index) => value === null ? '' :
    `${index === 0 || samples[index - 1] === null ? 'M' : 'L'}${x(index)} ${y(value)}`,
  ).join(' ')
}
</script>

<template>
  <figure class="sp-chart" aria-label="Демонстрационные данные: температура ESP32-017 растёт до перезагрузки, температура ESP32-024 остаётся стабильной.">
    <figcaption>
      <h2>Температура устройств</h2>
      <div class="sp-chart-legend sp-mono">
        <span v-for="device in devices" :key="device.id" :class="`sp-legend--${device.tone}`">{{ device.id }}</span>
      </div>
    </figcaption>
    <svg viewBox="0 0 572 220" aria-hidden="true" fill="none">
      <text x="9" y="18">°C</text>
      <g v-for="tick in ticksY" :key="tick">
        <path class="sp-sv-grid" :d="`M44 ${y(tick)}H560`" />
        <text x="27" :y="y(tick) + 5" text-anchor="end">{{ tick }}</text>
      </g>
      <path class="sp-sv-grid" d="M44 34V166" />
      <path class="sp-sv-guide" :d="`M${x(9)} 27V166`" />
      <text class="sp-chart-event" :x="x(9) - 8" y="18" text-anchor="end">Перезагрузка</text>
      <path v-for="device in devices" :key="device.id" :class="`sp-sv-${device.tone}`" :d="path(device.samples)" />
      <text v-for="tick in ticksX" :key="tick" :x="x(tick)" y="200" :text-anchor="tick === 0 ? 'start' : tick === 12 ? 'end' : 'middle'">{{ `12:41:${String(tick).padStart(2, '0')}` }}</text>
    </svg>
  </figure>
</template>
