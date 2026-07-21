import { createRouter, createWebHistory } from 'vue-router'
// 消费者碰卡的两页入口保持同步导入，平台选择和评价页都进轻量首屏主包。
import LandingPlatformPage from '../views/landing/LandingPlatformPage.vue'
import LandingReviewPage from '../views/landing/LandingReviewPage.vue'
import Portal from '../views/Portal.vue'
import SchemeTestPage from '../views/SchemeTestPage.vue'
import type { Role } from '../stores/auth'
import { isTokenExpired, safeAdminRedirect } from '../utils/authSession'

// 商家/管理后台按需加载，消费者落地页不下载这些重代码（CRUD 表格/表单）
const MerchantLogin = () => import('../views/merchant/MerchantLogin.vue')
const MerchantConsole = () => import('../views/merchant/MerchantConsole.vue')
const AdminLogin = () => import('../views/admin/AdminLogin.vue')
const AdminConsole = () => import('../views/admin/AdminConsole.vue')

const TOKEN_KEY = 'ppk-token'
const ROLE_KEY = 'ppk-role'
const DEFAULT_TITLE = '评价助手'

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

function routeTitle(title: unknown) {
  return typeof title === 'string' && title.trim() ? title : DEFAULT_TITLE
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: Portal, meta: { title: '评价助手 - 入口' } },
    { path: '/landing/:token', name: 'landing-platforms', component: LandingPlatformPage, meta: { title: '选择评价平台 - 评价助手' } },
    { path: '/landing/:token/review/:platformCode', name: 'landing-review', component: LandingReviewPage, meta: { title: '消费者评价页 - 评价助手' } },
    { path: '/scheme-test', component: SchemeTestPage, meta: { title: '平台唤起测试 - 评价助手' } },
    { path: '/merchant/login', component: MerchantLogin, meta: { title: '商家登录 - 评价助手' } },
    { path: '/merchant/console', component: MerchantConsole, meta: { requiresAuth: true, role: 'merchant', title: '商家后台 - 评价助手' } },
    { path: '/admin/login', component: AdminLogin, meta: { title: '管理员登录 - 评价助手' } },
    { path: '/admin/console', redirect: '/admin/console/overview' },
    { path: '/admin/console/:section', name: 'admin-console-section', component: AdminConsole, meta: { requiresAuth: true, role: 'admin', title: '管理员后台 - 评价助手' } }
  ]
})

router.beforeEach((to) => {
  const requiredRole = to.meta.role as Role | undefined
  const token = localStorage.getItem(TOKEN_KEY)
  const role = (localStorage.getItem(ROLE_KEY) as Role | null) || ''

  if (to.path === '/merchant/login' || to.path === '/admin/login') {
    const loginRole: Role = to.path === '/admin/login' ? 'admin' : 'merchant'
    if (token && role === loginRole && !isTokenExpired(token)) {
      if (loginRole === 'admin') return safeAdminRedirect(to.query.redirect)
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

  const expired = !!token && isTokenExpired(token)
  if (!token || expired) {
    clearAuth()
    return {
      path: loginPath(requiredRole),
      query: { redirect: to.fullPath, ...(expired ? { reason: 'session_expired' } : {}) }
    }
  }

  if (requiredRole && role !== requiredRole) {
    clearAuth()
    return { path: loginPath(requiredRole), query: { redirect: to.fullPath } }
  }

  return true
})

router.afterEach((to) => {
  document.title = routeTitle(to.meta.title)
})

export default router
