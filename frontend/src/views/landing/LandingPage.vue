<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { publicApi } from '../../api/public'

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
const payload = ref<LandingData | null>(null)
const selectedTag = ref('')
const selectedPlatform = ref<LandingData['platformLinks'][number] | null>(null)
// 顾客可在发布前编辑成自己的话
const editedContent = ref('')

async function trackEvent(payloadData: Record<string, unknown>) {
  try {
    await publicApi.createEvent(String(route.params.token), payloadData)
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
  await fetchReview(selectedTag.value, 'review_switch')
}

async function choosePlatform(link: LandingData['platformLinks'][number]) {
  if (!payload.value || switching.value) return
  if (selectedPlatform.value?.platformCode === link.platformCode && payload.value.review) return
  selectedPlatform.value = link
  selectedTag.value = ''
  payload.value.review = null
  editedContent.value = ''
  await fetchReview('', 'platform_select')
}

async function fetchReview(tag: string, action: string) {
  if (!payload.value || !selectedPlatform.value) return
  switching.value = true
  try {
    const { data } = await publicApi.switchReview(String(route.params.token), {
      platformCode: selectedPlatform.value.platformCode,
      tag: tag || undefined,
      sessionId: payload.value.sessionId
    })
    const review = data.data.review
    payload.value.review = review
    payload.value.remainingDispatchableCount = data.data.remainingDispatchableCount
    await trackEvent({
      sessionId: payload.value.sessionId,
      reviewItemId: review.id,
      actionType: action,
      platformCode: selectedPlatform.value.platformCode
    })
  } catch (err: any) {
    alert(err?.response?.data?.message || '暂无推荐文案，请稍后再试')
  } finally {
    switching.value = false
  }
}

async function copyReview() {
  const text = editedContent.value.trim()
  if (!payload.value || !payload.value.review || !text) return
  try {
    await navigator.clipboard.writeText(text)
    await trackEvent({
      sessionId: payload.value.sessionId,
      reviewItemId: payload.value.review.id,
      actionType: 'review_copy'
    })
    alert('已复制，可直接去平台发布')
  } catch {
    alert('复制失败，请手动长按复制')
  }
}

async function jump(link: { platformCode: string; targetUrl: string; backupUrl?: string }) {
  const text = editedContent.value.trim()
  if (!payload.value || !payload.value.review || !text) return
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    alert('文案未自动复制，请手动复制后发布')
  }
  await trackEvent({
    sessionId: payload.value.sessionId,
    reviewItemId: payload.value.review.id,
    actionType: 'platform_link_click',
    platformCode: link.platformCode
  })
  try {
    window.location.href = link.targetUrl
  } catch {
    if (link.backupUrl) {
      window.location.href = link.backupUrl
      return
    }
    alert('跳转失败，请稍后重试')
  }
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

    <div class="card error-card" v-else-if="error">{{ error }}</div>

    <template v-else-if="payload">
      <header class="store-head">
        <h1 class="store-name">{{ payload.storeName }}</h1>
        <p class="sub">碰一碰，几秒生成一条你的真实评价，可改成自己的话再发</p>
      </header>

      <!-- 步骤 1：选平台 -->
      <section class="card">
        <h2 class="step"><span class="no">1</span>选择要发布的平台</h2>
        <div v-if="payload.platformLinks.length" class="choice-grid">
          <button
            v-for="link in payload.platformLinks"
            :key="link.id"
            class="choice"
            :class="{ active: selectedPlatform?.platformCode === link.platformCode }"
            :disabled="switching"
            @click="choosePlatform(link)"
          >{{ link.platformName || link.buttonText }}</button>
        </div>
        <p v-else class="muted">当前门店还没有配置可用平台，请联系商家。</p>
      </section>

      <!-- 步骤 2 + 3：选体验 + 文案 -->
      <section class="card" v-if="selectedPlatform && payload.review">
        <h2 class="step"><span class="no">2</span>你点了什么 / 体验如何</h2>
        <div v-if="payload.keywords && payload.keywords.length" class="chips">
          <button
            v-for="kw in payload.keywords"
            :key="kw.id"
            class="chip"
            :class="{ active: selectedTag === kw.keyword }"
            :disabled="switching"
            @click="pickByTag(kw.keyword)"
          >{{ kw.keyword }}</button>
        </div>

        <h2 class="step mt"><span class="no">3</span>推荐文案（可改成你自己的话）</h2>
        <div class="review-wrap" :class="{ busy: switching }">
          <textarea
            v-model="editedContent"
            class="review-box"
            rows="9"
            :disabled="switching"
            placeholder="正在生成…"
          ></textarea>
          <div class="busy-mask" v-if="switching">换文案中…</div>
        </div>
        <div class="actions">
          <button class="act-switch" :disabled="switching" @click="switchReview">🔄 换一换</button>
          <div class="act-row">
            <button class="act-copy" :disabled="!editedContent.trim() || switching" @click="copyReview">复制</button>
            <button class="act-go" :disabled="!editedContent.trim() || switching" @click="jump(selectedPlatform)">
              {{ selectedPlatform.buttonText || '去发布' }}
            </button>
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
            <img :src="image.thumbnailUrl || image.imageUrl || image.url" loading="lazy" alt="" />
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
  padding-bottom: calc(24px + env(safe-area-inset-bottom));
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
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.05);
  margin-bottom: 14px;
}
.error-card {
  color: #991b1b;
  text-align: center;
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
  min-height: 52px;
  border: 1.5px solid #e2e8f0;
  border-radius: 12px;
  background: #f8fafc;
  color: #1f2937;
  font-size: 15px;
  font-weight: 600;
}
.choice.active {
  border-color: #3b82f6;
  background: #eff6ff;
  color: #1d4ed8;
}
.choice:disabled {
  opacity: 0.6;
}

/* 体验标签 chips */
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.chip {
  min-height: 40px;
  padding: 8px 16px;
  border-radius: 999px;
  border: 1.5px solid #e2e8f0;
  background: #fff;
  color: #334155;
  font-size: 14px;
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
  width: 100%;
  min-height: 44px;
  background: #eef2ff;
  color: #1e40af;
  border: none;
  border-radius: 12px;
  font-size: 15px;
  margin-bottom: 10px;
}
.act-row {
  display: flex;
  gap: 10px;
}
.act-copy {
  flex: 0 0 96px;
  min-height: 50px;
  background: #eef2ff;
  color: #1e40af;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
}
.act-go {
  flex: 1;
  min-height: 50px;
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
</style>
