import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminLogin from './AdminLogin.vue'

const login = vi.hoisted(() => vi.fn())
vi.mock('../../api/admin', () => ({ adminApi: { login } }))

async function mountLogin(url = '/admin/login') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/admin/login', component: AdminLogin },
      { path: '/admin/console/:section', component: { template: '<p>Admin</p>' } }
    ]
  })
  await router.push(url)
  await router.isReady()
  const wrapper = mount(AdminLogin, { global: { plugins: [createPinia(), router] } })
  return { wrapper, router }
}

describe('AdminLogin', () => {
  beforeEach(() => {
    localStorage.clear()
    login.mockReset()
  })

  it('shows inline validation instead of opening a browser alert', async () => {
    const { wrapper } = await mountLogin()
    expect((wrapper.get('input[autocomplete="username"]').element as HTMLInputElement).value).toBe('')
    expect((wrapper.get('input[autocomplete="current-password"]').element as HTMLInputElement).value).toBe('')
    await wrapper.get('input[autocomplete="username"]').setValue('')
    await wrapper.get('input[autocomplete="current-password"]').setValue('')
    await wrapper.get('form').trigger('submit')

    expect(wrapper.get('[role="alert"]').text()).toContain('请输入管理员账号和密码')
    expect(login).not.toHaveBeenCalled()
  })

  it('restores the requested admin page after successful login', async () => {
    login.mockResolvedValue({ data: { data: { token: 'admin-token' } } })
    const { wrapper, router } = await mountLogin('/admin/login?redirect=/admin/console/stores')
    await wrapper.get('input[autocomplete="username"]').setValue('admin')
    await wrapper.get('input[autocomplete="current-password"]').setValue('secret')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(login).toHaveBeenCalledWith({ account: 'admin', password: 'secret' })
    expect(router.currentRoute.value.fullPath).toBe('/admin/console/stores')
    expect(localStorage.getItem('ppk-role')).toBe('admin')
  })

  it('explains why the user returned to login after session expiry', async () => {
    const { wrapper } = await mountLogin('/admin/login?reason=session_expired')
    expect(wrapper.get('[role="status"]').text()).toContain('登录状态已过期')
  })
})
