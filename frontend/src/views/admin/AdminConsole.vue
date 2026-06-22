<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { adminApi } from '../../api/admin'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const merchants = ref<any[]>([])
const stores = ref<any[]>([])
const tags = ref<any[]>([])
const tasks = ref<any[]>([])
const stats = ref<Record<string, number>>({})
const selectedStoreId = ref(0)
const loading = ref(false)
const error = ref('')
const notice = ref('')
const activeStores = computed(() => stores.value.filter((store) => store.status === 1))

function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
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
    if (!activeStores.value.some((store) => store.id === selectedStoreId.value)) {
      selectedStoreId.value = activeStores.value[0]?.id || 0
    }
  } catch (err: any) {
    error.value = messageFrom(err, '后台数据加载失败')
  } finally {
    loading.value = false
  }
}

async function runAction(action: () => Promise<unknown>, success: string) {
  error.value = ''
  notice.value = ''
  try {
    await action()
    notice.value = success
    await loadAll()
  } catch (err: any) {
    error.value = messageFrom(err, '操作失败')
  }
}

async function createTag() {
  await runAction(() => adminApi.createTag({}), '标签已创建')
}

async function bindTag(tagId: number) {
  if (!selectedStoreId.value) {
    error.value = '请先选择要绑定的门店'
    return
  }
  await runAction(() => adminApi.bindTag(tagId, selectedStoreId.value), '标签已绑定')
}

async function toggleMerchantStatus(item: any) {
  const nextStatus = item.status === 1 ? 0 : 1
  await runAction(() => adminApi.updateMerchantStatus(item.id, nextStatus), '商家状态已更新')
}

async function toggleStoreStatus(item: any) {
  const nextStatus = item.status === 1 ? 0 : 1
  await runAction(() => adminApi.updateStoreStatus(item.id, nextStatus), '门店状态已更新')
}

async function toggleTagStatus(tag: any) {
  const nextStatus = tag.status === 'disabled' ? (tag.storeId ? 'bound' : 'unbound') : 'disabled'
  await runAction(() => adminApi.updateTagStatus(tag.id, nextStatus), '标签状态已更新')
}

function numericStatusText(status: number) {
  return status === 1 ? '启用' : '禁用'
}

function tagStatusText(status: string) {
  if (status === 'bound') return '已绑定'
  if (status === 'unbound') return '未绑定'
  if (status === 'disabled') return '禁用'
  return status
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
      <div class="row">
        <button class="secondary" :disabled="loading" @click="loadAll">刷新</button>
        <button class="secondary" @click="logout">退出登录</button>
      </div>
    </div>
    <p v-if="error" class="alert">{{ error }}</p>
    <p v-else-if="notice" class="notice">{{ notice }}</p>

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
          <select v-model.number="selectedStoreId">
            <option :value="0" disabled>选择要绑定的门店</option>
            <option v-for="store in activeStores" :key="store.id" :value="store.id">
              {{ store.id }} - {{ store.storeName }}
            </option>
          </select>
          <button :disabled="loading" @click="createTag">创建标签</button>
        </div>
        <table>
          <thead><tr><th>标签</th><th>状态</th><th>门店</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="tag in tags" :key="tag.id">
              <td>{{ tag.tagCode }}</td>
              <td>{{ tagStatusText(tag.status) }}</td>
              <td>{{ tag.storeId || '-' }}</td>
              <td>
                <div class="row">
                  <button class="secondary" :disabled="loading" @click="bindTag(tag.id)">绑定</button>
                  <button class="secondary" :disabled="loading" @click="toggleTagStatus(tag)">
                    {{ tag.status === 'disabled' ? '启用' : '禁用' }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="grid-2">
      <div class="card">
        <h2>商家列表</h2>
        <table>
          <thead><tr><th>ID</th><th>名称</th><th>账号</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="item in merchants" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.merchantName }}</td>
              <td>{{ item.account }}</td>
              <td>{{ numericStatusText(item.status) }}</td>
              <td>
                <button class="secondary" :disabled="loading" @click="toggleMerchantStatus(item)">
                  {{ item.status === 1 ? '禁用' : '启用' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="card">
        <h2>门店列表</h2>
        <table>
          <thead><tr><th>ID</th><th>门店</th><th>平台风格</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="item in stores" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.storeName }}</td>
              <td>{{ item.primaryPlatformStyle }}</td>
              <td>{{ numericStatusText(item.status) }}</td>
              <td>
                <button class="secondary" :disabled="loading" @click="toggleStoreStatus(item)">
                  {{ item.status === 1 ? '禁用' : '启用' }}
                </button>
              </td>
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
