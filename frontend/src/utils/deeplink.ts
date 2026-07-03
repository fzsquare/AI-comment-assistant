// 顾客点「去发布」：跳转到后端已判定好的最终链接。
// 后端会按短链 302 结果决定继续用官方短链，还是改走直达唤醒链接。
// 前端不自造 scheme，避免拉错 App、唤不起或引入额外延迟。
export function openPlatform(_platformCode: string, webUrl: string, fallbackUrl?: string): void {
  const url = (webUrl || fallbackUrl || '').trim()
  if (url) {
    window.location.href = url
  }
}
