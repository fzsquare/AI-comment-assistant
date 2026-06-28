import { createRouter, createWebHistory } from 'vue-router'
// 消费者碰卡的入口，保持同步导入 → 进首屏主包，开得最快
import LandingPage from '../views/landing/LandingPage.vue'
import Portal from '../views/Portal.vue'
import type { Role } from '../stores/auth'

// 商家/管理后台按需加载，消费者落地页不下载这些重代码（CRUD 表格/表单）
const MerchantLogin = () => import('../views/merchant/MerchantLogin.vue')
const MerchantConsole = () => import('../views/merchant/MerchantConsole.vue')
const AdminLogin = () => import('../views/admin/AdminLogin.vue')
const AdminConsole = () => import('../views/admin/AdminConsole.vue')

const TOKEN_KEY = 'ppk-token'
const ROLE_KEY = 'ppk-role'

function loginPath(role?: Role) {
  return role === 'admin' ? '/admin/login' : '/merchant/login'
}

function homePath(role: Role) {
  if (role === 'admin') {
    return '/admin/console'
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
  history: createWebHistory(),
  routes: [
    { path: '/', component: Portal },
    { path: '/landing/:token', component: LandingPage },
    { path: '/merchant/login', component: MerchantLogin },
    { path: '/merchant/console', component: MerchantConsole, meta: { requiresAuth: true, role: 'merchant' } },
    { path: '/admin/login', component: AdminLogin },
    { path: '/admin/console', component: AdminConsole, meta: { requiresAuth: true, role: 'admin' } }
  ]
})

router.beforeEach((to) => {
  const requiredRole = to.meta.role as Role | undefined
  const token = localStorage.getItem(TOKEN_KEY)
  const role = (localStorage.getItem(ROLE_KEY) as Role | null) || ''

  if (to.path === '/merchant/login' || to.path === '/admin/login') {
    if (token && (role === 'merchant' || role === 'admin')) {
      return homePath(role)
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
    if (role === 'merchant' || role === 'admin') {
      return homePath(role)
    }
    clearAuth()
    return { path: loginPath(requiredRole), query: { redirect: to.fullPath } }
  }

  return true
})

export default router
