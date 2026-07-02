<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { publicApi } from '../../api/public'
import { openPlatform } from '../../utils/deeplink'
import { copyToClipboard } from '../../utils/clipboard'

type Keyword = { id: number; keyword: string }

type LandingData = {
  sessionId: string
  storeName: string
  primaryPlatformStyle: string
  review?: { id: number; content: string; platformStyle?: string } | null
  keywords: Keyword[]
  images: Array<{ id: number; imageUrl?: string; url?: string; thumbnailUrl?: string }>
  platformLinks: Array<{ id: number; platformCode: string; platformName: string; buttonText: string; targetUrl: string; backupUrl?: string }>
  remainingDispatchableCount: number
}

const route = useRoute()
const loading = ref(true)
const switching = ref(false)
const error = ref('')
const reviewStateMessage = ref('')
const payload = ref<LandingData | null>(null)
const selectedTag = ref('')
const selectedPlatform = ref<LandingData['platformLinks'][number] | null>(null)
// 顾客可在发布前编辑成自己的话
const editedContent = ref('')
const acceptedReviewIds = ref<number[]>([])

function platformDisplayName(link?: LandingData['platformLinks'][number] | null) {
  return link?.platformName || link?.buttonText || '平台'
}

const platformActionLabel = computed(() => `复制并打开${platformDisplayName(selectedPlatform.value)}`)
const tagSummaryLabel = computed(() => selectedTag.value ? `已按“${selectedTag.value}”调整` : '可选：按这次体验换个角度')

async function trackEvent(payloadData: Record<string, unknown>) {
  try {
    await publicApi.createEvent(String(route.params.token), {
      ...payloadData,
      clientUserAgent: navigator.userAgent || ''
    })
  } catch (err) {
    console.warn('event tracking failed', err)
  }
}

// 文案变化时，把可编辑框同步成最新文案
watch(
  () => payload.value?.review?.content,
  (v) => {
    editedContent.value = v || ''
  }
)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await publicApi.initLanding(String(route.params.token))
    payload.value = data.data
    if (payload.value) {
      selectedPlatform.value = null
      selectedTag.value = ''
      editedContent.value = ''
      reviewStateMessage.value = ''
      await trackEvent({
        sessionId: payload.value.sessionId,
        actionType: 'page_view'
      })
    }
  } catch (err: any) {
    error.value = err?.response?.data?.message || '页面加载失败'
  } finally {
    loading.value = false
  }
}

// 顾客点“我点了什么”→ 取一条对应标签的文案
async function pickByTag(keyword: string) {
  if (!payload.value || !selectedPlatform.value || switching.value) return
  selectedTag.value = keyword
  await fetchReview(keyword, 'review_pick_by_tag')
}

async function switchReview() {
  if (!payload.value || !selectedPlatform.value || switching.value) return
  const currentReview = payload.value.review
  if (currentReview && !acceptedReviewIds.value.includes(currentReview.id)) {
    await trackEvent({
      sessionId: payload.value.sessionId,
      reviewItemId: currentReview.id,
      actionType: 'review_reject',
      platformCode: selectedPlatform.value.platformCode,
      editedContent: editedContent.value.trim()
    })
  }
  await fetchReview(selectedTag.value, 'review_switch')
}

async function choosePlatform(link: LandingData['platformLinks'][number]) {
  if (!payload.value || switching.value) return
  if (selectedPlatform.value?.platformCode === link.platformCode && payload.value.review) return
  selectedPlatform.value = link
  acceptedReviewIds.value = []
  selectedTag.value = ''
  payload.value.review = null
  editedContent.value = ''
  reviewStateMessage.value = ''
  await fetchReview('', 'platform_select')
}

async function fetchReview(tag: string, action: string) {
  if (!payload.value || !selectedPlatform.value) return
  switching.value = true
  reviewStateMessage.value = ''
  try {
    const { data } = await publicApi.switchReview(String(route.params.token), {
      platformCode: selectedPlatform.value.platformCode,
      tag: tag || undefined,
      sessionId: payload.value.sessionId
    })
    const review = data.data.review
    payload.value.review = review
    payload.value.remainingDispatchableCount = data.data.remainingDispatchableCount
    reviewStateMessage.value = ''
    await trackEvent({
      sessionId: payload.value.sessionId,
      reviewItemId: review.id,
      actionType: action,
      platformCode: selectedPlatform.value.platformCode
    })
  } catch (err: any) {
    reviewStateMessage.value = err?.response?.data?.message || '暂时没有可用文案，请找店员处理。'
  } finally {
    switching.value = false
  }
}

async function copyReview() {
  const text = editedContent.value.trim()
  if (!payload.value || !payload.value.review || !text) return
  const ok = await copyToClipboard(text)
  if (ok) {
    acceptedReviewIds.value = [...acceptedReviewIds.value, payload.value.review.id]
    await trackEvent({
      sessionId: payload.value.sessionId,
      reviewItemId: payload.value.review.id,
      actionType: 'review_copy',
      platformCode: selectedPlatform.value?.platformCode,
      editedContent: text
    })
    alert('已复制，可以粘贴到平台里再改。')
  } else {
    alert('复制失败，请手动长按选中文案复制')
  }
}

async function jump(link: { platformCode: string; targetUrl: string; backupUrl?: string }) {
  const text = editedContent.value.trim()
  if (!payload.value || !payload.value.review || !text) return
  if (!link.targetUrl && !link.backupUrl) {
    reviewStateMessage.value = '该平台暂时没有店铺链接，请换一个来源或联系店员。'
    return
  }
  if (!(await copyToClipboard(text))) {
    alert('文案未自动复制，请长按文案手动复制')
  }
  await trackEvent({
    sessionId: payload.value.sessionId,
    reviewItemId: payload.value.review.id,
    actionType: 'platform_link_click',
    platformCode: link.platformCode,
    editedContent: text
  })
  acceptedReviewIds.value = [...acceptedReviewIds.value, payload.value.review.id]
  // deeplink 唤起对应 App，唤不起回退商家网页链接
  openPlatform(link.platformCode, link.targetUrl, link.backupUrl)
}

onMounted(load)
</script>

<template>
  <div class="landing">
    <!-- 加载骨架屏：开得快、不白屏 -->
    <template v-if="loading">
      <div class="card sk">
        <div class="sk-line w50"></div>
        <div class="sk-line w30"></div>
      </div>
      <div class="card sk">
        <div class="sk-line w40"></div>
        <div class="sk-pill-row">
          <span class="sk-pill"></span><span class="sk-pill"></span><span class="sk-pill"></span>
        </div>
        <div class="sk-block"></div>
      </div>
    </template>

    <section class="card error-card" v-else-if="error" aria-labelledby="landing-error-title">
      <div class="error-mark" aria-hidden="true">!</div>
      <h1 id="landing-error-title">该商家暂不可用</h1>
      <p>{{ error || '当前商家未激活或服务器落地页不可访问。' }}</p>
      <dl class="error-meta">
        <div>
          <dt>访问路由</dt>
          <dd>/landing/{{ route.params.token }}</dd>
        </div>
        <div>
          <dt>处理方式</dt>
          <dd>停止评价流程，不跳转商家官方链接</dd>
        </div>
      </dl>
    </section>

    <template v-else-if="payload">
      <header class="store-head">
        <h1 class="store-name">{{ payload.storeName }}</h1>
        <p class="sub">系统先帮你整理一版评价草稿，你可以改成自己的话</p>
      </header>

      <!-- 步骤 1：选平台 -->
      <section class="card">
        <h2 class="step"><span class="no">1</span>选择你想评价的平台</h2>
        <div v-if="payload.platformLinks.length" class="choice-grid">
          <button
            v-for="link in payload.platformLinks"
            :key="link.id"
            class="choice"
            :class="{ active: selectedPlatform?.platformCode === link.platformCode }"
            :disabled="switching"
            :aria-pressed="selectedPlatform?.platformCode === link.platformCode"
            @click="choosePlatform(link)"
          >{{ link.platformName || link.buttonText }}</button>
        </div>
        <p v-else class="muted">当前门店还没有配置可用平台，请联系商家。</p>
      </section>

      <section class="card review-status-card" v-if="selectedPlatform && !payload.review && (switching || reviewStateMessage)" role="status">
        <h2 class="step"><span class="no">2</span>{{ switching ? '正在整理评价' : '暂时没有可用文案' }}</h2>
        <p class="helper">
          {{ switching ? '我们正在按你选择的平台准备一版可编辑草稿。' : reviewStateMessage }}
        </p>
      </section>

      <!-- 步骤 2：直接给出可编辑文案，体验标签折叠为可选项 -->
      <section class="card review-card" v-if="selectedPlatform && payload.review">
        <div class="review-heading">
          <h2 class="step"><span class="no">2</span>评价草稿</h2>
          <span class="platform-badge">{{ platformDisplayName(selectedPlatform) }}</span>
        </div>
        <p class="helper">可直接用，也可以改成你自己的话。</p>
        <div class="review-wrap" :class="{ busy: switching }">
          <textarea
            v-model="editedContent"
            class="review-box"
            rows="9"
            :disabled="switching"
            placeholder="正在整理…"
          ></textarea>
          <div class="busy-mask" v-if="switching" role="status">换文案中…</div>
        </div>
        <details class="tag-panel" v-if="payload.keywords && payload.keywords.length">
          <summary>{{ tagSummaryLabel }}</summary>
          <div class="chips">
            <button
              v-for="kw in payload.keywords"
              :key="kw.id"
              class="chip"
              :class="{ active: selectedTag === kw.keyword }"
              :disabled="switching"
              :aria-pressed="selectedTag === kw.keyword"
              @click="pickByTag(kw.keyword)"
            >{{ kw.keyword }}</button>
          </div>
        </details>
        <div class="actions">
          <button class="act-go" :disabled="!editedContent.trim() || switching" @click="jump(selectedPlatform)">
            {{ platformActionLabel }}
          </button>
          <div class="act-row secondary-actions">
            <button class="act-switch" :disabled="switching" @click="switchReview">换个说法</button>
            <button class="act-copy" :disabled="!editedContent.trim() || switching" @click="copyReview">仅复制</button>
          </div>
        </div>
      </section>

      <!-- 配图素材 -->
      <section class="card" v-if="payload.images.length">
        <h2 class="step">配图素材（长按图片保存）</h2>
        <div class="img-row">
          <a
            v-for="image in payload.images"
            :key="image.id"
            :href="image.imageUrl || image.url"
            target="_blank"
            rel="noreferrer"
          >
            <img :src="image.thumbnailUrl || image.imageUrl || image.url" loading="lazy" alt="店铺配图" />
          </a>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.landing {
  max-width: 640px;
  margin: 0 auto;
  padding: 16px 14px;
  padding-left: max(14px, env(safe-area-inset-left));
  padding-right: max(14px, env(safe-area-inset-right));
  padding-bottom: calc(24px + env(safe-area-inset-bottom));
  min-height: 100dvh;
}

.store-head {
  padding: 4px 4px 12px;
}
.store-name {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
}
.sub {
  margin: 6px 0 0;
  color: #667085;
  font-size: 14px;
  line-height: 1.5;
}

.card {
  background: #fff;
  border-radius: 16px;
  padding: 18px 16px;
  border: 1px solid rgba(219, 228, 240, 0.72);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.05);
  margin-bottom: 14px;
}
.error-card {
  margin-top: 18vh;
  padding: 28px 24px;
}
.error-card h1 {
  color: #111827;
  font-size: 22px;
  line-height: 1.3;
  margin: 18px 0 8px;
}
.error-card p {
  color: #667085;
  font-size: 14px;
  line-height: 1.7;
  margin: 0;
}
.error-mark {
  align-items: center;
  background: #fef2f2;
  border-radius: 12px;
  color: #b42318;
  display: inline-flex;
  font-size: 24px;
  font-weight: 800;
  height: 52px;
  justify-content: center;
  width: 52px;
}
.error-meta {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  display: grid;
  gap: 10px;
  margin: 18px 0 0;
  padding: 12px;
}
.error-meta div {
  display: grid;
  gap: 8px;
  grid-template-columns: 84px minmax(0, 1fr);
}
.error-meta dt {
  color: #667085;
  font-size: 12px;
}
.error-meta dd {
  color: #111827;
  font-size: 12px;
  font-weight: 700;
  margin: 0;
  overflow-wrap: anywhere;
}
.muted {
  color: #667085;
}

.step {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 14px;
  font-size: 16px;
  font-weight: 600;
}
.step.mt {
  margin-top: 18px;
}
.step .no {
  flex: 0 0 auto;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #3b82f6;
  color: #fff;
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

/* 平台选择 */
.choice-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 10px;
}
.choice {
  min-height: 54px;
  border: 1.5px solid #e2e8f0;
  border-radius: 12px;
  background: #f8fafc;
  color: #1f2937;
  font-size: 15px;
  font-weight: 600;
  touch-action: manipulation;
}
.choice.active {
  border-color: #3b82f6;
  background: #eff6ff;
  color: #1d4ed8;
}
.choice:disabled {
  opacity: 0.6;
}

.review-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.review-heading .step {
  margin-bottom: 0;
}
.platform-badge {
  flex: 0 0 auto;
  padding: 5px 10px;
  border-radius: 999px;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: 13px;
  font-weight: 600;
}
.helper {
  margin: 10px 0 12px;
  color: #667085;
  font-size: 14px;
  line-height: 1.5;
}

/* 体验标签 chips */
.tag-panel {
  margin-top: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  background: #f8fafc;
}
.tag-panel summary {
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
  color: #475569;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  list-style: none;
}
.tag-panel summary::-webkit-details-marker {
  display: none;
}
.tag-panel summary::after {
  content: '展开';
  color: #1d4ed8;
  font-size: 13px;
  font-weight: 600;
}
.tag-panel[open] summary {
  border-bottom: 1px solid #e2e8f0;
}
.tag-panel[open] summary::after {
  content: '收起';
}
.tag-panel .chips {
  padding: 12px;
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.chip {
  min-height: 44px;
  padding: 8px 16px;
  border-radius: 999px;
  border: 1.5px solid #e2e8f0;
  background: #fff;
  color: #334155;
  font-size: 14px;
  touch-action: manipulation;
}
.chip.active {
  border-color: #3b82f6;
  background: #3b82f6;
  color: #fff;
}

/* 文案框 */
.review-wrap {
  position: relative;
}
.review-box {
  width: 100%;
  min-height: 180px;
  font-size: 16px;
  line-height: 1.75;
  padding: 14px;
  border: 1px solid #dbe2ea;
  border-radius: 14px;
  background: #fcfdff;
  resize: vertical;
}
.busy-mask {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.65);
  border-radius: 14px;
  color: #475569;
  font-size: 14px;
}
/* 文案下方的操作组：全部在正常文档流里，无固定浮层 → 不会压住任何内容 */
.actions {
  margin-top: 14px;
}
.act-switch {
  flex: 1 1 0;
  min-height: 48px;
  background: #f1f5f9;
  color: #334155;
  border: none;
  border-radius: 12px;
  font-size: 15px;
  margin-bottom: 0;
}
.act-row {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}
.act-copy {
  flex: 1 1 0;
  min-height: 48px;
  background: #f1f5f9;
  color: #334155;
  border: none;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
}
.act-go {
  width: 100%;
  min-height: 52px;
  background: #3b82f6;
  color: #fff;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
}
.actions button:disabled {
  opacity: 0.5;
}

/* 配图横向滑动 */
.img-row {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  padding-bottom: 4px;
}
.img-row a {
  flex: 0 0 auto;
}
.img-row img {
  width: 150px;
  height: 110px;
  object-fit: cover;
  border-radius: 12px;
  flex: 0 0 auto;
}

/* 骨架屏 */
.sk-line,
.sk-pill,
.sk-block {
  background: linear-gradient(90deg, #eef1f6 25%, #e3e8f0 37%, #eef1f6 63%);
  background-size: 400% 100%;
  animation: sk 1.3s ease infinite;
  border-radius: 8px;
}
.sk-line {
  height: 16px;
  margin-bottom: 12px;
}
.w50 { width: 50%; }
.w40 { width: 40%; }
.w30 { width: 30%; }
.sk-pill-row {
  display: flex;
  gap: 10px;
  margin: 14px 0;
}
.sk-pill {
  width: 64px;
  height: 32px;
  border-radius: 999px;
}
.sk-block {
  height: 150px;
  border-radius: 14px;
}
@keyframes sk {
  0% { background-position: 100% 50%; }
  100% { background-position: 0 50%; }
}

@media (max-width: 380px) {
  .act-row {
    flex-direction: column;
  }
  .act-copy {
    flex: 1 1 auto;
  }
}
</style>
