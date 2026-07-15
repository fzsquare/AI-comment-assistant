<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { merchantApi } from '../../api/merchant'
import type { GenerationPreferences, PublishStats, PublishStatsRange } from '../../api/merchant'
import { useAuthStore } from '../../stores/auth'
import MerchantEffectDashboard from './MerchantEffectDashboard.vue'

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
const dashboard = ref<PublishStats | null>(null)
const analyticsPlatformCode = ref('')
const analyticsRange = ref<PublishStatsRange>('7d')
const dashboardLoading = ref(false)
const dashboardError = ref('')
let dashboardRequestId = 0
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
const activePlatformLinks = computed(() =>
  [...links.value]
    .filter((item) => item.status === 1)
    .sort((a, b) => Number(a.sortNo || 0) - Number(b.sortNo || 0))
)
const analyticsPlatformOptions = computed(() => [
  { platformCode: '', platformName: '全部平台' },
  ...activePlatformLinks.value.map((item) => ({
    platformCode: item.platformCode || '',
    platformName: item.platformName || item.platformCode || '未命名平台'
  }))
])
function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [storeRes, keywordRes, suggestRes, imageRes, linkRes, reviewRes, prefRes] = await Promise.all([
      merchantApi.getStoreDetail(),
      merchantApi.listKeywords(),
      merchantApi.getKeywordSuggestions(),
      merchantApi.listImages(),
      merchantApi.listPlatformLinks(),
      merchantApi.listReviews(),
      merchantApi.getGenerationPreferences()
    ])
    Object.assign(storeForm, storeRes.data.data)
    keywords.value = keywordRes.data.data
    suggestedTags.value = suggestRes.data.data?.tags || []
    images.value = imageRes.data.data
    links.value = linkRes.data.data
    reviews.value = reviewRes.data.data
    syncAnalyticsPlatformSelection()
    await loadDashboardStats(false)
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

async function loadDashboardStats(updateLoading = true) {
  const requestId = ++dashboardRequestId
  dashboardLoading.value = true
  dashboardError.value = ''
  try {
    const statsRes = await merchantApi.getPublishStats(analyticsPlatformCode.value, analyticsRange.value)
    if (requestId !== dashboardRequestId) return
    dashboard.value = statsRes.data.data
  } catch (err: any) {
    if (requestId !== dashboardRequestId) return
    dashboardError.value = messageFrom(err, '看板数据加载失败')
  } finally {
    if (requestId === dashboardRequestId) {
      dashboardLoading.value = false
      if (updateLoading) loading.value = false
    }
  }
}

function syncAnalyticsPlatformSelection() {
  if (!analyticsPlatformCode.value) return
  const exists = activePlatformLinks.value.some((item) => item.platformCode === analyticsPlatformCode.value)
  if (!exists) {
    analyticsPlatformCode.value = ''
  }
}

async function selectAnalyticsPlatform(platformCode: string) {
  if (analyticsPlatformCode.value === platformCode) return
  analyticsPlatformCode.value = platformCode
  await loadDashboardStats()
}

async function selectAnalyticsRange(range: PublishStatsRange) {
  if (analyticsRange.value === range) return
  analyticsRange.value = range
  await loadDashboardStats()
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

function logout() {
  auth.clear()
  location.href = import.meta.env.BASE_URL + 'merchant/login'
}

onMounted(async () => {
  await loadAll()
})
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

    <MerchantEffectDashboard
      :stats="dashboard"
      :loading="dashboardLoading"
      :error="dashboardError"
      :store-name="storeForm.storeName"
      :platform-options="analyticsPlatformOptions"
      @range-change="selectAnalyticsRange"
      @platform-change="selectAnalyticsPlatform"
      @retry="loadDashboardStats"
    />

    <section class="value-shell settings-shell" aria-label="评价内容优化">
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
      <details class="card fold-card" data-effect-target="platform-links">
        <summary>
          <span>
            <strong>跳转任务</strong>
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

      <details class="card fold-card" data-effect-target="nfc-guidance">
        <summary>
          <span>
            <strong>NFC 使用建议</strong>
            <small>检查卡片摆放和店员引导是否容易被顾客接受</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body guidance-copy">
          <p>把卡片放在顾客核销后自然能看到的位置，由店员说明“贴一下即可选择符合真实体验的评价，是否发布由你决定”。</p>
          <p>避免要求顾客交出手机，也不要把平台点击表述为已经发布成功。</p>
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

      <details class="card fold-card" data-effect-target="reviews">
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
  border: 1px solid var(--border);
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
.analytics-meta {
  display: grid;
  gap: 4px;
  justify-items: end;
}
.updated {
  text-align: right;
}
.platform-filter {
  align-items: center;
  border-top: 1px solid var(--border);
  display: grid;
  gap: 12px;
  grid-template-columns: minmax(0, 1fr) auto;
  margin-top: 18px;
  padding-top: 16px;
}
.platform-filter span {
  color: var(--muted);
  display: block;
  font-size: 13px;
}
.platform-filter strong {
  color: var(--text);
  display: block;
  font-size: 20px;
  line-height: 1.2;
  margin-top: 2px;
}
.platform-select-wrap {
  align-items: center;
  display: flex;
  gap: 8px;
}
.platform-select-wrap label {
  color: var(--muted);
  font-size: 13px;
  font-weight: 700;
}
.platform-select-wrap select {
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text);
  font: inherit;
  font-weight: 800;
  min-height: 40px;
  min-width: 150px;
  padding: 8px 12px;
}
.platform-select-wrap select:disabled {
  opacity: 0.65;
}
.platform-chip {
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 999px;
  color: var(--primary-strong);
  display: inline-flex;
  font-size: 12px;
  font-weight: 800;
  line-height: 1;
  margin-right: 6px;
  padding: 3px 7px;
  vertical-align: 1px;
}
.metric-zone {
  align-items: end;
  border-bottom: 1px solid var(--border);
  border-top: 1px solid var(--border);
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
  background: var(--surface-subtle);
  border: 1px solid var(--border);
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
.secondary-metrics small,
.compact-metrics small {
  color: var(--muted);
  display: block;
  font-size: 12px;
  font-weight: 700;
  line-height: 1.35;
  margin-top: 5px;
}
.secondary-metrics small b {
  font-size: inherit;
}
.secondary-metrics .growth-pair {
  align-items: center;
  display: flex;
  gap: 6px;
}
.secondary-metrics .growth-pair span {
  display: inline;
  font-size: inherit;
}
.up {
  color: var(--success-text);
}
.down {
  color: var(--danger);
}
.flat {
  color: var(--muted);
}
.secondary-metrics small.up,
.compact-metrics small.up {
  color: var(--success-text);
}
.secondary-metrics small.down,
.compact-metrics small.down {
  color: var(--danger);
}
.secondary-metrics small.flat,
.compact-metrics small.flat {
  color: var(--muted);
}
.secondary-metrics .growth-pair .up {
  color: var(--success-text);
}
.secondary-metrics .growth-pair .down {
  color: var(--danger);
}
.secondary-metrics .growth-pair .flat {
  color: var(--muted);
}
.dashboard-stack {
  display: grid;
  gap: 12px;
  margin-bottom: 18px;
}
.device-panel,
.publish-summary {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px;
}
.insight-head {
  align-items: start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  margin-bottom: 12px;
}
.insight-head h3 {
  margin: 0;
}
.insight-head p {
  margin: 3px 0 0;
}
.insight-head strong {
  color: var(--text);
  font-size: 22px;
  line-height: 1;
}
.device-collapsible {
  overflow: hidden;
  padding: 0;
}
.device-summary {
  align-items: center;
  cursor: pointer;
  display: grid;
  gap: 12px;
  grid-template-columns: minmax(0, 1fr) auto auto;
  list-style: none;
  min-height: 64px;
  padding: 14px;
}
.device-summary::-webkit-details-marker {
  display: none;
}
.device-summary h3 {
  margin: 0;
}
.device-summary p {
  margin: 3px 0 0;
}
.device-total {
  color: var(--text);
  font-size: 22px;
  font-weight: 850;
  line-height: 1;
}
.device-toggle {
  color: var(--primary-strong);
  font-size: 13px;
  font-weight: 800;
  min-width: 36px;
  text-align: right;
}
.device-collapsible[open] .when-closed,
.device-collapsible:not([open]) .when-open {
  display: none;
}
.device-content {
  border-top: 1px solid var(--border);
  padding: 14px;
}
.device-bars {
  display: grid;
  gap: 12px;
}
.device-row-head {
  align-items: center;
  display: flex;
  gap: 10px;
  justify-content: space-between;
  margin-bottom: 6px;
}
.device-row-head span {
  color: var(--text);
  font-weight: 800;
}
.device-row-head b {
  color: var(--muted);
  font-size: 13px;
  font-weight: 700;
}
.device-track {
  background: #e5edf7;
  border-radius: 999px;
  height: 10px;
  overflow: hidden;
}
.device-track span {
  background: #2563eb;
  border-radius: inherit;
  display: block;
  height: 100%;
}
.compact-metrics {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  margin: 0;
}
.compact-metrics div {
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
}
.compact-metrics dt {
  color: var(--muted);
  font-size: 13px;
  margin: 0;
}
.compact-metrics dd {
  color: var(--text);
  font-size: 22px;
  font-weight: 850;
  line-height: 1.1;
  margin: 4px 0 0;
}
.trend-section {
  border-bottom: 1px solid var(--border);
  margin-bottom: 18px;
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
  background: var(--surface-subtle);
  border: 1px solid var(--border);
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
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}
.chart-wrap {
  background: var(--chart-surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  padding: 14px 14px 8px;
  position: relative;
}
.line-chart-svg {
  display: block;
  height: 240px;
  width: 100%;
}
.chart-grid-line {
  stroke: #dbeafe;
  stroke-width: 1;
  stroke-dasharray: 5 5;
  vector-effect: non-scaling-stroke;
}
.chart-y-tick {
  stroke: #94a3b8;
  stroke-linecap: round;
  stroke-width: 1.4;
  vector-effect: non-scaling-stroke;
}
.chart-axis-text {
  fill: #64748b;
  font-size: 11px;
  font-weight: 700;
}
.chart-y-axis-text {
  fill: #334155;
  font-size: 12px;
  font-weight: 850;
  paint-order: stroke fill;
  stroke: rgba(248, 250, 252, 0.92);
  stroke-linejoin: round;
  stroke-width: 3px;
}
.chart-x-tick {
  stroke: #94a3b8;
  stroke-linecap: round;
  stroke-width: 1.2;
  vector-effect: non-scaling-stroke;
}
.chart-x-axis-text {
  fill: #475569;
  font-size: 11px;
  font-weight: 750;
  paint-order: stroke fill;
  stroke: rgba(248, 250, 252, 0.84);
  stroke-linejoin: round;
  stroke-width: 2.5px;
}
.chart-x-axis-text.dense {
  font-size: 10px;
  font-weight: 800;
}
.chart-area {
  fill: url(#merchantTrendArea);
}
.trend-line {
  filter: drop-shadow(0 8px 12px rgba(37, 99, 235, 0.14));
  stroke: #2563eb;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 3.5;
  vector-effect: non-scaling-stroke;
}
.chart-point-group {
  cursor: pointer;
  outline: none;
}
.chart-point-group:focus .trend-dot {
  fill: #2563eb;
  stroke: #fff;
  stroke-width: 3;
}
.chart-hover-line {
  stroke: #bfdbfe;
  stroke-dasharray: 4 4;
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}
.trend-dot {
  fill: #fff;
  stroke: #2563eb;
  stroke-width: 2.4;
  transition: fill 0.18s ease, r 0.18s ease, stroke 0.18s ease;
  vector-effect: non-scaling-stroke;
}
.trend-dot:hover {
  fill: #2563eb;
  stroke: #fff;
}
.chart-tooltip {
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 8px;
  box-sizing: border-box;
  box-shadow: var(--shadow-popover);
  color: var(--text);
  display: grid;
  gap: 2px;
  font-size: 12px;
  max-width: min(220px, calc(100% - 24px));
  opacity: 0;
  padding: 8px 10px;
  pointer-events: none;
  position: absolute;
  transform: translate(12px, -100%);
  transition: opacity 0.18s ease;
  width: max-content;
  z-index: 2;
}
.chart-tooltip.visible {
  opacity: 1;
}
.chart-tooltip strong {
  font-size: 12px;
}
.chart-tooltip span {
  color: var(--muted);
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
  border-top: 1px solid var(--border);
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
  background: var(--surface-subtle);
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
  color: var(--primary-strong);
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
.fold-card summary:hover {
  background: var(--surface-subtle);
}
.fold-card[open] summary {
  background: var(--surface-subtle);
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
  border-top: 1px solid var(--border);
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
  background: var(--primary-soft);
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
  .platform-filter,
  .metric-zone,
  .dashboard-stack,
  .optimization-panel {
    grid-template-columns: 1fr;
  }
  .analytics-meta {
    justify-items: start;
  }
  .platform-select-wrap {
    align-items: stretch;
    flex-direction: column;
  }
  .platform-select-wrap select {
    width: 100%;
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
