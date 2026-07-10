const APP_LAUNCH_TIMEOUT_MS = 1800

function isAppScheme(url: string): boolean {
  return /^[a-z][a-z\d+.-]*:\/\//i.test(url) && !/^https?:\/\//i.test(url)
}

// 已验证的 Scheme 优先唤起 App；页面未进入后台时，回退后台配置的官方 URL。
export function openPlatform(_platformCode: string, openUrl: string, fallbackUrl?: string): void {
  const primaryUrl = openUrl.trim()
  const officialUrl = (fallbackUrl || '').trim()

  if (!primaryUrl) {
    if (officialUrl) window.location.href = officialUrl
    return
  }

  if (!isAppScheme(primaryUrl) || !officialUrl || primaryUrl === officialUrl) {
    window.location.href = primaryUrl
    return
  }

  let appLaunchDetected = false
  let fallbackTimer: number | undefined

  const cleanup = () => {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    window.removeEventListener('pagehide', onPageHide)
    if (fallbackTimer !== undefined) window.clearTimeout(fallbackTimer)
  }
  const markAppLaunch = () => {
    appLaunchDetected = true
    cleanup()
  }
  const onVisibilityChange = () => {
    if (document.visibilityState === 'hidden') markAppLaunch()
  }
  const onPageHide = () => markAppLaunch()

  document.addEventListener('visibilitychange', onVisibilityChange)
  window.addEventListener('pagehide', onPageHide)
  fallbackTimer = window.setTimeout(() => {
    cleanup()
    if (!appLaunchDetected) window.location.href = officialUrl
  }, APP_LAUNCH_TIMEOUT_MS)
  window.location.href = primaryUrl
}
