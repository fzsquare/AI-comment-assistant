<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
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
const reviewText = ref('')
const reviewPlatformCode = ref('')
const keywords = ref<any[]>([])
const images = ref<any[]>([])
const links = ref<any[]>([])
const reviews = ref<any[]>([])
const tasks = ref<any[]>([])
const loading = ref(false)
const error = ref('')
const notice = ref('')

function messageFrom(err: any, fallback: string) {
  return err?.response?.data?.message || err?.message || fallback
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [storeRes, keywordRes, imageRes, linkRes, reviewRes, taskRes] = await Promise.all([
      merchantApi.getStoreDetail(),
      merchantApi.listKeywords(),
      merchantApi.listImages(),
      merchantApi.listPlatformLinks(),
      merchantApi.listReviews(),
      merchantApi.listTasks()
    ])
    Object.assign(storeForm, storeRes.data.data)
    keywords.value = keywordRes.data.data
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

async function addKeyword() {
  const value = keyword.value.trim()
  if (!value) return
  if (await runAction(() => merchantApi.createKeyword({ keyword: value, sortNo: keywords.value.length + 1 }), '关键词已添加')) {
    keyword.value = ''
  }
}

async function addImage() {
  const value = imageUrl.value.trim()
  if (!value) return
  if (await runAction(() => merchantApi.createImage({ imageUrl: value, thumbnailUrl: value, sortNo: images.value.length + 1 }), '图片已添加')) {
    imageUrl.value = ''
  }
}

async function addPlatformLink() {
  if (!platformForm.platformCode.trim() || !platformForm.targetUrl.trim()) {
    error.value = '请填写平台编码和主跳转链接'
    return
  }
  if (await runAction(() => merchantApi.createPlatformLink(platformForm), '平台入口已新增')) {
    platformForm.platformCode = ''
    platformForm.platformName = ''
    platformForm.buttonText = ''
    platformForm.targetUrl = ''
    platformForm.backupUrl = ''
    platformForm.sortNo = links.value.length + 1
    platformForm.status = 1
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
  await runAction(() => merchantApi.updatePlatformLinkStatus(item.id, nextStatus), '平台入口状态已更新')
}

async function deletePlatformLink(id: number) {
  if (!window.confirm('确认删除这个平台入口？')) return
  await runAction(() => merchantApi.deletePlatformLink(id), '平台入口已删除')
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
  location.href = '/merchant/login'
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
        <div class="row">
          <input v-model="keyword" placeholder="新增关键词" />
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
        <div class="row">
          <input v-model="imageUrl" placeholder="图片 URL" />
          <button :disabled="loading" @click="addImage">添加</button>
        </div>
        <div class="row">
          <div v-for="item in images" :key="item.id" class="image-item">
            <img :src="item.thumbnailUrl || item.imageUrl" />
            <button class="danger" :disabled="loading" @click="deleteImage(item.id)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <div class="grid-2">
      <div class="card">
        <h2>平台入口配置</h2>
        <input v-model="platformForm.platformCode" placeholder="平台编码" />
        <div style="height: 8px"></div>
        <input v-model="platformForm.platformName" placeholder="平台名称" />
        <div style="height: 8px"></div>
        <input v-model="platformForm.buttonText" placeholder="按钮文案" />
        <div style="height: 8px"></div>
        <input v-model="platformForm.targetUrl" placeholder="主跳转链接" />
        <div style="height: 8px"></div>
        <input v-model="platformForm.backupUrl" placeholder="备用链接" />
        <div style="height: 8px"></div>
        <button :disabled="loading" @click="addPlatformLink">新增平台入口</button>
        <ul>
          <li v-for="item in links" :key="item.id" class="list-action">
            <span>{{ item.buttonText }} - {{ item.targetUrl }}（{{ numericStatusText(item.status) }}）</span>
            <span class="row">
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
