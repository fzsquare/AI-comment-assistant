import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { publicApi } from '../../api/public'
import { copyToClipboard } from '../../utils/clipboard'
import { openPlatform } from '../../utils/deeplink'
import LandingReviewPage from './LandingReviewPage.vue'
import { ensureLandingSession, selectLandingPlatform } from './landingFlow'

vi.mock('../../api/public', () => ({
  publicApi: {
    initLanding: vi.fn(),
    switchReview: vi.fn(),
    createEvent: vi.fn(),
    drawLottery: vi.fn()
  }
}))
vi.mock('../../utils/clipboard', () => ({ copyToClipboard: vi.fn() }))
vi.mock('../../utils/deeplink', () => ({ openPlatform: vi.fn() }))

const platformLink = {
  id: 1,
  platformCode: 'meituan',
  platformName: '美团',
  buttonText: '打开美团',
  targetUrl: 'https://example.com/meituan',
  backupUrl: 'https://example.com/meituan',
  openUrl: 'imeituan://',
  openMode: 'app_link'
}
const landingPayload = {
  sessionId: 'ignored-new-session',
  storeName: '巷子里的椒麻鸡',
  primaryPlatformStyle: 'meituan',
  review: null,
  keywords: [{ id: 1, keyword: '服务热情' }],
  images: [],
  platformLinks: [platformLink],
  remainingDispatchableCount: 10
}

async function mountPage(withSession = true) {
  if (withSession) {
    ensureLandingSession('store-token', 'saved-session')
    selectLandingPlatform('store-token', 'meituan')
  }
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/landing/:token', component: { template: '<div>平台页</div>' } },
      { path: '/landing/:token/review/:platformCode', component: LandingReviewPage }
    ]
  })
  await router.push('/landing/store-token/review/meituan')
  await router.isReady()
  const wrapper = mount(LandingReviewPage, { global: { plugins: [router] } })
  await flushPromises()
  return { router, wrapper }
}

describe('LandingReviewPage', () => {
  beforeEach(() => {
    sessionStorage.clear()
    vi.mocked(publicApi.initLanding).mockResolvedValue({ data: { data: landingPayload } } as never)
    vi.mocked(publicApi.switchReview).mockResolvedValue({
      data: { data: { review: { id: 88, content: '服务很热情，团购核销也很顺。', platformStyle: 'meituan' }, remainingDispatchableCount: 9 } }
    } as never)
    vi.mocked(publicApi.createEvent).mockResolvedValue({ data: { data: { saved: true } } } as never)
    vi.mocked(publicApi.drawLottery).mockResolvedValue({ data: { data: { enabled: true, drawn: true, won: true, prizeName: '招牌小吃一份', prizeImageUrl: '' } } } as never)
    vi.mocked(copyToClipboard).mockResolvedValue(true)
  })

  it('redirects to platform selection without requesting a review when the session is missing', async () => {
    const { router } = await mountPage(false)

    expect(router.currentRoute.value.fullPath).toBe('/landing/store-token')
    expect(publicApi.switchReview).not.toHaveBeenCalled()
  })

  it('redirects without requesting a review when the selected platform does not match the route', async () => {
    ensureLandingSession('store-token', 'saved-session')
    selectLandingPlatform('store-token', 'dianping')

    const { router } = await mountPage(false)

    expect(router.currentRoute.value.fullPath).toBe('/landing/store-token')
    expect(publicApi.switchReview).not.toHaveBeenCalled()
  })

  it('redirects without requesting a review when the selected platform is no longer available', async () => {
    vi.mocked(publicApi.initLanding).mockResolvedValueOnce({
      data: { data: { ...landingPayload, platformLinks: [] } }
    } as never)

    const { router } = await mountPage()

    expect(router.currentRoute.value.fullPath).toBe('/landing/store-token')
    expect(publicApi.switchReview).not.toHaveBeenCalled()
  })

  it('loads a platform-specific review using the saved session', async () => {
    const { wrapper } = await mountPage()

    expect(publicApi.switchReview).toHaveBeenCalledWith('store-token', {
      platformCode: 'meituan',
      sessionId: 'saved-session'
    })
    expect(wrapper.get('textarea').element.value).toBe('服务很热情，团购核销也很顺。')
    expect(wrapper.text()).toContain('是否发布由你决定')
  })

  it('copies, records both funnel events, announces success, and opens the backend URL', async () => {
    const { wrapper } = await mountPage()

    await wrapper.get('[data-testid="primary-platform-action"]').trigger('click')
    await flushPromises()

    expect(publicApi.createEvent).toHaveBeenCalledWith('store-token', expect.objectContaining({ actionType: 'review_copy' }))
    expect(publicApi.createEvent).toHaveBeenCalledWith('store-token', expect.objectContaining({ actionType: 'platform_link_click' }))
    expect(wrapper.get('[role="status"]').text()).toContain('已复制，正在打开美团')
    expect(openPlatform).toHaveBeenCalledWith('meituan', 'imeituan://', 'https://example.com/meituan')
  })

  it('shows the immediate gift result only after the customer returns from the platform', async () => {
    const { wrapper } = await mountPage()

    await wrapper.get('[data-testid="primary-platform-action"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="lottery-result"]').exists()).toBe(false)

    window.dispatchEvent(new Event('focus'))
    await flushPromises()

    expect(publicApi.drawLottery).toHaveBeenCalledWith('store-token', { sessionId: 'saved-session' })
    expect(wrapper.get('[data-testid="lottery-result"]').text()).toContain('招牌小吃一份')
    expect(wrapper.get('[data-testid="lottery-result"]').text()).toContain('向身边店员出示本页面领取')
  })

  it('shows a recoverable alert and does not open the platform when copy fails', async () => {
    vi.mocked(copyToClipboard).mockResolvedValue(false)
    const { wrapper } = await mountPage()

    await wrapper.get('[data-testid="primary-platform-action"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('长按评价内容手动复制')
    expect(openPlatform).not.toHaveBeenCalled()
  })

  it('copies without opening the platform when the secondary copy action is used', async () => {
    const { wrapper } = await mountPage()

    await wrapper.get('.landing-secondary-actions button:last-child').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="status"]').text()).toContain('已复制，可以在美团中粘贴')
    expect(publicApi.createEvent).toHaveBeenCalledWith('store-token', expect.objectContaining({ actionType: 'review_copy' }))
    expect(openPlatform).not.toHaveBeenCalled()
  })

  it('shows a recoverable alert when the platform has no usable link', async () => {
    vi.mocked(publicApi.initLanding).mockResolvedValueOnce({
      data: {
        data: {
          ...landingPayload,
          platformLinks: [{ ...platformLink, targetUrl: '', backupUrl: '', openUrl: '' }]
        }
      }
    } as never)
    const { wrapper } = await mountPage()

    await wrapper.get('[data-testid="primary-platform-action"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('没有可用的门店入口')
    expect(openPlatform).not.toHaveBeenCalled()
  })

  it('shows a recoverable alert when the initial review request fails', async () => {
    vi.mocked(publicApi.switchReview).mockRejectedValueOnce(new Error('offline'))

    const { wrapper } = await mountPage()

    expect(wrapper.get('[role="alert"]').text()).toContain('暂时没有可用评价')
    expect(wrapper.get('.landing-back-link').text()).toContain('重新选择平台')
  })

  it('clears a server-rejected session and returns to platform selection', async () => {
    vi.mocked(publicApi.switchReview).mockRejectedValueOnce({
      response: { data: { message: '会话已失效，请刷新页面后重试' } }
    })

    const { router } = await mountPage()

    expect(router.currentRoute.value.fullPath).toBe('/landing/store-token')
    expect(sessionStorage.getItem('ppk-landing-session:store-token')).toBeNull()
  })

  it('returns to platform selection when the open review page crosses the local session limit', async () => {
    const { router, wrapper } = await mountPage()
    const stored = JSON.parse(sessionStorage.getItem('ppk-landing-session:store-token') || '{}')
    sessionStorage.setItem('ppk-landing-session:store-token', JSON.stringify({
      ...stored,
      createdAt: Date.now() - 23 * 60 * 60 * 1000
    }))

    await wrapper.get('[data-testid="primary-platform-action"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/landing/store-token')
    expect(copyToClipboard).not.toHaveBeenCalled()
  })

  it('opens the platform without waiting for analytics to finish', async () => {
    vi.mocked(publicApi.createEvent).mockImplementation(() => new Promise(() => {}) as never)
    const { wrapper } = await mountPage()

    await wrapper.get('[data-testid="primary-platform-action"]').trigger('click')
    await flushPromises()

    expect(openPlatform).toHaveBeenCalledTimes(1)
  })

  it('switches review content without waiting for rejection analytics', async () => {
    vi.mocked(publicApi.createEvent).mockImplementation(() => new Promise(() => {}) as never)
    const { wrapper } = await mountPage()

    await wrapper.get('.landing-secondary-actions button').trigger('click')
    await flushPromises()

    expect(publicApi.switchReview).toHaveBeenCalledTimes(2)
  })

  it('restores the previous tag label when a tag-specific review request fails', async () => {
    const { wrapper } = await mountPage()
    vi.mocked(publicApi.switchReview).mockRejectedValueOnce(new Error('offline'))

    await wrapper.get('.landing-chip').trigger('click')
    await flushPromises()

    expect(wrapper.get('.landing-tag-panel summary').text()).toContain('可选：换个符合体验的说法')
    expect(wrapper.get('[role="alert"]').text()).toContain('暂时没有可用评价')
  })

  it('ignores rapid repeated primary-action clicks while copying', async () => {
    let resolveCopy!: (result: boolean) => void
    vi.mocked(copyToClipboard).mockImplementation(() => new Promise<boolean>((resolve) => {
      resolveCopy = resolve
    }))
    const { wrapper } = await mountPage()
    const action = wrapper.get('[data-testid="primary-platform-action"]')

    await action.trigger('click')
    await action.trigger('click')

    expect(copyToClipboard).toHaveBeenCalledTimes(1)
    resolveCopy(true)
    await flushPromises()
    expect(openPlatform).toHaveBeenCalledTimes(1)
  })

  it('returns the sticky action area to document flow while the textarea is focused', async () => {
    const { wrapper } = await mountPage()
    const textarea = wrapper.get('textarea')

    await textarea.trigger('focus')
    expect(wrapper.get('[data-testid="review-actions"]').classes()).toContain('is-editing')

    await textarea.trigger('blur')
    expect(wrapper.get('[data-testid="review-actions"]').classes()).not.toContain('is-editing')
  })
})
