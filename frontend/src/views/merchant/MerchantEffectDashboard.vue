<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { PublishDailyPoint, PublishStats, PublishStatsRange } from '../../api/merchant'

type PlatformOption = { platformCode: string; platformName: string }

const props = withDefaults(defineProps<{
  stats: PublishStats | null
  loading: boolean
  error: string
  storeName: string
  platformOptions: PlatformOption[]
}>(), {
  platformOptions: () => []
})

const emit = defineEmits<{
  (event: 'range-change', value: PublishStatsRange): void
  (event: 'platform-change', value: string): void
  (event: 'retry'): void
}>()

const isReady = computed(() => props.stats?.dataState === 'ready')
const isPlatformFiltered = computed(() => Boolean(props.stats?.platformCode))
const stageByCode = computed(() => new Map((props.stats?.funnel || []).map((stage) => [stage.code, stage])))
const metricItems = computed(() => [
  { code: 'page_view', label: '贴卡访问', value: stageByCode.value.get('page_view')?.count || 0, note: isPlatformFiltered.value ? '全店进入评价流程的唯一会话' : '进入评价流程的唯一会话' },
  { code: 'platform_select', label: '选择平台', value: stageByCode.value.get('platform_select')?.count || 0, note: '选择了目标评价平台' },
  { code: 'review_copy', label: '复制评价', value: stageByCode.value.get('review_copy')?.count || 0, note: '已复制符合体验的评价' },
  { code: 'platform_link_click', label: '平台点击', value: stageByCode.value.get('platform_link_click')?.count || 0, note: '已点击打开平台，非确认发布' }
])
const maxFunnelCount = computed(() => Math.max(...(props.stats?.funnel || []).map((stage) => stage.count), 1))
const chartSvg = ref<SVGSVGElement | null>(null)
const chartWidth = ref(720)
const chartHeight = 260
const plot = computed(() => ({ left: 52, right: Math.max(chartWidth.value - 22, 180), top: 24, bottom: 214 }))
const chartMax = computed(() => {
  const values = (props.stats?.dailySeries || []).flatMap((point) => [point.pageViews, point.reviewCopies, point.platformLinkClicks])
  return Math.max(...values, 1)
})
const chartTicks = computed(() => [0, 0.25, 0.5, 0.75, 1].map((ratio) => ({
  value: Math.round(chartMax.value * (1 - ratio)),
  y: plot.value.top + (plot.value.bottom - plot.value.top) * ratio
})))

function pointsFor(key: keyof Pick<PublishDailyPoint, 'pageViews' | 'reviewCopies' | 'platformLinkClicks'>) {
  const series = props.stats?.dailySeries || []
  return series.map((point, index) => {
    const x = series.length === 1 ? (plot.value.left + plot.value.right) / 2 : plot.value.left + ((plot.value.right - plot.value.left) * index) / (series.length - 1)
    const y = plot.value.bottom - (Number(point[key]) / chartMax.value) * (plot.value.bottom - plot.value.top)
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

const visitPoints = computed(() => pointsFor('pageViews'))
const copyPoints = computed(() => pointsFor('reviewCopies'))
const clickPoints = computed(() => pointsFor('platformLinkClicks'))
const dateLabels = computed(() => {
  const series = props.stats?.dailySeries || []
  const compactStep = series.length > 10 ? Math.ceil(Math.max(series.length - 1, 1) / 3) : Math.ceil(Math.max(series.length - 1, 1) / 2)
  const step = chartWidth.value < 420 ? compactStep : series.length > 10 ? 5 : 1
  return series.map((point, index) => ({ point, index })).filter(({ index }) => index === 0 || index === series.length - 1 || index % step === 0).map(({ point, index }) => ({
    label: point.date.slice(5).replace('-', '/'),
    x: series.length === 1 ? (plot.value.left + plot.value.right) / 2 : plot.value.left + ((plot.value.right - plot.value.left) * index) / (series.length - 1)
  }))
})
const chartAriaLabel = computed(() => {
  const totals = metricItems.value
  return `${props.stats?.rangeStart || ''} 至 ${props.stats?.rangeEnd || ''}：${totals.map((item) => `${item.label} ${item.value} 次`).join('，')}`
})
const stateCopy = computed(() => {
  if (!props.stats) return ''
  if (props.stats.dataState === 'empty' && props.stats.platformCode) return `还没有顾客选择${props.stats.platformName}，贴卡访问仍按全店入口展示。`
  if (props.stats.dataState === 'empty') return '还没有顾客贴卡访问，先从 NFC 摆放和店员引导开始。'
  if (props.stats.dataState === 'accumulating') return '数据积累中：先看真实数量，暂不展示趋势、转化率或掉点结论。'
  if (props.stats.platformCode) return `${props.stats.rangeStart} 至 ${props.stats.rangeEnd}，贴卡访问为全店入口，后续指标按${props.stats.platformName}筛选。`
  return `${props.stats.rangeStart} 至 ${props.stats.rangeEnd}，所有数字、漏斗和趋势使用同一统计口径。`
})

let chartResizeObserver: ResizeObserver | null = null

function updateChartWidth() {
  const width = chartSvg.value?.getBoundingClientRect().width
  if (width) chartWidth.value = Math.max(Math.round(width), 280)
}

watch([isReady, chartSvg], async ([ready]) => {
  chartResizeObserver?.disconnect()
  chartResizeObserver = null
  if (!ready) return
  await nextTick()
  updateChartWidth()
  if (chartSvg.value && typeof ResizeObserver !== 'undefined') {
    chartResizeObserver = new ResizeObserver(updateChartWidth)
    chartResizeObserver.observe(chartSvg.value)
  }
}, { immediate: true })

onBeforeUnmount(() => chartResizeObserver?.disconnect())

function formatNumber(value: number) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function funnelWidth(count: number) {
  return `${Math.max(count > 0 ? 8 : 0, (count / maxFunnelCount.value) * 100)}%`
}

function onPlatformChange(event: Event) {
  emit('platform-change', (event.target as HTMLSelectElement).value)
}

function focusRecommendationTarget() {
  const target = props.stats?.recommendation.actionTarget
  if (!target || typeof document === 'undefined') return
  const mediaQuery = typeof window.matchMedia === 'function' ? window.matchMedia('(prefers-reduced-motion: reduce)') : null
  const reduceMotion = mediaQuery?.matches === true
  const targetElement = document.querySelector<HTMLElement>(`[data-effect-target="${target}"]`)
  if (!targetElement) return
  if (targetElement instanceof HTMLDetailsElement) targetElement.open = true
  const focusTarget = targetElement instanceof HTMLDetailsElement
    ? targetElement.querySelector<HTMLElement>('summary')
    : targetElement
  if (focusTarget && !focusTarget.matches('summary, button, a, input, select, textarea, [tabindex]')) {
    focusTarget.setAttribute('tabindex', '-1')
  }
  targetElement.scrollIntoView({ behavior: reduceMotion ? 'auto' : 'smooth', block: 'center' })
  focusTarget?.focus({ preventScroll: true })
}
</script>

<template>
  <section class="effect-dashboard" aria-labelledby="effect-dashboard-title">
    <header class="effect-header">
      <div>
        <p class="effect-eyebrow">{{ storeName || '商家' }} · 真实使用效果</p>
        <h2 id="effect-dashboard-title">顾客评价转化</h2>
        <p v-if="stats" class="effect-source">统计时区 {{ stats.timezone }} · 更新至 {{ new Date(stats.updatedAt).toLocaleString('zh-CN', { hour12: false }) }}</p>
      </div>
      <div class="effect-filters" aria-label="效果数据筛选">
        <div class="range-switch" role="group" aria-label="时间范围">
          <button type="button" data-range="7d" :aria-pressed="(stats?.range || '7d') === '7d'" @click="emit('range-change', '7d')">近 7 天</button>
          <button type="button" data-range="30d" :aria-pressed="stats?.range === '30d'" @click="emit('range-change', '30d')">近 30 天</button>
        </div>
        <label for="effect-platform">数据平台</label>
        <select id="effect-platform" :value="stats?.platformCode || ''" :disabled="loading" @change="onPlatformChange">
          <option v-for="item in platformOptions" :key="item.platformCode || 'all'" :value="item.platformCode">{{ item.platformName }}</option>
        </select>
      </div>
    </header>

    <div v-if="loading && !stats" class="effect-loading" aria-busy="true" aria-live="polite">
      <span></span><span></span><span></span><span></span>
      <p>正在加载效果数据…</p>
    </div>

    <div v-else-if="error && !stats" class="effect-error" role="alert">
      <strong>看板数据加载失败</strong>
      <p>{{ error }}</p>
      <button type="button" data-testid="retry-dashboard" @click="emit('retry')">重新加载</button>
    </div>

    <template v-else-if="stats">
      <div v-if="error" class="effect-error effect-error-inline" role="alert">
        <strong>新筛选加载失败，当前仍显示上一次成功数据</strong>
        <p>{{ error }}</p>
        <button type="button" data-testid="retry-dashboard" @click="emit('retry')">重新加载</button>
      </div>
      <p class="effect-state" data-testid="data-state" :class="`state-${stats.dataState}`">{{ stateCopy }}</p>

      <div class="effect-metrics" aria-label="评价流程关键数字">
        <article v-for="item in metricItems" :key="item.code" :data-metric="item.code">
          <span>{{ item.label }}</span>
          <strong>{{ formatNumber(item.value) }}</strong>
          <small>{{ item.note }}</small>
        </article>
      </div>

      <div class="effect-grid">
        <section v-if="isReady" class="trend-panel" data-effect-target="trend" aria-labelledby="daily-trend-title">
          <div class="panel-heading">
            <div>
              <h3 id="daily-trend-title">每日趋势</h3>
              <p>{{ isPlatformFiltered ? '全店访问与所选平台后续行为的变化' : '访问、复制与平台点击的同口径变化' }}</p>
            </div>
            <div class="trend-legend" aria-hidden="true"><span class="visit">访问</span><span class="copy">复制</span><span class="click">平台点击</span></div>
          </div>
          <svg ref="chartSvg" data-testid="daily-trend" class="daily-chart" :viewBox="`0 0 ${chartWidth} ${chartHeight}`" role="img" :aria-label="chartAriaLabel">
            <g aria-hidden="true">
              <g v-for="tick in chartTicks" :key="tick.y">
                <line class="grid-line" :x1="plot.left" :x2="plot.right" :y1="tick.y" :y2="tick.y" />
                <text class="axis-label" :x="plot.left - 10" :y="tick.y + 4" text-anchor="end">{{ tick.value }}</text>
              </g>
              <text v-for="label in dateLabels" :key="`${label.x}-${label.label}`" class="axis-label" :x="label.x" :y="242" text-anchor="middle">{{ label.label }}</text>
            </g>
            <polyline class="series-line visits" :points="visitPoints" />
            <polyline class="series-line copies" :points="copyPoints" />
            <polyline class="series-line clicks" :points="clickPoints" />
          </svg>
          <table class="sr-only">
            <caption>{{ chartAriaLabel }}</caption>
            <thead><tr><th>日期</th><th>访问</th><th>复制</th><th>平台点击</th></tr></thead>
            <tbody><tr v-for="point in stats.dailySeries" :key="point.date"><th>{{ point.date }}</th><td>{{ point.pageViews }}</td><td>{{ point.reviewCopies }}</td><td>{{ point.platformLinkClicks }}</td></tr></tbody>
          </table>
        </section>

        <section class="funnel-panel" data-effect-target="funnel" aria-labelledby="funnel-title">
          <div class="panel-heading">
            <div><h3 id="funnel-title">四段漏斗</h3><p>每一步按唯一会话计算</p></div>
          </div>
          <ol class="funnel-list">
            <li v-for="(stage, index) in stats.funnel" :key="stage.code" data-funnel-stage>
              <div class="funnel-copy"><span><b>{{ index + 1 }}</b>{{ stage.label }}</span><strong>{{ formatNumber(stage.count) }}</strong></div>
              <div class="funnel-track" aria-hidden="true"><span :style="{ width: funnelWidth(stage.count) }"></span></div>
              <p v-if="isReady && stage.conversionAvailable" data-conversion-rate>{{ stage.conversionLabel || '上一步转化率' }} {{ stage.conversionRate.toFixed(1) }}%</p>
              <p v-else-if="index === 0">流程起点</p>
            </li>
          </ol>
        </section>
      </div>

      <section class="recommendation" data-testid="recommendation" data-effect-target="recommendation">
        <div>
          <span>首要建议</span>
          <h3>{{ stats.recommendation.title }}</h3>
          <p>{{ stats.recommendation.message }}</p>
        </div>
        <button type="button" @click="focusRecommendationTarget">{{ stats.recommendation.actionLabel }}</button>
      </section>
    </template>
  </section>
</template>

<style scoped>
.effect-dashboard { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; margin-bottom: 16px; padding: 20px; }
.effect-header { align-items: end; display: flex; gap: 20px; justify-content: space-between; }
.effect-header h2 { font-size: 28px; margin: 2px 0 0; }
.effect-eyebrow, .effect-source { color: var(--muted); font-size: 14px; margin: 0; }
.effect-source { margin-top: 5px; }
.effect-filters { align-items: center; display: flex; flex-wrap: wrap; gap: 8px; }
.effect-filters label { color: var(--muted); font-size: 14px; font-weight: 700; }
.effect-filters select { min-height: 44px; min-width: 140px; }
.range-switch { background: var(--surface-subtle); border: 1px solid var(--border); border-radius: 8px; display: flex; padding: 3px; }
.range-switch button { background: transparent; color: var(--muted); min-height: 44px; padding: 8px 12px; }
.range-switch button[aria-pressed="true"] { background: var(--surface); box-shadow: 0 1px 2px rgb(15 23 42 / 10%); color: var(--text); }
.effect-state { background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 6px; color: #334155; margin: 18px 0 14px; padding: 8px 12px; }
.effect-state.state-accumulating { background: #fffbeb; border-color: #fde68a; }
.effect-state.state-empty { background: #f8fafc; border-color: #cbd5e1; }
.effect-metrics { display: grid; gap: 10px; grid-template-columns: repeat(4, minmax(0, 1fr)); }
.effect-metrics article { border-top: 2px solid #cbd5e1; padding: 12px 4px 10px; }
.effect-metrics span, .effect-metrics small { color: var(--muted); display: block; font-size: 14px; }
.effect-metrics strong { display: block; font-size: 32px; line-height: 1.1; margin: 5px 0; }
.effect-grid { display: grid; gap: 14px; grid-template-columns: minmax(0, 2fr) minmax(280px, 1fr); margin-top: 14px; }
.trend-panel, .funnel-panel { border: 1px solid var(--border); border-radius: 8px; padding: 16px; }
.panel-heading { align-items: start; display: flex; gap: 12px; justify-content: space-between; }
.panel-heading h3 { font-size: 18px; margin: 0; }
.panel-heading p { color: var(--muted); font-size: 14px; margin: 3px 0 0; }
.trend-legend { display: flex; flex-wrap: wrap; gap: 12px; }
.trend-legend span { color: var(--muted); font-size: 14px; }
.trend-legend span::before { border-radius: 999px; content: ''; display: inline-block; height: 3px; margin-right: 5px; vertical-align: middle; width: 18px; }
.trend-legend .visit::before { background: #2563eb; }.trend-legend .copy::before { background: #d97706; }.trend-legend .click::before { background: #059669; }
.daily-chart { display: block; height: 260px; margin-top: 8px; width: 100%; }
.grid-line { stroke: #e2e8f0; stroke-dasharray: 4 5; vector-effect: non-scaling-stroke; }
.axis-label { fill: #64748b; font-size: 12px; font-weight: 650; }
.series-line { fill: none; stroke-linecap: round; stroke-linejoin: round; stroke-width: 3; vector-effect: non-scaling-stroke; }
.series-line.visits { stroke: #2563eb; }.series-line.copies { stroke: #d97706; }.series-line.clicks { stroke: #059669; }
.funnel-list { display: grid; gap: 12px; list-style: none; margin: 14px 0 0; padding: 0; }
.funnel-copy { align-items: center; display: flex; gap: 10px; justify-content: space-between; }
.funnel-copy span { align-items: center; display: flex; font-size: 16px; font-weight: 750; gap: 8px; }
.funnel-copy b { align-items: center; background: #e2e8f0; border-radius: 50%; display: inline-flex; font-size: 12px; height: 24px; justify-content: center; width: 24px; }
.funnel-copy strong { font-size: 20px; }
.funnel-track { background: #e2e8f0; border-radius: 999px; height: 8px; margin-top: 6px; overflow: hidden; }
.funnel-track span { background: #2563eb; border-radius: inherit; display: block; height: 100%; }
.funnel-list p { color: var(--muted); font-size: 14px; margin: 4px 0 0 32px; }
.recommendation { align-items: center; background: #f8fafc; border: 1px solid #cbd5e1; border-radius: 8px; display: flex; gap: 20px; justify-content: space-between; margin-top: 14px; padding: 14px 16px; }
.recommendation span { color: var(--muted); font-size: 14px; font-weight: 700; }.recommendation h3 { font-size: 18px; margin: 2px 0; }.recommendation p { color: #475569; font-size: 14px; margin: 0; }
.effect-loading { display: grid; gap: 10px; grid-template-columns: repeat(4, 1fr); margin-top: 18px; }
.effect-loading span { animation: pulse 1.4s ease-in-out infinite; background: #e2e8f0; height: 92px; }.effect-loading p { color: var(--muted); grid-column: 1 / -1; }
.effect-error { background: #fff7ed; border: 1px solid #fed7aa; margin-top: 18px; padding: 16px; }.effect-error p { margin: 4px 0 12px; }
.effect-error-inline { border-radius: 6px; }
.sr-only { border: 0; clip: rect(0, 0, 0, 0); height: 1px; margin: -1px; overflow: hidden; padding: 0; position: absolute; white-space: nowrap; width: 1px; }
@keyframes pulse { 50% { opacity: .45; } }
@media (prefers-reduced-motion: reduce) { .effect-loading span { animation: none; } }
@media (max-width: 820px) {
  .effect-dashboard { padding: 16px; }.effect-header { align-items: stretch; flex-direction: column; }.effect-filters { align-items: stretch; display: grid; grid-template-columns: 1fr 1fr; }.range-switch { grid-column: 1 / -1; }.range-switch button { flex: 1; }.effect-metrics { grid-template-columns: 1fr 1fr; }.effect-grid { display: flex; flex-direction: column; }.funnel-panel { order: 1; }.trend-panel { order: 2; }.recommendation { align-items: stretch; flex-direction: column; }.effect-loading { grid-template-columns: 1fr 1fr; }
}
@media (max-width: 420px) { .effect-metrics { grid-template-columns: 1fr; }.effect-header h2 { font-size: 24px; }.effect-metrics strong { font-size: 28px; } }
</style>
