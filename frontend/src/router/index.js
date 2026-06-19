import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/home' },
  { path: '/home', name: 'home', component: () => import('../views/Home.vue'), meta: { tab: 0 } },
  { path: '/category', name: 'category', component: () => import('../views/Category.vue'), meta: { tab: 1 } },
  { path: '/discover', name: 'discover', component: () => import('../views/Discover.vue'), meta: { tab: 2 } },
  { path: '/cart', name: 'cart', component: () => import('../views/Cart.vue'), meta: { tab: 3, auth: true } },
  { path: '/mine', name: 'mine', component: () => import('../views/Mine.vue'), meta: { tab: 4 } },
  { path: '/product/:id', name: 'product', component: () => import('../views/ProductDetail.vue') },
  { path: '/search', name: 'search', component: () => import('../views/Search.vue') },
  { path: '/login', name: 'login', component: () => import('../views/Login.vue') },
  { path: '/orders', name: 'orders', component: () => import('../views/Orders.vue'), meta: { auth: true } },
  { path: '/checkout', name: 'checkout', component: () => import('../views/Checkout.vue'), meta: { auth: true } },
  { path: '/admin', name: 'admin', component: () => import('../views/Admin.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Simple auth guard.
router.beforeEach((to, from, next) => {
  if (to.meta.auth && !localStorage.getItem('jd_token')) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else {
    next()
  }
})

export default router
