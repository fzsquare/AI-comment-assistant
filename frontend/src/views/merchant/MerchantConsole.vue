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
const platformForm = reactive({ platformCode: 'meituan', platformName: '美团', buttonText: '去美团评论', targetUrl: 'https://www.meituan.com/', backupUrl: '', sortNo: 1, status: 1 })
const reviewText = ref('')
const keywords = ref<any[]>([])
const images = ref<any[]>([])
const links = ref<any[]>([])
const reviews = ref<any[]>([])
const tasks = ref<any[]>([])

async function loadAll() {
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
}

async function saveStore() {
  await merchantApi.updateStoreDetail(storeForm)
  alert('门店信息已保存')
}

async function addKeyword() {
  if (!keyword.value) return
  await merchantApi.createKeyword({ keyword: keyword.value, sortNo: keywords.value.length + 1 })
  keyword.value = ''
  await loadAll()
}

async function addImage() {
  if (!imageUrl.value) return
  await merchantApi.createImage({ imageUrl: imageUrl.value, thumbnailUrl: imageUrl.value, sortNo: images.value.length + 1 })
  imageUrl.value = ''
  await loadAll()
}

async function addPlatformLink() {
  await merchantApi.createPlatformLink(platformForm)
  await loadAll()
}

async function addReview() {
  if (!reviewText.value) return
  await merchantApi.createReview({ content: reviewText.value, status: 'available' })
  reviewText.value = ''
  await loadAll()
}

async function generateReviews() {
  await merchantApi.generateReviews(10)
  await loadAll()
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
      <button class="secondary" @click="logout">退出登录</button>
    </div>

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
        <button @click="saveStore">保存</button>
      </div>

      <div class="card">
        <h2>AI 生成任务</h2>
        <button @click="generateReviews">生成 10 条评价</button>
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
          <button @click="addKeyword">添加</button>
        </div>
        <ul>
          <li v-for="item in keywords" :key="item.id">{{ item.keyword }}</li>
        </ul>
      </div>

      <div class="card">
        <h2>图片管理</h2>
        <div class="row">
          <input v-model="imageUrl" placeholder="图片 URL" />
          <button @click="addImage">添加</button>
        </div>
        <div class="row">
          <img v-for="item in images" :key="item.id" :src="item.thumbnailUrl || item.imageUrl" style="width: 120px; height: 80px; object-fit: cover; border-radius: 10px" />
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
        <button @click="addPlatformLink">新增平台入口</button>
        <ul>
          <li v-for="item in links" :key="item.id">{{ item.buttonText }} - {{ item.targetUrl }}</li>
        </ul>
      </div>

      <div class="card">
        <h2>评价管理</h2>
        <textarea v-model="reviewText" placeholder="新增手工评价"></textarea>
        <div style="height: 8px"></div>
        <button @click="addReview">添加评价</button>
        <ul>
          <li v-for="item in reviews.slice(0, 8)" :key="item.id">{{ item.content }}</li>
        </ul>
      </div>
    </div>
  </div>
</template>
