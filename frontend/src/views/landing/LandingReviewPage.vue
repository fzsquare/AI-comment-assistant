<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  publicApi,
  type LandingData,
  type LandingPlatformLink,
  type LandingReview,
  type LotteryDrawResult
} from '../../api/public'
import { copyToClipboard } from '../../utils/clipboard'
import { openPlatform } from '../../utils/deeplink'
import { discardLandingSession, isLandingSessionError, readLandingSession, trackLandingEvent } from './landingFlow'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const switching = ref(false)
const actionPending = ref(false)
const isEditing = ref(false)
const error = ref('')
const actionError = ref('')
const actionStatus = ref('')
const payload = ref<LandingData | null>(null)
const platform = ref<LandingPlatformLink | null>(null)
const review = ref<LandingReview | null>(null)
const selectedTag = ref('')
const editedContent = ref('')
const reviewEditor = ref<HTMLTextAreaElement | null>(null)
const reviewActions = ref<HTMLElement | null>(null)
const acceptedReviewIds = new Set<number>()
const lotteryResult = ref<LotteryDrawResult | null>(null)
const lotteryDrawing = ref(false)

const token = computed(() => String(route.params.token || ''))
const platformCode = computed(() => String(route.params.platformCode || ''))
const platformName = computed(() => platform.value?.platformName || platform.value?.buttonText || '平台')
const primaryActionLabel = computed(() => `复制并打开${platformName.value}`)
const tagSummaryLabel = computed(() => selectedTag.value ? `已按“${selectedTag.value}”调整` : '可选：换个符合体验的说法')

function platformSelectionPath() {
  return { path: `/landing/${encodeURIComponent(token.value)}` }
}

function lotteryPendingKey() {
  return `ppk-lottery-pending:${token.value}`
}

function markLotteryPending(sessionId: string) {
  sessionStorage.setItem(lotteryPendingKey(), sessionId)
}

function pendingLotterySessionId() {
  return sessionStorage.getItem(lotteryPendingKey()) || ''
}

async function drawPendingLottery() {
  const session = readLandingSession(token.value)
  const pendingSessionID = pendingLotterySessionId()
  if (!session || !pendingSessionID || pendingSessionID !== session.sessionId || lotteryDrawing.value || lotteryResult.value) return
  lotteryDrawing.value = true
  try {
    const { data } = await publicApi.drawLottery(token.value, { sessionId: session.sessionId })
    if (data.data.enabled && data.data.drawn) lotteryResult.value = data.data
    sessionStorage.removeItem(lotteryPendingKey())
  } catch (drawError: any) {
    if (isLandingSessionError(drawError)) {
      discardLandingSession(token.value, session.sessionId)
      sessionStorage.removeItem(lotteryPendingKey())
    }
  } finally {
    lotteryDrawing.value = false
  }
}

async function redirectToPlatformSelection() {
  await router.replace(platformSelectionPath())
}

function clearActionMessage() {
  actionError.value = ''
  actionStatus.value = ''
}

function keepEditorAboveActions() {
  const editorRect = reviewEditor.value?.getBoundingClientRect()
  const actionsRect = reviewActions.value?.getBoundingClientRect()
  if (!editorRect || !actionsRect) return
  const overlap = editorRect.bottom + 16 - actionsRect.top
  if (overlap > 0) window.scrollBy({ top: overlap, behavior: 'auto' })
}

async function handleEditorBlur() {
  isEditing.value = false
  await nextTick()
  requestAnimationFrame(() => requestAnimationFrame(keepEditorAboveActions))
}

async function load() {
  const session = readLandingSession(token.value)
  if (!session || session.selectedPlatformCode !== platformCode.value) {
    await redirectToPlatformSelection()
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await publicApi.initLanding(token.value)
    const selected = data.data.platformLinks.find((link) => link.platformCode === platformCode.value)
    if (!selected) {
      await redirectToPlatformSelection()
      return
    }
    payload.value = data.data
    platform.value = selected
    await fetchReview('', '')
  } catch (loadError: any) {
    error.value = loadError?.response?.data?.message || '评价内容加载失败，请稍后重试。'
  } finally {
    loading.value = false
  }
}

async function fetchReview(tag: string, actionType: string) {
  const session = readLandingSession(token.value)
  if (!session) {
    await redirectToPlatformSelection()
    return false
  }
  if (!platform.value || switching.value) return false
  switching.value = true
  clearActionMessage()
  try {
    const { data } = await publicApi.switchReview(token.value, {
      platformCode: platform.value.platformCode,
      ...(tag ? { tag } : {}),
      sessionId: session.sessionId
    })
    review.value = data.data.review
    editedContent.value = data.data.review.content
    if (actionType) {
      void trackLandingEvent(token.value, session.sessionId, {
        reviewItemId: data.data.review.id,
        actionType,
        platformCode: platform.value.platformCode
      })
    }
    return true
  } catch (reviewError: any) {
    if (isLandingSessionError(reviewError)) {
      discardLandingSession(token.value, session.sessionId)
      await redirectToPlatformSelection()
      return false
    }
    actionError.value = reviewError?.response?.data?.message || '暂时没有可用评价，请返回选择平台或联系店员。'
    return false
  } finally {
    switching.value = false
  }
}

function rejectCurrentReviewIfNeeded() {
  const session = readLandingSession(token.value)
  if (!session || !review.value || acceptedReviewIds.has(review.value.id) || !platform.value) return
  void trackLandingEvent(token.value, session.sessionId, {
    reviewItemId: review.value.id,
    actionType: 'review_reject',
    platformCode: platform.value.platformCode,
    editedContent: editedContent.value.trim()
  })
}

async function switchReview() {
  if (switching.value) return
  rejectCurrentReviewIfNeeded()
  await fetchReview(selectedTag.value, 'review_switch')
}

async function pickByTag(keyword: string) {
  if (switching.value) return
  rejectCurrentReviewIfNeeded()
  if (await fetchReview(keyword, 'review_pick_by_tag')) {
    selectedTag.value = keyword
  }
}

function recordCopy(sessionId: string, text: string) {
  if (!review.value || !platform.value) return
  acceptedReviewIds.add(review.value.id)
  void trackLandingEvent(token.value, sessionId, {
    reviewItemId: review.value.id,
    actionType: 'review_copy',
    platformCode: platform.value.platformCode,
    editedContent: text
  })
}

async function copyReview() {
  const session = readLandingSession(token.value)
  const text = editedContent.value.trim()
  if (!session) {
    await redirectToPlatformSelection()
    return
  }
  if (!review.value || !text || actionPending.value) return
  actionPending.value = true
  clearActionMessage()
  try {
    if (!(await copyToClipboard(text))) {
      actionError.value = '未能自动复制，请长按评价内容手动复制。'
      return
    }
    actionStatus.value = `已复制，可以在${platformName.value}中粘贴并按真实体验修改。`
    recordCopy(session.sessionId, text)
  } finally {
    actionPending.value = false
  }
}

async function copyAndOpenPlatform() {
  const session = readLandingSession(token.value)
  const link = platform.value
  const text = editedContent.value.trim()
  if (!session) {
    await redirectToPlatformSelection()
    return
  }
  if (!review.value || !link || !text || actionPending.value) return
  actionPending.value = true
  clearActionMessage()
  try {
    if (!link.openUrl && !link.targetUrl && !link.backupUrl) {
      actionError.value = '该平台暂时没有可用的门店入口，请返回重新选择或联系店员。'
      return
    }
    if (!(await copyToClipboard(text))) {
      actionError.value = '未能自动复制，请长按评价内容手动复制后再试。'
      return
    }

    actionStatus.value = `已复制，正在打开${platformName.value}…`
    recordCopy(session.sessionId, text)
    void trackLandingEvent(token.value, session.sessionId, {
      reviewItemId: review.value.id,
      actionType: 'platform_link_click',
      platformCode: link.platformCode,
      editedContent: text
    })
    markLotteryPending(session.sessionId)
    openPlatform(link.platformCode, link.openUrl || link.targetUrl, link.backupUrl || link.targetUrl)
  } finally {
    actionPending.value = false
  }
}

onMounted(() => {
  void load()
  window.addEventListener('focus', drawPendingLottery)
})

onBeforeUnmount(() => window.removeEventListener('focus', drawPendingLottery))
</script>

<template>
  <main class="landing-page landing-review-page" aria-labelledby="review-page-title">
    <nav class="landing-back-row" aria-label="评价平台导航">
      <RouterLink :to="platformSelectionPath()" class="landing-back-link">← 重新选择平台</RouterLink>
    </nav>

    <template v-if="loading">
      <header class="landing-store-head" aria-busy="true" aria-label="正在加载评价内容">
        <span class="landing-skeleton landing-skeleton-title"></span>
        <span class="landing-skeleton landing-skeleton-copy"></span>
      </header>
      <section class="landing-panel">
        <span class="landing-skeleton landing-skeleton-heading"></span>
        <span class="landing-skeleton landing-skeleton-review"></span>
      </section>
    </template>

    <section v-else-if="error" class="landing-panel landing-error" role="alert">
      <span class="landing-error-mark" aria-hidden="true">!</span>
      <h1 id="review-page-title">评价内容暂不可用</h1>
      <p>{{ error }}</p>
      <button type="button" class="landing-secondary-button" @click="load">重新加载</button>
    </section>

    <template v-else-if="payload && platform">
      <header class="landing-store-head landing-review-head">
        <div>
          <p class="landing-eyebrow">{{ platformName }}</p>
          <h1 id="review-page-title">{{ payload.storeName }}</h1>
        </div>
      </header>

      <section class="landing-panel landing-review-panel" aria-labelledby="review-editor-title">
        <div class="landing-section-heading">
          <div>
            <p class="landing-eyebrow">按你的真实体验修改</p>
            <h2 id="review-editor-title">把这次体验整理成一段评价</h2>
          </div>
        </div>

        <div v-if="review" class="landing-review-wrap" :class="{ 'is-busy': switching }">
          <textarea
            ref="reviewEditor"
            v-model="editedContent"
            class="landing-review-editor"
            rows="9"
            :disabled="switching"
            aria-label="可编辑的评价内容"
            @focus="isEditing = true"
            @blur="handleEditorBlur"
          ></textarea>
          <div v-if="switching" class="landing-busy-mask" role="status">正在换个说法…</div>
        </div>
        <div v-else-if="switching" class="landing-review-placeholder" role="status">正在按{{ platformName }}的表达方式整理评价…</div>

        <details v-if="payload.keywords.length" class="landing-tag-panel">
          <summary>{{ tagSummaryLabel }}</summary>
          <div class="landing-chip-list">
            <button
              v-for="keyword in payload.keywords"
              :key="keyword.id"
              type="button"
              class="landing-chip"
              :class="{ 'is-active': selectedTag === keyword.keyword }"
              :aria-pressed="selectedTag === keyword.keyword"
              :disabled="switching"
              @click="pickByTag(keyword.keyword)"
            >{{ keyword.keyword }}</button>
          </div>
        </details>

        <p v-if="actionError" class="landing-action-message is-error" role="alert">{{ actionError }}</p>
        <div v-if="actionStatus" class="landing-action-message is-success" role="status" aria-live="polite">
          <span>{{ actionStatus }}</span>
          <button type="button" aria-label="关闭提示" @click="actionStatus = ''">关闭</button>
        </div>
      </section>

      <details v-if="payload.images.length" class="landing-optional-media">
        <summary>可选：查看门店配图素材</summary>
        <div class="landing-image-list">
          <a
            v-for="image in payload.images"
            :key="image.id"
            :href="image.imageUrl || image.url"
            target="_blank"
            rel="noreferrer"
          >
            <img :src="image.thumbnailUrl || image.imageUrl || image.url" loading="lazy" alt="门店配图" />
          </a>
        </div>
      </details>

      <p class="landing-trust landing-review-trust">内容可以修改，是否发布由你决定。</p>

      <div
        v-if="review"
        ref="reviewActions"
        data-testid="review-actions"
        class="landing-review-actions"
        :class="{ 'is-editing': isEditing }"
      >
        <button
          type="button"
          class="landing-primary-button"
          data-testid="primary-platform-action"
          :disabled="switching || actionPending || !editedContent.trim()"
          @click="copyAndOpenPlatform"
        >{{ primaryActionLabel }}</button>
        <div class="landing-secondary-actions">
          <button type="button" :disabled="switching || actionPending" @click="switchReview">换个说法</button>
          <button type="button" :disabled="switching || actionPending || !editedContent.trim()" @click="copyReview">仅复制</button>
        </div>
      </div>

      <div v-if="lotteryResult" class="landing-lottery-mask" role="dialog" aria-modal="true" aria-labelledby="lottery-result-title" data-testid="lottery-result">
        <section class="landing-lottery-result">
          <p class="landing-eyebrow">到店互动抽奖</p>
          <template v-if="lotteryResult.won">
            <p class="landing-lottery-spark" aria-hidden="true">✦</p>
            <h2 id="lottery-result-title">恭喜抽中</h2>
            <img v-if="lotteryResult.prizeImageUrl" class="landing-lottery-prize-image" :src="lotteryResult.prizeImageUrl" :alt="lotteryResult.prizeName" />
            <strong>{{ lotteryResult.prizeName }}</strong>
            <p>请向身边店员出示本页面领取</p>
          </template>
          <template v-else>
            <p class="landing-lottery-spark" aria-hidden="true">♡</p>
            <h2 id="lottery-result-title">谢谢参与</h2>
            <p>感谢你分享这次真实体验。</p>
          </template>
          <button type="button" class="landing-primary-button" @click="lotteryResult = null">我知道了</button>
        </section>
      </div>
    </template>
  </main>
</template>

<style scoped src="./landing.css"></style>
