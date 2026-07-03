<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { adminApi } from '../../api/admin'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const form = reactive({ account: 'admin', password: '123456' })
const submitting = ref(false)

async function submit() {
  if (submitting.value) return
  submitting.value = true
  try {
    const { data } = await adminApi.login(form)
    auth.setAuth(data.data.token, 'admin')
    router.push('/admin/console')
  } catch (err: any) {
    alert(err?.response?.data?.message || '登录失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="auth-page">
    <form class="auth-card" @submit.prevent="submit">
      <p class="auth-kicker">系统管理台</p>
      <h1>管理员登录</h1>
      <p class="auth-sub">管理商家、门店、NFC 卡片与生成任务。</p>

      <label class="field">
        <span>账号</span>
        <input v-model.trim="form.account" autocomplete="username" inputmode="text" />
      </label>

      <label class="field">
        <span>密码</span>
        <input v-model="form.password" type="password" autocomplete="current-password" />
      </label>

      <button class="primary-action" type="submit" :disabled="submitting">
        {{ submitting ? '登录中…' : '进入管理员后台' }}
      </button>

      <p class="auth-hint">演示账号：admin / 123456</p>
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
  padding: 22px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: none;
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
  margin: 8px 0 18px;
  color: var(--muted);
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
.auth-hint {
  margin: 14px 0 0;
  color: var(--muted);
  font-size: 13px;
  text-align: center;
}
</style>
