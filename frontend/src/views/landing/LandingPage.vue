<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { publicApi } from '../../api/public'

type LandingData = {
  sessionId: string
  storeName: string
  primaryPlatformStyle: string
  review: { id: number; content: string }
  images: Array<{ id: number; imageUrl?: string; url?: string; thumbnailUrl?: string }>
  platformLinks: Array<{ id: number; platformCode: string; platformName: string; buttonText: string; targetUrl: string; backupUrl?: string }>
  remainingDispatchableCount: number
}

const route = useRoute()
const loading = ref(true)
const error = ref('')
const payload = ref<LandingData | null>(null)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await publicApi.initLanding(String(route.params.token))
    payload.value = data.data
    if (payload.value) {
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

async function copyReview() {
  if (!payload.value) return
  try {
    await navigator.clipboard.writeText(payload.value.review.content)
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

async function switchReview() {
  if (!payload.value) return
  try {
    const { data } = await publicApi.switchReview(String(route.params.token), {
      currentReviewId: payload.value.review.id,
      sessionId: payload.value.sessionId
    })
    payload.value.review = data.data.review
    payload.value.remainingDispatchableCount = data.data.remainingDispatchableCount
    await publicApi.createEvent(String(route.params.token), {
      sessionId: payload.value.sessionId,
      reviewItemId: payload.value.review.id,
      actionType: 'review_switch'
    })
  } catch (err: any) {
    alert(err?.response?.data?.message || '暂无推荐文案，请稍后再试')
  }
}

async function jump(link: { platformCode: string; targetUrl: string; backupUrl?: string }) {
  if (!payload.value) return
  try {
    await navigator.clipboard.writeText(payload.value.review.content)
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
        <div style="white-space: pre-wrap; font-size: 18px; line-height: 1.7; margin: 16px 0;">{{ payload.review.content }}</div>
        <div class="row">
          <button @click="copyReview">复制文案</button>
          <button class="secondary" @click="switchReview">换一换</button>
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
          <button v-for="link in payload.platformLinks" :key="link.id" @click="jump(link)">{{ link.buttonText }}</button>
        </div>
      </div>
    </template>
  </div>
</template>
