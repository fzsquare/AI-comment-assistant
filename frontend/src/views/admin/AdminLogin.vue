<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { adminApi } from '../../api/admin'
import { useAuthStore } from '../../stores/auth'
import { safeAdminRedirect } from '../../utils/authSession'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const isMock = import.meta.env.VITE_USE_MOCK === 'true'
const form = reactive({ account: isMock ? 'admin' : '', password: isMock ? '123456' : '' })
const submitting = ref(false)
const showPassword = ref(false)
const error = ref('')

const sessionMessage = computed(() => {
  if (route.query.reason === 'session_expired') return '登录状态已过期，请重新登录后继续。'
  if (route.query.reason === 'logged_out') return '你已安全退出管理员后台。'
  return ''
})

async function submit() {
  if (submitting.value) return
  const account = form.account.trim()
  if (!account || !form.password) {
    error.value = '请输入管理员账号和密码。'
    return
  }
  error.value = ''
  submitting.value = true
  try {
    const { data } = await adminApi.login({ account, password: form.password })
    auth.setAuth(data.data.token, 'admin')
    await router.replace(safeAdminRedirect(route.query.redirect))
  } catch (err: any) {
    error.value = err?.response?.data?.message || '登录失败，请检查账号和密码后重试。'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="auth-page">
    <form class="auth-card" @submit.prevent="submit">
      <div class="auth-brand" aria-hidden="true">评</div>
      <p class="auth-kicker">评价助手 · 管理控制台</p>
      <h1>管理员登录</h1>
      <p class="auth-sub">登录后管理商家、门店和运营数据。</p>

      <p v-if="sessionMessage" class="auth-status" role="status">{{ sessionMessage }}</p>
      <p v-if="error" class="auth-error" role="alert">{{ error }}</p>

      <label class="field">
        <span>账号</span>
        <input v-model="form.account" autocomplete="username" inputmode="text" placeholder="请输入管理员账号" />
      </label>

      <label class="field">
        <span>密码</span>
        <span class="password-field">
          <input
            v-model="form.password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="current-password"
            placeholder="请输入密码"
          />
          <button
            type="button"
            class="password-toggle"
            :aria-pressed="showPassword"
            :aria-label="showPassword ? '隐藏密码' : '显示密码'"
            @click="showPassword = !showPassword"
          >{{ showPassword ? '隐藏' : '显示' }}</button>
        </span>
      </label>

      <button class="primary-action" type="submit" :disabled="submitting">
        {{ submitting ? '登录中…' : '进入管理员后台' }}
      </button>

      <p v-if="isMock" class="auth-hint">Mock 演示账号：admin / 123456</p>
    </form>
  </main>
</template>

<style scoped>
.auth-page {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: calc(20px + env(safe-area-inset-top)) 16px calc(20px + env(safe-area-inset-bottom));
}
.auth-card {
  width: min(100%, 420px);
  padding: 28px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: none;
}
.auth-brand {
  align-items: center;
  background: #0f172a;
  border-radius: 8px;
  color: #fff;
  display: flex;
  font-size: 16px;
  font-weight: 900;
  height: 40px;
  justify-content: center;
  margin-bottom: 20px;
  width: 40px;
}
.auth-kicker {
  margin: 0 0 8px;
  color: #2563eb;
  font-size: 13px;
  font-weight: 800;
}
h1 {
  margin: 0;
  color: #0f172a;
  font-size: 26px;
  line-height: 1.2;
}
.auth-sub {
  margin: 8px 0 22px;
  color: var(--muted);
}
.auth-status,
.auth-error {
  border: 1px solid;
  border-radius: 6px;
  font-size: 14px;
  margin: 0 0 16px;
  padding: 10px 12px;
}
.auth-status {
  background: #eff6ff;
  border-color: #bfdbfe;
  color: #1e40af;
}
.auth-error {
  background: #fef2f2;
  border-color: #fecaca;
  color: #991b1b;
}
.field {
  display: grid;
  gap: 7px;
  margin-bottom: 14px;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 700;
}
.primary-action {
  width: 100%;
  margin-top: 4px;
  min-height: 50px;
  font-weight: 800;
}
.password-field {
  display: grid;
  position: relative;
}
.password-field input {
  padding-right: 64px;
}
.password-toggle {
  align-self: center;
  background: transparent;
  border: 0;
  color: #2563eb;
  font-size: 13px;
  font-weight: 800;
  min-height: 36px;
  padding: 0 10px;
  position: absolute;
  right: 4px;
}
.auth-hint {
  margin: 14px 0 0;
  color: var(--muted);
  font-size: 13px;
  text-align: center;
}
</style>
