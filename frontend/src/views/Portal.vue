<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
const go = (path: string) => router.push(path)
</script>

<template>
  <div class="portal">
    <header class="head">
      <h1>NFC 评价助手</h1>
      <p class="sub">到店碰一碰，几秒生成一条真实评价</p>
    </header>

    <section class="entries">
      <button class="entry merchant" @click="go('/merchant/login')">
        <span class="ico">🏪</span>
        <span class="t">商家入口</span>
        <span class="d">管理店铺、上传图片、配置标签、生成评价</span>
      </button>

      <button class="entry admin" @click="go('/admin/login')">
        <span class="ico">🛠️</span>
        <span class="t">管理员入口</span>
        <span class="d">管理商家 / 门店 / NFC 标签，查看统计</span>
      </button>

      <div class="entry customer">
        <span class="ico">📱</span>
        <span class="t">顾客入口</span>
        <span class="d">顾客无需登录，到店碰一碰桌上的 NFC 卡即可打开</span>
        <button class="demo" @click="go('/landing/11111111-1111-4111-8111-111111111111')">查看顾客页演示</button>
      </div>
    </section>

    <p class="foot">企业内部使用 · 请通过对应入口登录</p>
  </div>
</template>

<style scoped>
.portal {
  max-width: 560px;
  margin: 0 auto;
  padding: 32px 16px calc(24px + env(safe-area-inset-bottom));
}
.head {
  text-align: center;
  margin-bottom: 24px;
}
.head h1 {
  margin: 0;
  font-size: 24px;
}
.sub {
  margin: 8px 0 0;
  color: #667085;
  font-size: 14px;
}
.entries {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.entry {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  text-align: left;
  gap: 4px;
  width: 100%;
  background: #fff;
  border: 1px solid #eef0f4;
  border-radius: 16px;
  padding: 20px 18px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.05);
  cursor: pointer;
}
.entry:hover {
  border-color: #c7d2fe;
}
.entry .ico {
  font-size: 26px;
}
.entry .t {
  font-size: 17px;
  font-weight: 700;
  color: #1f2937;
}
.entry .d {
  font-size: 13px;
  color: #667085;
  line-height: 1.5;
}
.customer {
  cursor: default;
}
.customer:hover {
  border-color: #eef0f4;
}
.demo {
  margin-top: 10px;
  align-self: stretch;
  min-height: 44px;
  background: #eef2ff;
  color: #1e40af;
  border: none;
  border-radius: 10px;
  font-size: 14px;
  cursor: pointer;
}
.foot {
  text-align: center;
  color: #98a2b3;
  font-size: 12px;
  margin-top: 28px;
}
</style>
