<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { merchantApi } from '../../api/merchant'
import type { GenerationPreferences, PublishStats, PublishTrendPoint } from '../../api/merchant'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const storeForm = reactive({
  storeName: '',
  industryType: '',
  storeIntro: '',
  address: '',
  primaryPlatformStyle: 'xiaohongshu',
  brandTone: ''
})
const keyword = ref('')
const imageUrl = ref('')
const platformForm = reactive({ platformCode: '', platformName: '', buttonText: '', targetUrl: '', backupUrl: '', sortNo: 1, status: 1 })
const editingPlatformLinkId = ref<number | null>(null)
const reviewText = ref('')
const reviewPlatformCode = ref('')
const keywords = ref<any[]>([])
const suggestedTags = ref<string[]>([])
const images = ref<any[]>([])
const links = ref<any[]>([])
const reviews = ref<any[]>([])
const tasks = ref<any[]>([])
const dashboard = ref<PublishStats | null>(null)
const activeTrend = ref<'week' | 'month'>('week')
const optimizationOpen = ref(false)
const customFocusKeyword = ref('')
const preferenceSaving = ref(false)
const generating = ref(false)
const generationNotice = ref('')
const preferenceForm = reactive<GenerationPreferences>({
  configured: false,
  focusKeywords: [],
  styleCodes: ['natural'],
  diversityDimensions: ['customer_identity'],
  referenceReviews: [''],
  lengthVariance: 'wide'
})
const loading = ref(false)
const error = ref('')
const notice = ref('')
const isEditingPlatformLink = computed(() => editingPlatformLinkId.value !== null)

const platformPresets: Record<string, { name: string; buttonText: string }> = {
  dianping: { name: '大众点评', buttonText: '去大众点评发布' },
  meituan: { name: '美团', buttonText: '去美团发布' },
  xiaohongshu: { name: '小红书', buttonText: '去小红书发布' },
  douyin: { name: '抖音', buttonText: '去抖音发布' }
}

const styleOptions = [
  { code: 'natural', label: '自然随手写' },
  { code: 'detail_rich', label: '细节丰富' },
  { code: 'young_casual', label: '年轻口语' },
  { code: 'restrained', label: '稍微克制' },
  { code: 'regular_customer', label: '像老顾客' }
]

const diversityOptions = [
  { code: 'customer_identity', label: '顾客身份', sample: '新客 / 老客 / 上班族' },
  { code: 'dining_scene', label: '到店场景', sample: '午餐 / 聚餐 / 路过' },
  { code: 'content_angle', label: '内容角度', sample: '菜品 / 服务 / 环境' },
  { code: 'expression_structure', label: '表达结构', sample: '开头 / 转折 / 收尾' }
]

const styleLabels = computed(() => Object.fromEntries(styleOptions.map((item) => [item.code, item.label])))
const diversityLabels = computed(() => Object.fromEntries(diversityOptions.map((item) => [item.code, item.label])))
const availableSuggestions = computed(() =>
  suggestedTags.value.filter((t) => !keywords.value.some((k) => k.keyword === t))
)
const preferenceKeywordOptions = computed(() => {
  const all = [...keywords.value.map((k) => k.keyword), ...suggestedTags.value]
  return Array.from(new Set(all.map((v) => String(v || '').trim()).filter(Boolean))).slice(0, 24)
})
const selectedStyleLabels = computed(() =>
  preferenceForm.styleCodes.map((code) => styleLabels.value[code] || code).join('、') || '自然随手写'
)
const selectedDiversityLabels = computed(() =>
  preferenceForm.diversityDimensions.map((code) => diversityLabels.value[code] || code).join('、') || '顾客身份'
)
const preferenceSummary = computed(() => {
  const focus = preferenceForm.focusKeywords.length ? preferenceForm.focusKeywords.join('、') : '未设置重点'
  const refs = cleanReferenceReviews().length
  return `重点：${focus}；方向：${selectedDiversityLabels.value}；语气：${selectedStyleLabels.value}；参考评论 ${refs} 条`
})
const storeInitial = computed(() => (storeForm.storeName || '店').trim().slice(0, 1))
const trendSeries = computed(() => {
  const source = activeTrend.value === 'week' ? dashboard.value?.weeklySeries : dashboard.value?.monthlySeries
  return (source || []).map((item) => ({
    label: activeTrend.value === 'week' ? `${shortDate(item.weekStart)}-${shortDate(item.weekEnd)}` : item.month || '',
    count: item.count || 0
  }))
})
const trendHasData = computed(() => trendSeries.value.some((item) => item.count > 0))
const chartPoints = computed(() => buildChartPoints(trendSeries.value))
const chartPolyline = computed(() => chartPoints.value.map((p) => `${p.x},${p.y}`).join(' '))
const chartAria = computed(() => {
  const mode = activeTrend.value === 'week' ? '按周' : '按月'
  const total = trendSeries.value.reduce((sum, item) => sum + item.count, 0)
  return `${mode}引导发布趋势，共 ${trendSeries.value.length} 个数据点，合计 ${total} 次`
})
const dashboardStatusText = computed(() => {
  if (!dashboard.value) return '数据加载中'
  if (!dashboard.value.platformLinksConfigured) return '先配置平台链接，顾客才能去发布'
  if (dashboard.value.totalPublishClicks === 0) return '平台链接已启用，等待顾客点击去发布'
  return `累计引导顾客去发布 ${formatNumber(dashboard.value.totalPublishClicks)} 次`
})
const updatedText = computed(() => {
  if (!dashboard.value?.updatedAt) return ''
  const d = new Date(dashboard.value.updatedAt)
  if (Number.isNaN(d.getTime())) return ''
  return `数据更新至 ${d.toLocaleString('zh-CN', { hour12: false })}`
})

function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [storeRes, keywordRes, suggestRes, imageRes, linkRes, reviewRes, taskRes, statsRes, prefRes] = await Promise.all([
      merchantApi.getStoreDetail(),
      merchantApi.listKeywords(),
      merchantApi.getKeywordSuggestions(),
      merchantApi.listImages(),
      merchantApi.listPlatformLinks(),
      merchantApi.listReviews(),
      merchantApi.listTasks(),
      merchantApi.getPublishStats(),
      merchantApi.getGenerationPreferences()
    ])
    Object.assign(storeForm, storeRes.data.data)
    keywords.value = keywordRes.data.data
    suggestedTags.value = suggestRes.data.data?.tags || []
    images.value = imageRes.data.data
    links.value = linkRes.data.data
    reviews.value = reviewRes.data.data
    tasks.value = taskRes.data.data
    dashboard.value = statsRes.data.data
    applyPreferences(prefRes.data.data)
    if (!reviewPlatformCode.value && links.value.length > 0) {
      reviewPlatformCode.value = links.value.find((item) => item.status === 1)?.platformCode || links.value[0].platformCode
    }
  } catch (err: any) {
    error.value = messageFrom(err, '商家后台数据加载失败')
  } finally {
    loading.value = false
  }
}

async function runAction(action: () => Promise<unknown>, success: string, reload = true) {
  error.value = ''
  notice.value = ''
  try {
    await action()
    notice.value = success
    if (reload) {
      await loadAll()
    }
    return true
  } catch (err: any) {
    error.value = messageFrom(err, '操作失败')
    return false
  }
}

async function saveStore() {
  await runAction(() => merchantApi.updateStoreDetail(storeForm), '门店信息已保存')
}

function resetPlatformForm() {
  editingPlatformLinkId.value = null
  platformForm.platformCode = ''
  platformForm.platformName = ''
  platformForm.buttonText = ''
  platformForm.targetUrl = ''
  platformForm.backupUrl = ''
  platformForm.sortNo = links.value.length + 1
  platformForm.status = 1
}

function applyPlatformPreset() {
  const preset = platformPresets[platformForm.platformCode]
  if (!preset) return
  if (!platformForm.platformName.trim()) platformForm.platformName = preset.name
  if (!platformForm.buttonText.trim()) platformForm.buttonText = preset.buttonText
}

async function addKeyword() {
  const value = keyword.value.trim()
  if (!value) return
  if (await runAction(() => merchantApi.createKeyword({ keyword: value, sortNo: keywords.value.length + 1 }), '关键词已添加')) {
    keyword.value = ''
  }
}

async function addSuggested(tag: string) {
  await runAction(() => merchantApi.createKeyword({ keyword: tag, sortNo: keywords.value.length + 1 }), '已添加推荐标签')
}

async function addImage() {
  const value = imageUrl.value.trim()
  if (!value) return
  if (await runAction(() => merchantApi.createImage({ imageUrl: value, thumbnailUrl: value, sortNo: images.value.length + 1 }), '图片已添加')) {
    imageUrl.value = ''
  }
}

async function onPickImage(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files && input.files[0]
  if (!file) return
  if (file.size > 5 * 1024 * 1024) {
    error.value = '图片需在 5MB 以内'
    input.value = ''
    return
  }
  await runAction(() => merchantApi.uploadImageFile(file), '图片已上传')
  input.value = ''
}

function editPlatformLink(item: any) {
  editingPlatformLinkId.value = item.id
  platformForm.platformCode = item.platformCode || ''
  platformForm.platformName = item.platformName || ''
  platformForm.buttonText = item.buttonText || ''
  platformForm.targetUrl = item.targetUrl || ''
  platformForm.backupUrl = item.backupUrl || ''
  platformForm.sortNo = item.sortNo || 1
  platformForm.status = item.status === 0 ? 0 : 1
}

async function savePlatformLink() {
  applyPlatformPreset()
  if (!platformForm.platformCode.trim() || !platformForm.targetUrl.trim()) {
    error.value = '请填写平台编码和客户端跳转链接'
    return
  }
  const payload = {
    platformCode: platformForm.platformCode.trim(),
    platformName: platformForm.platformName.trim() || platformForm.platformCode.trim(),
    buttonText: platformForm.buttonText.trim() || '去发布',
    targetUrl: platformForm.targetUrl.trim(),
    backupUrl: platformForm.backupUrl.trim(),
    sortNo: platformForm.sortNo || links.value.length + 1,
    status: platformForm.status || 1
  }
  const action = isEditingPlatformLink.value
    ? () => merchantApi.updatePlatformLink(editingPlatformLinkId.value as number, payload)
    : () => merchantApi.createPlatformLink(payload)
  const success = isEditingPlatformLink.value ? '客户端跳转链接已保存' : '客户端跳转链接已新增'
  if (await runAction(action, success)) {
    resetPlatformForm()
  }
}

async function addReview() {
  const value = reviewText.value.trim()
  if (!value) return
  if (!reviewPlatformCode.value) {
    error.value = '请先选择评价平台'
    return
  }
  if (await runAction(() => merchantApi.createReview({ content: value, status: 'available', platformCode: reviewPlatformCode.value }), '评价已添加')) {
    reviewText.value = ''
  }
}

async function generateReviews() {
  if (!reviewPlatformCode.value) {
    error.value = '请先选择评价平台'
    return
  }
  generating.value = true
  generationNotice.value = ''
  try {
    await merchantApi.generateReviews(reviewPlatformCode.value, 10)
    generationNotice.value = '已按当前生成方向新增可用评论'
    notice.value = '评价生成任务已完成'
    await loadAll()
  } catch (err: any) {
    error.value = messageFrom(err, '评价生成失败，已保存的生成方向不会丢失')
  } finally {
    generating.value = false
  }
}

async function savePreferences(generateAfter = false) {
  error.value = ''
  notice.value = ''
  preferenceSaving.value = true
  try {
    const payload = preferencePayload()
    const res = await merchantApi.saveGenerationPreferences(payload)
    applyPreferences(res.data.data)
    notice.value = generateAfter ? '生成方向已保存，正在生成评论' : '生成方向已保存'
    if (generateAfter) {
      await generateReviews()
    }
  } catch (err: any) {
    error.value = messageFrom(err, '生成方向保存失败')
  } finally {
    preferenceSaving.value = false
  }
}

function applyPreferences(data: GenerationPreferences) {
  preferenceForm.configured = !!data.configured
  preferenceForm.focusKeywords = [...(data.focusKeywords || [])]
  preferenceForm.styleCodes = data.styleCodes?.length ? [...data.styleCodes] : ['natural']
  preferenceForm.diversityDimensions = data.diversityDimensions?.length ? [...data.diversityDimensions] : ['customer_identity']
  preferenceForm.referenceReviews = data.referenceReviews?.length ? [...data.referenceReviews] : ['']
  preferenceForm.lengthVariance = data.lengthVariance || 'wide'
  preferenceForm.updatedAt = data.updatedAt
}

function preferencePayload(): GenerationPreferences {
  return {
    focusKeywords: preferenceForm.focusKeywords.map((v) => v.trim()).filter(Boolean).slice(0, 8),
    styleCodes: preferenceForm.styleCodes.length ? preferenceForm.styleCodes.slice(0, 3) : ['natural'],
    diversityDimensions: preferenceForm.diversityDimensions.length ? preferenceForm.diversityDimensions.slice(0, 4) : ['customer_identity'],
    referenceReviews: cleanReferenceReviews(),
    lengthVariance: 'wide'
  }
}

function toggleFocusKeyword(tag: string) {
  tag = tag.trim()
  if (!tag) return
  const i = preferenceForm.focusKeywords.indexOf(tag)
  if (i >= 0) {
    preferenceForm.focusKeywords.splice(i, 1)
    return
  }
  if (preferenceForm.focusKeywords.length >= 8) {
    error.value = '重点方向最多选择 8 个'
    return
  }
  preferenceForm.focusKeywords.push(tag)
}

function addCustomFocusKeyword() {
  const value = customFocusKeyword.value.trim()
  if (!value) return
  toggleFocusKeyword(value)
  customFocusKeyword.value = ''
}

function toggleStyle(code: string) {
  const i = preferenceForm.styleCodes.indexOf(code)
  if (i >= 0) {
    if (preferenceForm.styleCodes.length === 1) return
    preferenceForm.styleCodes.splice(i, 1)
    return
  }
  if (preferenceForm.styleCodes.length >= 3) {
    error.value = '语气最多选择 3 个'
    return
  }
  preferenceForm.styleCodes.push(code)
}

function toggleDiversityDimension(code: string) {
  const i = preferenceForm.diversityDimensions.indexOf(code)
  if (i >= 0) {
    if (preferenceForm.diversityDimensions.length === 1) return
    preferenceForm.diversityDimensions.splice(i, 1)
    return
  }
  if (preferenceForm.diversityDimensions.length >= 4) {
    error.value = '多样化方向最多选择 4 个'
    return
  }
  preferenceForm.diversityDimensions.push(code)
}

function addReferenceReview() {
  if (preferenceForm.referenceReviews.length >= 5) {
    error.value = '参考评论最多 5 条'
    return
  }
  preferenceForm.referenceReviews.push('')
}

function removeReferenceReview(index: number) {
  preferenceForm.referenceReviews.splice(index, 1)
  if (preferenceForm.referenceReviews.length === 0) {
    preferenceForm.referenceReviews.push('')
  }
}

function cleanReferenceReviews() {
  return preferenceForm.referenceReviews.map((v) => v.trim()).filter(Boolean).slice(0, 5)
}

async function deleteKeyword(id: number) {
  if (!window.confirm('确认删除这个关键词？')) return
  await runAction(() => merchantApi.deleteKeyword(id), '关键词已删除')
}

async function deleteImage(id: number) {
  if (!window.confirm('确认删除这张图片？')) return
  await runAction(() => merchantApi.deleteImage(id), '图片已删除')
}

async function togglePlatformLinkStatus(item: any) {
  const nextStatus = item.status === 1 ? 0 : 1
  await runAction(() => merchantApi.updatePlatformLinkStatus(item.id, nextStatus), '跳转链接状态已更新')
}

async function deletePlatformLink(id: number) {
  if (!window.confirm('确认删除这个客户端跳转链接？')) return
  await runAction(() => merchantApi.deletePlatformLink(id), '跳转链接已删除')
  if (editingPlatformLinkId.value === id) resetPlatformForm()
}

async function deleteReview(id: number) {
  if (!window.confirm('确认删除这条评价？')) return
  await runAction(() => merchantApi.deleteReview(id), '评价已删除')
}

function numericStatusText(status: number) {
  return status === 1 ? '启用' : '禁用'
}

function formatNumber(value: number | undefined) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function shortDate(value?: string) {
  if (!value) return ''
  const parts = value.split('-')
  return parts.length === 3 ? `${parts[1]}.${parts[2]}` : value
}

function buildChartPoints(series: { label: string; count: number }[]) {
  if (!series.length) return []
  const width = 320
  const top = 18
  const bottom = 96
  const max = Math.max(...series.map((item) => item.count), 1)
  return series.map((item, index) => {
    const x = series.length === 1 ? width / 2 : 18 + (index * (width - 36)) / (series.length - 1)
    const y = bottom - (item.count / max) * (bottom - top)
    return { ...item, x: Number(x.toFixed(2)), y: Number(y.toFixed(2)) }
  })
}

function logout() {
  auth.clear()
  location.href = import.meta.env.BASE_URL + 'merchant/login'
}

onMounted(loadAll)
</script>

<template>
  <div class="page merchant-console">
    <div class="row merchant-header">
      <h1>商家后台</h1>
      <div class="row header-actions">
        <button class="secondary" :disabled="loading" @click="loadAll">刷新</button>
        <button class="secondary" @click="logout">退出登录</button>
      </div>
    </div>
    <p v-if="error" class="alert">{{ error }}</p>
    <p v-else-if="notice" class="notice">{{ notice }}</p>

    <section class="value-shell" aria-labelledby="publish-title">
      <div class="store-strip">
        <div class="store-mark" aria-hidden="true">{{ storeInitial }}</div>
        <div class="store-copy">
          <p class="eyebrow">{{ storeForm.industryType || '商家' }}</p>
          <h2 id="publish-title">{{ storeForm.storeName || '商家价值看板' }}</h2>
        </div>
        <p class="updated">{{ updatedText }}</p>
      </div>

      <div class="metric-zone" :aria-busy="loading && !dashboard">
        <div>
          <p class="metric-label">累计引导发布</p>
          <p class="hero-number">{{ formatNumber(dashboard?.totalPublishClicks) }}</p>
          <p class="metric-status">{{ dashboardStatusText }}</p>
        </div>
        <div class="secondary-metrics">
          <div>
            <span>本周</span>
            <strong>{{ formatNumber(dashboard?.currentWeekPublishClicks) }}</strong>
          </div>
          <div>
            <span>本月</span>
            <strong>{{ formatNumber(dashboard?.currentMonthPublishClicks) }}</strong>
          </div>
        </div>
      </div>

      <div class="trend-section">
        <div class="trend-head">
          <div>
            <h3>发布趋势</h3>
            <p class="muted">
              {{ activeTrend === 'week' ? '最近 12 周' : '最近 12 个月' }}
            </p>
          </div>
          <div class="trend-tabs" role="tablist" aria-label="趋势维度">
            <button type="button" :class="{ active: activeTrend === 'week' }" :aria-pressed="activeTrend === 'week'" @click="activeTrend = 'week'">按周</button>
            <button type="button" :class="{ active: activeTrend === 'month' }" :aria-pressed="activeTrend === 'month'" @click="activeTrend = 'month'">按月</button>
          </div>
        </div>

        <div class="chart-wrap">
          <svg viewBox="0 0 320 120" role="img" :aria-label="chartAria">
            <line x1="18" y1="96" x2="302" y2="96" class="axis" />
            <polyline v-if="chartPoints.length" :points="chartPolyline" class="trend-line" fill="none" />
            <g v-for="point in chartPoints" :key="point.label">
              <circle :cx="point.x" :cy="point.y" r="3.6" class="trend-dot" />
              <text v-if="point.count > 0" :x="point.x" :y="point.y - 8" text-anchor="middle" class="point-label">{{ point.count }}</text>
            </g>
          </svg>
          <div class="chart-labels" aria-hidden="true">
            <span v-for="item in trendSeries.filter((_, i) => i === 0 || i === trendSeries.length - 1 || i === Math.floor(trendSeries.length / 2))" :key="item.label">
              {{ item.label }}
            </span>
          </div>
          <p v-if="!trendHasData" class="empty-note">
            {{ dashboard?.platformLinksConfigured ? '有顾客点击去发布后，这里会出现趋势' : '先新增平台链接，顾客才能去发布' }}
          </p>
          <table class="sr-only">
            <caption>{{ activeTrend === 'week' ? '按周引导发布数据' : '按月引导发布数据' }}</caption>
            <tbody>
              <tr v-for="item in trendSeries" :key="item.label">
                <th>{{ item.label }}</th>
                <td>{{ item.count }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="optimization-panel">
        <div>
          <h3>优化下一批评论</h3>
          <p class="muted">{{ preferenceSummary }}</p>
          <p v-if="generationNotice" class="inline-success">{{ generationNotice }}</p>
        </div>
        <div class="row optimization-actions">
          <button class="secondary" type="button" @click="optimizationOpen = !optimizationOpen">
            {{ optimizationOpen ? '收起' : '去优化' }}
          </button>
          <button type="button" :disabled="preferenceSaving || generating" @click="savePreferences(true)">保存并生成 10 条</button>
        </div>
      </div>

      <div v-if="optimizationOpen" class="preference-form">
        <div class="form-block">
          <label class="field-label" for="custom-focus">重点方向</label>
          <div class="tag-grid">
            <button
              v-for="tag in preferenceKeywordOptions"
              :key="tag"
              type="button"
              class="select-chip"
              :class="{ selected: preferenceForm.focusKeywords.includes(tag) }"
              :aria-pressed="preferenceForm.focusKeywords.includes(tag)"
              @click="toggleFocusKeyword(tag)"
            >{{ tag }}</button>
          </div>
          <div class="row action-row compact-row">
            <input id="custom-focus" v-model="customFocusKeyword" maxlength="40" placeholder="新增重点，如 上菜快" />
            <button type="button" class="secondary" @click="addCustomFocusKeyword">添加重点</button>
          </div>
        </div>

        <div class="form-block">
          <p class="field-label">多样化方向</p>
          <div class="tag-grid dimension-grid">
            <button
              v-for="item in diversityOptions"
              :key="item.code"
              type="button"
              class="select-chip dimension-chip"
              :class="{ selected: preferenceForm.diversityDimensions.includes(item.code) }"
              :aria-pressed="preferenceForm.diversityDimensions.includes(item.code)"
              @click="toggleDiversityDimension(item.code)"
            >
              <span>{{ item.label }}</span>
              <small>{{ item.sample }}</small>
            </button>
          </div>
        </div>

        <div class="form-block">
          <p class="field-label">语气</p>
          <div class="tag-grid">
            <button
              v-for="item in styleOptions"
              :key="item.code"
              type="button"
              class="select-chip"
              :class="{ selected: preferenceForm.styleCodes.includes(item.code) }"
              :aria-pressed="preferenceForm.styleCodes.includes(item.code)"
              @click="toggleStyle(item.code)"
            >{{ item.label }}</button>
          </div>
        </div>

        <div class="form-block">
          <div class="row form-title-row">
            <label class="field-label">参考评论</label>
            <button type="button" class="secondary small-btn" @click="addReferenceReview">新增一条</button>
          </div>
          <div class="reference-list">
            <div v-for="(_, index) in preferenceForm.referenceReviews" :key="index" class="reference-row">
              <textarea v-model="preferenceForm.referenceReviews[index]" maxlength="300" placeholder="贴一条真实顾客评论，AI 只学习表达方式"></textarea>
              <button type="button" class="danger small-btn" @click="removeReferenceReview(index)">删除</button>
            </div>
          </div>
          <p class="privacy-note">请勿粘贴手机号、微信号、订单号等个人信息。</p>
        </div>

        <div class="row form-actions">
          <button type="button" :disabled="preferenceSaving" @click="savePreferences(false)">
            {{ preferenceSaving ? '保存中' : '保存生成方向' }}
          </button>
          <button type="button" class="secondary" :disabled="preferenceSaving || generating" @click="savePreferences(true)">
            {{ generating ? '生成中' : '保存并生成 10 条' }}
          </button>
        </div>
      </div>
    </section>

    <div class="fold-grid">
      <details class="card fold-card">
        <summary>
          <span>
            <strong>跳转链接</strong>
            <small>顾客点击去发布时打开的商家链接</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
          <input v-model.trim="platformForm.platformCode" list="platform-codes" placeholder="平台编码，如 dianping" @change="applyPlatformPreset" />
          <datalist id="platform-codes">
            <option value="dianping">大众点评</option>
            <option value="meituan">美团</option>
            <option value="xiaohongshu">小红书</option>
            <option value="douyin">抖音</option>
          </datalist>
          <div class="field-gap"></div>
          <input v-model="platformForm.platformName" placeholder="平台名称" />
          <div class="field-gap"></div>
          <input v-model="platformForm.buttonText" placeholder="按钮文案" />
          <div class="field-gap"></div>
          <input v-model.trim="platformForm.targetUrl" placeholder="客户端跳转链接" />
          <div class="field-gap"></div>
          <input v-model.trim="platformForm.backupUrl" placeholder="备用链接（选填）" />
          <div class="field-gap"></div>
          <div class="row action-row">
            <button :disabled="loading" @click="savePlatformLink">
              {{ isEditingPlatformLink ? '保存跳转链接' : '新增跳转链接' }}
            </button>
            <button v-if="isEditingPlatformLink" class="secondary" :disabled="loading" @click="resetPlatformForm">取消编辑</button>
          </div>
          <ul class="link-list">
            <li v-for="item in links" :key="item.id" class="list-action">
              <span>{{ item.buttonText }} - {{ item.targetUrl }}（{{ numericStatusText(item.status) }}）</span>
              <span class="row link-actions">
                <button class="secondary" :disabled="loading" @click="editPlatformLink(item)">编辑</button>
                <button class="secondary" :disabled="loading" @click="togglePlatformLinkStatus(item)">
                  {{ item.status === 1 ? '禁用' : '启用' }}
                </button>
                <button class="danger" :disabled="loading" @click="deletePlatformLink(item.id)">删除</button>
              </span>
            </li>
          </ul>
        </div>
      </details>

      <details class="card fold-card">
        <summary>
          <span>
            <strong>AI 生成任务</strong>
            <small>按当前生成方向补充评价池</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
          <select v-model="reviewPlatformCode">
            <option value="" disabled>选择评价平台</option>
            <option v-for="item in links" :key="item.id" :value="item.platformCode">
              {{ item.platformName || item.platformCode }}
            </option>
          </select>
          <div class="field-gap"></div>
          <button :disabled="loading || generating" @click="generateReviews">{{ generating ? '生成中' : '生成 10 条评价' }}</button>
          <table>
            <thead><tr><th>ID</th><th>类型</th><th>状态</th><th>成功数</th></tr></thead>
            <tbody>
              <tr v-for="task in tasks" :key="task.id">
                <td>{{ task.id }}</td>
                <td>{{ task.triggerType }}</td>
                <td>{{ task.status }}</td>
                <td>{{ task.successCount }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </details>

      <details class="card fold-card">
        <summary>
          <span>
            <strong>关键词管理</strong>
            <small>顾客选择标签，用来生成更贴合的评价</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
          <div v-if="availableSuggestions.length" class="suggestions">
            <p class="muted">本行业推荐标签（点击添加）：</p>
            <div class="row suggestion-row">
              <button
                v-for="tag in availableSuggestions"
                :key="tag"
                class="suggest-chip"
                :disabled="loading"
                @click="addSuggested(tag)"
              >+ {{ tag }}</button>
            </div>
          </div>
          <div class="row action-row">
            <input v-model="keyword" placeholder="自定义关键词" />
            <button :disabled="loading" @click="addKeyword">添加</button>
          </div>
          <ul>
            <li v-for="item in keywords" :key="item.id" class="list-action">
              <span>{{ item.keyword }}</span>
              <button class="danger" :disabled="loading" @click="deleteKeyword(item.id)">删除</button>
            </li>
          </ul>
        </div>
      </details>

      <details class="card fold-card">
        <summary>
          <span>
            <strong>图片管理</strong>
            <small>落地页展示的店铺或菜品图片</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
          <label class="upload-btn">
            <span>上传图片</span>
            <input type="file" accept="image/*" :disabled="loading" @change="onPickImage" style="display: none" />
          </label>
          <details class="inline-details">
            <summary>或：贴图片 URL</summary>
            <div class="row action-row">
              <input v-model="imageUrl" placeholder="图片 URL" />
              <button :disabled="loading" @click="addImage">添加</button>
            </div>
          </details>
          <div class="row image-list">
            <div v-for="item in images" :key="item.id" class="image-item">
              <img :src="item.thumbnailUrl || item.imageUrl" alt="店铺图片" />
              <button class="danger" :disabled="loading" @click="deleteImage(item.id)">删除</button>
            </div>
          </div>
        </div>
      </details>

      <details class="card fold-card">
        <summary>
          <span>
            <strong>店铺信息</strong>
            <small>门店名称、行业、地址和品牌语气</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
          <input v-model="storeForm.storeName" placeholder="门店名称" />
          <div class="field-gap"></div>
          <input v-model="storeForm.industryType" placeholder="行业类型" />
          <div class="field-gap"></div>
          <input v-model="storeForm.address" placeholder="门店地址" />
          <div class="field-gap"></div>
          <input v-model="storeForm.primaryPlatformStyle" placeholder="主平台风格" />
          <div class="field-gap"></div>
          <textarea v-model="storeForm.storeIntro" placeholder="门店简介"></textarea>
          <div class="field-gap"></div>
          <input v-model="storeForm.brandTone" placeholder="品牌调性" />
          <div class="field-gap"></div>
          <button :disabled="loading" @click="saveStore">保存</button>
        </div>
      </details>

      <details class="card fold-card">
        <summary>
          <span>
            <strong>评价管理</strong>
            <small>手工补充和维护可用评价</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
          <select v-model="reviewPlatformCode">
            <option value="" disabled>选择评价平台</option>
            <option v-for="item in links" :key="item.id" :value="item.platformCode">
              {{ item.platformName || item.platformCode }}
            </option>
          </select>
          <div class="field-gap"></div>
          <textarea v-model="reviewText" placeholder="新增手工评价"></textarea>
          <div class="field-gap"></div>
          <button :disabled="loading" @click="addReview">添加评价</button>
          <ul>
            <li v-for="item in reviews.slice(0, 8)" :key="item.id" class="list-action">
              <span>{{ item.platformStyle }} - {{ item.content }}</span>
              <button class="danger" :disabled="loading" @click="deleteReview(item.id)">删除</button>
            </li>
          </ul>
        </div>
      </details>
    </div>
  </div>
</template>

<style scoped>
.merchant-header {
  align-items: center;
  justify-content: space-between;
}
.header-actions {
  align-items: center;
}
.merchant-console .card {
  border-radius: 8px;
  box-shadow: none;
}
.value-shell {
  background: var(--surface);
  border: 1px solid rgba(219, 228, 240, 0.9);
  border-radius: 8px;
  margin-bottom: 16px;
  padding: 20px;
}
.store-strip {
  align-items: center;
  display: grid;
  gap: 12px;
  grid-template-columns: auto minmax(0, 1fr) auto;
}
.store-mark {
  align-items: center;
  background: #0f172a;
  border-radius: 8px;
  color: #fff;
  display: inline-flex;
  font-size: 20px;
  font-weight: 800;
  height: 48px;
  justify-content: center;
  width: 48px;
}
.store-copy h2,
.trend-head h3,
.optimization-panel h3 {
  margin: 0;
}
.eyebrow,
.updated {
  color: var(--muted);
  font-size: 13px;
  margin: 0;
}
.updated {
  text-align: right;
}
.metric-zone {
  align-items: end;
  border-bottom: 1px solid var(--border-soft);
  border-top: 1px solid var(--border-soft);
  display: grid;
  gap: 18px;
  grid-template-columns: minmax(0, 1fr) auto;
  margin: 18px 0;
  padding: 18px 0;
}
.metric-label {
  color: var(--muted);
  font-size: 14px;
  margin: 0 0 4px;
}
.hero-number {
  font-size: 56px;
  font-weight: 850;
  line-height: 1;
  margin: 0;
}
.metric-status {
  color: #334155;
  margin: 8px 0 0;
}
.secondary-metrics {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(2, minmax(96px, 1fr));
}
.secondary-metrics div {
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 10px 12px;
}
.secondary-metrics span {
  color: var(--muted);
  display: block;
  font-size: 13px;
}
.secondary-metrics strong {
  display: block;
  font-size: 24px;
  line-height: 1.15;
  margin-top: 2px;
}
.trend-section {
  border-bottom: 1px solid var(--border-soft);
  padding-bottom: 18px;
}
.trend-head {
  align-items: center;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  margin-bottom: 12px;
}
.trend-head p {
  margin: 2px 0 0;
}
.trend-tabs {
  background: #f8fafc;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  display: inline-flex;
  padding: 3px;
}
.trend-tabs button {
  background: transparent;
  border-radius: 6px;
  color: var(--muted);
  min-height: 36px;
  padding: 6px 12px;
}
.trend-tabs button.active {
  background: var(--surface);
  color: var(--text);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}
.chart-wrap {
  position: relative;
}
.chart-wrap svg {
  display: block;
  height: 180px;
  width: 100%;
}
.axis {
  stroke: #cbd5e1;
  stroke-width: 1;
}
.trend-line {
  stroke: var(--primary);
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 3;
}
.trend-dot {
  fill: var(--surface);
  stroke: var(--primary-strong);
  stroke-width: 2;
}
.point-label {
  fill: var(--text);
  font-size: 10px;
  font-weight: 700;
}
.chart-labels {
  color: var(--muted);
  display: flex;
  font-size: 12px;
  justify-content: space-between;
  margin-top: -14px;
}
.empty-note {
  color: var(--muted);
  margin: 8px 0 0;
}
.optimization-panel {
  align-items: center;
  display: grid;
  gap: 12px;
  grid-template-columns: minmax(0, 1fr) auto;
  padding-top: 18px;
}
.optimization-panel p {
  margin: 4px 0 0;
}
.optimization-actions {
  justify-content: flex-end;
}
.inline-success {
  color: var(--success-text);
  font-weight: 700;
}
.preference-form {
  border-top: 1px solid var(--border-soft);
  margin-top: 18px;
  padding-top: 18px;
}
.form-block {
  margin-bottom: 16px;
}
.field-label {
  color: var(--text);
  display: block;
  font-size: 14px;
  font-weight: 800;
  margin: 0 0 8px;
}
.tag-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.select-chip {
  background: #f8fafc;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: #334155;
  min-height: 40px;
  padding: 8px 12px;
}
.select-chip.selected {
  background: var(--primary-soft);
  border-color: #93c5fd;
  color: var(--primary-strong);
  font-weight: 800;
}
.dimension-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.dimension-chip {
  align-items: flex-start;
  display: grid;
  gap: 4px;
  justify-items: start;
  min-height: 58px;
  text-align: left;
}
.dimension-chip small {
  color: var(--muted);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.35;
}
.dimension-chip.selected small {
  color: #1d4ed8;
}
.compact-row {
  margin-top: 10px;
}
.form-title-row {
  align-items: center;
  justify-content: space-between;
}
.reference-list {
  display: grid;
  gap: 10px;
}
.reference-row {
  align-items: start;
  display: grid;
  gap: 8px;
  grid-template-columns: minmax(0, 1fr) auto;
}
.privacy-note {
  color: var(--muted);
  font-size: 13px;
  margin: 8px 0 0;
}
.small-btn {
  min-height: 38px;
  padding: 7px 12px;
}
.form-actions {
  align-items: center;
}
.field-gap {
  height: 8px;
}
.action-row {
  align-items: center;
}
.link-list {
  margin-top: 14px;
  padding-left: 0;
}
.link-actions {
  flex: 0 0 auto;
}
.suggestions {
  margin-bottom: 12px;
}
.suggestions p {
  margin: 0 0 8px;
}
.suggestion-row {
  gap: 8px;
}
.image-list {
  margin-top: 12px;
}
.inline-details {
  margin: 10px 0;
}
.inline-details summary {
  color: var(--muted);
  cursor: pointer;
  font-size: 14px;
}
.inline-details .row {
  margin-top: 8px;
}
.fold-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.fold-card {
  margin-bottom: 0;
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
.fold-card summary strong {
  display: block;
  font-size: 16px;
}
.fold-card summary small {
  color: var(--muted);
  display: block;
  font-size: 12px;
  margin-top: 2px;
}
.fold-hint {
  color: var(--primary-strong);
  flex: 0 0 auto;
  font-size: 13px;
  font-weight: 700;
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
.upload-btn {
  align-items: center;
  background: #3b82f6;
  border-radius: 8px;
  color: #fff;
  cursor: pointer;
  display: inline-flex;
  font-size: 15px;
  font-weight: 700;
  justify-content: center;
  min-height: 48px;
  min-width: 132px;
  padding: 10px 16px;
  touch-action: manipulation;
}
.upload-btn:hover {
  background: #2563eb;
}
.suggest-chip {
  background: #f0f7ff;
  border: 1px dashed #93c5fd;
  border-radius: 8px;
  color: #1d4ed8;
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
  min-height: 40px;
  padding: 8px 14px;
}
.suggest-chip:hover {
  background: #dbeafe;
}
.suggest-chip:disabled {
  opacity: 0.6;
}
.sr-only {
  height: 1px;
  margin: -1px;
  overflow: hidden;
  position: absolute;
  width: 1px;
}

@media (max-width: 720px) {
  .merchant-header {
    display: grid;
    grid-template-columns: 1fr;
  }
  .header-actions,
  .action-row,
  .form-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }
  .header-actions button,
  .action-row button,
  .form-actions button {
    width: 100%;
  }
  .store-strip,
  .metric-zone,
  .optimization-panel {
    grid-template-columns: 1fr;
  }
  .updated {
    text-align: left;
  }
  .hero-number {
    font-size: 44px;
  }
  .secondary-metrics,
  .fold-grid {
    grid-template-columns: 1fr;
  }
  .trend-head {
    align-items: stretch;
    flex-direction: column;
  }
  .trend-tabs {
    width: 100%;
  }
  .trend-tabs button {
    flex: 1;
  }
  .optimization-actions {
    justify-content: stretch;
  }
  .reference-row {
    grid-template-columns: 1fr;
  }
  .upload-btn {
    width: 100%;
  }
  .suggest-chip,
  .select-chip {
    flex: 1 1 calc(50% - 8px);
    min-height: 44px;
  }
  .dimension-grid {
    grid-template-columns: 1fr;
  }
  .link-actions {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    width: 100%;
  }
  .link-actions button {
    width: 100%;
  }
}
</style>
