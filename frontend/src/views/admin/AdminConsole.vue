<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { adminApi } from '../../api/admin'
import type { AdminStats, AdminStore, ExternalStoreReviewMatch, ReviewCrawlBatch } from '../../api/admin'
import type { DeviceBreakdownItem, ReviewGenerationTask } from '../../api/merchant'
import { copyToClipboard } from '../../utils/clipboard'
import { analyticsSourceLabel } from '../../utils/analyticsSource'
import { generationFailureReason, generationLogPreview, generationStageSummary } from '../../utils/generationAudit'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const merchants = ref<any[]>([])
const storeTypes = ref<any[]>([])
const stores = ref<AdminStore[]>([])
const tasks = ref<ReviewGenerationTask[]>([])
const stats = ref<AdminStats>(emptyStats())
const loading = ref(false)
const error = ref('')
const notice = ref('')
const editingStoreId = ref<number | null>(null)
const crawlPanelStore = ref<AdminStore | null>(null)
const crawlBatches = ref<ReviewCrawlBatch[]>([])
const crawlMatches = ref<ExternalStoreReviewMatch[]>([])
const crawlLoading = ref(false)
const selectedStoreId = ref<number | null>(null)
const storeSearch = ref('')
type StoreStatusFilter = 'all' | 'active' | 'inactive' | 'official_configured' | 'official_missing' | 'nfc_error' | 'crawl_failed' | 'no_crawl' | 'no_visit' | 'low_conversion'
const storeStatusFilter = ref<StoreStatusFilter>('all')
const configPanelOpen = ref(false)
const activeDetailTab = ref<'overview' | 'usage' | 'conversion' | 'advice' | 'config'>('overview')
const hoveredOpsChartIndex = ref<number | null>(null)
const opsChartSvg = ref<SVGSVGElement | null>(null)
const opsChartSize = reactive({ width: 600, height: 220 })
let opsResizeObserver: ResizeObserver | null = null

const platformOptions = [
  { code: 'dianping', name: '大众点评' },
  { code: 'meituan', name: '美团' },
  { code: 'xiaohongshu', name: '小红书' },
  { code: 'douyin', name: '抖音' }
]
const detailTabs = [
  { key: 'overview', label: '概览' },
  { key: 'usage', label: '使用明细' },
  { key: 'conversion', label: '转化分析' },
  { key: 'advice', label: '运营建议' },
  { key: 'config', label: '配置资料' }
] as const
const navItems = [
  { label: '运营总览', href: '#ops-overview' },
  { label: '商家列表', href: '#merchant-ledger' },
  { label: '商家详情', href: '#merchant-detail' },
  { label: '运营建议', href: '#ops-advice' },
  { label: '配置资料', href: '#config-drawer-anchor' }
]

// 自定义类型只能挂在 9 个预置行业之一之下（生成/隔离基准）
const presetTypes = computed(() => storeTypes.value.filter((t) => t.isPreset))
const isEditingStore = computed(() => editingStoreId.value !== null)
const globalDeviceItems = computed(() => stats.value.deviceStats?.items || [])
const globalTopDevice = computed(() => dominantDevice(globalDeviceItems.value))
const updatedText = computed(() => {
  if (!stats.value.updatedAt) return ''
  const d = new Date(stats.value.updatedAt)
  if (Number.isNaN(d.getTime())) return ''
  return `更新至 ${d.toLocaleString('zh-CN', { hour12: false })}`
})
const analyticsSourceText = computed(() => analyticsSourceLabel(stats.value.dataSourceLabel))
const deviceSummaryText = computed(() => {
  if (!stats.value.totalCustomerVisits) return '暂无顾客访问数据'
  if (!globalTopDevice.value) return '设备结构待积累'
  return `${globalTopDevice.value.label}最多，占 ${formatPercent(globalTopDevice.value.percent)}`
})
const topStores = computed(() => {
  return [...stores.value]
    .sort((a, b) => (b.analytics?.totalCustomerVisits || 0) - (a.analytics?.totalCustomerVisits || 0))
    .slice(0, 5)
})
const opsChartSeries = computed(() => {
  const source = topStores.value.length ? [...topStores.value].reverse() : stores.value.slice(0, 5)
  return source.map((item) => ({
    label: item.storeName,
    visits: item.analytics?.totalCustomerVisits || 0,
    clicks: item.analytics?.totalPublishClicks || 0
  }))
})
const opsChartHasData = computed(() => opsChartSeries.value.some((item) => item.visits > 0 || item.clicks > 0))
const opsChartMetrics = computed(() => buildOpsChartMetrics(opsChartSize.width, opsChartSize.height))
const opsChartViewBox = computed(() => `0 0 ${opsChartMetrics.value.width} ${opsChartMetrics.value.height}`)
const opsChartScale = computed(() => buildOpsChartScale(opsChartSeries.value, opsChartMetrics.value))
const opsVisitPoints = computed(() => buildOpsChartPoints(opsChartSeries.value, 'visits', opsChartScale.value.max, opsChartMetrics.value))
const opsClickPoints = computed(() => buildOpsChartPoints(opsChartSeries.value, 'clicks', opsChartScale.value.max, opsChartMetrics.value))
const opsVisitPolyline = computed(() => opsVisitPoints.value.map((p) => `${p.x},${p.y}`).join(' '))
const opsClickPolyline = computed(() => opsClickPoints.value.map((p) => `${p.x},${p.y}`).join(' '))
const opsAreaPolygon = computed(() => {
  if (!opsVisitPoints.value.length) return ''
  const metrics = opsChartMetrics.value
  return `${opsVisitPolyline.value} ${metrics.plotRight},${metrics.plotBottom} ${metrics.plotLeft},${metrics.plotBottom}`
})
const activeOpsChartPoint = computed(() => {
  if (hoveredOpsChartIndex.value === null) return null
  return opsVisitPoints.value[hoveredOpsChartIndex.value] || null
})
const opsTooltipStyle = computed(() => {
  const point = activeOpsChartPoint.value
  if (!point) return {}
  const metrics = opsChartMetrics.value
  return {
    left: `${(point.x / metrics.width) * 100}%`,
    top: `${(point.y / metrics.height) * 100}%`
  }
})
const opsChartAria = computed(() => {
  if (!opsChartHasData.value) return '暂无商家访问和官方点击分布数据'
  return `商家使用曲线：${opsChartSeries.value.map((item) => `${item.label}访问${item.visits}次，官方点击${item.clicks}次`).join('；')}`
})
const filteredStores = computed(() => {
  const query = storeSearch.value.trim().toLowerCase()
  return stores.value.filter((item) => {
    return storeMatchesStatusFilter(item, storeStatusFilter.value) && storeMatchesSearch(item, query)
  })
})
const selectedStore = computed(() => {
  if (!filteredStores.value.length) return null
  return filteredStores.value.find((item) => item.id === selectedStoreId.value) || filteredStores.value[0] || null
})
const selectedMerchant = computed(() => selectedStore.value ? merchantForStore(selectedStore.value) : {})
const activeStores = computed(() => stores.value.filter((item) => isStoreActive(item)))
const inactiveStoreCount = computed(() => stores.value.filter((item) => !isStoreActive(item)).length)
const officialConfiguredCount = computed(() => stores.value.filter((item) => hasOfficialLink(item)).length)
const officialMissingCount = computed(() => stores.value.filter((item) => !hasOfficialLink(item)).length)
const activeOfficialMissingCount = computed(() => activeStores.value.filter((item) => !hasOfficialLink(item)).length)
const nfcRouteUsableCount = computed(() => stores.value.filter((item) => nfcRouteStatus(item) === 'usable').length)
const nfcRouteErrorCount = computed(() => stores.value.filter((item) => nfcRouteStatus(item) !== 'usable').length)
const activeNoVisitCount = computed(() => activeStores.value.filter((item) => (item.analytics?.currentWeekCustomerVisits || 0) === 0).length)
const highVisitLowConversionCount = computed(() =>
  activeStores.value.filter((item) => {
    const visits = item.analytics?.totalCustomerVisits || 0
    return visits >= 50 && conversionRate(item) < 15
  }).length
)
const noCrawlConfigCount = computed(() => stores.value.filter((item) => !item.reviewCrawl?.enabled).length)
const pendingWorkCount = computed(() =>
  inactiveStoreCount.value +
  activeOfficialMissingCount.value +
  nfcRouteErrorCount.value +
  activeNoVisitCount.value +
  highVisitLowConversionCount.value +
  stats.value.crawlFailedStoreCount +
  noCrawlConfigCount.value
)
const workItems = computed(() => [
  { label: '未激活商家', count: inactiveStoreCount.value, tone: inactiveStoreCount.value > 0 ? 'warn' : 'stable', filter: 'inactive' as StoreStatusFilter },
  { label: '已激活但未配置官方链接', count: activeOfficialMissingCount.value, tone: activeOfficialMissingCount.value > 0 ? 'danger' : 'stable', filter: 'official_missing' as StoreStatusFilter },
  { label: 'NFC 路由不可用', count: nfcRouteErrorCount.value, tone: nfcRouteErrorCount.value > 0 ? 'danger' : 'stable', filter: 'nfc_error' as StoreStatusFilter },
  { label: '已激活但近 7 天无访问', count: activeNoVisitCount.value, tone: activeNoVisitCount.value > 0 ? 'warn' : 'stable', filter: 'no_visit' as StoreStatusFilter },
  { label: '访问高但发布点击低', count: highVisitLowConversionCount.value, tone: highVisitLowConversionCount.value > 0 ? 'warn' : 'stable', filter: 'low_conversion' as StoreStatusFilter },
  { label: '采集失败', count: stats.value.crawlFailedStoreCount, tone: stats.value.crawlFailedStoreCount > 0 ? 'warn' : 'stable', filter: 'crawl_failed' as StoreStatusFilter },
  { label: '未启用评论采集', count: noCrawlConfigCount.value, tone: noCrawlConfigCount.value > 0 ? 'neutral' : 'stable', filter: 'no_crawl' as StoreStatusFilter }
])
const globalDeviceAria = computed(() => {
  if (!globalDeviceItems.value.length) return '暂无全局访问设备数据'
  return `全局访问设备占比：${globalDeviceItems.value.map((item) => `${item.label}${formatPercent(item.percent)}`).join('，')}`
})

// 新建自定义类型
const newType = reactive({ name: '', industryCode: '' })
// 新建/编辑店铺（含商家账号）
const newStore = reactive({
  account: '',
  password: '',
  merchantName: '',
  contactName: '',
  typeId: 0,
  storeName: '',
  address: '',
  storeIntro: '',
  primaryPlatformStyle: 'dianping',
  brandTone: '',
  platformUrl: '',
  reviewCrawlPlatformCode: 'meituan',
  reviewCrawlExternalShopId: '',
  reviewCrawlEnabled: false
})
const selectedStoreIdConfig = reactive({
  storeId: 0,
  platformCode: 'meituan',
  externalShopId: '',
  enabled: false
})
const lastCreated = ref<{ storeName: string; uuid: string; landingUrl: string; account: string } | null>(null)

function emptyStats(): AdminStats {
  return {
    merchantCount: 0,
    storeCount: 0,
    tagCount: 0,
    taskCount: 0,
    enabledMerchantCount: 0,
    disabledMerchantCount: 0,
    currentWeekNewMerchants: 0,
    currentMonthNewMerchants: 0,
    enabledStoreCount: 0,
    disabledStoreCount: 0,
    crawlEnabledStoreCount: 0,
    crawlFailedStoreCount: 0,
    crawlDataAccumulatingCount: 0,
    totalCustomerVisits: 0,
    currentWeekCustomerVisits: 0,
    currentMonthCustomerVisits: 0,
    totalPublishClicks: 0,
    currentWeekPublishClicks: 0,
    currentMonthPublishClicks: 0,
    deviceStats: { totalCount: 0, items: [] },
    dataSource: '',
    dataSourceLabel: '',
    updatedAt: ''
  }
}

function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

function typeName(typeId?: number) {
  return storeTypes.value.find((t) => t.id === typeId)?.name || '-'
}

function formatNumber(value: number | undefined) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function formatPercent(value: number | undefined) {
  return `${Number(value || 0).toFixed(1)}%`
}

function shortStoreLabel(value: string) {
  const text = String(value || '').trim()
  if (text.length <= 5) return text || '-'
  return `${text.slice(0, 5)}…`
}

function niceOpsChartMax(value: number) {
  const raw = Math.max(value * 1.15, 4)
  const magnitude = Math.pow(10, Math.floor(Math.log10(raw)))
  const fraction = raw / magnitude
  const niceFraction = fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 5 ? 5 : 10
  return niceFraction * magnitude
}

function buildOpsChartMetrics(width: number, height: number) {
  const safeWidth = Math.max(Math.round(width || 0), 260)
  const safeHeight = Math.max(Math.round(height || 0), 180)
  const plotLeft = safeWidth < 420 ? 38 : 54
  const plotRight = Math.max(plotLeft + 120, safeWidth - 22)
  const plotTop = 32
  const plotBottom = Math.max(plotTop + 80, safeHeight - 38)
  return {
    width: safeWidth,
    height: safeHeight,
    plotLeft,
    plotRight,
    plotTop,
    plotBottom,
    plotHeight: plotBottom - plotTop
  }
}

function updateOpsChartSize() {
  const rect = opsChartSvg.value?.getBoundingClientRect()
  if (!rect?.width || !rect?.height) return
  opsChartSize.width = Math.round(rect.width)
  opsChartSize.height = Math.round(rect.height)
}

function startOpsChartObserver() {
  updateOpsChartSize()
  if (!opsChartSvg.value || typeof ResizeObserver === 'undefined') return
  opsResizeObserver = new ResizeObserver(updateOpsChartSize)
  opsResizeObserver.observe(opsChartSvg.value)
}

function buildOpsChartScale(series: { visits: number; clicks: number }[], metrics: ReturnType<typeof buildOpsChartMetrics>) {
  const maxValue = Math.max(...series.flatMap((item) => [item.visits, item.clicks]), 1)
  const max = niceOpsChartMax(maxValue)
  const steps = 4
  const ticks = Array.from({ length: steps + 1 }, (_, index) => {
    const value = Math.round((max * (steps - index)) / steps)
    const y = metrics.plotTop + (index * metrics.plotHeight) / steps
    return { value, y: Number(y.toFixed(2)) }
  })
  return { max, ticks }
}

function buildOpsChartPoints(series: { label: string; visits: number; clicks: number }[], key: 'visits' | 'clicks', maxValue: number, metrics: ReturnType<typeof buildOpsChartMetrics>) {
  if (!series.length) return []
  return series.map((item, index) => {
    const x = series.length === 1 ? (metrics.plotLeft + metrics.plotRight) / 2 : metrics.plotLeft + (index * (metrics.plotRight - metrics.plotLeft)) / (series.length - 1)
    const y = metrics.plotBottom - (item[key] / Math.max(maxValue, 1)) * metrics.plotHeight
    return { ...item, x: Number(x.toFixed(2)), y: Number(y.toFixed(2)) }
  })
}

function opsAxisAnchor(x: number) {
  const metrics = opsChartMetrics.value
  if (x <= metrics.plotLeft + 1) return 'start'
  if (x >= metrics.plotRight - 1) return 'end'
  return 'middle'
}

function dominantDevice(items: DeviceBreakdownItem[]) {
  return items.reduce<DeviceBreakdownItem | null>((best, item) => {
    if (!best || item.count > best.count) return item
    return best
  }, null)
}

function deviceBarStyle(item: DeviceBreakdownItem) {
  return { width: `${Math.min(Math.max(item.percent, item.count > 0 ? 4 : 0), 100)}%` }
}

function primaryDeviceText(item: AdminStore) {
  const top = dominantDevice(item.analytics?.deviceStats?.items || [])
  if (!top) return '暂无设备数据'
  return `${top.label} ${formatPercent(top.percent)}`
}

function platformDisplayName(code?: string) {
  return platformOptions.find((item) => item.code === code)?.name || code || '-'
}

function storeMatchesStatusFilter(item: AdminStore, filter: StoreStatusFilter) {
  if (filter === 'active') return isStoreActive(item)
  if (filter === 'inactive') return !isStoreActive(item)
  if (filter === 'official_configured') return hasOfficialLink(item)
  if (filter === 'official_missing') return !hasOfficialLink(item)
  if (filter === 'nfc_error') return nfcRouteStatus(item) !== 'usable'
  if (filter === 'crawl_failed') return item.reviewCrawl?.lastStatus === 'failed'
  if (filter === 'no_crawl') return !item.reviewCrawl?.enabled
  if (filter === 'no_visit') return isStoreActive(item) && (item.analytics?.currentWeekCustomerVisits || 0) === 0
  if (filter === 'low_conversion') {
    const visits = item.analytics?.totalCustomerVisits || 0
    return isStoreActive(item) && visits >= 50 && conversionRate(item) < 15
  }
  return true
}

function storeMatchesSearch(item: AdminStore, query: string) {
  if (!query) return true
  const merchant = merchantForStore(item)
  return [
    item.storeName,
    item.industryType,
    item.merchantAccount,
    item.merchantName,
    item.contactName,
    merchant.account,
    item.reviewCrawl?.externalShopId,
    item.platformUrl,
    item.uuid
  ].some((value) => String(value || '').toLowerCase().includes(query))
}

function isStoreActive(item: AdminStore) {
  return item.status === 1
}

function activationStatusText(item: AdminStore) {
  return isStoreActive(item) ? '已激活' : '未激活'
}

function hasOfficialLink(item: AdminStore) {
  return !!String(item.platformUrl || '').trim()
}

function officialLinkStatusText(item: AdminStore) {
  return hasOfficialLink(item) ? '已配置' : '未配置'
}

function officialLinkStatusClass(item: AdminStore) {
  return hasOfficialLink(item) ? 'ready' : 'missing'
}

function nfcRouteStatus(item: AdminStore) {
  const status = item.nfcCardStatus?.primaryStatus
  if (status === 'usable' || status === 'unwritten' || status === 'unusable') return status
  if (!item.uuid) return 'unwritten'
  if (!isStoreActive(item)) return 'unusable'
  return 'unusable'
}

function nfcRouteStatusText(item: AdminStore) {
  const status = nfcRouteStatus(item)
  if (status === 'usable') return '可用'
  if (status === 'unwritten') return '未写入'
  return '不可用'
}

function nfcRouteStatusClass(item: AdminStore) {
  const status = nfcRouteStatus(item)
  if (status === 'usable') return 'enabled'
  if (status === 'unwritten') return 'disabled'
  return 'error'
}

function nfcRouteStatusDetail(item: AdminStore) {
  const status = item.nfcCardStatus
  if (!status) return '等待后端返回 NFC 状态'
  if (status.primaryStatus === 'usable') return `已写入 ${formatNumber(status.writtenCount)} 张`
  if (status.primaryStatus === 'unwritten') return '还没有绑定可用 NFC 标签'
  if (status.routeStatus === 'store_inactive') return '商家未激活，落地页不可用'
  if (status.disabledCount > 0) return `已禁用 ${formatNumber(status.disabledCount)} 张标签`
  return '请检查 NFC 标签绑定和落地页路由'
}

function storeVisitText(item: AdminStore) {
  const visits = item.analytics?.totalCustomerVisits || 0
  const publishes = item.analytics?.totalPublishClicks || 0
  return `${formatNumber(visits)} 访问 / ${formatNumber(publishes)} 发布`
}

function conversionRate(item: AdminStore) {
  const visits = item.analytics?.totalCustomerVisits || 0
  if (!visits) return 0
  return ((item.analytics?.totalPublishClicks || 0) / visits) * 100
}

function conversionRateText(item: AdminStore) {
  const visits = item.analytics?.totalCustomerVisits || 0
  if (!visits) return '-'
  return formatPercent(conversionRate(item))
}

function merchantIdText(item: AdminStore) {
  return item.reviewCrawl?.externalShopId || '-'
}

function merchantAccountText(item: AdminStore) {
  return item.merchantAccount || merchantForStore(item).account || '-'
}

function primaryRecommendation(item: AdminStore) {
  if (isStoreActive(item) && !hasOfficialLink(item)) return '先配置商家官方链接，恢复发布跳转链路'
  if (nfcRouteStatus(item) !== 'usable') return '检查商家激活状态和服务器落地页路由'
  if ((item.analytics?.currentWeekCustomerVisits || 0) === 0) return '检查 NFC 摆放位置和员工引导'
  if (conversionRate(item) > 0 && conversionRate(item) < 15) return '优化官方跳转按钮和评价文案'
  if (!item.reviewCrawl?.externalShopId) return '补充商家 ID，便于评论采集和 C 验证'
  return '持续观察使用率和发布转化'
}

function crawlStatusText(status?: string) {
  const map: Record<string, string> = {
    never_run: '未采集',
    running: '采集中',
    success: '已采集',
    failed: '采集失败'
  }
  return map[status || ''] || status || '未配置'
}

function crawlConfigText(item: AdminStore) {
  const cfg = item.reviewCrawl
  if (!cfg || !cfg.externalShopId) return '未配置'
  const enabledText = cfg.enabled ? '启用' : '未启用'
  return `${platformDisplayName(cfg.platformCode)} ${cfg.externalShopId} · ${enabledText}`
}

function statusToneClass(tone: string) {
  return {
    danger: tone === 'danger',
    warn: tone === 'warn',
    stable: tone === 'stable',
    neutral: tone === 'neutral'
  }
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString('zh-CN', { hour12: false })
}

function shortDateTime(value?: string) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}

function normalizeAbsoluteUrl(value: string) {
  const trimmed = (value || '').trim()
  if (!trimmed) return ''
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  if (trimmed.startsWith('/')) return `${location.origin}${trimmed}`
  return `${location.origin}/${trimmed.replace(/^\/+/, '')}`
}

function storeLandingUrl(item: any) {
  return normalizeAbsoluteUrl(item.landingUrl || `${import.meta.env.BASE_URL}landing/${item.uuid}`)
}

function merchantForStore(item: any) {
  return merchants.value.find((merchant) => merchant.id === item.merchantUserId) || {}
}

function syncStoreIdConfig(item: AdminStore | null) {
  selectedStoreIdConfig.storeId = item?.id || 0
  selectedStoreIdConfig.platformCode = item?.reviewCrawl?.platformCode || 'meituan'
  selectedStoreIdConfig.externalShopId = item?.reviewCrawl?.externalShopId || ''
  selectedStoreIdConfig.enabled = !!item?.reviewCrawl?.enabled
}

function resetStoreForm() {
  editingStoreId.value = null
  newStore.account = ''
  newStore.password = ''
  newStore.merchantName = ''
  newStore.contactName = ''
  newStore.typeId = storeTypes.value[0]?.id || 0
  newStore.storeName = ''
  newStore.address = ''
  newStore.storeIntro = ''
  newStore.primaryPlatformStyle = 'dianping'
  newStore.brandTone = ''
  newStore.platformUrl = ''
  newStore.reviewCrawlPlatformCode = 'meituan'
  newStore.reviewCrawlExternalShopId = ''
  newStore.reviewCrawlEnabled = false
}

function openCreateStore() {
  resetStoreForm()
  lastCreated.value = null
  configPanelOpen.value = true
  activeDetailTab.value = 'config'
}

function closeConfigPanel() {
  configPanelOpen.value = false
  resetStoreForm()
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [merchantRes, typeRes, storeRes, taskRes, statsRes] = await Promise.all([
      adminApi.listMerchants(),
      adminApi.listStoreTypes(),
      adminApi.listStores(),
      adminApi.listTasks(),
      adminApi.getStats()
    ])
    merchants.value = merchantRes.data.data
    storeTypes.value = typeRes.data.data
    stores.value = storeRes.data.data
    tasks.value = taskRes.data.data
    stats.value = statsRes.data.data
    if (!newStore.typeId) newStore.typeId = storeTypes.value[0]?.id || 0
    if (!newType.industryCode) newType.industryCode = presetTypes.value[0]?.code || ''
    if (stores.value.length && !stores.value.some((item) => item.id === selectedStoreId.value)) {
      selectedStoreId.value = stores.value[0].id
    }
    if (!stores.value.length) {
      selectedStoreId.value = null
    }
    syncStoreIdConfig(selectedStore.value)
  } catch (err: any) {
    error.value = messageFrom(err, '后台数据加载失败')
  } finally {
    loading.value = false
  }
}

async function runAction(action: () => Promise<unknown>, success: string) {
  error.value = ''
  notice.value = ''
  try {
    await action()
    notice.value = success
    await loadAll()
  } catch (err: any) {
    error.value = messageFrom(err, '操作失败')
  }
}

async function createStoreType() {
  if (!newType.name.trim() || !newType.industryCode) {
    error.value = '请填写类型名称并选择行业基准'
    return
  }
  await runAction(() => adminApi.createStoreType({ name: newType.name.trim(), industryCode: newType.industryCode }), '类型已创建')
  newType.name = ''
}

function editStore(item: any) {
  const merchant = merchantForStore(item)
  selectedStoreId.value = item.id
  editingStoreId.value = item.id
  configPanelOpen.value = true
  activeDetailTab.value = 'config'
  lastCreated.value = null
  newStore.account = item.merchantAccount || merchant.account || ''
  newStore.password = ''
  newStore.merchantName = item.merchantName || merchant.merchantName || item.storeName || ''
  newStore.contactName = item.contactName || merchant.contactName || ''
  newStore.typeId = item.typeId || 0
  newStore.storeName = item.storeName || ''
  newStore.address = item.address || ''
  newStore.storeIntro = item.storeIntro || ''
  newStore.primaryPlatformStyle = item.primaryPlatformStyle || 'dianping'
  newStore.brandTone = item.brandTone || ''
  newStore.platformUrl = item.platformUrl || ''
  newStore.reviewCrawlPlatformCode = item.reviewCrawl?.platformCode || 'meituan'
  newStore.reviewCrawlExternalShopId = item.reviewCrawl?.externalShopId || ''
  newStore.reviewCrawlEnabled = !!item.reviewCrawl?.enabled
}

async function saveStoreForm() {
  const account = newStore.account.trim()
  const password = newStore.password.trim()
  const storeName = newStore.storeName.trim()
  if (!account || !storeName || !newStore.typeId || (!isEditingStore.value && !password)) {
    error.value = isEditingStore.value ? '登录账号、门店名、类型为必填' : '登录账号、密码、门店名、类型为必填'
    return
  }

  const payload = {
    account,
    password,
    merchantName: newStore.merchantName.trim() || undefined,
    contactName: newStore.contactName.trim() || undefined,
    typeId: newStore.typeId,
    storeName,
    address: newStore.address.trim() || undefined,
    storeIntro: newStore.storeIntro.trim() || undefined,
    primaryPlatformStyle: newStore.primaryPlatformStyle,
    brandTone: newStore.brandTone.trim() || undefined,
    platformUrl: newStore.platformUrl.trim() || undefined,
    reviewCrawlPlatformCode: newStore.reviewCrawlPlatformCode,
    reviewCrawlExternalShopId: newStore.reviewCrawlExternalShopId.trim() || undefined,
    reviewCrawlEnabled: newStore.reviewCrawlEnabled
  }

  error.value = ''
  notice.value = ''
  try {
    if (isEditingStore.value) {
      await adminApi.updateStore(editingStoreId.value as number, {
        ...payload,
        password: password || undefined
      })
      notice.value = '门店资料已保存'
      resetStoreForm()
      configPanelOpen.value = false
      await loadAll()
      return
    }

    const { data } = await adminApi.createStore(payload)
    const created = data.data
    selectedStoreId.value = created.store.id
    lastCreated.value = {
      storeName: created.store.storeName,
      uuid: created.store.uuid,
      landingUrl: normalizeAbsoluteUrl(created.landingUrl || `${import.meta.env.BASE_URL}landing/${created.store.uuid}`),
      account: created.merchant.account
    }
    notice.value = '门店已创建'
    resetStoreForm()
    configPanelOpen.value = false
    await loadAll()
  } catch (err: any) {
    error.value = messageFrom(err, isEditingStore.value ? '保存失败' : '创建失败')
  }
}

async function toggleMerchantStatus(item: any) {
  await runAction(() => adminApi.updateMerchantStatus(item.id, item.status === 1 ? 0 : 1), '商家状态已更新')
}

async function deleteMerchant(item: any) {
  if (!window.confirm(`确认删除商家「${item.merchantName || item.account}」？关联门店、评价、图片、平台入口和生成任务会一起删除，NFC 物料会解绑保留。`)) return
  await runAction(() => adminApi.deleteMerchant(item.id), '商家已删除')
}

async function toggleStoreStatus(item: any) {
  await runAction(() => adminApi.updateStoreStatus(item.id, item.status === 1 ? 0 : 1), '门店状态已更新')
}

async function deleteStore(item: any) {
  if (!window.confirm(`确认删除门店「${item.storeName}」？关联商家账号、评价、图片、平台入口和生成任务会一起删除，NFC 物料会解绑保留。`)) return
  await runAction(() => adminApi.deleteStore(item.id), '门店已删除')
  if (editingStoreId.value === item.id) {
    resetStoreForm()
    configPanelOpen.value = false
  }
  if (selectedStoreId.value === item.id) selectedStoreId.value = null
}

async function runStoreReviewCrawl(item: AdminStore) {
  selectedStoreId.value = item.id
  await runAction(() => adminApi.runStoreReviewCrawl(item.id), '评论采集已同步')
  if (crawlPanelStore.value?.id === item.id) {
    await loadReviewCrawl(item)
  }
}

async function loadReviewCrawl(item: AdminStore) {
  selectedStoreId.value = item.id
  syncStoreIdConfig(item)
  crawlLoading.value = true
  error.value = ''
  try {
    const [batchRes, matchRes] = await Promise.all([
      adminApi.listStoreReviewCrawlBatches(item.id),
      adminApi.listStoreReviewCrawlMatches(item.id)
    ])
    crawlPanelStore.value = item
    crawlBatches.value = batchRes.data.data
    crawlMatches.value = matchRes.data.data
  } catch (err: any) {
    error.value = messageFrom(err, '评论采集数据加载失败')
  } finally {
    crawlLoading.value = false
  }
}

function selectStore(item: AdminStore) {
  selectedStoreId.value = item.id
  syncStoreIdConfig(item)
}

function clearStoreFilters() {
  storeSearch.value = ''
  storeStatusFilter.value = 'all'
}

async function applyWorkFilter(filter: StoreStatusFilter) {
  storeSearch.value = ''
  storeStatusFilter.value = filter
  await nextTick()
  selectedStoreId.value = filteredStores.value[0]?.id || null
}

async function saveSelectedStoreIdConfig() {
  const item = selectedStore.value
  if (!item) return
  const externalShopId = selectedStoreIdConfig.externalShopId.trim()
  if (selectedStoreIdConfig.enabled && !externalShopId) {
    error.value = '启用评论采集需要填写商家 ID / 外部店铺 ID'
    return
  }
  const merchant = merchantForStore(item)
  const account = item.merchantAccount || merchant.account || ''
  if (!account || !item.storeName || !item.typeId) {
    error.value = '当前商家资料不完整，无法保存商家 ID'
    return
  }

  error.value = ''
  notice.value = ''
  try {
    await adminApi.updateStore(item.id, {
      account,
      merchantName: item.merchantName || merchant.merchantName || item.storeName,
      contactName: item.contactName || merchant.contactName || undefined,
      typeId: item.typeId,
      storeName: item.storeName,
      address: item.address || undefined,
      storeIntro: item.storeIntro || undefined,
      primaryPlatformStyle: item.primaryPlatformStyle || 'dianping',
      brandTone: item.brandTone || undefined,
      platformUrl: item.platformUrl || undefined,
      reviewCrawlPlatformCode: selectedStoreIdConfig.platformCode,
      reviewCrawlExternalShopId: externalShopId || undefined,
      reviewCrawlEnabled: selectedStoreIdConfig.enabled
    })
    notice.value = '商家 ID 配置已保存'
    await loadAll()
  } catch (err: any) {
    error.value = messageFrom(err, '商家 ID 保存失败')
  }
}

async function copyText(text: string) {
  if (await copyToClipboard(text)) {
    notice.value = '已复制到剪贴板'
  } else {
    error.value = '复制失败，请手动选中复制：' + text
  }
}

function numericStatusText(status: number) {
  return status === 1 ? '已激活' : '未激活'
}

function logout() {
  auth.clear()
  location.href = import.meta.env.BASE_URL + 'admin/login'
}

onMounted(async () => {
  await nextTick()
  startOpsChartObserver()
  await loadAll()
})

onBeforeUnmount(() => {
  opsResizeObserver?.disconnect()
})
</script>

<template>
  <div class="admin-console">
    <aside class="ops-sidebar" aria-label="商家运营导航">
      <div class="brand-block">
        <strong>商家运营控制台</strong>
        <span>NFC 评价助手 · 管理员</span>
      </div>
      <nav class="side-nav">
        <a v-for="item in navItems" :key="item.href" :href="item.href">
          <i aria-hidden="true"></i>
          <span>{{ item.label }}</span>
        </a>
      </nav>
      <div class="side-summary">
        <span>核心状态</span>
        <strong>{{ formatNumber(activeStores.length) }} / {{ formatNumber(stats.storeCount) }}</strong>
        <small>已激活商家</small>
      </div>
    </aside>

    <main class="ops-main">
      <header class="ops-topbar">
        <div>
          <p class="eyebrow">Admin Workspace</p>
          <h1>商家运营控制台</h1>
          <p class="muted">激活状态、官方链接配置和 NFC 路由可用性独立管理。</p>
        </div>
        <div class="topbar-actions">
          <button class="secondary" :disabled="loading" @click="loadAll">刷新</button>
          <button type="button" @click="openCreateStore">新建商家</button>
          <button class="secondary" @click="logout">退出登录</button>
        </div>
      </header>

      <p v-if="error" class="alert">{{ error }}</p>
      <p v-else-if="notice" class="notice">{{ notice }}</p>
      <span id="config-drawer-anchor" class="sr-only">配置资料</span>

      <section id="ops-overview" class="screen-card overview-screen" aria-labelledby="overview-title">
        <div class="screen-head">
          <div>
            <h2 id="overview-title">商家运营总览</h2>
            <p class="muted">{{ updatedText || '等待数据更新' }}</p>
            <p class="data-source">来源：{{ analyticsSourceText }}</p>
          </div>
          <strong :class="['workload-pill', pendingWorkCount > 0 ? 'warn' : 'stable']">
            待处理 {{ formatNumber(pendingWorkCount) }}
          </strong>
        </div>

        <div class="kpi-grid" aria-label="总览指标">
          <div class="kpi">
            <span>总商家</span>
            <strong>{{ formatNumber(stats.storeCount) }}</strong>
            <small>本周新增 {{ formatNumber(stats.currentWeekNewMerchants) }}</small>
          </div>
          <div class="kpi">
            <span>已激活</span>
            <strong>{{ formatNumber(activeStores.length) }}</strong>
            <small>未激活 {{ formatNumber(inactiveStoreCount) }}</small>
          </div>
          <div class="kpi">
            <span>官方链接已配置</span>
            <strong>{{ formatNumber(officialConfiguredCount) }}</strong>
            <small>未配置 {{ formatNumber(officialMissingCount) }}</small>
          </div>
          <div class="kpi">
            <span>NFC 路由可用</span>
            <strong>{{ formatNumber(nfcRouteUsableCount) }}</strong>
            <small>异常 {{ formatNumber(nfcRouteErrorCount) }}</small>
          </div>
          <div class="kpi">
            <span>NFC 访问</span>
            <strong>{{ formatNumber(stats.totalCustomerVisits) }}</strong>
            <small>本月 {{ formatNumber(stats.currentMonthCustomerVisits) }}</small>
          </div>
          <div class="kpi">
            <span>平均发布转化率</span>
            <strong>{{ stats.totalCustomerVisits ? formatPercent((stats.totalPublishClicks / stats.totalCustomerVisits) * 100) : '-' }}</strong>
            <small>发布点击 / NFC 访问</small>
          </div>
        </div>

        <div class="overview-grid">
          <section class="ops-panel" aria-labelledby="risk-title">
            <div class="panel-head">
              <div>
                <h3 id="risk-title">风险队列</h3>
                <p class="muted">按优先级处理</p>
              </div>
              <strong>{{ formatNumber(pendingWorkCount) }}</strong>
            </div>
            <div class="work-list">
              <button
                v-for="item in workItems"
                :key="item.label"
                type="button"
                :class="['work-item', statusToneClass(item.tone)]"
                :disabled="item.count === 0"
                @click="applyWorkFilter(item.filter)"
              >
                <span>{{ item.label }}</span>
                <b>{{ formatNumber(item.count) }}</b>
              </button>
            </div>
          </section>

          <section class="ops-panel" aria-labelledby="trend-title">
            <div class="panel-head">
              <div>
                <h3 id="trend-title">转化链路</h3>
                <p class="muted">NFC 访问、官方点击、发布转化</p>
              </div>
            </div>
            <div class="mini-funnel">
              <div>
                <span>NFC 访问</span>
                <strong>{{ formatNumber(stats.totalCustomerVisits) }}</strong>
              </div>
              <div>
                <span>官方点击</span>
                <strong>{{ formatNumber(stats.totalPublishClicks) }}</strong>
              </div>
              <div>
                <span>发布转化</span>
                <strong>{{ stats.totalCustomerVisits ? formatPercent((stats.totalPublishClicks / stats.totalCustomerVisits) * 100) : '-' }}</strong>
              </div>
            </div>
            <div class="ops-line-chart" @pointerleave="hoveredOpsChartIndex = null">
              <div class="ops-chart-legend" aria-hidden="true">
                <span><i class="visit"></i>NFC 访问</span>
                <span><i class="click"></i>官方点击</span>
              </div>
              <svg ref="opsChartSvg" class="ops-line-chart-svg" :viewBox="opsChartViewBox" role="img" :aria-label="opsChartAria">
                <defs>
                  <linearGradient id="adminOpsVisitArea" x1="0%" y1="0%" x2="0%" y2="100%">
                    <stop offset="0%" stop-color="#3b82f6" stop-opacity="0.2" />
                    <stop offset="76%" stop-color="#3b82f6" stop-opacity="0.04" />
                    <stop offset="100%" stop-color="#3b82f6" stop-opacity="0" />
                  </linearGradient>
                </defs>
                <g>
                  <g v-for="tick in opsChartScale.ticks" :key="tick.value">
                    <line class="ops-chart-grid-line" :x1="opsChartMetrics.plotLeft" :x2="opsChartMetrics.plotRight" :y1="tick.y" :y2="tick.y" />
                    <line class="ops-chart-y-tick" :x1="opsChartMetrics.plotLeft - 6" :x2="opsChartMetrics.plotLeft" :y1="tick.y" :y2="tick.y" />
                    <text class="ops-chart-axis-text ops-chart-y-axis-text" :x="opsChartMetrics.plotLeft - 12" :y="tick.y + 4" text-anchor="end">{{ formatNumber(tick.value) }}</text>
                  </g>
                </g>
                <polygon v-if="opsAreaPolygon" class="ops-chart-area" :points="opsAreaPolygon" />
                <polyline v-if="opsVisitPoints.length" class="ops-chart-line visit" :points="opsVisitPolyline" fill="none" />
                <polyline v-if="opsClickPoints.length" class="ops-chart-line click" :points="opsClickPolyline" fill="none" />
                <g
                  v-for="(point, index) in opsVisitPoints"
                  :key="point.label"
                  class="ops-chart-point-group"
                  tabindex="0"
                  focusable="true"
                  @pointerenter="hoveredOpsChartIndex = index"
                  @focus="hoveredOpsChartIndex = index"
                  @blur="hoveredOpsChartIndex = null"
                >
                  <rect class="ops-chart-hit" :x="point.x - 22" :y="opsChartMetrics.plotTop" width="44" :height="opsChartMetrics.plotHeight" />
                  <line
                    v-if="hoveredOpsChartIndex === index"
                    class="ops-chart-hover-line"
                    :x1="point.x"
                    :x2="point.x"
                    :y1="opsChartMetrics.plotTop"
                    :y2="opsChartMetrics.plotBottom"
                  />
                  <circle :cx="point.x" :cy="point.y" :r="hoveredOpsChartIndex === index ? 5.8 : 4.4" class="ops-chart-dot visit" />
                  <circle
                    v-if="opsClickPoints[index]"
                    :cx="opsClickPoints[index].x"
                    :cy="opsClickPoints[index].y"
                    :r="hoveredOpsChartIndex === index ? 5.4 : 4"
                    class="ops-chart-dot click"
                  />
                </g>
                <g aria-hidden="true">
                  <text
                    v-for="point in opsVisitPoints"
                    :key="point.label"
                    class="ops-chart-axis-text"
                    :x="point.x"
                    :y="opsChartMetrics.height - 20"
                    :text-anchor="opsAxisAnchor(point.x)"
                  >
                    {{ shortStoreLabel(point.label) }}
                  </text>
                </g>
              </svg>
              <div v-if="activeOpsChartPoint" class="ops-chart-tooltip visible" :style="opsTooltipStyle">
                <strong>{{ activeOpsChartPoint.label }}</strong>
                <span>NFC 访问 {{ formatNumber(activeOpsChartPoint.visits) }}</span>
                <span>官方点击 {{ formatNumber(activeOpsChartPoint.clicks) }}</span>
              </div>
              <p v-if="!opsChartHasData" class="empty-note">暂无商家使用曲线数据。</p>
            </div>
          </section>

          <section class="ops-panel" aria-labelledby="rank-title">
            <div class="panel-head">
              <div>
                <h3 id="rank-title">商家访问排行</h3>
                <p class="muted">{{ deviceSummaryText }}</p>
              </div>
            </div>
            <ol v-if="topStores.length" class="store-rank">
              <li v-for="item in topStores" :key="item.id">
                <button type="button" @click="selectStore(item)">
                  <span>
                    <b>{{ item.storeName }}</b>
                    <small>{{ primaryDeviceText(item) }} · 转化 {{ conversionRateText(item) }}</small>
                  </span>
                  <strong>{{ formatNumber(item.analytics?.totalCustomerVisits) }}</strong>
                </button>
              </li>
            </ol>
            <p v-else class="empty-note">暂无门店数据。</p>
          </section>
        </div>
      </section>

      <section id="merchant-ledger" class="screen-card ledger-screen" aria-labelledby="ledger-title">
        <div class="screen-head ledger-head">
          <div>
            <h2 id="ledger-title">商家管理列表</h2>
            <p class="muted">共 {{ formatNumber(filteredStores.length) }} 条，当前选中 {{ selectedStore?.storeName || '-' }}</p>
          </div>
          <div class="ledger-tools">
            <input v-model="storeSearch" type="search" placeholder="搜索商家、账号、门店、商家 ID、官方链接" />
            <select v-model="storeStatusFilter" aria-label="商家状态筛选">
              <option value="all">全部</option>
              <option value="active">已激活</option>
              <option value="inactive">未激活</option>
              <option value="official_configured">官方链接已配置</option>
              <option value="official_missing">官方链接未配置</option>
              <option value="nfc_error">NFC 异常</option>
              <option value="no_visit">近 7 天无访问</option>
              <option value="low_conversion">访问高转化低</option>
              <option value="crawl_failed">采集失败</option>
              <option value="no_crawl">未启用采集</option>
            </select>
            <button class="secondary" type="button" @click="clearStoreFilters">重置</button>
          </div>
        </div>

        <div class="ledger-layout">
          <div class="ledger-table">
            <table class="desktop-table">
              <thead>
                <tr>
                  <th>商家名称</th>
                  <th>商家 ID</th>
                  <th>激活状态</th>
                  <th>官方链接</th>
                  <th>NFC 链接</th>
                  <th>NFC 访问</th>
                  <th>官方点击</th>
                  <th>转化率</th>
                  <th>最近使用</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in filteredStores" :key="item.id" :class="{ selected: selectedStore?.id === item.id }" @click="selectStore(item)">
                  <td>
                    <strong>{{ item.storeName }}</strong>
                    <span class="subtext">账号 {{ merchantAccountText(item) }} · 门店 ID {{ item.id }} · {{ typeName(item.typeId) }}</span>
                  </td>
                  <td>
                    <strong>{{ merchantIdText(item) }}</strong>
                    <span class="subtext">外部平台店铺 ID</span>
                  </td>
                  <td><span :class="['status-pill', isStoreActive(item) ? 'enabled' : 'disabled']">{{ activationStatusText(item) }}</span></td>
                  <td>
                    <span :class="['status-pill', officialLinkStatusClass(item)]">{{ officialLinkStatusText(item) }}</span>
                    <span class="subtext">{{ platformDisplayName(item.primaryPlatformStyle) }}</span>
                  </td>
                  <td>
                    <span :class="['status-pill', nfcRouteStatusClass(item)]">{{ nfcRouteStatusText(item) }}</span>
                    <span class="subtext">{{ nfcRouteStatusDetail(item) }}</span>
                  </td>
                  <td><strong>{{ formatNumber(item.analytics?.totalCustomerVisits) }}</strong><span class="subtext">本月 {{ formatNumber(item.analytics?.currentMonthCustomerVisits) }}</span></td>
                  <td>{{ formatNumber(item.analytics?.totalPublishClicks) }}</td>
                  <td><strong>{{ conversionRateText(item) }}</strong></td>
                  <td>{{ shortDateTime(item.updatedAt || item.createdAt) }}</td>
                  <td class="actions-cell">
                    <span class="table-actions">
                      <button class="secondary" :disabled="loading" @click.stop="editStore(item)">编辑</button>
                      <button class="secondary" :disabled="loading" @click.stop="loadReviewCrawl(item)">采集</button>
                      <button class="secondary" :disabled="loading" @click.stop="toggleStoreStatus(item)">
                        {{ isStoreActive(item) ? '停用' : '激活' }}
                      </button>
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
            <p v-if="!filteredStores.length" class="empty-note ledger-empty">没有匹配的商家。</p>

            <div class="mobile-store-list" aria-label="门店列表">
              <article v-for="item in filteredStores" :key="item.id" class="mobile-store-item" @click="selectStore(item)">
                <div class="mobile-store-head">
                  <div>
                    <strong>{{ item.storeName }}</strong>
                    <span>账号 {{ merchantAccountText(item) }} · 商家 ID {{ merchantIdText(item) }}</span>
                  </div>
                  <b :class="['status-pill', isStoreActive(item) ? 'enabled' : 'disabled']">{{ activationStatusText(item) }}</b>
                </div>
                <dl class="mobile-store-meta">
                  <div>
                    <dt>官方链接</dt>
                    <dd><span :class="['status-pill', officialLinkStatusClass(item)]">{{ officialLinkStatusText(item) }}</span></dd>
                  </div>
                  <div>
                    <dt>NFC 链接</dt>
                    <dd>
                      <span :class="['status-pill', nfcRouteStatusClass(item)]">{{ nfcRouteStatusText(item) }}</span>
                      <small>{{ nfcRouteStatusDetail(item) }}</small>
                    </dd>
                  </div>
                  <div>
                    <dt>NFC 访问</dt>
                    <dd>{{ formatNumber(item.analytics?.totalCustomerVisits) }}</dd>
                  </div>
                  <div>
                    <dt>官方点击</dt>
                    <dd>{{ formatNumber(item.analytics?.totalPublishClicks) }}</dd>
                  </div>
                </dl>
                <div class="mobile-store-actions">
                  <button class="secondary" :disabled="loading" @click.stop="editStore(item)">编辑</button>
                  <button class="secondary" :disabled="loading" @click.stop="copyText(storeLandingUrl(item))">复制服务器链接</button>
                  <button class="secondary" :disabled="loading" @click.stop="toggleStoreStatus(item)">
                    {{ isStoreActive(item) ? '停用' : '激活' }}
                  </button>
                </div>
              </article>
            </div>
          </div>

          <aside v-if="selectedStore" id="merchant-detail" class="detail-aside" aria-label="当前选中商家摘要">
            <div class="detail-head">
              <div>
                <p class="eyebrow">当前选中商家</p>
                <h3>{{ selectedStore.storeName }}</h3>
                <span>账号 {{ merchantAccountText(selectedStore) }} · 门店 ID {{ selectedStore.id }}</span>
              </div>
              <b :class="['status-pill', isStoreActive(selectedStore) ? 'enabled' : 'disabled']">{{ activationStatusText(selectedStore) }}</b>
            </div>

            <dl class="status-grid">
              <div>
                <dt>官方链接</dt>
                <dd><span :class="['status-pill', officialLinkStatusClass(selectedStore)]">{{ officialLinkStatusText(selectedStore) }}</span></dd>
              </div>
              <div>
                <dt>NFC 链接</dt>
                <dd>
                  <span :class="['status-pill', nfcRouteStatusClass(selectedStore)]">{{ nfcRouteStatusText(selectedStore) }}</span>
                  <small>{{ nfcRouteStatusDetail(selectedStore) }}</small>
                </dd>
              </div>
              <div>
                <dt>商家 ID</dt>
                <dd>{{ merchantIdText(selectedStore) }}</dd>
              </div>
            </dl>

            <section class="merchant-id-panel" aria-labelledby="merchant-id-panel-title">
              <div class="merchant-id-head">
                <div>
                  <h4 id="merchant-id-panel-title">商家 ID 配置</h4>
                  <p class="muted">主页面直接配置，不放入弹窗。</p>
                </div>
                <span :class="['status-pill', merchantIdText(selectedStore) === '-' ? 'missing' : 'ready']">
                  {{ merchantIdText(selectedStore) === '-' ? '未配置' : '已配置' }}
                </span>
              </div>
              <div class="merchant-id-form">
                <label class="fld">商家 ID 平台
                  <select v-model="selectedStoreIdConfig.platformCode">
                    <option value="meituan">美团</option>
                  </select>
                </label>
                <label class="fld">商家 ID / 外部店铺 ID
                  <input v-model="selectedStoreIdConfig.externalShopId" placeholder="如 1953748828" inputmode="numeric" />
                </label>
                <label class="check-field">
                  <input v-model="selectedStoreIdConfig.enabled" type="checkbox" />
                  <span>启用每 7 天评论采集</span>
                </label>
                <button type="button" :disabled="loading" @click="saveSelectedStoreIdConfig">保存商家 ID</button>
              </div>
            </section>

            <div class="detail-metrics">
              <div>
                <span>NFC 访问</span>
                <strong>{{ formatNumber(selectedStore.analytics?.totalCustomerVisits) }}</strong>
                <small>本月 {{ formatNumber(selectedStore.analytics?.currentMonthCustomerVisits) }}</small>
              </div>
              <div>
                <span>官方点击</span>
                <strong>{{ formatNumber(selectedStore.analytics?.totalPublishClicks) }}</strong>
                <small>本周 {{ formatNumber(selectedStore.analytics?.currentWeekPublishClicks) }}</small>
              </div>
              <div>
                <span>发布转化</span>
                <strong>{{ conversionRateText(selectedStore) }}</strong>
                <small>{{ platformDisplayName(selectedStore.primaryPlatformStyle) }}</small>
              </div>
            </div>

            <nav class="detail-tabs" aria-label="商家详情标签">
              <button
                v-for="tab in detailTabs"
                :key="tab.key"
                type="button"
                :class="{ active: activeDetailTab === tab.key }"
                :aria-pressed="activeDetailTab === tab.key"
                @click="activeDetailTab = tab.key"
              >
                {{ tab.label }}
              </button>
            </nav>

            <div class="detail-tab-panel">
              <template v-if="activeDetailTab === 'overview'">
                <div class="detail-block">
                  <h4>服务器落地链接 / NFC 写入链接</h4>
                  <p class="detail-url">{{ storeLandingUrl(selectedStore) }}</p>
                  <button class="secondary" type="button" @click="copyText(storeLandingUrl(selectedStore))">复制服务器链接</button>
                </div>
                <div class="detail-block">
                  <h4>商家官方链接</h4>
                  <p class="detail-url">{{ selectedStore.platformUrl || '未配置' }}</p>
                </div>
              </template>

              <template v-else-if="activeDetailTab === 'usage'">
                <div class="usage-list">
                  <div><span>近 7 天访问</span><strong>{{ formatNumber(selectedStore.analytics?.currentWeekCustomerVisits) }}</strong></div>
                  <div><span>近 30 天访问</span><strong>{{ formatNumber(selectedStore.analytics?.currentMonthCustomerVisits) }}</strong></div>
                  <div><span>主力设备</span><strong>{{ primaryDeviceText(selectedStore) }}</strong></div>
                </div>
              </template>

              <template v-else-if="activeDetailTab === 'conversion'">
                <div class="conversion-funnel">
                  <div><span>NFC 访问</span><strong>{{ formatNumber(selectedStore.analytics?.totalCustomerVisits) }}</strong></div>
                  <div><span>官方点击</span><strong>{{ formatNumber(selectedStore.analytics?.totalPublishClicks) }}</strong></div>
                  <div><span>发布转化率</span><strong>{{ conversionRateText(selectedStore) }}</strong></div>
                </div>
              </template>

              <template v-else-if="activeDetailTab === 'advice'">
                <div id="ops-advice" class="advice-card">
                  <span>主要建议</span>
                  <strong>{{ primaryRecommendation(selectedStore) }}</strong>
                </div>
              </template>

              <template v-else>
                <div class="detail-block">
                  <h4>配置资料</h4>
                  <dl class="config-summary">
                    <div><dt>商家名称</dt><dd>{{ selectedStore.merchantName || selectedMerchant.merchantName || selectedStore.storeName }}</dd></div>
                    <div><dt>商家 ID</dt><dd>{{ merchantIdText(selectedStore) }}</dd></div>
                    <div><dt>评论采集</dt><dd>{{ crawlConfigText(selectedStore) }}</dd></div>
                    <div><dt>官方链接</dt><dd>{{ selectedStore.platformUrl || '未配置' }}</dd></div>
                  </dl>
                </div>
              </template>
            </div>

            <div class="detail-actions">
              <button class="secondary" type="button" :disabled="loading" @click="editStore(selectedStore)">编辑配置</button>
              <button class="secondary" type="button" :disabled="loading || !selectedStore.reviewCrawl?.enabled" @click="runStoreReviewCrawl(selectedStore)">同步评论</button>
              <button class="secondary" type="button" :disabled="loading" @click="loadReviewCrawl(selectedStore)">采集明细</button>
              <button type="button" :disabled="loading" @click="toggleStoreStatus(selectedStore)">
                {{ isStoreActive(selectedStore) ? '停用商家' : '激活商家' }}
              </button>
            </div>
          </aside>
        </div>

        <section v-if="crawlPanelStore" class="crawl-panel" aria-labelledby="crawl-panel-title">
          <div class="panel-head">
            <div>
              <h3 id="crawl-panel-title">{{ crawlPanelStore.storeName }} 评论采集</h3>
              <p class="muted">批次数据和 C 验证明细仅管理员可见。</p>
            </div>
            <button class="secondary" :disabled="crawlLoading" @click="crawlPanelStore = null">关闭</button>
          </div>
          <p v-if="crawlLoading" class="empty-note">加载中...</p>
          <div v-else class="crawl-tables">
            <div>
              <h4>最近批次</h4>
              <table>
                <thead><tr><th>批次</th><th>状态</th><th>类型</th><th>入库</th><th>C 验证</th><th>完成时间</th></tr></thead>
                <tbody>
                  <tr v-for="batch in crawlBatches" :key="batch.id">
                    <td>#{{ batch.id }}</td>
                    <td>{{ crawlStatusText(batch.status) }}</td>
                    <td>{{ batch.isBaseline ? '基线' : '周期' }}</td>
                    <td>{{ formatNumber(batch.insertedRowCount) }}</td>
                    <td>{{ formatNumber(batch.matchedReviewCount) }}</td>
                    <td>{{ formatDateTime(batch.finishedAt) }}</td>
                  </tr>
                </tbody>
              </table>
              <p v-if="!crawlBatches.length" class="empty-note">暂无采集批次。</p>
            </div>
            <div>
              <h4>C 验证明细</h4>
              <table>
                <thead><tr><th>评论时间</th><th>用户</th><th>匹配分</th><th>内容</th></tr></thead>
                <tbody>
                  <tr v-for="match in crawlMatches" :key="match.id">
                    <td>{{ formatDateTime(match.reviewTime) }}</td>
                    <td>{{ match.userName || '-' }}</td>
                    <td>{{ formatPercent((match.matchScore || 0) * 100) }}</td>
                    <td class="match-content">{{ match.content || '-' }}</td>
                  </tr>
                </tbody>
              </table>
              <p v-if="!crawlMatches.length" class="empty-note">暂无 C 验证明细。</p>
            </div>
          </div>
        </section>
      </section>

      <section class="fold-grid" aria-label="低频管理">
        <details class="fold-card">
          <summary>
            <span>
              <strong>类型标签管理</strong>
              <small>推荐标签与行业隔离基准</small>
            </span>
            <span class="fold-hint">展开</span>
          </summary>
          <div class="fold-body">
            <div class="inline-form">
              <input v-model="newType.name" placeholder="自定义类型名称，如 苍蝇馆子" />
              <select v-model="newType.industryCode">
                <option v-for="t in presetTypes" :key="t.code" :value="t.code">基准：{{ t.name }}</option>
              </select>
              <button :disabled="loading" @click="createStoreType">新增类型</button>
            </div>
            <table>
              <thead><tr><th>类型</th><th>行业基准</th><th>来源</th></tr></thead>
              <tbody>
                <tr v-for="t in storeTypes" :key="t.id">
                  <td>{{ t.name }}</td>
                  <td>{{ t.industryCode }}</td>
                  <td>{{ t.isPreset ? '预置' : '自定义' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </details>

        <details class="fold-card">
          <summary>
            <span>
              <strong>商家账号</strong>
              <small>账号启停和删除</small>
            </span>
            <span class="fold-hint">展开</span>
          </summary>
          <div class="fold-body">
            <table>
              <thead><tr><th>ID</th><th>名称</th><th>账号</th><th>状态</th><th>操作</th></tr></thead>
              <tbody>
                <tr v-for="item in merchants" :key="item.id">
                  <td>{{ item.id }}</td>
                  <td>{{ item.merchantName }}</td>
                  <td>{{ item.account }}</td>
                  <td>{{ numericStatusText(item.status) }}</td>
                  <td>
                    <span class="table-actions compact">
                      <button class="secondary" :disabled="loading" @click="toggleMerchantStatus(item)">
                        {{ item.status === 1 ? '禁用' : '启用' }}
                      </button>
                      <button class="danger" :disabled="loading" @click="deleteMerchant(item)">删除</button>
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </details>

        <details class="fold-card">
          <summary>
            <span>
              <strong>生成任务</strong>
              <small>评价生成记录、失败原因和 agent 调用日志</small>
            </span>
            <span class="fold-hint">展开</span>
          </summary>
          <div class="fold-body">
            <table>
              <thead><tr><th>ID</th><th>门店</th><th>平台</th><th>类型</th><th>状态</th><th>入池</th><th>失败原因</th><th>最近日志</th></tr></thead>
              <tbody>
                <tr v-for="item in tasks" :key="item.id">
                  <td>{{ item.id }}</td>
                  <td>{{ item.storeId }}</td>
                  <td>{{ platformDisplayName(item.platformStyle) }}</td>
                  <td>{{ item.triggerType }}</td>
                  <td>
                    <strong>{{ item.status }}</strong>
                    <span class="subtext">目标 {{ formatNumber(item.targetCount) }}</span>
                  </td>
                  <td>
                    <strong>{{ formatNumber(item.successCount) }}</strong>
                    <span class="subtext">原始 {{ formatNumber(item.generatedRawCount) }} / 过滤 {{ formatNumber(item.duplicateFilteredCount) }}</span>
                  </td>
                  <td class="audit-reason">{{ generationFailureReason(item) }}</td>
                  <td class="audit-cell">
                    <span class="subtext">{{ generationStageSummary(item) }}</span>
                    <span v-for="log in generationLogPreview(item)" :key="log.id" :class="['audit-line', log.level]">
                      {{ formatDateTime(log.createdAt) }} · {{ log.stage }} · {{ log.message }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </details>
      </section>
    </main>

    <div v-if="configPanelOpen" class="drawer-backdrop" role="presentation" @click.self="closeConfigPanel">
      <aside class="config-drawer" role="dialog" aria-modal="true" aria-labelledby="config-panel-title">
        <div class="drawer-head">
          <div>
            <p class="eyebrow">配置资料</p>
            <h2 id="config-panel-title">{{ isEditingStore ? '编辑商家配置' : '新建商家' }}</h2>
          </div>
          <button class="secondary" type="button" @click="closeConfigPanel">关闭</button>
        </div>

        <div class="config-form-grid">
          <section class="config-section" aria-labelledby="account-config-title">
            <h3 id="account-config-title">商家账号</h3>
            <div class="form-grid compact">
              <label class="fld">商家登录账号
                <input v-model="newStore.account" placeholder="商家登录用，如 laozhang001" />
              </label>
              <label class="fld">{{ isEditingStore ? '商家登录密码（留空不修改）' : '商家登录密码' }}
                <input v-model="newStore.password" type="password" placeholder="商家登录用" />
              </label>
              <label class="fld">商家名称
                <input v-model="newStore.merchantName" placeholder="默认同门店名" />
              </label>
              <label class="fld">联系人
                <input v-model="newStore.contactName" placeholder="选填" />
              </label>
            </div>
          </section>

          <section class="config-section" aria-labelledby="store-config-title">
            <h3 id="store-config-title">门店资料</h3>
            <div class="form-grid compact">
              <label class="fld">门店类型
                <select v-model.number="newStore.typeId">
                  <option v-for="t in storeTypes" :key="t.id" :value="t.id">{{ t.name }}</option>
                </select>
              </label>
              <label class="fld">门店名称
                <input v-model="newStore.storeName" placeholder="如 老张川菜馆" />
              </label>
              <label class="fld wide">门店地址
                <input v-model="newStore.address" placeholder="选填" />
              </label>
              <label class="fld">门店简介
                <input v-model="newStore.storeIntro" placeholder="选填，用于生成评价上下文" />
              </label>
              <label class="fld">品牌语气
                <input v-model="newStore.brandTone" placeholder="如 轻松自然、专业细致" />
              </label>
            </div>
          </section>

          <section class="config-section important" aria-labelledby="official-link-config-title">
            <h3 id="official-link-config-title">商家官方链接</h3>
            <div class="form-grid compact">
              <label class="fld">官方平台
                <select v-model="newStore.primaryPlatformStyle">
                  <option v-for="p in platformOptions" :key="p.code" :value="p.code">{{ p.name }}</option>
                </select>
              </label>
              <label class="fld wide">商家官方链接 URL
                <input v-model="newStore.platformUrl" placeholder="https://... 平台店铺、评价页或官方分享链接" />
              </label>
            </div>
          </section>

        </div>

        <div v-if="isEditingStore && selectedStore" class="delivery-preview" aria-label="服务器落地链接与 NFC 写入链接">
          <div>
            <span>服务器落地链接</span>
            <code>{{ storeLandingUrl(selectedStore) }}</code>
          </div>
          <div>
            <span>NFC 写入链接</span>
            <code>{{ storeLandingUrl(selectedStore) }}</code>
          </div>
          <button class="secondary" type="button" @click="copyText(storeLandingUrl(selectedStore))">复制服务器链接</button>
        </div>

        <div v-if="lastCreated" class="created" role="status">
          <p>已创建「{{ lastCreated.storeName }}」，商家账号：<b>{{ lastCreated.account }}</b></p>
          <p>UUID：<code>{{ lastCreated.uuid }}</code></p>
          <p class="url-line">
            <span>服务器落地链接：</span>
            <code>{{ lastCreated.landingUrl }}</code>
            <button class="secondary" @click="copyText(lastCreated.landingUrl)">复制</button>
          </p>
        </div>

        <div class="drawer-actions">
          <button :disabled="loading" @click="saveStoreForm">
            {{ isEditingStore ? '保存配置资料' : '创建商家并生成服务器链接' }}
          </button>
          <button class="secondary" type="button" :disabled="loading" @click="closeConfigPanel">取消</button>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.admin-console {
  background: #eef3f8;
  color: var(--text);
  display: grid;
  grid-template-columns: 248px minmax(0, 1fr);
  min-height: 100vh;
}

.ops-sidebar {
  background: #0f172a;
  color: #e5edf8;
  display: grid;
  grid-template-rows: auto 1fr auto;
  min-height: 100vh;
  padding: 22px 18px;
  position: sticky;
  top: 0;
}

.brand-block {
  border-bottom: 1px solid rgba(226, 232, 240, 0.16);
  padding-bottom: 20px;
}

.brand-block strong,
.brand-block span,
.side-summary span,
.side-summary small,
.subtext {
  display: block;
}

.brand-block strong {
  color: #fff;
  font-size: 18px;
}

.brand-block span,
.side-summary span,
.side-summary small {
  color: #94a3b8;
  font-size: 12px;
  margin-top: 4px;
}

.side-nav {
  display: grid;
  gap: 6px;
  align-content: start;
  padding: 22px 0;
}

.side-nav a {
  align-items: center;
  border-radius: 8px;
  color: #cbd5e1;
  display: grid;
  gap: 10px;
  grid-template-columns: 10px 1fr;
  min-height: 38px;
  padding: 8px 10px;
  text-decoration: none;
}

.side-nav a:first-child,
.side-nav a:hover {
  background: rgba(37, 99, 235, 0.18);
  color: #fff;
}

.side-nav i {
  background: currentColor;
  border-radius: 999px;
  height: 7px;
  width: 7px;
}

.side-summary {
  border: 1px solid rgba(226, 232, 240, 0.16);
  border-radius: 8px;
  padding: 14px;
}

.side-summary strong {
  color: #fff;
  display: block;
  font-size: 28px;
  line-height: 1;
  margin-top: 8px;
}

.ops-main {
  display: grid;
  gap: 16px;
  min-width: 0;
  padding: 24px;
}

.ops-topbar,
.screen-head,
.panel-head,
.detail-head,
.drawer-head {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.ops-topbar h1,
.screen-head h2,
.panel-head h3,
.detail-head h3,
.drawer-head h2 {
  margin: 0;
}

.ops-topbar .muted,
.screen-head .muted,
.panel-head .muted {
  margin-top: 4px;
}

.topbar-actions {
  display: flex;
  flex: 0 0 auto;
  gap: 8px;
}

.topbar-actions button,
.drawer-head button {
  min-height: 40px;
  padding: 8px 12px;
}

.screen-card,
.fold-card {
  background: var(--surface);
  border: 1px solid rgba(219, 228, 240, 0.86);
  border-radius: 8px;
  box-shadow: none;
  min-width: 0;
}

.screen-card {
  padding: 18px;
}

.muted,
.eyebrow,
.subtext,
.empty-note,
.data-source {
  color: var(--muted);
  font-size: 13px;
  margin: 0;
}

.audit-reason {
  max-width: 320px;
  white-space: normal;
}

.audit-cell {
  min-width: 280px;
}

.audit-line {
  color: var(--muted);
  display: block;
  font-size: 12px;
  line-height: 1.5;
  margin-top: 4px;
  white-space: normal;
}

.audit-line.error {
  color: #b91c1c;
}

.eyebrow {
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sr-only {
  height: 1px;
  margin: -1px;
  overflow: hidden;
  position: absolute;
  width: 1px;
}

.workload-pill,
.status-pill {
  border-radius: 999px;
  display: inline-flex;
  flex: 0 0 auto;
  font-size: 12px;
  font-weight: 800;
  line-height: 1;
  padding: 6px 9px;
}

.workload-pill.warn {
  background: #fffbeb;
  color: #92400e;
}

.workload-pill.stable,
.status-pill.enabled {
  background: var(--success-bg);
  color: var(--success-text);
}

.status-pill.disabled {
  background: #f1f5f9;
  color: #64748b;
}

.status-pill.ready {
  background: #eff6ff;
  color: #1d4ed8;
}

.status-pill.missing {
  background: #fff7ed;
  color: #c2410c;
}

.status-pill.error {
  background: #fef2f2;
  color: #b91c1c;
}

.kpi-grid {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  margin-top: 16px;
}

.kpi,
.ops-panel,
.detail-aside,
.detail-metrics div,
.detail-url,
.usage-list div,
.conversion-funnel div,
.advice-card,
.config-section {
  border: 1px solid var(--border-soft);
  border-radius: 8px;
}

.kpi {
  background: #fbfdff;
  min-width: 0;
  padding: 12px;
}

.kpi span,
.kpi small,
.detail-metrics span,
.detail-metrics small,
.usage-list span,
.conversion-funnel span,
.advice-card span,
.config-summary dt,
.status-grid dt,
.mobile-store-meta dt,
.delivery-preview span {
  color: var(--muted);
  display: block;
  font-size: 12px;
}

.kpi strong {
  display: block;
  font-size: 24px;
  line-height: 1.1;
  margin: 4px 0 6px;
}

.overview-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: minmax(250px, 0.9fr) minmax(300px, 1fr) minmax(300px, 1fr);
  margin-top: 14px;
}

.ops-panel {
  background: #fbfdff;
  padding: 14px;
}

.panel-head {
  margin-bottom: 12px;
}

.panel-head strong {
  font-size: 24px;
  line-height: 1;
}

.work-list,
.device-bars,
.store-rank,
.usage-list,
.conversion-funnel {
  display: grid;
  gap: 8px;
}

.work-item {
  align-items: center;
  background: #fff;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  color: var(--text);
  display: flex;
  justify-content: space-between;
  min-height: 42px;
  padding: 8px 10px;
  text-align: left;
}

.work-item:hover {
  background: #f8fafc;
}

.work-item:disabled {
  cursor: default;
  opacity: 0.66;
}

.work-item:disabled:hover {
  background: inherit;
}

.work-item span {
  color: currentColor;
  font-size: 13px;
}

.work-item b {
  font-size: 18px;
}

.work-item.danger {
  background: #fef2f2;
  border-color: #fecaca;
  color: #991b1b;
}

.work-item.warn {
  background: #fffbeb;
  border-color: #fde68a;
  color: #92400e;
}

.work-item.stable {
  background: var(--success-bg);
  border-color: #bbf7d0;
  color: var(--success-text);
}

.work-item.neutral {
  background: #f8fafc;
  color: var(--text);
}

.mini-funnel {
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-bottom: 14px;
}

.mini-funnel div {
  background: #fff;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 10px;
}

.mini-funnel span {
  color: var(--muted);
  display: block;
  font-size: 12px;
}

.mini-funnel strong {
  display: block;
  font-size: 20px;
  line-height: 1.1;
  margin-top: 4px;
}

.ops-line-chart {
  background:
    linear-gradient(180deg, rgba(239, 246, 255, 0.86) 0%, rgba(255, 255, 255, 0.96) 48%),
    #fff;
  border: 1px solid #dbeafe;
  border-radius: 8px;
  overflow: hidden;
  padding: 10px 10px 6px;
  position: relative;
}

.ops-chart-legend {
  align-items: center;
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-bottom: 2px;
}

.ops-chart-legend span {
  align-items: center;
  color: var(--muted);
  display: inline-flex;
  font-size: 12px;
  font-weight: 700;
  gap: 5px;
}

.ops-chart-legend i {
  border-radius: 999px;
  display: inline-block;
  height: 8px;
  width: 8px;
}

.ops-chart-legend i.visit {
  background: #2563eb;
}

.ops-chart-legend i.click {
  background: #10b981;
}

.ops-line-chart-svg {
  display: block;
  height: 220px;
  width: 100%;
}

.ops-chart-grid-line {
  stroke: #dbeafe;
  stroke-dasharray: 5 5;
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}

.ops-chart-y-tick {
  stroke: #94a3b8;
  stroke-linecap: round;
  stroke-width: 1.4;
  vector-effect: non-scaling-stroke;
}

.ops-chart-axis-text {
  fill: #64748b;
  font-size: 11px;
  font-weight: 700;
}

.ops-chart-y-axis-text {
  fill: #334155;
  font-size: 12px;
  font-weight: 850;
  paint-order: stroke fill;
  stroke: rgba(248, 250, 252, 0.92);
  stroke-linejoin: round;
  stroke-width: 3px;
}

.ops-chart-area {
  fill: url(#adminOpsVisitArea);
}

.ops-chart-line {
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 3.2;
  vector-effect: non-scaling-stroke;
}

.ops-chart-line.visit {
  filter: drop-shadow(0 8px 12px rgba(37, 99, 235, 0.14));
  stroke: #2563eb;
}

.ops-chart-line.click {
  stroke: #10b981;
}

.ops-chart-point-group {
  cursor: pointer;
  outline: none;
}

.ops-chart-hit {
  fill: transparent;
}

.ops-chart-hover-line {
  stroke: #bfdbfe;
  stroke-dasharray: 4 4;
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}

.ops-chart-dot {
  fill: #fff;
  stroke-width: 2.3;
  transition: fill 0.18s ease, r 0.18s ease, stroke 0.18s ease;
  vector-effect: non-scaling-stroke;
}

.ops-chart-dot.visit {
  stroke: #2563eb;
}

.ops-chart-dot.click {
  stroke: #10b981;
}

.ops-chart-point-group:focus .ops-chart-dot.visit,
.ops-chart-dot.visit:hover {
  fill: #2563eb;
  stroke: #fff;
}

.ops-chart-point-group:focus .ops-chart-dot.click,
.ops-chart-dot.click:hover {
  fill: #10b981;
  stroke: #fff;
}

.ops-chart-tooltip {
  background: #fff;
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  box-shadow: 0 14px 34px rgba(37, 99, 235, 0.18);
  color: var(--text);
  display: grid;
  gap: 2px;
  font-size: 12px;
  opacity: 0;
  padding: 8px 10px;
  pointer-events: none;
  position: absolute;
  transform: translate(12px, -100%);
  transition: opacity 0.18s ease;
  z-index: 2;
}

.ops-chart-tooltip.visible {
  opacity: 1;
}

.ops-chart-tooltip span {
  color: var(--muted);
}

.device-row-head {
  align-items: center;
  display: flex;
  gap: 10px;
  justify-content: space-between;
  margin-bottom: 6px;
}

.device-row-head span {
  font-weight: 800;
}

.device-row-head b {
  color: var(--muted);
  font-size: 13px;
}

.device-track {
  background: #e5edf7;
  border-radius: 999px;
  height: 9px;
  overflow: hidden;
}

.device-track span {
  background: #2563eb;
  border-radius: inherit;
  display: block;
  height: 100%;
}

.store-rank {
  list-style: none;
  margin: 0;
  padding: 0;
}

.store-rank button {
  align-items: center;
  background: #fff;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  color: var(--text);
  display: flex;
  gap: 12px;
  justify-content: space-between;
  min-height: 54px;
  padding: 9px 10px;
  text-align: left;
  width: 100%;
}

.store-rank button:hover {
  background: #f8fafc;
  border-color: #bfdbfe;
}

.store-rank b,
.store-rank small {
  display: block;
}

.store-rank small {
  color: var(--muted);
  font-size: 12px;
  margin-top: 2px;
}

.store-rank strong {
  font-size: 20px;
}

.ledger-head {
  align-items: flex-start;
  margin-bottom: 14px;
}

.ledger-tools {
  display: grid;
  gap: 8px;
  grid-template-columns: minmax(240px, 1fr) minmax(160px, auto) auto;
  min-width: min(100%, 680px);
}

.ledger-layout {
  align-items: start;
  display: grid;
  gap: 14px;
  grid-template-columns: minmax(0, 1fr) 360px;
}

.ledger-table {
  min-width: 0;
  overflow-x: auto;
}

.ledger-table table,
.crawl-tables table,
.fold-card table {
  display: table;
  min-width: 100%;
}

.ledger-table table {
  min-width: 1080px;
}

.ledger-table tr {
  cursor: pointer;
}

.ledger-table tbody tr:hover,
.ledger-table tbody tr.selected {
  background: #f8fafc;
}

.ledger-table tbody tr.selected td:first-child {
  box-shadow: inset 3px 0 0 var(--primary);
}

.ledger-table th,
.ledger-table td {
  font-size: 13px;
  padding: 10px 8px;
  vertical-align: top;
}

.ledger-table td strong {
  color: var(--text);
}

.actions-cell {
  white-space: normal;
}

.table-actions {
  display: inline-grid;
  gap: 6px;
  grid-template-columns: repeat(3, auto);
}

.table-actions.compact {
  grid-template-columns: repeat(2, auto);
}

.table-actions button {
  min-height: 34px;
  padding: 6px 10px;
}

.ledger-empty {
  border: 1px dashed var(--border);
  border-radius: 8px;
  padding: 16px;
}

.mobile-store-list {
  display: none;
}

.detail-aside {
  background: #f8fafc;
  display: grid;
  gap: 12px;
  padding: 14px;
  position: sticky;
  top: 16px;
}

.detail-head h3 {
  font-size: 20px;
  margin-top: 3px;
}

.detail-head span {
  color: var(--muted);
  display: block;
  font-size: 12px;
  margin-top: 3px;
}

.status-grid,
.detail-metrics,
.config-summary {
  display: grid;
  gap: 8px;
  margin: 0;
}

.status-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.status-grid div,
.config-summary div {
  min-width: 0;
}

.status-grid dd,
.config-summary dd {
  color: var(--text);
  font-weight: 800;
  margin: 2px 0 0;
  overflow-wrap: anywhere;
}

.merchant-id-panel {
  background: #fff;
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  display: grid;
  gap: 10px;
  padding: 12px;
}

.merchant-id-head {
  align-items: flex-start;
  display: flex;
  gap: 10px;
  justify-content: space-between;
}

.merchant-id-head h4 {
  margin: 0 0 3px;
}

.merchant-id-form {
  display: grid;
  gap: 9px;
  grid-template-columns: minmax(0, 0.8fr) minmax(0, 1fr);
}

.merchant-id-form .check-field,
.merchant-id-form button {
  grid-column: 1 / -1;
}

.merchant-id-form button {
  min-height: 40px;
  padding: 8px 10px;
}

.detail-metrics {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.detail-metrics div,
.usage-list div,
.conversion-funnel div {
  background: #fff;
  min-width: 0;
  padding: 9px;
}

.detail-metrics strong,
.usage-list strong,
.conversion-funnel strong {
  display: block;
  font-size: 20px;
  line-height: 1.1;
  margin-top: 3px;
  overflow-wrap: anywhere;
}

.detail-tabs {
  background: #eaf0f7;
  border-radius: 8px;
  display: grid;
  gap: 3px;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  padding: 3px;
}

.detail-tabs button {
  background: transparent;
  border-radius: 6px;
  color: #475569;
  font-size: 12px;
  min-height: 34px;
  padding: 6px 4px;
}

.detail-tabs button.active {
  background: #fff;
  color: var(--primary-strong);
}

.detail-tab-panel {
  min-height: 160px;
}

.detail-block {
  border-top: 1px solid var(--border-soft);
  display: grid;
  gap: 8px;
  padding-top: 12px;
}

.detail-block:first-child {
  border-top: 0;
  padding-top: 0;
}

.detail-block h4 {
  margin: 0;
}

.detail-url {
  background: #fff;
  margin: 0;
  overflow-wrap: anywhere;
  padding: 9px;
}

.advice-card {
  background: #fff7ed;
  color: #9a3412;
  display: grid;
  gap: 4px;
  padding: 12px;
}

.advice-card strong {
  color: #7c2d12;
}

.detail-actions {
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.detail-actions button {
  min-height: 40px;
  padding: 8px 10px;
}

.crawl-panel {
  border-top: 1px solid var(--border-soft);
  margin-top: 18px;
  padding-top: 16px;
}

.crawl-panel h3,
.crawl-panel h4 {
  margin: 0 0 6px;
}

.crawl-tables {
  display: grid;
  gap: 14px;
  grid-template-columns: 1fr;
  overflow-x: auto;
}

.match-content {
  max-width: 360px;
  overflow-wrap: anywhere;
  white-space: normal;
}

.fold-grid {
  display: grid;
  gap: 12px;
}

.fold-card {
  padding: 0;
}

.fold-card summary {
  align-items: center;
  cursor: pointer;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  list-style: none;
  padding: 16px 18px;
}

.fold-card summary::-webkit-details-marker {
  display: none;
}

.fold-card summary strong,
.fold-card summary small {
  display: block;
}

.fold-card summary small {
  color: var(--muted);
  font-size: 12px;
  margin-top: 2px;
}

.fold-hint {
  color: var(--primary-strong);
  flex: 0 0 auto;
  font-size: 13px;
  font-weight: 800;
}

.fold-card[open] .fold-hint {
  color: var(--muted);
  font-size: 0;
}

.fold-card[open] .fold-hint::before {
  content: "收起";
  font-size: 13px;
}

.fold-body {
  border-top: 1px solid var(--border-soft);
  padding: 14px 18px 18px;
}

.inline-form {
  display: grid;
  gap: 8px;
  grid-template-columns: minmax(200px, 1fr) minmax(180px, 0.5fr) auto;
  margin-bottom: 12px;
}

.drawer-backdrop {
  align-items: stretch;
  background: rgba(15, 23, 42, 0.36);
  display: flex;
  inset: 0;
  justify-content: flex-end;
  position: fixed;
  z-index: 50;
}

.config-drawer {
  background: var(--surface);
  border-left: 1px solid var(--border);
  box-shadow: -18px 0 40px rgba(15, 23, 42, 0.16);
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto auto auto;
  gap: 14px;
  max-width: min(760px, 100vw);
  overflow-y: auto;
  padding: 20px;
  width: 760px;
}

.drawer-head {
  border-bottom: 1px solid var(--border-soft);
  padding-bottom: 12px;
}

.config-form-grid {
  align-content: start;
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.config-section {
  background: #fff;
  min-width: 0;
  padding: 14px;
}

.config-section.important {
  background: #f8fafc;
}

.config-section h3 {
  font-size: 15px;
  margin: 0 0 10px;
}

.form-grid.compact {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.fld,
.check-field {
  color: #374151;
  display: block;
  font-size: 13px;
}

.fld input,
.fld select {
  display: block;
  margin-top: 4px;
  width: 100%;
}

.fld.wide,
.check-field.wide {
  grid-column: 1 / -1;
}

.check-field {
  align-items: center;
  display: inline-flex;
  gap: 8px;
  min-height: 42px;
}

.check-field input {
  height: 18px;
  width: 18px;
}

.delivery-preview {
  border-top: 1px solid var(--border-soft);
  display: grid;
  gap: 10px;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  padding-top: 14px;
}

.delivery-preview code,
.created code,
.url-line code {
  background: #f8fafc;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  display: block;
  font-size: 12px;
  overflow-wrap: anywhere;
  padding: 8px;
  white-space: normal;
}

.created {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  font-size: 13px;
  padding: 12px;
}

.created p {
  margin: 0 0 8px;
}

.created p:last-child {
  margin-bottom: 0;
}

.url-line {
  align-items: center;
  display: grid;
  gap: 8px;
  grid-template-columns: auto minmax(0, 1fr) auto;
}

.drawer-actions {
  background: var(--surface);
  border-top: 1px solid var(--border-soft);
  bottom: 0;
  display: grid;
  gap: 8px;
  grid-template-columns: minmax(0, 1fr) auto;
  padding-top: 12px;
  position: sticky;
}

.mobile-store-item {
  border-top: 1px solid var(--border-soft);
  padding: 14px 0;
}

.mobile-store-item:first-child {
  border-top: 0;
  padding-top: 0;
}

.mobile-store-head {
  align-items: flex-start;
  display: flex;
  gap: 10px;
  justify-content: space-between;
}

.mobile-store-head strong {
  display: block;
  font-size: 17px;
}

.mobile-store-head span {
  color: var(--muted);
  display: block;
  font-size: 13px;
  margin-top: 3px;
}

.mobile-store-meta {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 12px 0;
}

.mobile-store-meta dd {
  margin: 0;
}

.mobile-store-actions {
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.mobile-store-actions button {
  min-height: 42px;
  padding: 8px 10px;
}

@media (max-width: 1180px) {
  .kpi-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .overview-grid,
  .ledger-layout {
    grid-template-columns: 1fr;
  }

  .detail-aside {
    position: static;
  }
}

@media (max-width: 860px) {
  .admin-console {
    grid-template-columns: 1fr;
  }

  .ops-sidebar {
    min-height: auto;
    position: static;
  }

  .side-nav {
    grid-template-columns: repeat(5, minmax(0, 1fr));
    overflow-x: auto;
  }

  .side-nav a {
    grid-template-columns: 1fr;
    justify-items: center;
    min-width: 96px;
    text-align: center;
  }

  .side-nav i {
    display: none;
  }

  .side-summary {
    display: none;
  }

  .ops-topbar,
  .screen-head,
  .ledger-head {
    display: grid;
    grid-template-columns: 1fr;
  }

  .topbar-actions,
  .ledger-tools,
  .inline-form,
  .delivery-preview,
  .url-line {
    grid-template-columns: 1fr;
  }

  .topbar-actions {
    display: grid;
  }
}

@media (max-width: 640px) {
  .ops-main {
    padding: 14px;
  }

  .screen-card {
    padding: 14px;
  }

  .kpi-grid,
  .mini-funnel,
  .detail-metrics,
  .status-grid,
  .detail-actions,
  .merchant-id-form,
  .config-form-grid,
  .form-grid.compact,
  .mobile-store-meta,
  .mobile-store-actions {
    grid-template-columns: 1fr;
  }

  .desktop-table {
    display: none !important;
  }

  .mobile-store-list {
    display: block;
  }

  .detail-tabs {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .config-drawer {
    max-width: 100vw;
    width: 100vw;
  }
}
</style>
