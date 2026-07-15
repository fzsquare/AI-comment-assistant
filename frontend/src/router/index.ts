import { createRouter, createWebHistory } from 'vue-router'
// 消费者碰卡的两页入口保持同步导入，平台选择和评价页都进轻量首屏主包。
import LandingPlatformPage from '../views/landing/LandingPlatformPage.vue'
import LandingReviewPage from '../views/landing/LandingReviewPage.vue'
import Portal from '../views/Portal.vue'
import SchemeTestPage from '../views/SchemeTestPage.vue'
import type { Role } from '../stores/auth'

// 商家/管理后台按需加载，消费者落地页不下载这些重代码（CRUD 表格/表单）
const MerchantLogin = () => import('../views/merchant/MerchantLogin.vue')
const MerchantConsole = () => import('../views/merchant/MerchantConsole.vue')
const AdminLogin = () => import('../views/admin/AdminLogin.vue')
const AdminConsole = () => import('../views/admin/AdminConsole.vue')

const TOKEN_KEY = 'ppk-token'
const ROLE_KEY = 'ppk-role'

function loginPath(role?: Role) {
  if (role === 'admin') return '/admin/login'
  if (role === 'merchant') return '/merchant/login'
  return '/'
}

function homePath(role: Role) {
  if (role === 'admin') {
    return '/admin/console/overview'
  }
  if (role === 'merchant') {
    return '/merchant/console'
  }
  return '/merchant/login'
}

function clearAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(ROLE_KEY)
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: Portal },
    { path: '/landing/:token', name: 'landing-platforms', component: LandingPlatformPage },
    { path: '/landing/:token/review/:platformCode', name: 'landing-review', component: LandingReviewPage },
    { path: '/scheme-test', component: SchemeTestPage },
    { path: '/merchant/login', component: MerchantLogin },
    { path: '/merchant/console', component: MerchantConsole, meta: { requiresAuth: true, role: 'merchant' } },
    { path: '/admin/login', component: AdminLogin },
    { path: '/admin/console', redirect: '/admin/console/overview' },
    { path: '/admin/console/:section', name: 'admin-console-section', component: AdminConsole, meta: { requiresAuth: true, role: 'admin' } }
  ]
})

router.beforeEach((to) => {
  const requiredRole = to.meta.role as Role | undefined
  const token = localStorage.getItem(TOKEN_KEY)
  const role = (localStorage.getItem(ROLE_KEY) as Role | null) || ''

  if (to.path === '/merchant/login' || to.path === '/admin/login') {
    const loginRole: Role = to.path === '/admin/login' ? 'admin' : 'merchant'
    if (token && role === loginRole) {
      return homePath(loginRole)
    }
    if (token) {
      clearAuth()
    }
    return true
  }

  if (!to.meta.requiresAuth) {
    return true
  }

  if (!token) {
    clearAuth()
    return { path: loginPath(requiredRole), query: { redirect: to.fullPath } }
  }

  if (requiredRole && role !== requiredRole) {
    clearAuth()
    return { path: loginPath(requiredRole), query: { redirect: to.fullPath } }
  }

  return true
})

export default router
