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
  { path: '/pay', name: 'pay', component: () => import('../views/Pay.vue'), meta: { auth: true } },
  { path: '/logistics', name: 'logistics', component: () => import('../views/Logistics.vue'), meta: { auth: true } },
  { path: '/refunds', name: 'refunds', component: () => import('../views/Refunds.vue'), meta: { auth: true } },
  { path: '/coupons', name: 'coupons', component: () => import('../views/Coupons.vue') },
  { path: '/favorites', name: 'favorites', component: () => import('../views/Favorites.vue'), meta: { auth: true } },
  { path: '/addresses', name: 'addresses', component: () => import('../views/Addresses.vue'), meta: { auth: true } },
  { path: '/profile', name: 'profile', component: () => import('../views/EditProfile.vue'), meta: { auth: true } },
  { path: '/history', name: 'history', component: () => import('../views/History.vue'), meta: { auth: true } },
  { path: '/checkin', name: 'checkin', component: () => import('../views/CheckIn.vue'), meta: { auth: true } },
  { path: '/points-shop', name: 'pointShop', component: () => import('../views/PointShop.vue'), meta: { auth: true } },
  { path: '/lottery', name: 'lottery', component: () => import('../views/Lottery.vue'), meta: { auth: true } },
  { path: '/seckill', name: 'seckill', component: () => import('../views/Seckill.vue') },
  { path: '/group-buy', name: 'groupBuy', component: () => import('../views/GroupBuy.vue') },
  { path: '/presale', name: 'presale', component: () => import('../views/Presale.vue') },
  { path: '/shop/:name', name: 'shopRatings', component: () => import('../views/ShopRatings.vue') },
  { path: '/bundles', name: 'bundles', component: () => import('../views/Bundles.vue') },
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
