import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { api, getToken } from '@/api'
import { useUserStore } from '@/stores/user'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/upload' },
  { path: '/setup', component: () => import('@/pages/Setup.vue'), meta: { title: '首次设置', public: true } },
  { path: '/login', component: () => import('@/pages/Login.vue'), meta: { title: '登录', public: true } },
  { path: '/upload', component: () => import('@/pages/Upload.vue'), meta: { title: '上传' } },
  { path: '/gallery', component: () => import('@/pages/Gallery.vue'), meta: { title: '图库' } },
  { path: '/albums', component: () => import('@/pages/Albums.vue'), meta: { title: '相册' } },
  { path: '/tokens', component: () => import('@/pages/Tokens.vue'), meta: { title: 'API 令牌' } },
  { path: '/admin', component: () => import('@/pages/Admin.vue'), meta: { title: '管理后台', admin: true } },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

let setupChecked = false
let needsSetup = false

router.beforeEach(async (to, _from, next) => {
  // 1. 首次启动检查是否需要设置向导
  if (!setupChecked) {
    try {
      const r = await api.setupStatus()
      needsSetup = r.needs_setup
    } catch {}
    setupChecked = true
  }
  if (needsSetup && to.path !== '/setup') return next('/setup')
  if (!needsSetup && to.path === '/setup') return next('/login')

  // 2. 公开路由直接放行
  if (to.meta.public) return next()

  // 3. 强制登录：无 token 或 me 接口验证不过 → 跳 /login
  const userStore = useUserStore()
  if (!getToken()) return next('/login')
  if (!userStore.user) {
    await userStore.fetchMe()
    if (!userStore.user) return next('/login')
  }

  // 4. admin 路由
  if (to.meta.admin && !userStore.isAdmin) return next('/upload')

  next()
})
