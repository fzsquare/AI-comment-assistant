<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { publicApi, type LandingData, type LandingPlatformLink } from '../../api/public'
import {
  ensureLandingSession,
  markLandingPageViewed,
  selectLandingPlatform,
  trackLandingEvent
} from './landingFlow'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const navigating = ref(false)
const error = ref('')
const payload = ref<LandingData | null>(null)

function token() {
  return String(route.params.token || '')
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await publicApi.initLanding(token())
    payload.value = data.data
    const session = ensureLandingSession(token(), data.data.sessionId)
    if (!session.pageViewTracked) {
      // 先标记本 tab 已尝试上报，避免慢请求期间刷新造成重复 page_view。
      markLandingPageViewed(token())
      void trackLandingEvent(token(), session.sessionId, { actionType: 'page_view' })
    }
  } catch (loadError: any) {
    error.value = loadError?.response?.data?.message || '页面加载失败，请稍后重试。'
  } finally {
    loading.value = false
  }
}

async function choosePlatform(link: LandingPlatformLink) {
  if (navigating.value) return
  const session = selectLandingPlatform(token(), link.platformCode)
  if (!session) {
    error.value = '当前会话已失效，请刷新页面后重试。'
    return
  }
  navigating.value = true
  void trackLandingEvent(token(), session.sessionId, {
    actionType: 'platform_select',
    platformCode: link.platformCode
  })
  await router.push({
    name: 'landing-review',
    params: { token: token(), platformCode: link.platformCode }
  })
}

onMounted(load)
</script>

<template>
  <main class="landing-page" aria-labelledby="platform-page-title">
    <template v-if="loading">
      <header class="landing-store-head" aria-busy="true" aria-label="正在加载门店和评价平台">
        <span class="landing-skeleton landing-skeleton-title"></span>
        <span class="landing-skeleton landing-skeleton-copy"></span>
      </header>
      <section class="landing-panel landing-platform-panel">
        <span class="landing-skeleton landing-skeleton-heading"></span>
        <span v-for="index in 3" :key="index" class="landing-skeleton landing-skeleton-button"></span>
      </section>
    </template>

    <section v-else-if="error" class="landing-panel landing-error" role="alert">
      <span class="landing-error-mark" aria-hidden="true">!</span>
      <h1 id="platform-page-title">该门店暂不可用</h1>
      <p>{{ error }}</p>
      <button type="button" class="landing-secondary-button" @click="load">重新加载</button>
    </section>

    <template v-else-if="payload">
      <header class="landing-store-head">
        <p class="landing-eyebrow">本次到店体验</p>
        <h1 id="platform-page-title">{{ payload.storeName }}</h1>
      </header>

      <section class="landing-panel landing-platform-panel" aria-labelledby="platform-choice-title">
        <h2 id="platform-choice-title">选择评价平台</h2>
        <div v-if="payload.platformLinks.length" class="landing-platform-list">
          <button
            v-for="link in payload.platformLinks"
            :key="link.id"
            type="button"
            class="landing-platform-button"
            :data-platform-code="link.platformCode"
            :disabled="navigating"
            @click="choosePlatform(link)"
          >
            <span>{{ link.platformName || link.buttonText }}</span>
            <span aria-hidden="true">→</span>
          </button>
        </div>
        <div v-else class="landing-empty" role="status">
          <h2>当前没有可用平台</h2>
          <p>请联系店员确认门店评价入口。</p>
        </div>
      </section>

      <p class="landing-trust">我们只帮你整理真实体验，是否发布由你决定。</p>
    </template>
  </main>
</template>

<style scoped src="./landing.css"></style>
