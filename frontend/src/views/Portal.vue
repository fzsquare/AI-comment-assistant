<template>
  <main class="portal" aria-labelledby="portal-title">
    <header class="head">
      <p class="eyebrow">PPK NFC REVIEW</p>
      <h1 id="portal-title">评价助手</h1>
      <p class="sub">请使用分配给你的专属地址访问对应系统。</p>
    </header>

    <section class="entry-notice" aria-label="入口隔离提示">
      <div class="notice-icon" aria-hidden="true">
        <svg viewBox="0 0 24 24"><path d="M12 3.5 19 7v5.2c0 4.1-2.7 6.8-7 8.3-4.3-1.5-7-4.2-7-8.3V7l7-3.5Z" /><path d="M9 12.2 11.1 14l4-4.2" /></svg>
      </div>
      <div>
        <h2>入口已隔离</h2>
        <p>当前页面不提供管理员、商家和顾客落地页之间的跳转。</p>
      </div>
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
.entry-notice {
  align-items: center;
  background: #fff;
  border: 1px solid #dbe4f0;
  border-radius: 16px;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.06);
  display: grid;
  gap: 12px;
  grid-template-columns: 48px minmax(0, 1fr);
  min-height: 96px;
  padding: 16px;
}
.notice-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 14px;
  color: #1d4ed8;
  background: #dbeafe;
}
.notice-icon svg {
  width: 25px;
  height: 25px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.9;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.entry-notice h2 {
  color: #0f172a;
  font-size: 18px;
  font-weight: 800;
  margin: 0 0 4px;
}
.entry-notice p {
  color: #64748b;
  font-size: 14px;
  line-height: 1.35;
  margin: 0;
}
@media (min-width: 768px) {
  .portal {
    padding-top: 56px;
  }
}
</style>
