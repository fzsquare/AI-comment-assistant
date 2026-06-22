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
  <div class="page" style="max-width: 720px">
    <div class="card" v-if="loading">加载中...</div>
    <div class="card" v-else-if="error">{{ error }}</div>
    <template v-else-if="payload">
      <div class="card">
        <h1>{{ payload.storeName }}</h1>
        <p class="muted">请选择你要发布评价的平台</p>

        <div v-if="payload.platformLinks.length" class="platform-grid">
          <button
            v-for="link in payload.platformLinks"
            :key="link.id"
            :class="{ secondary: selectedPlatform?.platformCode !== link.platformCode }"
            :disabled="switching"
            @click="choosePlatform(link)"
          >
            {{ link.platformName || link.buttonText }}
          </button>
        </div>
        <p v-else class="muted">当前门店还没有配置可用平台，请联系商家。</p>
      </div>

      <div class="card" v-if="selectedPlatform && payload.review">
        <h2>{{ selectedPlatform.platformName }}评价文案</h2>
        <div v-if="payload.keywords && payload.keywords.length" style="margin: 12px 0;">
          <p class="muted" style="margin-bottom: 8px;">你点了什么 / 体验如何？选一个，帮你生成更贴合的评价：</p>
          <div class="row" style="flex-wrap: wrap; gap: 8px;">
            <button
              v-for="kw in payload.keywords"
              :key="kw.id"
              :class="{ secondary: selectedTag !== kw.keyword }"
              :disabled="switching"
              @click="pickByTag(kw.keyword)"
            >{{ kw.keyword }}</button>
          </div>
        </div>

        <p class="muted" style="margin: 12px 0 6px;">推荐文案（可改成你自己的话再发）：</p>
        <textarea
          v-model="editedContent"
          rows="8"
          style="width: 100%; font-size: 17px; line-height: 1.7; padding: 12px; border: 1px solid #ddd; border-radius: 12px; box-sizing: border-box; resize: vertical;"
        ></textarea>

        <div class="row" style="margin-top: 12px;">
          <button :disabled="!editedContent.trim()" @click="copyReview">复制文案</button>
          <button class="secondary" :disabled="switching" @click="switchReview">换一换</button>
        </div>
        <p class="muted" style="margin-top: 12px">{{ selectedPlatform.platformName }}剩余可发放文案：{{ payload.remainingDispatchableCount }}</p>
      </div>

      <div class="card" v-if="payload.images.length">
        <h2>配图素材</h2>
        <div class="row">
          <a v-for="image in payload.images" :key="image.id" :href="image.imageUrl || image.url" target="_blank" rel="noreferrer">
            <img :src="image.thumbnailUrl || image.imageUrl || image.url" style="width: 180px; height: 120px; object-fit: cover; border-radius: 12px" />
          </a>
        </div>
      </div>

      <div class="card" v-if="selectedPlatform && payload.review">
        <h2>去平台发布</h2>
        <div class="row">
          <button :disabled="!editedContent.trim()" @click="jump(selectedPlatform)">{{ selectedPlatform.buttonText }}</button>
        </div>
      </div>
    </template>
  </div>
</template>
