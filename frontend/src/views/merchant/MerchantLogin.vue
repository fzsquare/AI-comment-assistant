<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { merchantApi } from '../../api/merchant'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const form = reactive({ account: 'merchant', password: '123456' })
const submitting = ref(false)

async function submit() {
  if (submitting.value) return
  submitting.value = true
  try {
    const { data } = await merchantApi.login(form)
    auth.setAuth(data.data.token, 'merchant')
    router.push('/merchant/console')
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
      <p class="auth-kicker">店铺运营台</p>
      <h1>商家登录</h1>
      <p class="auth-sub">维护跳转链接、图片素材、关键词和评价池。</p>

      <label class="field">
        <span>账号</span>
        <input v-model.trim="form.account" autocomplete="username" inputmode="text" />
      </label>

      <label class="field">
        <span>密码</span>
        <input v-model="form.password" type="password" autocomplete="current-password" />
      </label>

      <button class="primary-action" type="submit" :disabled="submitting">
        {{ submitting ? '登录中…' : '进入商家后台' }}
      </button>

      <p class="auth-hint">演示账号：merchant / 123456</p>
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
.primary-action {
  width: 100%;
  margin-top: 4px;
  min-height: 50px;
  font-weight: 800;
}
.auth-hint {
  margin: 14px 0 0;
  color: #64748b;
  font-size: 13px;
  text-align: center;
}
</style>
