<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adminApi } from '../../api/admin'
import { copyToClipboard } from '../../utils/clipboard'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const merchants = ref<any[]>([])
const storeTypes = ref<any[]>([])
const stores = ref<any[]>([])
const tasks = ref<any[]>([])
const stats = ref<Record<string, number>>({})
const loading = ref(false)
const error = ref('')
const notice = ref('')
const editingStoreId = ref<number | null>(null)

const platformOptions = [
  { code: 'dianping', name: '大众点评' },
  { code: 'meituan', name: '美团' },
  { code: 'xiaohongshu', name: '小红书' },
  { code: 'douyin', name: '抖音' }
]

// 自定义类型只能挂在 9 个预置行业之一之下（生成/隔离基准）
const presetTypes = computed(() => storeTypes.value.filter((t) => t.isPreset))
const isEditingStore = computed(() => editingStoreId.value !== null)

// 新建自定义类型
const newType = reactive({ name: '', industryCode: '' })
// 新建/编辑店铺（含商家账号）
const newStore = reactive({
  account: '',
  password: '',
  merchantName: '',
  contactName: '',
  typeId: 0,
  storeName: '',
  address: '',
  storeIntro: '',
  primaryPlatformStyle: 'dianping',
  brandTone: '',
  platformUrl: ''
})
const lastCreated = ref<{ storeName: string; uuid: string; landingUrl: string; account: string } | null>(null)

function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

function typeName(typeId: number) {
  return storeTypes.value.find((t) => t.id === typeId)?.name || '-'
}

function normalizeAbsoluteUrl(value: string) {
  const trimmed = (value || '').trim()
  if (!trimmed) return ''
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  if (trimmed.startsWith('/')) return `${location.origin}${trimmed}`
  return `${location.origin}/${trimmed.replace(/^\/+/, '')}`
}

function storeLandingUrl(item: any) {
  return normalizeAbsoluteUrl(item.landingUrl || `${import.meta.env.BASE_URL}landing/${item.uuid}`)
}

function merchantForStore(item: any) {
  return merchants.value.find((merchant) => merchant.id === item.merchantUserId) || {}
}

function resetStoreForm() {
  editingStoreId.value = null
  newStore.account = ''
  newStore.password = ''
  newStore.merchantName = ''
  newStore.contactName = ''
  newStore.typeId = storeTypes.value[0]?.id || 0
  newStore.storeName = ''
  newStore.address = ''
  newStore.storeIntro = ''
  newStore.primaryPlatformStyle = 'dianping'
  newStore.brandTone = ''
  newStore.platformUrl = ''
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [merchantRes, typeRes, storeRes, taskRes, statsRes] = await Promise.all([
      adminApi.listMerchants(),
      adminApi.listStoreTypes(),
      adminApi.listStores(),
      adminApi.listTasks(),
      adminApi.getStats()
    ])
    merchants.value = merchantRes.data.data
    storeTypes.value = typeRes.data.data
    stores.value = storeRes.data.data
    tasks.value = taskRes.data.data
    stats.value = statsRes.data.data
    if (!newStore.typeId) newStore.typeId = storeTypes.value[0]?.id || 0
    if (!newType.industryCode) newType.industryCode = presetTypes.value[0]?.code || ''
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

function editStore(item: any) {
  const merchant = merchantForStore(item)
  editingStoreId.value = item.id
  lastCreated.value = null
  newStore.account = item.merchantAccount || merchant.account || ''
  newStore.password = ''
  newStore.merchantName = item.merchantName || merchant.merchantName || item.storeName || ''
  newStore.contactName = item.contactName || merchant.contactName || ''
  newStore.typeId = item.typeId || 0
  newStore.storeName = item.storeName || ''
  newStore.address = item.address || ''
  newStore.storeIntro = item.storeIntro || ''
  newStore.primaryPlatformStyle = item.primaryPlatformStyle || 'dianping'
  newStore.brandTone = item.brandTone || ''
  newStore.platformUrl = item.platformUrl || ''
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

async function saveStoreForm() {
  const account = newStore.account.trim()
  const password = newStore.password.trim()
  const storeName = newStore.storeName.trim()
  if (!account || !storeName || !newStore.typeId || (!isEditingStore.value && !password)) {
    error.value = isEditingStore.value ? '登录账号、门店名、类型为必填' : '登录账号、密码、门店名、类型为必填'
    return
  }

  const payload = {
    account,
    password,
    merchantName: newStore.merchantName.trim() || undefined,
    contactName: newStore.contactName.trim() || undefined,
    typeId: newStore.typeId,
    storeName,
    address: newStore.address.trim() || undefined,
    storeIntro: newStore.storeIntro.trim() || undefined,
    primaryPlatformStyle: newStore.primaryPlatformStyle,
    brandTone: newStore.brandTone.trim() || undefined,
    platformUrl: newStore.platformUrl.trim() || undefined
  }

  error.value = ''
  notice.value = ''
  try {
    if (isEditingStore.value) {
      await adminApi.updateStore(editingStoreId.value as number, {
        ...payload,
        password: password || undefined
      })
      notice.value = '门店资料已保存'
      resetStoreForm()
      await loadAll()
      return
    }

    const { data } = await adminApi.createStore(payload)
    const created = data.data
    lastCreated.value = {
      storeName: created.store.storeName,
      uuid: created.store.uuid,
      landingUrl: normalizeAbsoluteUrl(created.landingUrl || `${import.meta.env.BASE_URL}landing/${created.store.uuid}`),
      account: created.merchant.account
    }
    notice.value = '门店已创建'
    resetStoreForm()
    await loadAll()
  } catch (err: any) {
    error.value = messageFrom(err, isEditingStore.value ? '保存失败' : '创建失败')
  }
}

async function toggleMerchantStatus(item: any) {
  await runAction(() => adminApi.updateMerchantStatus(item.id, item.status === 1 ? 0 : 1), '商家状态已更新')
}

async function deleteMerchant(item: any) {
  if (!window.confirm(`确认删除商家「${item.merchantName || item.account}」？关联门店、评价、图片、平台入口和生成任务会一起删除，NFC 卡片会解绑保留。`)) return
  await runAction(() => adminApi.deleteMerchant(item.id), '商家已删除')
}

async function toggleStoreStatus(item: any) {
  await runAction(() => adminApi.updateStoreStatus(item.id, item.status === 1 ? 0 : 1), '门店状态已更新')
}

async function deleteStore(item: any) {
  if (!window.confirm(`确认删除门店「${item.storeName}」？关联商家账号、评价、图片、平台入口和生成任务会一起删除，NFC 卡片会解绑保留。`)) return
  await runAction(() => adminApi.deleteStore(item.id), '门店已删除')
  if (editingStoreId.value === item.id) resetStoreForm()
}

async function copyText(text: string) {
  if (await copyToClipboard(text)) {
    notice.value = '已复制到剪贴板'
  } else {
    error.value = '复制失败，请手动选中复制：' + text
  }
}

function numericStatusText(status: number) {
  return status === 1 ? '启用' : '禁用'
}

function logout() {
  auth.clear()
  location.href = import.meta.env.BASE_URL + 'admin/login'
}

onMounted(loadAll)
</script>

<template>
  <div class="page admin-console">
    <div class="row admin-header">
      <h1>管理员后台</h1>
      <div class="row header-actions">
        <button class="secondary" :disabled="loading" @click="loadAll">刷新</button>
        <button class="secondary" @click="logout">退出登录</button>
      </div>
    </div>
    <p v-if="error" class="alert">{{ error }}</p>
    <p v-else-if="notice" class="notice">{{ notice }}</p>

    <div class="stat-strip" aria-label="平台统计">
      <div><span>商家</span><strong>{{ stats.merchantCount || 0 }}</strong></div>
      <div><span>门店</span><strong>{{ stats.storeCount || 0 }}</strong></div>
      <div><span>交付 URL</span><strong>{{ stats.storeCount || 0 }}</strong></div>
      <div><span>生成任务</span><strong>{{ stats.taskCount || 0 }}</strong></div>
    </div>

    <div class="card">
      <div class="section-head">
        <div>
          <h2>{{ isEditingStore ? '编辑门店资料与登录账号' : '新建门店并生成交付 URL' }}</h2>
          <p class="muted">固定 URL 绑定门店，交付时写入 NFC 卡贴。</p>
        </div>
        <button v-if="isEditingStore" class="secondary" :disabled="loading" @click="resetStoreForm">取消编辑</button>
      </div>
      <div class="grid-2 form-grid">
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
          <label class="fld">客户端跳转链接
            <input v-model="newStore.platformUrl" placeholder="平台店铺或分享链接，可后续补充" />
          </label>
          <label class="fld">门店地址<input v-model="newStore.address" placeholder="选填" /></label>
        </div>
        <div>
          <label class="fld">商家登录账号<input v-model="newStore.account" placeholder="商家登录用" /></label>
          <label class="fld">{{ isEditingStore ? '商家登录密码（留空不修改）' : '商家登录密码' }}
            <input v-model="newStore.password" type="password" placeholder="商家登录用" />
          </label>
          <label class="fld">商家名称<input v-model="newStore.merchantName" placeholder="选填，默认同门店名" /></label>
          <label class="fld">联系人<input v-model="newStore.contactName" placeholder="选填" /></label>
          <label class="fld">门店简介<input v-model="newStore.storeIntro" placeholder="选填" /></label>
          <label class="fld">品牌语气<input v-model="newStore.brandTone" placeholder="选填，如 轻松自然" /></label>
        </div>
      </div>
      <div class="row form-actions">
        <button :disabled="loading" @click="saveStoreForm">
          {{ isEditingStore ? '保存修改' : '创建门店并生成 URL' }}
        </button>
        <button v-if="isEditingStore" class="secondary" :disabled="loading" @click="resetStoreForm">取消</button>
      </div>

      <div v-if="lastCreated" class="created" role="status">
        <p>已创建「{{ lastCreated.storeName }}」，商家账号：<b>{{ lastCreated.account }}</b></p>
        <p>UUID：<code>{{ lastCreated.uuid }}</code></p>
        <p class="url-line">
          <span>交付 URL：</span>
          <code>{{ lastCreated.landingUrl }}</code>
          <button class="secondary" @click="copyText(lastCreated.landingUrl)">复制</button>
        </p>
      </div>
    </div>

    <div class="card">
      <h2>门店列表</h2>
      <table class="desktop-table">
        <thead><tr><th>门店</th><th>类型</th><th>商家账号</th><th>交付 URL</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="item in stores" :key="item.id">
            <td>
              <strong>{{ item.storeName }}</strong>
              <span class="subtext">ID {{ item.id }}</span>
            </td>
            <td>{{ typeName(item.typeId) }}</td>
            <td>{{ item.merchantAccount || merchantForStore(item).account || '-' }}</td>
            <td class="url-cell">
              <code>{{ storeLandingUrl(item) }}</code>
              <button class="link" @click="copyText(storeLandingUrl(item))">复制 URL</button>
            </td>
            <td>{{ numericStatusText(item.status) }}</td>
            <td>
              <span class="table-actions">
                <button class="secondary" :disabled="loading" @click="editStore(item)">编辑</button>
                <button class="secondary" :disabled="loading" @click="toggleStoreStatus(item)">
                  {{ item.status === 1 ? '禁用' : '启用' }}
                </button>
                <button class="danger" :disabled="loading" @click="deleteStore(item)">删除</button>
              </span>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="mobile-store-list" aria-label="门店列表">
        <article v-for="item in stores" :key="item.id" class="mobile-store-item">
          <div class="mobile-store-head">
            <div>
              <strong>{{ item.storeName }}</strong>
              <span>ID {{ item.id }} · {{ typeName(item.typeId) }}</span>
            </div>
            <b :class="['status-pill', item.status === 1 ? 'enabled' : 'disabled']">{{ numericStatusText(item.status) }}</b>
          </div>
          <dl class="mobile-store-meta">
            <div>
              <dt>商家账号</dt>
              <dd>{{ item.merchantAccount || merchantForStore(item).account || '-' }}</dd>
            </div>
            <div>
              <dt>交付 URL</dt>
              <dd>
                <code>{{ storeLandingUrl(item) }}</code>
                <button class="link" @click="copyText(storeLandingUrl(item))">复制 URL</button>
              </dd>
            </div>
          </dl>
          <div class="mobile-store-actions">
            <button class="secondary" :disabled="loading" @click="editStore(item)">编辑</button>
            <button class="secondary" :disabled="loading" @click="toggleStoreStatus(item)">
              {{ item.status === 1 ? '禁用' : '启用' }}
            </button>
            <button class="danger" :disabled="loading" @click="deleteStore(item)">删除</button>
          </div>
        </article>
      </div>
    </div>

    <div class="fold-grid">
      <details class="card fold-card">
        <summary>
          <span>
            <strong>类型标签管理</strong>
            <small>低频配置，用于推荐标签与行业隔离基准</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
          <div class="row inline-form">
            <input v-model="newType.name" placeholder="自定义类型名称，如 苍蝇馆子" />
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
      </details>

      <details class="card fold-card">
        <summary>
          <span>
            <strong>商家账号</strong>
            <small>账号禁用、删除等管理动作</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
          <table>
            <thead><tr><th>ID</th><th>名称</th><th>账号</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="item in merchants" :key="item.id">
                <td>{{ item.id }}</td>
                <td>{{ item.merchantName }}</td>
                <td>{{ item.account }}</td>
                <td>{{ numericStatusText(item.status) }}</td>
                <td>
                  <span class="table-actions compact">
                    <button class="secondary" :disabled="loading" @click="toggleMerchantStatus(item)">
                      {{ item.status === 1 ? '禁用' : '启用' }}
                    </button>
                    <button class="danger" :disabled="loading" @click="deleteMerchant(item)">删除</button>
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </details>

      <details class="card fold-card">
        <summary>
          <span>
            <strong>生成任务</strong>
            <small>查看评价生成记录和成功数量</small>
          </span>
          <span class="fold-hint">展开</span>
        </summary>
        <div class="fold-body">
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
      </details>
    </div>
  </div>
</template>

<style scoped>
.admin-header {
  align-items: center;
  justify-content: space-between;
}
.header-actions {
  align-items: center;
}
.stat-strip {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: 16px;
}
.stat-strip div {
  background: var(--surface);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 12px;
}
.stat-strip span,
.subtext {
  color: var(--muted);
  display: block;
  font-size: 12px;
}
.stat-strip strong {
  display: block;
  font-size: 24px;
  line-height: 1.1;
  margin-top: 4px;
}
.muted { color: #6b7280; font-size: 13px; margin: 0 0 10px; }
.inline-form {
  align-items: center;
  gap: 8px;
}
.inline-form input {
  flex: 1 1 180px;
}
.inline-form select {
  flex: 0 1 180px;
}
.section-head {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  margin-bottom: 6px;
}
.section-head h2 {
  margin-bottom: 4px;
}
.form-grid {
  align-items: start;
}
.fld { display: block; margin-bottom: 10px; font-size: 13px; color: #374151; }
.fld input, .fld select { display: block; width: 100%; margin-top: 4px; }
.form-actions {
  margin-top: 6px;
}
.created {
  margin-top: 14px;
  padding: 12px;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  font-size: 13px;
}
.created p {
  margin: 0 0 8px;
}
.created p:last-child {
  margin-bottom: 0;
}
.created code {
  background: #fff;
  padding: 4px 6px;
  border-radius: 6px;
}
.url-line {
  align-items: center;
  display: grid;
  gap: 8px;
  grid-template-columns: auto minmax(0, 1fr) auto;
}
.url-line code,
.url-cell code {
  overflow-wrap: anywhere;
  white-space: normal;
}
.url-cell {
  min-width: 280px;
  white-space: normal;
}
.link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 34px;
  margin-left: 6px;
  padding: 0 10px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  color: #2563eb;
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
}
.table-actions {
  display: inline-flex;
  gap: 8px;
}
.table-actions.compact {
  gap: 6px;
}
.table-actions button {
  min-height: 38px;
  padding: 8px 12px;
}
.mobile-store-list {
  display: none;
}
.mobile-store-item {
  border-top: 1px solid var(--border-soft);
  padding: 14px 0;
}
.mobile-store-item:first-child {
  border-top: 0;
  padding-top: 0;
}
.mobile-store-item:last-child {
  padding-bottom: 0;
}
.mobile-store-head {
  align-items: flex-start;
  display: flex;
  gap: 10px;
  justify-content: space-between;
}
.mobile-store-head strong {
  display: block;
  font-size: 17px;
}
.mobile-store-head span {
  color: var(--muted);
  display: block;
  font-size: 13px;
  margin-top: 3px;
}
.status-pill {
  border-radius: 999px;
  flex: 0 0 auto;
  font-size: 12px;
  padding: 4px 9px;
}
.status-pill.enabled {
  background: var(--success-bg);
  color: var(--success-text);
}
.status-pill.disabled {
  background: #f1f5f9;
  color: var(--muted);
}
.mobile-store-meta {
  display: grid;
  gap: 10px;
  margin: 12px 0;
}
.mobile-store-meta div {
  min-width: 0;
}
.mobile-store-meta dt {
  color: var(--muted);
  font-size: 12px;
  margin-bottom: 3px;
}
.mobile-store-meta dd {
  margin: 0;
  min-width: 0;
}
.mobile-store-meta code {
  display: block;
  overflow-wrap: anywhere;
  white-space: normal;
}
.mobile-store-actions {
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.mobile-store-actions button {
  min-height: 44px;
  padding: 8px 10px;
  width: 100%;
}
.fold-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: 1fr;
}
.fold-card {
  margin-bottom: 0;
  padding: 0;
}
.fold-card summary {
  align-items: center;
  cursor: pointer;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  list-style: none;
  padding: 16px 18px;
}
.fold-card summary::-webkit-details-marker {
  display: none;
}
.fold-card summary strong {
  display: block;
  font-size: 16px;
}
.fold-card summary small {
  color: var(--muted);
  display: block;
  font-size: 12px;
  margin-top: 2px;
}
.fold-hint {
  color: var(--primary-strong);
  flex: 0 0 auto;
  font-size: 13px;
  font-weight: 700;
}
.fold-card[open] .fold-hint {
  color: var(--muted);
}
.fold-card[open] .fold-hint::before {
  content: "收起";
}
.fold-card[open] .fold-hint {
  font-size: 0;
}
.fold-card[open] .fold-hint::before {
  font-size: 13px;
}
.fold-body {
  border-top: 1px solid var(--border-soft);
  padding: 14px 18px 18px;
}

@media (max-width: 640px) {
  .admin-header,
  .section-head,
  .url-line {
    display: grid;
    grid-template-columns: 1fr;
  }
  .header-actions,
  .form-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }
  .header-actions button,
  .form-actions button,
  .section-head button {
    width: 100%;
  }
  .form-actions button:only-child {
    grid-column: 1 / -1;
  }
  .stat-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .stat-strip div {
    padding: 10px;
  }
  .stat-strip strong {
    font-size: 20px;
  }
  .inline-form {
    display: grid;
    grid-template-columns: 1fr;
  }
  .link {
    width: 100%;
    min-height: 44px;
    margin: 8px 0 0;
  }
  .desktop-table {
    display: none;
  }
  .mobile-store-list {
    display: block;
  }
  .table-actions {
    display: grid;
    grid-template-columns: repeat(3, minmax(68px, 1fr));
    min-width: 220px;
  }
  .table-actions.compact {
    grid-template-columns: repeat(2, minmax(76px, 1fr));
    min-width: 160px;
  }
  .table-actions button {
    min-height: 44px;
    width: 100%;
  }
}
</style>
