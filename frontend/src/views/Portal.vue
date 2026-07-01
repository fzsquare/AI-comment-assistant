<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
const go = (path: string) => router.push(path)
</script>

<template>
  <main class="portal" aria-labelledby="portal-title">
    <header class="head">
      <p class="eyebrow">PPK NFC REVIEW</p>
      <h1 id="portal-title">评价助手</h1>
      <p class="sub">选择身份后直接进入对应工作台</p>
    </header>

    <section class="entries" aria-label="入口">
      <button class="entry primary" type="button" @click="go('/merchant/login')">
        <span class="entry-icon" aria-hidden="true">
          <svg viewBox="0 0 24 24"><path d="M4 10.5 5.4 5h13.2l1.4 5.5" /><path d="M5 10.5v8h14v-8" /><path d="M9 18.5v-5h6v5" /><path d="M3.5 10.5h17" /></svg>
        </span>
        <span class="entry-copy">
          <span class="t">商家入口</span>
          <span class="d">店铺、图片、平台链接、评价池</span>
        </span>
        <span class="chev" aria-hidden="true">›</span>
      </button>

      <button class="entry" type="button" @click="go('/admin/login')">
        <span class="entry-icon" aria-hidden="true">
          <svg viewBox="0 0 24 24"><path d="M12 3.5 19 7v5.2c0 4.1-2.7 6.8-7 8.3-4.3-1.5-7-4.2-7-8.3V7l7-3.5Z" /><path d="M9 12.2 11.1 14l4-4.2" /></svg>
        </span>
        <span class="entry-copy">
          <span class="t">管理员入口</span>
          <span class="d">商家、门店、NFC 卡片、统计</span>
        </span>
        <span class="chev" aria-hidden="true">›</span>
      </button>

      <button class="entry ghost" type="button" @click="go('/landing/11111111-1111-4111-8111-111111111111')">
        <span class="entry-icon" aria-hidden="true">
          <svg viewBox="0 0 24 24"><path d="M8 2.8h8a2 2 0 0 1 2 2v14.4a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V4.8a2 2 0 0 1 2-2Z" /><path d="M10 18h4" /><path d="M8.5 6h7" /></svg>
        </span>
        <span class="entry-copy">
          <span class="t">顾客入口</span>
          <span class="d">落地页演示</span>
        </span>
        <span class="chev" aria-hidden="true">›</span>
      </button>
    </section>
  </main>
</template>

<style scoped>
.portal {
  min-height: 100dvh;
  max-width: 520px;
  margin: 0 auto;
  padding: calc(28px + env(safe-area-inset-top)) 16px calc(24px + env(safe-area-inset-bottom));
}
.head {
  margin-bottom: 22px;
}
.eyebrow {
  margin: 0 0 8px;
  color: #2563eb;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0;
}
.head h1 {
  margin: 0;
  color: #0f172a;
  font-size: 30px;
  line-height: 1.15;
}
.sub {
  margin: 8px 0 0;
  color: #475569;
  font-size: 16px;
}
.entries {
  display: grid;
  gap: 12px;
}
.entry {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr) 24px;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 86px;
  padding: 16px;
  color: #0f172a;
  text-align: left;
  background: #fff;
  border: 1px solid #dbe4f0;
  border-radius: 16px;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.06);
  touch-action: manipulation;
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;
}
.entry:active {
  transform: scale(0.99);
}
.entry.primary {
  border-color: #bfdbfe;
  background: #eff6ff;
}
.entry.ghost {
  box-shadow: none;
}
.entry-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 14px;
  color: #1d4ed8;
  background: #dbeafe;
}
.entry-icon svg {
  width: 25px;
  height: 25px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.9;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.entry-copy {
  display: grid;
  gap: 3px;
  min-width: 0;
}
.t {
  font-size: 17px;
  font-weight: 800;
}
.d {
  color: #64748b;
  font-size: 14px;
  line-height: 1.35;
}
.chev {
  color: #64748b;
  font-size: 28px;
  line-height: 1;
}
@media (min-width: 768px) {
  .portal {
    padding-top: 56px;
  }
}
</style>
