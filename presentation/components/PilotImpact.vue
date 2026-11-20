<script setup lang="ts">
const metrics = [
  {
    label: 'Сокращение MTTR',
    detail: 'Среднее время восстановления',
    min: 30,
    max: 60,
  },
  {
    label: 'Снижение затрат',
    detail: 'На выездную диагностику',
    min: 50,
    max: 70,
  },
]
const ticks = [0, 25, 50, 75, 100]
</script>

<template>
  <div class="sp-impact">
    <div class="sp-impact-copy">
      <h1 class="sp-statement">Меньше простоя,<br><span class="sp-accent">меньше выездов</span></h1>
      <p class="sp-intro">Две гипотезы, которые проверим на пилоте.</p>
      <div class="sp-impact-metrics">
        <MetricBlock
          v-for="metric in metrics"
          :key="metric.label"
          :value="`${metric.min}–${metric.max}%`"
          :label="metric.label"
          :detail="metric.detail"
        />
      </div>
    </div>
    <div class="sp-impact-data">
      <p class="sp-impact-eyebrow sp-mono">КАК ПРОВЕРЯЕМ</p>
      <p class="sp-impact-method">Сравним время восстановления и затраты на выезды до и после подключения Sparkly.</p>
      <figure class="sp-card sp-impact-chart">
        <figcaption>
          <h2>Целевые диапазоны снижения</h2>
          <p>Относительно исходного уровня</p>
        </figcaption>
        <div class="sp-range-plot" aria-hidden="true">
          <div v-for="metric in metrics" :key="metric.label" class="sp-range-row">
            <span>{{ metric.label }}</span>
            <div class="sp-range-track" :style="{ '--range-start': metric.min + '%', '--range-size': (metric.max - metric.min) + '%' }">
              <div class="sp-range-fill"><span>{{ metric.min }}%</span><span>{{ metric.max }}%</span></div>
            </div>
          </div>
          <div class="sp-range-axis sp-mono">
            <span v-for="tick in ticks" :key="tick" :style="{ left: tick + '%' }">{{ tick }}%</span>
          </div>
        </div>
      </figure>
    </div>
  </div>
</template>
