import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { publicApi } from '../../api/public'
import LandingPlatformPage from './LandingPlatformPage.vue'

vi.mock('../../api/public', () => ({
  publicApi: {
    initLanding: vi.fn(),
    switchReview: vi.fn(),
    createEvent: vi.fn()
  }
}))

const landingPayload = {
  sessionId: 'server-session-1',
  storeName: '巷子里的椒麻鸡',
  primaryPlatformStyle: 'meituan',
  review: null,
  keywords: [],
  images: [],
  platformLinks: [
    {
      id: 1,
      platformCode: 'meituan',
      platformName: '美团',
      buttonText: '打开美团',
      targetUrl: 'https://example.com/meituan',
      openUrl: 'imeituan://'
    }
  ],
  remainingDispatchableCount: 10
}

async function mountPage() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/landing/:token', component: LandingPlatformPage },
      { path: '/landing/:token/review/:platformCode', name: 'landing-review', component: { template: '<div>评论页</div>' } }
    ]
  })
  await router.push('/landing/store-token')
  await router.isReady()
  const wrapper = mount(LandingPlatformPage, { global: { plugins: [router] } })
  await flushPromises()
  return { router, wrapper }
}

describe('LandingPlatformPage', () => {
  beforeEach(() => {
    sessionStorage.clear()
    vi.mocked(publicApi.initLanding).mockResolvedValue({ data: { data: landingPayload } } as never)
    vi.mocked(publicApi.createEvent).mockResolvedValue({ data: { data: { saved: true } } } as never)
  })

  it('shows only the store and platform choices, then opens the platform review route', async () => {
    const { router, wrapper } = await mountPage()

    expect(wrapper.get('h1').text()).toBe('巷子里的椒麻鸡')
    expect(wrapper.text()).toContain('选择评价平台')
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(publicApi.createEvent).toHaveBeenCalledWith('store-token', expect.objectContaining({
      sessionId: 'server-session-1',
      actionType: 'page_view'
    }))

    await wrapper.get('[data-platform-code="meituan"]').trigger('click')
    await flushPromises()

    expect(publicApi.createEvent).toHaveBeenCalledWith('store-token', expect.objectContaining({
      sessionId: 'server-session-1',
      actionType: 'platform_select',
      platformCode: 'meituan'
    }))
    expect(router.currentRoute.value.fullPath).toBe('/landing/store-token/review/meituan')
  })

  it('does not record page_view twice when the same tab reloads the platform page', async () => {
    const first = await mountPage()
    first.wrapper.unmount()
    vi.mocked(publicApi.initLanding).mockResolvedValue({
      data: { data: { ...landingPayload, sessionId: 'server-session-2' } }
    } as never)

    await mountPage()

    const pageViews = vi.mocked(publicApi.createEvent).mock.calls.filter(([, body]) => body.actionType === 'page_view')
    expect(pageViews).toHaveLength(1)
    expect(sessionStorage.getItem('ppk-landing-session:store-token')).toContain('server-session-1')
  })

  it('does not keep platform choices behind the loading skeleton while analytics is slow', async () => {
    vi.mocked(publicApi.createEvent).mockImplementation(() => new Promise(() => {}) as never)

    const first = await mountPage()

    expect(first.wrapper.find('[data-platform-code="meituan"]').exists()).toBe(true)
    first.wrapper.unmount()
    await mountPage()

    const pageViews = vi.mocked(publicApi.createEvent).mock.calls.filter(([, body]) => body.actionType === 'page_view')
    expect(pageViews).toHaveLength(1)
  })

  it('shows a recoverable alert when landing initialization fails', async () => {
    vi.mocked(publicApi.initLanding).mockRejectedValueOnce(new Error('offline'))

    const { wrapper } = await mountPage()

    expect(wrapper.get('[role="alert"]').text()).toContain('页面加载失败')
    expect(wrapper.get('button').text()).toBe('重新加载')
  })
})
