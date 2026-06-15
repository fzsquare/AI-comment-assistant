<script setup lang="ts">
import { reactive } from 'vue'
import { useRouter } from 'vue-router'
import { adminApi } from '../../api/admin'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const form = reactive({ account: 'admin', password: '123456' })

async function submit() {
  try {
    const { data } = await adminApi.login(form)
    auth.setAuth(data.data.token, 'admin')
    router.push('/admin/console')
  } catch (err: any) {
    alert(err?.response?.data?.message || '登录失败')
  }
}
</script>

<template>
  <div class="page" style="max-width: 420px">
    <div class="card">
      <h1>管理员登录</h1>
      <div style="margin-bottom: 12px">
        <label>账号</label>
        <input v-model="form.account" />
      </div>
      <div style="margin-bottom: 12px">
        <label>密码</label>
        <input v-model="form.password" type="password" />
      </div>
      <button @click="submit" style="width: 100%">登录</button>
      <p class="muted" style="margin-top: 12px">默认演示账号：admin / 123456</p>
    </div>
  </div>
</template>
