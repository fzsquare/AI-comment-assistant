<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'

type PlatformScheme = {
  code: string
  name: string
  scheme: string
}

const platforms: PlatformScheme[] = [
  { code: 'meituan', name: '美团', scheme: 'imeituan://' },
  { code: 'dianping', name: '大众点评', scheme: 'dianping://' },
  { code: 'douyin', name: '抖音', scheme: 'snssdk1128://' },
  { code: 'xiaohongshu', name: '小红书', scheme: 'xhsdiscover://' }
]

const customScheme = ref('')
const activeScheme = ref('')
const lastAttemptAt = ref('')
const status = ref<'idle' | 'waiting' | 'backgrounded' | 'no-transition'>('idle')
let resultTimer = 0

const statusText = computed(() => {
  if (status.value === 'waiting') return '已发送拉起请求，正在等待系统响应'
  if (status.value === 'backgrounded') return '检测到页面进入后台，系统已切离测试页'
  if (status.value === 'no-transition') return '未检测到页面进入后台，请在真机确认 App 是否已打开'
  return '请选择一个平台开始测试'
})

function cleanup() {
  window.clearTimeout(resultTimer)
  document.removeEventListener('visibilitychange', markBackgrounded)
  window.removeEventListener('pagehide', markBackgrounded)
}

function markBackgrounded() {
  if (document.hidden) {
    status.value = 'backgrounded'
    cleanup()
  }
}

function launch(scheme: string) {
  const normalized = scheme.trim()
  if (!normalized) return

  cleanup()
  activeScheme.value = normalized
  lastAttemptAt.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
  status.value = 'waiting'
  document.addEventListener('visibilitychange', markBackgrounded)
  window.addEventListener('pagehide', markBackgrounded)
  resultTimer = window.setTimeout(() => {
    if (status.value === 'waiting') status.value = 'no-transition'
    cleanup()
  }, 2200)
  window.location.href = normalized
}

function launchCustom() {
  launch(customScheme.value)
}

onBeforeUnmount(cleanup)
</script>

<template>
  <main class="scheme-test page">
    <header class="page-head">
      <p class="eyebrow">设备验证</p>
      <h1>App 拉起测试</h1>
      <p class="sub">点击后将直接请求系统打开对应 App。</p>
    </header>

    <section class="status-panel" aria-live="polite">
      <span class="status-dot" :class="status"></span>
      <div>
        <strong>{{ statusText }}</strong>
        <p v-if="activeScheme">{{ activeScheme }}</p>
        <p v-else>尚未发起请求</p>
      </div>
      <time v-if="lastAttemptAt">{{ lastAttemptAt }}</time>
    </section>

    <section aria-label="平台 Scheme 测试">
      <div class="platform-list">
        <article v-for="platform in platforms" :key="platform.code" class="platform-row">
          <div>
            <h2>{{ platform.name }}</h2>
            <code>{{ platform.scheme }}</code>
          </div>
          <button type="button" @click="launch(platform.scheme)">拉起</button>
        </article>
      </div>
    </section>

    <section class="custom-panel">
      <label for="custom-scheme">自定义 Scheme</label>
      <div class="custom-action">
        <input id="custom-scheme" v-model="customScheme" inputmode="url" placeholder="例如 appname://" @keyup.enter="launchCustom" />
        <button type="button" :disabled="!customScheme.trim()" @click="launchCustom">测试</button>
      </div>
    </section>
  </main>
</template>

<style scoped>
.scheme-test {
  max-width: 680px;
  padding-top: 40px;
}

.page-head {
  margin-bottom: 24px;
}

.eyebrow {
  color: var(--primary-strong);
  font-size: 13px;
  font-weight: 700;
  margin: 0 0 8px;
}

h1 {
  margin-bottom: 8px;
}

.sub {
  color: var(--muted);
  margin: 0;
}

.status-panel {
  align-items: center;
  background: var(--surface);
  border: 1px solid var(--border);
  display: grid;
  gap: 12px;
  grid-template-columns: 12px minmax(0, 1fr) auto;
  margin-bottom: 18px;
  padding: 16px;
}

.status-dot {
  background: #94a3b8;
  border-radius: 50%;
  height: 10px;
  width: 10px;
}

.status-dot.waiting { background: #d97706; }
.status-dot.backgrounded { background: #16a34a; }
.status-dot.no-transition { background: #b45309; }

.status-panel strong {
  color: var(--text);
  display: block;
  font-size: 14px;
}

.status-panel p,
.status-panel time {
  color: var(--muted);
  font-size: 13px;
  margin: 4px 0 0;
}

.status-panel code {
  overflow-wrap: anywhere;
}

.platform-list {
  border-top: 1px solid var(--border);
}

.platform-row {
  align-items: center;
  border-bottom: 1px solid var(--border);
  display: flex;
  gap: 16px;
  justify-content: space-between;
  min-height: 78px;
  padding: 14px 0;
}

.platform-row h2 {
  font-size: 16px;
  margin: 0 0 5px;
}

.platform-row code {
  color: var(--muted);
  font-size: 13px;
}

.platform-row button,
.custom-action button {
  flex: 0 0 auto;
}

.custom-panel {
  margin-top: 24px;
}

.custom-panel label {
  color: var(--text-secondary);
  display: block;
  font-size: 14px;
  font-weight: 700;
  margin-bottom: 8px;
}

.custom-action {
  display: flex;
  gap: 10px;
}

.custom-action input {
  min-width: 0;
}

@media (max-width: 480px) {
  .scheme-test {
    padding-top: 24px;
  }

  .status-panel {
    align-items: start;
    grid-template-columns: 12px minmax(0, 1fr);
  }

  .status-panel time {
    grid-column: 2;
    margin-top: 0;
  }

  .custom-action {
    flex-direction: column;
  }

  .custom-action button {
    width: 100%;
  }
}
</style>
