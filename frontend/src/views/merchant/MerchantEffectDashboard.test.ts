import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { PublishStats } from '../../api/merchant'
import MerchantEffectDashboard from './MerchantEffectDashboard.vue'

function stats(overrides: Partial<PublishStats> = {}): PublishStats {
  return {
    range: '7d',
    rangeStart: '2026-07-09',
    rangeEnd: '2026-07-15',
    dataState: 'ready',
    uniqueSessions: 32,
    funnel: [
      { code: 'page_view', label: '贴卡访问', count: 32, conversionRate: 0, conversionAvailable: false },
      { code: 'platform_select', label: '选择平台', count: 26, conversionRate: 81.3, conversionAvailable: true },
      { code: 'review_copy', label: '复制评价', count: 19, conversionRate: 73.1, conversionAvailable: true },
      { code: 'platform_link_click', label: '平台点击', count: 15, conversionRate: 78.9, conversionAvailable: true }
    ],
    dailySeries: [
      { date: '2026-07-14', pageViews: 12, platformSelections: 10, reviewCopies: 8, platformLinkClicks: 6 },
      { date: '2026-07-15', pageViews: 20, platformSelections: 16, reviewCopies: 11, platformLinkClicks: 9 }
    ],
    recommendation: {
      code: 'funnel_drop',
      title: '优先改善“打开平台”',
      message: '这一环节是当前范围内最大的顾客流失点。',
      actionLabel: '查看漏斗',
      actionTarget: 'funnel'
    },
    platformCode: '',
    platformName: '全部平台',
    totalPublishClicks: 15,
    currentWeekPublishClicks: 15,
    currentMonthPublishClicks: 15,
    previousWeekPublishClicks: 0,
    previousMonthPublishClicks: 0,
    publishWeekGrowthPercent: 0,
    publishMonthGrowthPercent: 0,
    totalCustomerVisits: 32,
    currentWeekCustomerVisits: 32,
    currentMonthCustomerVisits: 32,
    previousWeekCustomerVisits: 0,
    previousMonthCustomerVisits: 0,
    visitWeekGrowthPercent: 0,
    visitMonthGrowthPercent: 0,
    updatedAt: '2026-07-15T15:00:00+08:00',
    dataSource: 'review_display_logs',
    dataSourceLabel: '客户端落地页事件日志',
    timezone: 'Asia/Shanghai',
    currentWeekStart: '2026-07-13',
    currentWeekEnd: '2026-07-19',
    currentMonthStart: '2026-07-01',
    currentMonthEnd: '2026-07-31',
    platformLinksConfigured: true,
    activePlatformLinkCount: 2,
    crawlDataReady: false,
    crawlDataMessage: '',
    weeklyGuidedShareReady: false,
    monthlyGuidedShareReady: false,
    weeklyGuidedSharePercent: 0,
    monthlyGuidedSharePercent: 0,
    deviceStats: { totalCount: 0, items: [] },
    weeklySeries: [],
    monthlySeries: [],
    partialErrors: [],
    ...overrides
  }
}

const platformOptions = [
  { platformCode: '', platformName: '全部平台' },
  { platformCode: 'meituan', platformName: '美团' }
]

describe('MerchantEffectDashboard', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })
  it('keeps copy analytics out of the three-stage merchant funnel and trend', () => {
    const wrapper = mount(MerchantEffectDashboard, {
      props: { stats: stats(), loading: false, error: '', storeName: '巷子里的椒麻鸡', platformOptions }
    })

    expect(wrapper.get('[data-metric="page_view"]').text()).toContain('32')
    expect(wrapper.get('[data-metric="platform_select"]').text()).toContain('26')
    expect(wrapper.get('[data-metric="platform_link_click"]').text()).toContain('15')
    expect(wrapper.find('[data-metric="review_copy"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-funnel-stage]')).toHaveLength(3)
    expect(wrapper.get('[data-testid="daily-trend"]').attributes('aria-label')).toContain('选择平台 26 次')
    expect(wrapper.get('[data-testid="daily-trend"]').attributes('aria-label')).not.toContain('复制评价')
    expect(wrapper.get('[data-testid="recommendation"]').text()).toContain('优先改善“打开平台”')
  })

  it('keeps merchant-facing review verification visible when platform crawl data is ready', () => {
    const wrapper = mount(MerchantEffectDashboard, {
      props: {
        stats: stats({
          crawlDataReady: true,
          weeklyGuidedShareReady: true,
          monthlyGuidedShareReady: true,
          weeklyGuidedSharePercent: 35.8,
          monthlyGuidedSharePercent: 29.7
        }),
        loading: false,
        error: '',
        storeName: '测试门店',
        platformOptions
      }
    })

    const verification = wrapper.get('[data-testid="review-verification"]')
    expect(verification.text()).toContain('评论结果验证')
    expect(verification.text()).toContain('本周引导评论占比')
    expect(verification.text()).toContain('35.8%')
    expect(verification.text()).toContain('本月引导评论占比')
    expect(verification.text()).toContain('29.7%')
    expect(verification.text()).toContain('不代表逐条确认发布')
  })

  it('keeps the review verification section visible while platform data accumulates', () => {
    const wrapper = mount(MerchantEffectDashboard, {
      props: {
        stats: stats({ crawlDataReady: false, crawlDataMessage: '数据积累中' }),
        loading: false,
        error: '',
        storeName: '测试门店',
        platformOptions
      }
    })

    const verification = wrapper.get('[data-testid="review-verification"]')
    expect(verification.text()).toContain('评论结果验证')
    expect(verification.text()).toContain('数据积累中')
    expect(verification.text()).not.toContain('0%')
  })

  it('trusts backend accumulating state and hides trend, percentages and drop-off conclusion', () => {
    const accumulating = stats({
      dataState: 'accumulating',
      uniqueSessions: 19,
      recommendation: {
        code: 'accumulating',
        title: '数据积累中',
        message: '先继续积累真实使用数据。',
        actionLabel: '查看原始数量',
        actionTarget: 'funnel'
      }
    })
    const wrapper = mount(MerchantEffectDashboard, {
      props: { stats: accumulating, loading: false, error: '', storeName: '测试门店', platformOptions }
    })

    expect(wrapper.text()).toContain('数据积累中')
    expect(wrapper.get('[data-metric="page_view"]').text()).toContain('32')
    expect(wrapper.find('[data-testid="daily-trend"]').exists()).toBe(false)
    expect(wrapper.find('[data-conversion-rate]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('最大的顾客流失点')
  })

  it('emits range and platform filters without locally rewriting the displayed payload', async () => {
    const payload = stats()
    const wrapper = mount(MerchantEffectDashboard, {
      props: { stats: payload, loading: false, error: '', storeName: '测试门店', platformOptions }
    })

    await wrapper.get('[data-range="30d"]').trigger('click')
    await wrapper.get('#effect-platform').setValue('meituan')

    expect(wrapper.emitted('range-change')).toEqual([['30d']])
    expect(wrapper.emitted('platform-change')).toEqual([['meituan']])
    expect(wrapper.get('[data-metric="page_view"]').text()).toContain('32')
  })

  it('shows loading, recoverable error and backend empty states distinctly', async () => {
    const loading = mount(MerchantEffectDashboard, {
      props: { stats: null, loading: true, error: '', storeName: '测试门店', platformOptions }
    })
    expect(loading.get('[aria-busy="true"]').text()).toContain('正在加载')

    const failed = mount(MerchantEffectDashboard, {
      props: { stats: null, loading: false, error: '网络暂时不可用', storeName: '测试门店', platformOptions }
    })
    expect(failed.get('[role="alert"]').text()).toContain('网络暂时不可用')
    await failed.get('[data-testid="retry-dashboard"]').trigger('click')
    expect(failed.emitted('retry')).toHaveLength(1)

    const empty = mount(MerchantEffectDashboard, {
      props: {
        stats: stats({ dataState: 'empty', uniqueSessions: 0, funnel: stats().funnel.map((stage) => ({ ...stage, count: 0 })) }),
        loading: false,
        error: '',
        storeName: '测试门店',
        platformOptions
      }
    })
    expect(empty.get('[data-testid="data-state"]').text()).toContain('还没有顾客贴卡访问')

    const platformEmpty = mount(MerchantEffectDashboard, {
      props: {
        stats: stats({ dataState: 'empty', uniqueSessions: 0, platformCode: 'meituan', platformName: '美团' }),
        loading: false,
        error: '',
        storeName: '测试门店',
        platformOptions
      }
    })
    expect(platformEmpty.get('[data-testid="data-state"]').text()).toContain('还没有顾客选择美团')
    expect(platformEmpty.get('[data-testid="data-state"]').text()).not.toContain('没有顾客贴卡')
  })

  it('disables smooth scrolling when the user prefers reduced motion', async () => {
    const scrollIntoView = vi.fn()
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: true }))
    Element.prototype.scrollIntoView = scrollIntoView
    const wrapper = mount(MerchantEffectDashboard, {
      attachTo: document.body,
      props: { stats: stats(), loading: false, error: '', storeName: '测试门店', platformOptions }
    })

    await wrapper.get('[data-testid="recommendation"] button').trigger('click')

    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'auto', block: 'center' })
    wrapper.unmount()
  })

  it('opens and focuses a collapsed recommendation target', async () => {
    const target = document.createElement('details')
    target.dataset.effectTarget = 'reviews'
    const summary = document.createElement('summary')
    summary.textContent = '评价管理'
    target.append(summary)
    document.body.append(target)
    const focus = vi.spyOn(summary, 'focus')
    Element.prototype.scrollIntoView = vi.fn()
    const wrapper = mount(MerchantEffectDashboard, {
      attachTo: document.body,
      props: {
        stats: stats({ recommendation: { code: 'content_blocker', title: '补充评价', message: '库存不足', actionLabel: '检查评价管理', actionTarget: 'reviews' } }),
        loading: false,
        error: '',
        storeName: '测试门店',
        platformOptions
      }
    })

    await wrapper.get('[data-testid="recommendation"] button').trigger('click')

    expect(target.open).toBe(true)
    expect(focus).toHaveBeenCalled()
    wrapper.unmount()
    target.remove()
  })
})
