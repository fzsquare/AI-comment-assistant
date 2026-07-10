// 顾客点「去发布」：打开后台配置并由后端原样下发的平台 URL。
export function openPlatform(_platformCode: string, openUrl: string, fallbackUrl?: string): void {
  const url = (openUrl || fallbackUrl || '').trim()
  if (url) {
    window.location.href = url
  }
}
