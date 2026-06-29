<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adminApi } from '../../api/admin'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const merchants = ref<any[]>([])
const storeTypes = ref<any[]>([])
const stores = ref<any[]>([])
const tags = ref<any[]>([])
const tasks = ref<any[]>([])
const stats = ref<Record<string, number>>({})
const loading = ref(false)
const error = ref('')
const notice = ref('')

const platformOptions = [
  { code: 'dianping', name: '大众点评' },
  { code: 'meituan', name: '美团' },
  { code: 'xiaohongshu', name: '小红书' },
  { code: 'douyin', name: '抖音' }
]

// 自定义类型只能挂在 9 个预置行业之一之下（生成/隔离基准）
const presetTypes = computed(() => storeTypes.value.filter((t) => t.isPreset))
const activeStores = computed(() => stores.value.filter((s) => s.status === 1))

// 新建自定义类型
const newType = reactive({ name: '', industryCode: '' })
// 新建店铺（含商家账号）
const newStore = reactive({
  account: '',
  password: '',
  merchantName: '',
  typeId: 0,
  storeName: '',
  address: '',
  storeIntro: '',
  primaryPlatformStyle: 'dianping'
})
const lastCreated = ref<{ storeName: string; uuid: string; landingUrl: string; account: string } | null>(null)

// 卡片库存：按门店查看/新建
const cardStoreId = ref(0)
const newCardCode = ref('')
const cardsOfStore = computed(() => tags.value.filter((t) => t.storeId === cardStoreId.value))

function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

function typeName(typeId: number) {
  return storeTypes.value.find((t) => t.id === typeId)?.name || '-'
}

function landingUrl(uuid: string) {
  return `${location.origin}/landing/${uuid}`
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [merchantRes, typeRes, storeRes, tagRes, taskRes, statsRes] = await Promise.all([
      adminApi.listMerchants(),
      adminApi.listStoreTypes(),
      adminApi.listStores(),
      adminApi.listTags(),
      adminApi.listTasks(),
      adminApi.getStats()
    ])
    merchants.value = merchantRes.data.data
    storeTypes.value = typeRes.data.data
    stores.value = storeRes.data.data
    tags.value = tagRes.data.data
    tasks.value = taskRes.data.data
    stats.value = statsRes.data.data
    if (!newStore.typeId) newStore.typeId = storeTypes.value[0]?.id || 0
    if (!newType.industryCode) newType.industryCode = presetTypes.value[0]?.code || ''
    if (!cardStoreId.value) cardStoreId.value = activeStores.value[0]?.id || 0
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

async function createStoreType() {
  if (!newType.name.trim() || !newType.industryCode) {
    error.value = '请填写类型名称并选择行业基准'
    return
  }
  await runAction(() => adminApi.createStoreType({ name: newType.name.trim(), industryCode: newType.industryCode }), '类型已创建')
  newType.name = ''
}

async function createStore() {
  if (!newStore.account.trim() || !newStore.password || !newStore.storeName.trim() || !newStore.typeId) {
    error.value = '登录账号、密码、门店名、类型为必填'
    return
  }
  error.value = ''
  notice.value = ''
  try {
    const { data } = await adminApi.createStore({
      account: newStore.account.trim(),
      password: newStore.password,
      merchantName: newStore.merchantName.trim() || undefined,
      typeId: newStore.typeId,
      storeName: newStore.storeName.trim(),
      address: newStore.address.trim() || undefined,
      storeIntro: newStore.storeIntro.trim() || undefined,
      primaryPlatformStyle: newStore.primaryPlatformStyle
    })
    const created = data.data
    lastCreated.value = {
      storeName: created.store.storeName,
      uuid: created.store.uuid,
      landingUrl: landingUrl(created.store.uuid),
      account: created.merchant.account
    }
    notice.value = '门店已创建'
    newStore.account = ''
    newStore.password = ''
    newStore.merchantName = ''
    newStore.storeName = ''
    newStore.address = ''
    newStore.storeIntro = ''
    await loadAll()
  } catch (err: any) {
    error.value = messageFrom(err, '创建失败')
  }
}

async function createCard() {
  if (!cardStoreId.value) {
    error.value = '请先选择门店'
    return
  }
  await runAction(
    () => adminApi.createTag({ storeId: cardStoreId.value, tagCode: newCardCode.value.trim() || undefined }),
    '卡片已创建'
  )
  newCardCode.value = ''
}

async function toggleMerchantStatus(item: any) {
  await runAction(() => adminApi.updateMerchantStatus(item.id, item.status === 1 ? 0 : 1), '商家状态已更新')
}

async function toggleStoreStatus(item: any) {
  await runAction(() => adminApi.updateStoreStatus(item.id, item.status === 1 ? 0 : 1), '门店状态已更新')
}

async function toggleTagStatus(tag: any) {
  const nextStatus = tag.status === 'disabled' ? (tag.storeId ? 'bound' : 'unbound') : 'disabled'
  await runAction(() => adminApi.updateTagStatus(tag.id, nextStatus), '卡片状态已更新')
}

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    notice.value = '已复制到剪贴板'
  } catch {
    notice.value = text
  }
}

function numericStatusText(status: number) {
  return status === 1 ? '启用' : '禁用'
}
function tagStatusText(status: string) {
  if (status === 'disabled') return '禁用'
  return '启用'
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
        <p>NFC 卡片数：{{ stats.tagCount || 0 }}</p>
        <p>任务数：{{ stats.taskCount || 0 }}</p>
      </div>

      <div class="card">
        <h2>类型标签管理</h2>
        <p class="muted">类型决定门店的行业（推荐标签 + 串味隔离基准）。预置 9 行业，可新增自定义。</p>
        <div class="row" style="gap: 8px; flex-wrap: wrap">
          <input v-model="newType.name" placeholder="自定义类型名称，如 苍蝇馆子" style="flex: 1; min-width: 160px" />
          <select v-model="newType.industryCode">
            <option v-for="t in presetTypes" :key="t.code" :value="t.code">基准：{{ t.name }}</option>
          </select>
          <button :disabled="loading" @click="createStoreType">新增类型</button>
        </div>
        <table>
          <thead><tr><th>类型</th><th>行业基准</th><th>来源</th></tr></thead>
          <tbody>
            <tr v-for="t in storeTypes" :key="t.id">
              <td>{{ t.name }}</td>
              <td>{{ t.industryCode }}</td>
              <td>{{ t.isPreset ? '预置' : '自定义' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="card">
      <h2>新建门店（在类型下创建店铺 + 商家账号）</h2>
      <p class="muted">创建后系统自动生成店铺 UUID，并给出写入 NFC 卡片的落地链接。仍保持一商家一店。</p>
      <div class="grid-2">
        <div>
          <label class="fld">门店类型
            <select v-model.number="newStore.typeId">
              <option v-for="t in storeTypes" :key="t.id" :value="t.id">{{ t.name }}</option>
            </select>
          </label>
          <label class="fld">门店名称<input v-model="newStore.storeName" placeholder="如 老张川菜馆" /></label>
          <label class="fld">主推平台
            <select v-model="newStore.primaryPlatformStyle">
              <option v-for="p in platformOptions" :key="p.code" :value="p.code">{{ p.name }}</option>
            </select>
          </label>
          <label class="fld">门店地址<input v-model="newStore.address" placeholder="选填" /></label>
        </div>
        <div>
          <label class="fld">商家登录账号<input v-model="newStore.account" placeholder="商家登录用" /></label>
          <label class="fld">商家登录密码<input v-model="newStore.password" type="password" placeholder="商家登录用" /></label>
          <label class="fld">商家名称<input v-model="newStore.merchantName" placeholder="选填，默认同门店名" /></label>
          <label class="fld">门店简介<input v-model="newStore.storeIntro" placeholder="选填" /></label>
        </div>
      </div>
      <button :disabled="loading" @click="createStore">创建门店</button>

      <div v-if="lastCreated" class="created">
        <p>✅ 已创建「{{ lastCreated.storeName }}」，商家账号：<b>{{ lastCreated.account }}</b></p>
        <p>UUID：<code>{{ lastCreated.uuid }}</code></p>
        <p class="row" style="gap: 8px; align-items: center">
          写入 NFC 的链接：<code style="flex: 1; word-break: break-all">{{ lastCreated.landingUrl }}</code>
          <button class="secondary" @click="copyText(lastCreated.landingUrl)">复制</button>
        </p>
      </div>
    </div>

    <div class="card">
      <h2>门店列表</h2>
      <table>
        <thead><tr><th>ID</th><th>门店</th><th>类型</th><th>UUID / 落地链接</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="item in stores" :key="item.id">
            <td>{{ item.id }}</td>
            <td>{{ item.storeName }}</td>
            <td>{{ typeName(item.typeId) }}</td>
            <td>
              <code style="font-size: 12px">{{ item.uuid }}</code>
              <button class="link" @click="copyText(landingUrl(item.uuid))">复制链接</button>
            </td>
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

    <div class="grid-2">
      <div class="card">
        <h2>卡片库存</h2>
        <p class="muted">同一门店可印多张卡，都指向该店 UUID 落地页。禁用仅作库存标记。</p>
        <div class="row" style="gap: 8px">
          <select v-model.number="cardStoreId">
            <option :value="0" disabled>选择门店</option>
            <option v-for="s in activeStores" :key="s.id" :value="s.id">{{ s.id }} - {{ s.storeName }}</option>
          </select>
          <input v-model="newCardCode" placeholder="卡片编码（选填，可填 NFC UID）" style="flex: 1" />
          <button :disabled="loading" @click="createCard">新建卡片</button>
        </div>
        <table>
          <thead><tr><th>卡片编码</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="tag in cardsOfStore" :key="tag.id">
              <td>{{ tag.tagCode }}</td>
              <td>{{ tagStatusText(tag.status) }}</td>
              <td>
                <button class="secondary" :disabled="loading" @click="toggleTagStatus(tag)">
                  {{ tag.status === 'disabled' ? '启用' : '禁用' }}
                </button>
              </td>
            </tr>
            <tr v-if="!cardsOfStore.length"><td colspan="3" class="muted">该门店暂无卡片</td></tr>
          </tbody>
        </table>
      </div>

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

<style scoped>
.muted { color: #6b7280; font-size: 13px; margin: 0 0 10px; }
.fld { display: block; margin-bottom: 10px; font-size: 13px; color: #374151; }
.fld input, .fld select { display: block; width: 100%; margin-top: 4px; }
.created { margin-top: 14px; padding: 12px; background: #f0fdf4; border: 1px solid #bbf7d0; border-radius: 8px; font-size: 13px; }
.created code { background: #fff; padding: 2px 6px; border-radius: 4px; }
.link { background: none; border: none; color: #2563eb; cursor: pointer; font-size: 12px; padding: 0 0 0 8px; }
</style>
