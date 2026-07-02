<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adminApi } from '../../api/admin'
import type { AdminStats, AdminStore, ExternalStoreReviewMatch, ReviewCrawlBatch } from '../../api/admin'
import type { DeviceBreakdownItem } from '../../api/merchant'
import { copyToClipboard } from '../../utils/clipboard'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const merchants = ref<any[]>([])
const storeTypes = ref<any[]>([])
const stores = ref<AdminStore[]>([])
const tasks = ref<any[]>([])
const stats = ref<AdminStats>(emptyStats())
const loading = ref(false)
const error = ref('')
const notice = ref('')
const editingStoreId = ref<number | null>(null)
const crawlPanelStore = ref<AdminStore | null>(null)
const crawlBatches = ref<ReviewCrawlBatch[]>([])
const crawlMatches = ref<ExternalStoreReviewMatch[]>([])
const crawlLoading = ref(false)

const platformOptions = [
  { code: 'dianping', name: '大众点评' },
  { code: 'meituan', name: '美团' },
  { code: 'xiaohongshu', name: '小红书' },
  { code: 'douyin', name: '抖音' }
]

// 自定义类型只能挂在 9 个预置行业之一之下（生成/隔离基准）
const presetTypes = computed(() => storeTypes.value.filter((t) => t.isPreset))
const isEditingStore = computed(() => editingStoreId.value !== null)
const globalDeviceItems = computed(() => stats.value.deviceStats?.items || [])
const globalTopDevice = computed(() => dominantDevice(globalDeviceItems.value))
const updatedText = computed(() => {
  if (!stats.value.updatedAt) return ''
  const d = new Date(stats.value.updatedAt)
  if (Number.isNaN(d.getTime())) return ''
  return `更新至 ${d.toLocaleString('zh-CN', { hour12: false })}`
})
const deviceSummaryText = computed(() => {
  if (!stats.value.totalCustomerVisits) return '暂无顾客访问数据'
  if (!globalTopDevice.value) return '设备结构待积累'
  return `${globalTopDevice.value.label}最多，占 ${formatPercent(globalTopDevice.value.percent)}`
})
const topStores = computed(() => {
  return [...stores.value]
    .sort((a, b) => (b.analytics?.totalCustomerVisits || 0) - (a.analytics?.totalCustomerVisits || 0))
    .slice(0, 5)
})
const globalDeviceAria = computed(() => {
  if (!globalDeviceItems.value.length) return '暂无全局访问设备数据'
  return `全局访问设备占比：${globalDeviceItems.value.map((item) => `${item.label}${formatPercent(item.percent)}`).join('，')}`
})

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
  platformUrl: '',
  reviewCrawlPlatformCode: 'meituan',
  reviewCrawlExternalShopId: '',
  reviewCrawlEnabled: false
})
const lastCreated = ref<{ storeName: string; uuid: string; landingUrl: string; account: string } | null>(null)

function emptyStats(): AdminStats {
  return {
    merchantCount: 0,
    storeCount: 0,
    tagCount: 0,
    taskCount: 0,
    totalCustomerVisits: 0,
    currentWeekCustomerVisits: 0,
    currentMonthCustomerVisits: 0,
    totalPublishClicks: 0,
    currentWeekPublishClicks: 0,
    currentMonthPublishClicks: 0,
    deviceStats: { totalCount: 0, items: [] },
    updatedAt: ''
  }
}

function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

function typeName(typeId?: number) {
  return storeTypes.value.find((t) => t.id === typeId)?.name || '-'
}

function formatNumber(value: number | undefined) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function formatPercent(value: number | undefined) {
  return `${Number(value || 0).toFixed(1)}%`
}

function dominantDevice(items: DeviceBreakdownItem[]) {
  return items.reduce<DeviceBreakdownItem | null>((best, item) => {
    if (!best || item.count > best.count) return item
    return best
  }, null)
}

function deviceBarStyle(item: DeviceBreakdownItem) {
  return { width: `${Math.min(Math.max(item.percent, item.count > 0 ? 4 : 0), 100)}%` }
}

function primaryDeviceText(item: AdminStore) {
  const top = dominantDevice(item.analytics?.deviceStats?.items || [])
  if (!top) return '暂无设备数据'
  return `${top.label} ${formatPercent(top.percent)}`
}

function platformDisplayName(code?: string) {
  return platformOptions.find((item) => item.code === code)?.name || code || '-'
}

function storeVisitText(item: AdminStore) {
  const visits = item.analytics?.totalCustomerVisits || 0
  const publishes = item.analytics?.totalPublishClicks || 0
  return `${formatNumber(visits)} 访问 / ${formatNumber(publishes)} 发布`
}

function crawlStatusText(status?: string) {
  const map: Record<string, string> = {
    never_run: '未采集',
    running: '采集中',
    success: '已采集',
    failed: '采集失败'
  }
  return map[status || ''] || status || '未配置'
}

function crawlConfigText(item: AdminStore) {
  const cfg = item.reviewCrawl
  if (!cfg || !cfg.externalShopId) return '未配置'
  const enabledText = cfg.enabled ? '启用' : '未启用'
  return `${platformDisplayName(cfg.platformCode)} ${cfg.externalShopId} · ${enabledText}`
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString('zh-CN', { hour12: false })
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
  newStore.reviewCrawlPlatformCode = 'meituan'
  newStore.reviewCrawlExternalShopId = ''
  newStore.reviewCrawlEnabled = false
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
  newStore.reviewCrawlPlatformCode = item.reviewCrawl?.platformCode || 'meituan'
  newStore.reviewCrawlExternalShopId = item.reviewCrawl?.externalShopId || ''
  newStore.reviewCrawlEnabled = !!item.reviewCrawl?.enabled
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
    platformUrl: newStore.platformUrl.trim() || undefined,
    reviewCrawlPlatformCode: newStore.reviewCrawlPlatformCode,
    reviewCrawlExternalShopId: newStore.reviewCrawlExternalShopId.trim() || undefined,
    reviewCrawlEnabled: newStore.reviewCrawlEnabled
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

async function runStoreReviewCrawl(item: AdminStore) {
  await runAction(() => adminApi.runStoreReviewCrawl(item.id), '评论采集已同步')
  if (crawlPanelStore.value?.id === item.id) {
    await loadReviewCrawl(item)
  }
}

async function loadReviewCrawl(item: AdminStore) {
  crawlLoading.value = true
  error.value = ''
  try {
    const [batchRes, matchRes] = await Promise.all([
      adminApi.listStoreReviewCrawlBatches(item.id),
      adminApi.listStoreReviewCrawlMatches(item.id)
    ])
    crawlPanelStore.value = item
    crawlBatches.value = batchRes.data.data
    crawlMatches.value = matchRes.data.data
  } catch (err: any) {
    error.value = messageFrom(err, '评论采集数据加载失败')
  } finally {
    crawlLoading.value = false
  }
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
      <div>
        <span>商家</span>
        <strong>{{ formatNumber(stats.merchantCount) }}</strong>
        <small>{{ formatNumber(stats.storeCount) }} 个门店 URL</small>
      </div>
      <div>
        <span>客户访问</span>
        <strong>{{ formatNumber(stats.totalCustomerVisits) }}</strong>
        <small>本月 {{ formatNumber(stats.currentMonthCustomerVisits) }}</small>
      </div>
      <div>
        <span>引导发布</span>
        <strong>{{ formatNumber(stats.totalPublishClicks) }}</strong>
        <small>本周 {{ formatNumber(stats.currentWeekPublishClicks) }}</small>
      </div>
      <div>
        <span>生成任务</span>
        <strong>{{ formatNumber(stats.taskCount) }}</strong>
        <small>{{ updatedText || '等待数据' }}</small>
      </div>
    </div>

    <div class="ops-grid" aria-label="运营总览">
      <section class="ops-panel" aria-labelledby="admin-device-title">
        <div class="panel-head">
          <div>
            <h2 id="admin-device-title">全局访问设备</h2>
            <p class="muted">{{ deviceSummaryText }}</p>
          </div>
          <strong>{{ formatNumber(stats.deviceStats?.totalCount) }}</strong>
        </div>
        <div v-if="globalDeviceItems.length" class="device-bars" role="img" :aria-label="globalDeviceAria">
          <div v-for="item in globalDeviceItems" :key="item.code" class="device-row">
            <div class="device-row-head">
              <span>{{ item.label }}</span>
              <b>{{ formatNumber(item.count) }} · {{ formatPercent(item.percent) }}</b>
            </div>
            <div class="device-track" aria-hidden="true">
              <span :style="deviceBarStyle(item)"></span>
            </div>
          </div>
        </div>
        <p v-else class="empty-note">顾客打开交付 URL 后，这里会出现设备结构。</p>
      </section>

      <section class="ops-panel" aria-labelledby="top-store-title">
        <div class="panel-head">
          <div>
            <h2 id="top-store-title">商家访问排行</h2>
            <p class="muted">按访问量排序，辅助判断交付后是否真实使用</p>
          </div>
        </div>
        <ol v-if="topStores.length" class="store-rank">
          <li v-for="item in topStores" :key="item.id">
            <span>
              <b>{{ item.storeName }}</b>
              <small>{{ primaryDeviceText(item) }}</small>
            </span>
            <strong>{{ formatNumber(item.analytics?.totalCustomerVisits) }}</strong>
          </li>
        </ol>
        <p v-else class="empty-note">暂无门店数据。</p>
      </section>
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
      <div class="crawl-config-grid" aria-label="评论采集配置">
        <label class="fld">采集平台
          <select v-model="newStore.reviewCrawlPlatformCode">
            <option value="meituan">美团</option>
          </select>
        </label>
        <label class="fld">美团商家 ID
          <input v-model="newStore.reviewCrawlExternalShopId" placeholder="如 1953748828" inputmode="numeric" />
        </label>
        <label class="check-field">
          <input v-model="newStore.reviewCrawlEnabled" type="checkbox" />
          <span>启用每 7 天评论采集</span>
        </label>
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
        <thead><tr><th>门店</th><th>商家账号</th><th>后台数据</th><th>主力设备</th><th>评论采集</th><th>交付 URL</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="item in stores" :key="item.id">
            <td>
              <strong>{{ item.storeName }}</strong>
              <span class="subtext">ID {{ item.id }} · {{ typeName(item.typeId) }}</span>
            </td>
            <td>{{ item.merchantAccount || merchantForStore(item).account || '-' }}</td>
            <td>
              <strong>{{ storeVisitText(item) }}</strong>
              <span class="subtext">本月访问 {{ formatNumber(item.analytics?.currentMonthCustomerVisits) }}</span>
            </td>
            <td>{{ primaryDeviceText(item) }}</td>
            <td>
              <strong>{{ crawlConfigText(item) }}</strong>
              <span class="subtext">
                {{ crawlStatusText(item.reviewCrawl?.lastStatus) }}
                <template v-if="item.reviewCrawl?.nextCrawlAt"> · 下次 {{ formatDateTime(item.reviewCrawl.nextCrawlAt) }}</template>
              </span>
            </td>
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
                <button class="secondary" :disabled="loading || !item.reviewCrawl?.enabled" @click="runStoreReviewCrawl(item)">同步评论</button>
                <button class="secondary" :disabled="loading" @click="loadReviewCrawl(item)">查看采集</button>
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
              <dt>后台数据</dt>
              <dd>{{ storeVisitText(item) }}</dd>
            </div>
            <div>
              <dt>主力设备</dt>
              <dd>{{ primaryDeviceText(item) }}</dd>
            </div>
            <div>
              <dt>评论采集</dt>
              <dd>
                {{ crawlConfigText(item) }}
                <span class="subtext">{{ crawlStatusText(item.reviewCrawl?.lastStatus) }}</span>
              </dd>
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
            <button class="secondary" :disabled="loading || !item.reviewCrawl?.enabled" @click="runStoreReviewCrawl(item)">同步评论</button>
            <button class="secondary" :disabled="loading" @click="loadReviewCrawl(item)">查看采集</button>
            <button class="danger" :disabled="loading" @click="deleteStore(item)">删除</button>
          </div>
        </article>
      </div>

      <section v-if="crawlPanelStore" class="crawl-panel" aria-labelledby="crawl-panel-title">
        <div class="panel-head">
          <div>
            <h3 id="crawl-panel-title">{{ crawlPanelStore.storeName }} 评论采集</h3>
            <p class="muted">批次数据和 C 验证明细仅管理员可见。</p>
          </div>
          <button class="secondary" :disabled="crawlLoading" @click="crawlPanelStore = null">关闭</button>
        </div>
        <p v-if="crawlLoading" class="empty-note">加载中...</p>
        <div v-else class="crawl-tables">
          <div>
            <h4>最近批次</h4>
            <table>
              <thead><tr><th>批次</th><th>状态</th><th>类型</th><th>入库</th><th>C 验证</th><th>完成时间</th></tr></thead>
              <tbody>
                <tr v-for="batch in crawlBatches" :key="batch.id">
                  <td>#{{ batch.id }}</td>
                  <td>{{ crawlStatusText(batch.status) }}</td>
                  <td>{{ batch.isBaseline ? '基线' : '周期' }}</td>
                  <td>{{ formatNumber(batch.insertedRowCount) }}</td>
                  <td>{{ formatNumber(batch.matchedReviewCount) }}</td>
                  <td>{{ formatDateTime(batch.finishedAt) }}</td>
                </tr>
              </tbody>
            </table>
            <p v-if="!crawlBatches.length" class="empty-note">暂无采集批次。</p>
          </div>
          <div>
            <h4>C 验证明细</h4>
            <table>
              <thead><tr><th>评论时间</th><th>用户</th><th>匹配分</th><th>内容</th></tr></thead>
              <tbody>
                <tr v-for="match in crawlMatches" :key="match.id">
                  <td>{{ formatDateTime(match.reviewTime) }}</td>
                  <td>{{ match.userName || '-' }}</td>
                  <td>{{ formatPercent((match.matchScore || 0) * 100) }}</td>
                  <td class="match-content">{{ match.content || '-' }}</td>
                </tr>
              </tbody>
            </table>
            <p v-if="!crawlMatches.length" class="empty-note">暂无 C 验证明细。</p>
          </div>
        </div>
      </section>
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
.stat-strip small {
  color: var(--muted);
  display: block;
  font-size: 12px;
  margin-top: 6px;
}
.ops-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: minmax(0, 1.15fr) minmax(280px, 0.85fr);
  margin-bottom: 16px;
}
.ops-panel {
  background: var(--surface);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 16px;
}
.panel-head {
  align-items: start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  margin-bottom: 12px;
}
.panel-head h2 {
  margin-bottom: 4px;
}
.panel-head strong {
  font-size: 24px;
  line-height: 1;
}
.device-bars {
  display: grid;
  gap: 12px;
}
.device-row-head {
  align-items: center;
  display: flex;
  gap: 10px;
  justify-content: space-between;
  margin-bottom: 6px;
}
.device-row-head span {
  color: var(--text);
  font-weight: 800;
}
.device-row-head b {
  color: var(--muted);
  font-size: 13px;
  font-weight: 700;
}
.device-track {
  background: #eef2f7;
  border-radius: 999px;
  height: 10px;
  overflow: hidden;
}
.device-track span {
  background: #2563eb;
  border-radius: inherit;
  display: block;
  height: 100%;
}
.empty-note {
  color: var(--muted);
  margin: 0;
}
.store-rank {
  display: grid;
  gap: 10px;
  list-style: none;
  margin: 0;
  padding: 0;
}
.store-rank li {
  align-items: center;
  background: #f8fafc;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  min-height: 52px;
  padding: 9px 10px;
}
.store-rank b,
.store-rank small {
  display: block;
}
.store-rank small {
  color: var(--muted);
  font-size: 12px;
  margin-top: 2px;
}
.store-rank strong {
  flex: 0 0 auto;
  font-size: 20px;
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
.crawl-config-grid {
  align-items: end;
  border-top: 1px solid var(--border-soft);
  display: grid;
  gap: 10px;
  grid-template-columns: minmax(150px, 0.7fr) minmax(220px, 1fr) minmax(180px, 0.8fr);
  margin-top: 8px;
  padding-top: 12px;
}
.check-field {
  align-items: center;
  color: #374151;
  display: inline-flex;
  font-size: 13px;
  gap: 8px;
  min-height: 42px;
}
.check-field input {
  height: 18px;
  width: 18px;
}
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
  display: inline-grid;
  gap: 8px;
  grid-template-columns: repeat(2, minmax(0, auto));
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
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
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
.crawl-panel {
  border-top: 1px solid var(--border-soft);
  margin-top: 18px;
  padding-top: 16px;
}
.crawl-panel h3,
.crawl-panel h4 {
  margin: 0 0 6px;
}
.crawl-tables {
  display: grid;
  gap: 14px;
  grid-template-columns: 1fr;
}
.match-content {
  max-width: 360px;
  overflow-wrap: anywhere;
  white-space: normal;
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
  .crawl-config-grid {
    grid-template-columns: 1fr;
  }
  .ops-grid {
    grid-template-columns: 1fr;
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
