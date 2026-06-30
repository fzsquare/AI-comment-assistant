<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { merchantApi } from '../../api/merchant'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const storeForm = reactive({
  storeName: '',
  industryType: '',
  storeIntro: '',
  address: '',
  primaryPlatformStyle: 'xiaohongshu',
  brandTone: ''
})
const keyword = ref('')
const imageUrl = ref('')
const platformForm = reactive({ platformCode: '', platformName: '', buttonText: '', targetUrl: '', backupUrl: '', sortNo: 1, status: 1 })
const editingPlatformLinkId = ref<number | null>(null)
const reviewText = ref('')
const reviewPlatformCode = ref('')
const keywords = ref<any[]>([])
const suggestedTags = ref<string[]>([])
const images = ref<any[]>([])
// 还没添加的推荐标签
const availableSuggestions = computed(() =>
  suggestedTags.value.filter((t) => !keywords.value.some((k) => k.keyword === t))
)
const links = ref<any[]>([])
const reviews = ref<any[]>([])
const tasks = ref<any[]>([])
const loading = ref(false)
const error = ref('')
const notice = ref('')
const isEditingPlatformLink = computed(() => editingPlatformLinkId.value !== null)

const platformPresets: Record<string, { name: string; buttonText: string }> = {
  dianping: { name: '大众点评', buttonText: '去大众点评发布' },
  meituan: { name: '美团', buttonText: '去美团发布' },
  xiaohongshu: { name: '小红书', buttonText: '去小红书发布' },
  douyin: { name: '抖音', buttonText: '去抖音发布' }
}

function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [storeRes, keywordRes, suggestRes, imageRes, linkRes, reviewRes, taskRes] = await Promise.all([
      merchantApi.getStoreDetail(),
      merchantApi.listKeywords(),
      merchantApi.getKeywordSuggestions(),
      merchantApi.listImages(),
      merchantApi.listPlatformLinks(),
      merchantApi.listReviews(),
      merchantApi.listTasks()
    ])
    Object.assign(storeForm, storeRes.data.data)
    keywords.value = keywordRes.data.data
    suggestedTags.value = suggestRes.data.data?.tags || []
    images.value = imageRes.data.data
    links.value = linkRes.data.data
    reviews.value = reviewRes.data.data
    tasks.value = taskRes.data.data
    if (!reviewPlatformCode.value && links.value.length > 0) {
      reviewPlatformCode.value = links.value[0].platformCode
    }
  } catch (err: any) {
    error.value = messageFrom(err, '商家后台数据加载失败')
  } finally {
    loading.value = false
  }
}

async function runAction(action: () => Promise<unknown>, success: string, reload = true) {
  error.value = ''
  notice.value = ''
  try {
    await action()
    notice.value = success
    if (reload) {
      await loadAll()
    }
    return true
  } catch (err: any) {
    error.value = messageFrom(err, '操作失败')
    return false
  }
}

async function saveStore() {
  await runAction(() => merchantApi.updateStoreDetail(storeForm), '门店信息已保存')
}

function resetPlatformForm() {
  editingPlatformLinkId.value = null
  platformForm.platformCode = ''
  platformForm.platformName = ''
  platformForm.buttonText = ''
  platformForm.targetUrl = ''
  platformForm.backupUrl = ''
  platformForm.sortNo = links.value.length + 1
  platformForm.status = 1
}

function applyPlatformPreset() {
  const preset = platformPresets[platformForm.platformCode]
  if (!preset) return
  if (!platformForm.platformName.trim()) platformForm.platformName = preset.name
  if (!platformForm.buttonText.trim()) platformForm.buttonText = preset.buttonText
}

async function addKeyword() {
  const value = keyword.value.trim()
  if (!value) return
  if (await runAction(() => merchantApi.createKeyword({ keyword: value, sortNo: keywords.value.length + 1 }), '关键词已添加')) {
    keyword.value = ''
  }
}

async function addSuggested(tag: string) {
  await runAction(() => merchantApi.createKeyword({ keyword: tag, sortNo: keywords.value.length + 1 }), '已添加推荐标签')
}

async function addImage() {
  const value = imageUrl.value.trim()
  if (!value) return
  if (await runAction(() => merchantApi.createImage({ imageUrl: value, thumbnailUrl: value, sortNo: images.value.length + 1 }), '图片已添加')) {
    imageUrl.value = ''
  }
}

async function onPickImage(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files && input.files[0]
  if (!file) return
  if (file.size > 5 * 1024 * 1024) {
    error.value = '图片需在 5MB 以内'
    input.value = ''
    return
  }
  await runAction(() => merchantApi.uploadImageFile(file), '图片已上传')
  input.value = '' // 允许再次选同一文件
}

function editPlatformLink(item: any) {
  editingPlatformLinkId.value = item.id
  platformForm.platformCode = item.platformCode || ''
  platformForm.platformName = item.platformName || ''
  platformForm.buttonText = item.buttonText || ''
  platformForm.targetUrl = item.targetUrl || ''
  platformForm.backupUrl = item.backupUrl || ''
  platformForm.sortNo = item.sortNo || 1
  platformForm.status = item.status === 0 ? 0 : 1
}

async function savePlatformLink() {
  applyPlatformPreset()
  if (!platformForm.platformCode.trim() || !platformForm.targetUrl.trim()) {
    error.value = '请填写平台编码和客户端跳转链接'
    return
  }
  const payload = {
    platformCode: platformForm.platformCode.trim(),
    platformName: platformForm.platformName.trim() || platformForm.platformCode.trim(),
    buttonText: platformForm.buttonText.trim() || '去发布',
    targetUrl: platformForm.targetUrl.trim(),
    backupUrl: platformForm.backupUrl.trim(),
    sortNo: platformForm.sortNo || links.value.length + 1,
    status: platformForm.status || 1
  }
  const action = isEditingPlatformLink.value
    ? () => merchantApi.updatePlatformLink(editingPlatformLinkId.value as number, payload)
    : () => merchantApi.createPlatformLink(payload)
  const success = isEditingPlatformLink.value ? '客户端跳转链接已保存' : '客户端跳转链接已新增'
  if (await runAction(action, success)) {
    resetPlatformForm()
  }
}

async function addReview() {
  const value = reviewText.value.trim()
  if (!value) return
  if (!reviewPlatformCode.value) {
    error.value = '请先选择评价平台'
    return
  }
  if (await runAction(() => merchantApi.createReview({ content: value, status: 'available', platformCode: reviewPlatformCode.value }), '评价已添加')) {
    reviewText.value = ''
  }
}

async function generateReviews() {
  if (!reviewPlatformCode.value) {
    error.value = '请先选择评价平台'
    return
  }
  await runAction(() => merchantApi.generateReviews(reviewPlatformCode.value, 10), '评价生成任务已完成')
}

async function deleteKeyword(id: number) {
  if (!window.confirm('确认删除这个关键词？')) return
  await runAction(() => merchantApi.deleteKeyword(id), '关键词已删除')
}

async function deleteImage(id: number) {
  if (!window.confirm('确认删除这张图片？')) return
  await runAction(() => merchantApi.deleteImage(id), '图片已删除')
}

async function togglePlatformLinkStatus(item: any) {
  const nextStatus = item.status === 1 ? 0 : 1
  await runAction(() => merchantApi.updatePlatformLinkStatus(item.id, nextStatus), '跳转链接状态已更新')
}

async function deletePlatformLink(id: number) {
  if (!window.confirm('确认删除这个客户端跳转链接？')) return
  await runAction(() => merchantApi.deletePlatformLink(id), '跳转链接已删除')
  if (editingPlatformLinkId.value === id) resetPlatformForm()
}

async function deleteReview(id: number) {
  if (!window.confirm('确认删除这条评价？')) return
  await runAction(() => merchantApi.deleteReview(id), '评价已删除')
}

function numericStatusText(status: number) {
  return status === 1 ? '启用' : '禁用'
}

function logout() {
  auth.clear()
  location.href = import.meta.env.BASE_URL + 'merchant/login'
}

onMounted(loadAll)
</script>

<template>
  <div class="page">
    <div class="row" style="justify-content: space-between; align-items: center">
      <h1>商家后台</h1>
      <div class="row">
        <button class="secondary" :disabled="loading" @click="loadAll">刷新</button>
        <button class="secondary" @click="logout">退出登录</button>
      </div>
    </div>
    <p v-if="error" class="alert">{{ error }}</p>
    <p v-else-if="notice" class="notice">{{ notice }}</p>

    <div class="grid-2">
      <div class="card">
        <h2>店铺信息</h2>
        <input v-model="storeForm.storeName" placeholder="门店名称" />
        <div style="height: 8px"></div>
        <input v-model="storeForm.industryType" placeholder="行业类型" />
        <div style="height: 8px"></div>
        <input v-model="storeForm.address" placeholder="门店地址" />
        <div style="height: 8px"></div>
        <input v-model="storeForm.primaryPlatformStyle" placeholder="主平台风格" />
        <div style="height: 8px"></div>
        <textarea v-model="storeForm.storeIntro" placeholder="门店简介"></textarea>
        <div style="height: 8px"></div>
        <input v-model="storeForm.brandTone" placeholder="品牌调性" />
        <div style="height: 8px"></div>
        <button :disabled="loading" @click="saveStore">保存</button>
      </div>

      <div class="card">
        <h2>AI 生成任务</h2>
        <select v-model="reviewPlatformCode">
          <option value="" disabled>选择评价平台</option>
          <option v-for="item in links" :key="item.id" :value="item.platformCode">
            {{ item.platformName || item.platformCode }}
          </option>
        </select>
        <div style="height: 8px"></div>
        <button :disabled="loading" @click="generateReviews">生成 10 条评价</button>
        <table>
          <thead><tr><th>ID</th><th>类型</th><th>状态</th><th>成功数</th></tr></thead>
          <tbody>
            <tr v-for="task in tasks" :key="task.id">
              <td>{{ task.id }}</td>
              <td>{{ task.triggerType }}</td>
              <td>{{ task.status }}</td>
              <td>{{ task.successCount }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="grid-2">
      <div class="card">
        <h2>关键词管理</h2>
        <p class="muted" style="margin: 0 0 8px">这些标签会显示给顾客选择，用于生成更贴合的评价。</p>
        <div v-if="availableSuggestions.length" style="margin-bottom: 10px">
          <p class="muted" style="margin: 0 0 6px">本行业推荐标签（点击添加）：</p>
          <div class="row" style="gap: 8px">
            <button
              v-for="tag in availableSuggestions"
              :key="tag"
              class="suggest-chip"
              :disabled="loading"
              @click="addSuggested(tag)"
            >+ {{ tag }}</button>
          </div>
        </div>
        <div class="row">
          <input v-model="keyword" placeholder="自定义关键词" />
          <button :disabled="loading" @click="addKeyword">添加</button>
        </div>
        <ul>
          <li v-for="item in keywords" :key="item.id" class="list-action">
            <span>{{ item.keyword }}</span>
            <button class="danger" :disabled="loading" @click="deleteKeyword(item.id)">删除</button>
          </li>
        </ul>
      </div>

      <div class="card">
        <h2>图片管理</h2>
        <p class="muted" style="margin: 0 0 8px">上传店铺/菜品图片，顾客落地页会展示，可长按保存：</p>
        <label class="upload-btn">
          <span>上传图片</span>
          <input type="file" accept="image/*" :disabled="loading" @change="onPickImage" style="display: none" />
        </label>
        <details style="margin: 10px 0">
          <summary class="muted" style="cursor: pointer">或：贴图片 URL</summary>
          <div class="row" style="margin-top: 8px">
            <input v-model="imageUrl" placeholder="图片 URL" />
            <button :disabled="loading" @click="addImage">添加</button>
          </div>
        </details>
        <div class="row">
          <div v-for="item in images" :key="item.id" class="image-item">
            <img :src="item.thumbnailUrl || item.imageUrl" alt="店铺图片" />
            <button class="danger" :disabled="loading" @click="deleteImage(item.id)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <div class="grid-2">
      <div class="card">
        <h2>客户端跳转链接</h2>
        <p class="muted" style="margin: 0 0 8px">顾客从落地页点按钮时打开这里配置的商家链接。</p>
        <input v-model.trim="platformForm.platformCode" list="platform-codes" placeholder="平台编码，如 dianping" @change="applyPlatformPreset" />
        <datalist id="platform-codes">
          <option value="dianping">大众点评</option>
          <option value="meituan">美团</option>
          <option value="xiaohongshu">小红书</option>
          <option value="douyin">抖音</option>
        </datalist>
        <div style="height: 8px"></div>
        <input v-model="platformForm.platformName" placeholder="平台名称" />
        <div style="height: 8px"></div>
        <input v-model="platformForm.buttonText" placeholder="按钮文案" />
        <div style="height: 8px"></div>
        <input v-model.trim="platformForm.targetUrl" placeholder="客户端跳转链接" />
        <div style="height: 8px"></div>
        <input v-model.trim="platformForm.backupUrl" placeholder="备用链接（选填）" />
        <div style="height: 8px"></div>
        <div class="row">
          <button :disabled="loading" @click="savePlatformLink">
            {{ isEditingPlatformLink ? '保存跳转链接' : '新增跳转链接' }}
          </button>
          <button v-if="isEditingPlatformLink" class="secondary" :disabled="loading" @click="resetPlatformForm">取消编辑</button>
        </div>
        <ul>
          <li v-for="item in links" :key="item.id" class="list-action">
            <span>{{ item.buttonText }} - {{ item.targetUrl }}（{{ numericStatusText(item.status) }}）</span>
            <span class="row">
              <button class="secondary" :disabled="loading" @click="editPlatformLink(item)">编辑</button>
              <button class="secondary" :disabled="loading" @click="togglePlatformLinkStatus(item)">
                {{ item.status === 1 ? '禁用' : '启用' }}
              </button>
              <button class="danger" :disabled="loading" @click="deletePlatformLink(item.id)">删除</button>
            </span>
          </li>
        </ul>
      </div>

      <div class="card">
        <h2>评价管理</h2>
        <select v-model="reviewPlatformCode">
          <option value="" disabled>选择评价平台</option>
          <option v-for="item in links" :key="item.id" :value="item.platformCode">
            {{ item.platformName || item.platformCode }}
          </option>
        </select>
        <div style="height: 8px"></div>
        <textarea v-model="reviewText" placeholder="新增手工评价"></textarea>
        <div style="height: 8px"></div>
        <button :disabled="loading" @click="addReview">添加评价</button>
        <ul>
          <li v-for="item in reviews.slice(0, 8)" :key="item.id" class="list-action">
            <span>{{ item.platformStyle }} - {{ item.content }}</span>
            <button class="danger" :disabled="loading" @click="deleteReview(item.id)">删除</button>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.upload-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 48px;
  min-width: 132px;
  padding: 10px 16px;
  background: #3b82f6;
  color: #fff;
  border-radius: 10px;
  cursor: pointer;
  font-size: 15px;
  font-weight: 700;
  touch-action: manipulation;
}
.upload-btn:hover {
  background: #2563eb;
}
.suggest-chip {
  min-height: 40px;
  padding: 8px 14px;
  border-radius: 999px;
  border: 1px dashed #93c5fd;
  background: #f0f7ff;
  color: #1d4ed8;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.suggest-chip:hover {
  background: #dbeafe;
}
.suggest-chip:disabled {
  opacity: 0.6;
}

@media (max-width: 640px) {
  .upload-btn {
    width: 100%;
  }
  .suggest-chip {
    flex: 1 1 calc(50% - 8px);
    min-height: 44px;
  }
}
</style>
