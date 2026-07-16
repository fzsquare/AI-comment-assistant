import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PublishStats } from '../../api/merchant'
import MerchantEffectDashboard from './MerchantEffectDashboard.vue'

function readyStats(overrides: Partial<PublishStats> = {}): PublishStats {
  return {
    range: '7d',
    rangeStart: '2026-07-10',
    rangeEnd: '2026-07-16',
    dataState: 'ready',
    uniqueSessions: 86,
    funnel: [
      { code: 'page_view', label: '贴卡访问', count: 86, conversionRate: 0, conversionAvailable: false },
      { code: 'platform_select', label: '选择平台', count: 70, conversionRate: 81.4, conversionAvailable: true, conversionLabel: '上一步转化率' },
      { code: 'review_copy', label: '复制评价', count: 52, conversionRate: 74.3, conversionAvailable: true, conversionLabel: '上一步转化率' },
      { code: 'platform_link_click', label: '平台点击', count: 41, conversionRate: 78.8, conversionAvailable: true, conversionLabel: '上一步转化率' }
    ],
    dailySeries: [
      { date: '2026-07-16', pageViews: 14, platformSelections: 10, reviewCopies: 8, platformLinkClicks: 6 }
    ],
    recommendation: {
      code: 'funnel_drop',
      title: '优先改善“复制评价”',
      message: '选择平台后到复制评价的流失最多。',
      actionLabel: '查看转化',
      actionTarget: 'funnel'
    },
    platformCode: '',
    platformName: '全部平台',
    updatedAt: '2026-07-16T13:50:12+08:00',
    timezone: 'Asia/Shanghai',
    ...overrides
  } as PublishStats
}

describe('MerchantEffectDashboard merchant-facing copy', () => {
  it('shows business outcomes without exposing analytics implementation language', () => {
    const wrapper = mount(MerchantEffectDashboard, {
      props: {
        stats: readyStats(),
        loading: false,
        error: '',
        storeName: '巷子里的椒麻鸡',
        platformOptions: [{ platformCode: '', platformName: '全部平台' }]
      }
    })

    expect(wrapper.text()).toContain('顾客转化')
    expect(wrapper.text()).toContain('打开平台')
    expect(wrapper.text()).toContain('81.4% 选择平台')
    expect(wrapper.find('[data-testid="data-state"]').exists()).toBe(false)
    expect(wrapper.text()).not.toMatch(/统计时区|唯一会话|统一统计口径|四段漏斗|平台点击|流程起点|上一步转化率|真实使用效果/)
  })

  it('keeps only the merchant-relevant scope hint when one platform is selected', () => {
    const wrapper = mount(MerchantEffectDashboard, {
      props: {
        stats: readyStats({ platformCode: 'meituan', platformName: '美团' }),
        loading: false,
        error: '',
        storeName: '巷子里的椒麻鸡',
        platformOptions: [{ platformCode: 'meituan', platformName: '美团' }]
      }
    })

    expect(wrapper.text()).toContain('访问为全店，其他为美团')
  })
})
