import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MerchantConsole from './MerchantConsole.vue'

const merchantApi = vi.hoisted(() => ({
  getStoreDetail: vi.fn(),
  listKeywords: vi.fn(),
  getKeywordSuggestions: vi.fn(),
  listImages: vi.fn(),
  listPlatformLinks: vi.fn(),
  listReviews: vi.fn(),
  getGenerationPreferences: vi.fn(),
  getPublishStats: vi.fn(),
  getLotteryConfig: vi.fn(),
  saveLotteryConfig: vi.fn()
}))

vi.mock('../../api/merchant', () => ({ merchantApi }))

function response(data: unknown) {
  return Promise.resolve({ data: { data } })
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function dashboard(range: '7d' | '30d', platformCode: string) {
  const visits = range === '30d' ? 130 : 35
  const selected = platformCode ? Math.round(visits * 0.7) : Math.round(visits * 0.8)
  const copies = Math.round(selected * 0.75)
  const clicks = Math.round(copies * 0.8)
  return {
    range,
    rangeStart: range === '30d' ? '2026-06-16' : '2026-07-09',
    rangeEnd: '2026-07-15',
    dataState: 'ready',
    uniqueSessions: visits,
    funnel: [
      { code: 'page_view', label: '贴卡访问', count: visits, conversionRate: 0, conversionAvailable: false },
      { code: 'platform_select', label: '选择平台', count: selected, conversionRate: 80, conversionAvailable: true },
      { code: 'review_copy', label: '复制评价', count: copies, conversionRate: 75, conversionAvailable: true },
      { code: 'platform_link_click', label: '平台点击', count: clicks, conversionRate: 80, conversionAvailable: true }
    ],
    dailySeries: [{ date: '2026-07-15', pageViews: visits, platformSelections: selected, reviewCopies: copies, platformLinkClicks: clicks }],
    recommendation: { code: 'healthy', title: '评价流程运行稳定', message: '继续观察。', actionLabel: '查看趋势', actionTarget: 'trend' },
    platformCode,
    platformName: platformCode ? '美团' : '全部平台',
    timezone: 'Asia/Shanghai',
    updatedAt: '2026-07-15T20:00:00+08:00'
  }
}

describe('MerchantConsole effect filters', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    merchantApi.getStoreDetail.mockReturnValue(response({ storeName: '巷子里的椒麻鸡', industryType: '餐饮' }))
    merchantApi.listKeywords.mockReturnValue(response([]))
    merchantApi.getKeywordSuggestions.mockReturnValue(response({ tags: [] }))
    merchantApi.listImages.mockReturnValue(response([]))
    merchantApi.listPlatformLinks.mockReturnValue(response([
      { id: 1, platformCode: 'meituan', platformName: '美团', status: 1, sortNo: 1 }
    ]))
    merchantApi.listReviews.mockReturnValue(response([]))
    merchantApi.getGenerationPreferences.mockReturnValue(response({
      configured: false,
      focusKeywords: [],
      styleCodes: ['natural'],
      diversityDimensions: ['customer_identity'],
      referenceReviews: [],
      lengthVariance: 'wide'
    }))
    merchantApi.getPublishStats.mockImplementation((platformCode: string, range: '7d' | '30d') => response(dashboard(range, platformCode)))
    merchantApi.getLotteryConfig.mockReturnValue(response({ enabled: false, prizes: [] }))
    merchantApi.saveLotteryConfig.mockImplementation((payload: unknown) => response(payload))
  })

  it('keeps numbers, funnel and trend on the same range and platform request', async () => {
    const wrapper = mount(MerchantConsole, { global: { plugins: [createPinia()] } })
    await flushPromises()

    expect(merchantApi.getPublishStats).toHaveBeenLastCalledWith('', '7d')
    expect(wrapper.get('[data-metric="page_view"]').text()).toContain('35')

    await wrapper.get('[data-range="30d"]').trigger('click')
    await flushPromises()
    expect(merchantApi.getPublishStats).toHaveBeenLastCalledWith('', '30d')
    expect(wrapper.get('[data-metric="page_view"]').text()).toContain('130')

    await wrapper.get('#effect-platform').setValue('meituan')
    await flushPromises()
    expect(merchantApi.getPublishStats).toHaveBeenLastCalledWith('meituan', '30d')
    expect(wrapper.get('[data-metric="platform_select"]').text()).toContain('91')
    expect(wrapper.findAll('[data-funnel-stage]')).toHaveLength(3)
    expect(wrapper.get('[data-testid="daily-trend"]').attributes('aria-label')).toContain('贴卡访问 130 次')
  })

  it('ignores an older dashboard response that arrives after the latest filter response', async () => {
    const wrapper = mount(MerchantConsole, { global: { plugins: [createPinia()] } })
    await flushPromises()

    const thirtyDay = deferred<Awaited<ReturnType<typeof response>>>()
    const sevenDay = deferred<Awaited<ReturnType<typeof response>>>()
    merchantApi.getPublishStats
      .mockImplementationOnce(() => thirtyDay.promise)
      .mockImplementationOnce(() => sevenDay.promise)

    await wrapper.get('[data-range="30d"]').trigger('click')
    await wrapper.get('[data-range="7d"]').trigger('click')
    sevenDay.resolve(await response(dashboard('7d', '')))
    await flushPromises()
    thirtyDay.resolve(await response(dashboard('30d', '')))
    await flushPromises()

    expect(wrapper.get('[data-range="7d"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('[data-metric="page_view"]').text()).toContain('35')
  })

  it('shows a recoverable inline error when a filter refresh fails with stale data present', async () => {
    const wrapper = mount(MerchantConsole, { global: { plugins: [createPinia()] } })
    await flushPromises()

    merchantApi.getPublishStats.mockRejectedValueOnce(new Error('筛选请求失败'))
    await wrapper.get('[data-range="30d"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('筛选请求失败')
    expect(wrapper.get('[data-testid="retry-dashboard"]').exists()).toBe(true)
    expect(wrapper.get('[data-metric="page_view"]').text()).toContain('35')
  })
})
