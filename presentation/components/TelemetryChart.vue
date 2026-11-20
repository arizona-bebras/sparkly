<script setup lang="ts">
// Synthetic temperatures sampled once a second. ESP32-017 has a gap around the reboot.
const devices = [
  { id: 'ESP32-017', tone: 'accent', samples: [37, 38, 37, 39, 40, 42, 44, 46, null, null, 43, 42, 41] },
  { id: 'ESP32-024', tone: 'signal', samples: [35, 35, 36, 35, 36, 36, 35, 37, 36, 37, 36, 36, 37] },
]
const rebootSecond = 9
const ticksY = [50, 45, 40, 35, 30]
const ticksX = [0, 3, 6, 9, 12]
const x = (second: number) => 60 + second * 50
const y = (degrees: number) => 58 + (50 - degrees) * 12.6
const baseline = y(30)

function segments(samples: (number | null)[]) {
  const groups: { second: number; degrees: number }[][] = []
  samples.forEach((degrees, second) => {
    if (degrees === null) return
    if (second === 0 || samples[second - 1] === null) groups.push([])
    groups[groups.length - 1].push({ second, degrees })
  })
  return groups
}

function line(group: ReturnType<typeof segments>[number]) {
  return group.map(({ second, degrees }, i) => `${i === 0 ? 'M' : 'L'}${x(second)} ${y(degrees)}`).join(' ')
}

const primary = segments(devices[0].samples)
const temperatureAreas = primary.map(group => {
  const first = group[0].second
  return `${line(group)}V${baseline}H${x(first)}Z`
})
const peak = primary[0].reduce((highest, sample) => sample.degrees > highest.degrees ? sample : highest)
</script>

<template>
  <figure class="sp-chart" :aria-label="`Демонстрационные данные: температура ESP32-017 достигает ${peak.degrees} °C перед перезагрузкой, температура ESP32-024 остаётся стабильной.`">
    <figcaption>
      <h2>Температура устройств</h2>
      <div class="sp-chart-legend sp-mono">
        <span v-for="device in devices" :key="device.id" :class="`sp-legend--${device.tone}`">{{ device.id }}</span>
      </div>
    </figcaption>
    <svg viewBox="0 0 688 364" aria-hidden="true" fill="none">
      <defs>
        <linearGradient id="sp-telemetry-area" x1="0" y1="0" x2="0" :y2="baseline" gradientUnits="userSpaceOnUse">
          <stop offset="0" stop-color="var(--sp-accent)" stop-opacity=".28" />
          <stop offset="1" stop-color="var(--sp-accent)" stop-opacity=".04" />
        </linearGradient>
      </defs>
      <text x="60" y="26">°C</text>
      <g v-for="tick in ticksY" :key="tick">
        <path class="sp-chart-grid" :d="`M60 ${y(tick)}H660`" />
        <text x="44" :y="y(tick) + 5" text-anchor="end">{{ tick }}</text>
      </g>
      <path class="sp-chart-axis" :d="`M60 58V${baseline}H660`" />
      <path v-for="area in temperatureAreas" :key="area" :d="area" fill="url(#sp-telemetry-area)" />
      <path class="sp-sv-guide" :d="`M${x(rebootSecond)} 40V${baseline}`" />
      <text class="sp-chart-event" :x="x(rebootSecond) - 10" y="26" text-anchor="end">Перезагрузка</text>
      <path v-for="device in devices" :key="device.id" :class="`sp-sv-${device.tone}`" :d="segments(device.samples).map(line).join(' ')" />
      <circle :cx="x(peak.second)" :cy="y(peak.degrees)" r="4.5" fill="var(--sp-accent)" />
      <text class="sp-chart-peak" :x="x(peak.second)" :y="y(peak.degrees) - 16" text-anchor="middle">{{ peak.degrees }} °C</text>
      <text v-for="tick in ticksX" :key="tick" :x="x(tick)" y="344" text-anchor="middle">{{ `12:41:${String(tick).padStart(2, '0')}` }}</text>
    </svg>
  </figure>
</template>
