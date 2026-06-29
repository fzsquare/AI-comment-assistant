// 平台唤端：用各平台 App scheme 拼 deeplink 唤起 App，唤不起（未装/失败）回退商家网页链接。
// scheme 里尽量带上商家网页链接；即便 scheme 不精确，网页回退也常能被平台 universal link
// 接管、直达店铺。需要调优时改这里即可。
const SCHEME_BUILDERS: Record<string, ((webUrl: string) => string) | undefined> = {
  dianping: (u) => `dianping://web?url=${encodeURIComponent(u)}`,
  meituan: (u) => `imeituan://www.meituan.com/web?url=${encodeURIComponent(u)}`,
  xiaohongshu: (u) => `xhsdiscover://webview?url=${encodeURIComponent(u)}`,
  douyin: (u) => `snssdk1128://webview?url=${encodeURIComponent(u)}`
}

export function buildDeeplink(platformCode: string, webUrl: string): string | null {
  const builder = SCHEME_BUILDERS[platformCode]
  return builder && webUrl ? builder(webUrl) : null
}

// 先尝试 deeplink 唤端；约 1.2s 内页面仍可见（App 未接管）则跳商家网页。
export function openPlatform(platformCode: string, webUrl: string, fallbackUrl?: string): void {
  const web = webUrl || fallbackUrl || ''
  const deeplink = buildDeeplink(platformCode, web)
  if (!deeplink) {
    if (web) window.location.href = web
    return
  }

  const start = Date.now()
  const fallback = () => {
    // App 已接管 → 页面被切到后台（hidden）→ 不再跳网页，避免“双开”
    if (document.hidden || document.visibilityState === 'hidden') return
    if (Date.now() - start > 2500) return
    if (web) window.location.href = web
  }
  const timer = window.setTimeout(fallback, 1200)
  const onHide = () => {
    if (document.hidden) window.clearTimeout(timer)
  }
  document.addEventListener('visibilitychange', onHide, { once: true })

  try {
    window.location.href = deeplink
  } catch {
    window.clearTimeout(timer)
    if (web) window.location.href = web
  }
}
