import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminConsole from './AdminConsole.vue'

const adminApi = vi.hoisted(() => ({
  listMerchants: vi.fn(),
  listStoreTypes: vi.fn(),
  listStores: vi.fn(),
  listTasks: vi.fn(),
  getStats: vi.fn(),
  listPlatformReviews: vi.fn()
}))
vi.mock('../../api/admin', () => ({ adminApi }))

function response(data: unknown) {
  return Promise.resolve({ data: { data } })
}

async function mountConsole() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/admin/console/:section', name: 'admin-console-section', component: AdminConsole }]
  })
  await router.push('/admin/console/overview')
  await router.isReady()
  const wrapper = mount(AdminConsole, { global: { plugins: [createPinia(), router] } })
  await flushPromises()
  return { wrapper, router }
}

describe('AdminConsole information architecture', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminApi.listMerchants.mockReturnValue(response([{ id: 1 }, { id: 2 }]))
    adminApi.listStoreTypes.mockReturnValue(response([{ id: 1, name: '餐饮', code: 'food', isPreset: true }]))
    adminApi.listStores.mockReturnValue(response([]))
    adminApi.listTasks.mockReturnValue(response([]))
    adminApi.listPlatformReviews.mockReturnValue(response({ items: [], total: 0, selectedCount: 0 }))
    adminApi.getStats.mockReturnValue(response({
      merchantCount: 2,
      storeCount: 3,
      enabledMerchantCount: 2,
      disabledMerchantCount: 0,
      currentWeekNewMerchants: 1,
      currentMonthNewMerchants: 2,
      enabledStoreCount: 2,
      disabledStoreCount: 1,
      tagCount: 0,
      taskCount: 0,
      crawlEnabledStoreCount: 0,
      crawlFailedStoreCount: 0,
      crawlDataAccumulatingCount: 0,
      totalCustomerVisits: 20,
      currentWeekCustomerVisits: 8,
      currentMonthCustomerVisits: 20,
      totalPublishClicks: 10,
      currentWeekPublishClicks: 4,
      currentMonthPublishClicks: 10,
      deviceStats: { totalCount: 0, items: [] },
      dataSource: 'real',
      dataSourceLabel: '真实数据',
      updatedAt: '2026-07-16T10:00:00+08:00'
    }))
  })

  it('uses task-oriented navigation without duplicating merchant detail as a primary section', async () => {
    const { wrapper } = await mountConsole()
    const nav = wrapper.get('.side-nav')
    expect(nav.text()).toContain('运营总览')
    expect(nav.text()).toContain('商家与门店')
    expect(nav.text()).toContain('平台评论库')
    expect(nav.text()).toContain('系统配置')
    expect(nav.text()).not.toContain('商家详情')
  })

  it('keeps merchant accounts and stores as separate overview metrics', async () => {
    const { wrapper } = await mountConsole()
    const metrics = wrapper.findAll('.kpi').map((item) => item.text())
    expect(metrics[0]).toContain('商家账号2')
    expect(metrics[1]).toContain('门店3')
  })
})
