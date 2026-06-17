<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { publicApi } from '../../api/public'

type Keyword = { id: number; keyword: string }

type LandingData = {
  sessionId: string
  storeName: string
  primaryPlatformStyle: string
  review: { id: number; content: string }
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
// 顾客可在发布前编辑成自己的话
const editedContent = ref('')

// 文案变化时，把可编辑框同步成最新文案
watch(
  () => payload.value?.review.content,
  (v) => {
    if (v != null) editedContent.value = v
  }
)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await publicApi.initLanding(String(route.params.token))
    payload.value = data.data
    if (payload.value) {
      editedContent.value = payload.value.review.content
      await publicApi.createEvent(String(route.params.token), {
        sessionId: payload.value.sessionId,
        reviewItemId: payload.value.review.id,
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
  if (!payload.value || switching.value) return
  selectedTag.value = keyword
  await fetchReview(keyword, 'review_pick_by_tag')
}

async function switchReview() {
  if (!payload.value || switching.value) return
  await fetchReview(selectedTag.value, 'review_switch')
}

async function fetchReview(tag: string, action: string) {
  if (!payload.value) return
  switching.value = true
  try {
    const { data } = await publicApi.switchReview(String(route.params.token), {
      tag: tag || undefined,
      sessionId: payload.value.sessionId
    })
    payload.value.review = data.data.review
    payload.value.remainingDispatchableCount = data.data.remainingDispatchableCount
    await publicApi.createEvent(String(route.params.token), {
      sessionId: payload.value.sessionId,
      reviewItemId: payload.value.review.id,
      actionType: action
      // 不把关键词标签塞进 platformCode（那是给平台跳转事件用的），避免污染分析
    })
  } catch (err: any) {
    alert(err?.response?.data?.message || '暂无推荐文案，请稍后再试')
  } finally {
    switching.value = false
  }
}

async function copyReview() {
  const text = editedContent.value.trim()
  if (!payload.value || !text) return
  try {
    await navigator.clipboard.writeText(text)
    await publicApi.createEvent(String(route.params.token), {
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
  if (!payload.value || !text) return
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    alert('文案未自动复制，请手动复制后发布')
  }
  await publicApi.createEvent(String(route.params.token), {
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
        <p class="muted">主平台风格：{{ payload.primaryPlatformStyle }}</p>

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
        <p class="muted" style="margin-top: 12px">剩余可发放文案：{{ payload.remainingDispatchableCount }}</p>
      </div>

      <div class="card" v-if="payload.images.length">
        <h2>配图素材</h2>
        <div class="row">
          <a v-for="image in payload.images" :key="image.id" :href="image.imageUrl || image.url" target="_blank" rel="noreferrer">
            <img :src="image.thumbnailUrl || image.imageUrl || image.url" style="width: 180px; height: 120px; object-fit: cover; border-radius: 12px" />
          </a>
        </div>
      </div>

      <div class="card" v-if="payload.platformLinks.length">
        <h2>去平台发布</h2>
        <div class="row">
          <button v-for="link in payload.platformLinks" :key="link.id" :disabled="!editedContent.trim()" @click="jump(link)">{{ link.buttonText }}</button>
        </div>
      </div>
    </template>
  </div>
</template>
