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

      <button class="switch-action" type="button" @click="router.push('/merchant/login')">
        切换到商家入口
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
  background: #fff;
  border: 1px solid #dbe4f0;
  border-radius: 18px;
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.08);
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
  color: #64748b;
}
.field {
  display: grid;
  gap: 7px;
  margin-bottom: 14px;
  color: #334155;
  font-size: 14px;
  font-weight: 700;
}
.primary-action,
.switch-action {
  width: 100%;
}
.primary-action {
  margin-top: 4px;
  min-height: 50px;
  font-weight: 800;
}
.switch-action {
  margin-top: 10px;
  background: #eef2ff;
  color: #1e40af;
}
.auth-hint {
  margin: 14px 0 0;
  color: #64748b;
  font-size: 13px;
  text-align: center;
}
</style>
