import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockAdapter } from './mock'

describe('merchant publish-stats mock', () => {
  afterEach(() => vi.useRealTimers())

  it('uses the production 20-session threshold for a low-volume platform', async () => {
    vi.useFakeTimers()
    const pending = mockAdapter({
      method: 'get',
      url: '/merchant/dashboard/publish-stats',
      params: { range: '7d', platformCode: 'douyin' },
      headers: {}
    } as never)
    await vi.runAllTimersAsync()
    const response = await pending
    const stats = response.data.data

    expect(stats.uniqueSessions).toBeLessThan(20)
    expect(stats.dataState).toBe('accumulating')
    expect(stats.recommendation.code).toBe('accumulating')
  })
})
