<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminApi } from '../../api/admin'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const merchants = ref<any[]>([])
const stores = ref<any[]>([])
const tags = ref<any[]>([])
const tasks = ref<any[]>([])
const stats = ref<Record<string, number>>({})
const bindStoreId = ref<number>(1)

async function loadAll() {
  const [merchantRes, storeRes, tagRes, taskRes, statsRes] = await Promise.all([
    adminApi.listMerchants(),
    adminApi.listStores(),
    adminApi.listTags(),
    adminApi.listTasks(),
    adminApi.getStats()
  ])
  merchants.value = merchantRes.data.data
  stores.value = storeRes.data.data
  tags.value = tagRes.data.data
  tasks.value = taskRes.data.data
  stats.value = statsRes.data.data
}

async function createTag() {
  await adminApi.createTag({})
  await loadAll()
}

async function bindTag(tagId: number) {
  await adminApi.bindTag(tagId, bindStoreId.value)
  await loadAll()
}

function logout() {
  auth.clear()
  location.href = '/admin/login'
}

onMounted(loadAll)
</script>

<template>
  <div class="page">
    <div class="row" style="justify-content: space-between; align-items: center">
      <h1>管理员后台</h1>
      <button class="secondary" @click="logout">退出登录</button>
    </div>

    <div class="grid-2">
      <div class="card">
        <h2>平台统计</h2>
        <p>商家数：{{ stats.merchantCount || 0 }}</p>
        <p>门店数：{{ stats.storeCount || 0 }}</p>
        <p>NFC 标签数：{{ stats.tagCount || 0 }}</p>
        <p>任务数：{{ stats.taskCount || 0 }}</p>
      </div>

      <div class="card">
        <h2>NFC 标签管理</h2>
        <div class="row">
          <input v-model.number="bindStoreId" placeholder="绑定门店 ID" />
          <button @click="createTag">创建标签</button>
        </div>
        <table>
          <thead><tr><th>标签</th><th>状态</th><th>门店</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="tag in tags" :key="tag.id">
              <td>{{ tag.tagCode }}</td>
              <td>{{ tag.status }}</td>
              <td>{{ tag.storeId || '-' }}</td>
              <td><button class="secondary" @click="bindTag(tag.id)">绑定</button></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="grid-2">
      <div class="card">
        <h2>商家列表</h2>
        <table>
          <thead><tr><th>ID</th><th>名称</th><th>账号</th><th>状态</th></tr></thead>
          <tbody>
            <tr v-for="item in merchants" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.merchantName }}</td>
              <td>{{ item.account }}</td>
              <td>{{ item.status }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="card">
        <h2>门店列表</h2>
        <table>
          <thead><tr><th>ID</th><th>门店</th><th>平台风格</th><th>状态</th></tr></thead>
          <tbody>
            <tr v-for="item in stores" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.storeName }}</td>
              <td>{{ item.primaryPlatformStyle }}</td>
              <td>{{ item.status }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="card">
      <h2>生成任务</h2>
      <table>
        <thead><tr><th>ID</th><th>门店</th><th>类型</th><th>状态</th><th>成功数</th></tr></thead>
        <tbody>
          <tr v-for="item in tasks" :key="item.id">
            <td>{{ item.id }}</td>
            <td>{{ item.storeId }}</td>
            <td>{{ item.triggerType }}</td>
            <td>{{ item.status }}</td>
            <td>{{ item.successCount }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
