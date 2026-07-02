export function analyticsSourceLabel(label?: string) {
  const value = String(label || '').trim()
  return value || '客户端落地页事件日志'
}
