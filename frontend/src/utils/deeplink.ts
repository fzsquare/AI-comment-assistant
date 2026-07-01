// 顾客点「去发布」：直接跳转商家分享链接本身。
// 平台分享短链（如 dpurl.cn）已配 universal link / app link：在手机浏览器打开会被
// 系统直接接管、拉起对应 App 到店铺；未装 App 则回退展示网页。
// 因此不自造 scheme —— 自造 scheme 反而可能拉错 App、唤不起或引入延迟。
// 行为等同于把链接粘到浏览器地址栏打开。
export function openPlatform(_platformCode: string, webUrl: string, fallbackUrl?: string): void {
  const url = (webUrl || fallbackUrl || '').trim()
  if (url) {
    window.location.href = url
  }
}
