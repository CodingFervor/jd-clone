<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProfile, getCheckInStatus, getOrders, listFavorites, getMyCoupons } from '../api'

// ---- Notification dots (个人中心消息红点) ----
// A red dot is shown on 售后服务 / 优惠券 / 收货地址 when that section has an
// "update", defined as: the user last visited it more than 1 hour ago (or has
// never visited it). Visiting a section records the timestamp and clears the
// dot. Timestamps live in localStorage under one key per section.
const NOTIF_SECTIONS = ['refunds', 'coupons', 'addresses']
const ONE_HOUR_MS = 60 * 60 * 1000
const notif = ref({ refunds: false, coupons: false, addresses: false })

function refreshNotif() {
  const now = Date.now()
  const out = {}
  for (const key of NOTIF_SECTIONS) {
    const last = Number(localStorage.getItem('jd_notif_' + key)) || 0
    out[key] = now - last > ONE_HOUR_MS
  }
  notif.value = out
}

// Navigate to a section and mark it visited (clearing its dot).
function goNotif(path, key) {
  try {
    localStorage.setItem('jd_notif_' + key, String(Date.now()))
  } catch (_) {}
  notif.value[key] = false
  router.push(path)
}

const router = useRouter()
const user = ref(null)
const cartCount = ref(0)
const loggedIn = ref(false)
const growthPoints = ref(0) // growth value derived from check-in points

// ---- Feature: 个人碳足迹 (Mine Carbon Footprint) ----
// Drives a fun green-themed card. Inputs are the check-in streak (days) and the
// count of completed orders, fetched alongside the quick stats.
const checkInStreak = ref(0)
const completedOrders = ref(0)
// CO2 saving from online shopping: avg 2.3 kg CO2 saved per completed order.
const CO2_PER_ORDER = 2.3
// Each tree absorbs ~20 kg CO2, used to convert savings into "equivalent trees".
const KG_CO2_PER_TREE = 20
// Green points: check-in streak × 10 + completed orders × 5.
const greenPoints = computed(() => checkInStreak.value * 10 + completedOrders.value * 5)
// Total CO2 saving in kg from completed orders.
const co2Saved = computed(() => Number((completedOrders.value * CO2_PER_ORDER).toFixed(1)))
// Equivalent trees planted, 1 decimal.
const treesEquivalent = computed(() => Number((co2Saved.value / KG_CO2_PER_TREE).toFixed(1)))

// ---- Quick stats dashboard (个人中心速览) ----
// Counts shown at the top of the Mine page: orders, favorites, coupons, points.
const stats = ref({ orders: 0, favorites: 0, coupons: 0, points: 0 })
function goStat(path) {
  router.push(path)
}

// ---- Dark mode (深色模式) ----
// Mirror of the flag stored in App.vue; initialized from localStorage so the
// switch reflects the real state on first render.
const darkMode = ref(localStorage.getItem('jd_dark_mode') === 'true')
function toggleDark(val) {
  darkMode.value = val
  localStorage.setItem('jd_dark_mode', String(val))
  // Let App.vue apply the class + body bg.
  if (window.__setJdDarkMode) window.__setJdDarkMode(val)
  else {
    const el = document.querySelector('.app-wrap')
    if (el) el.classList.toggle('dark-mode', val)
    document.body.classList.toggle('jd-dark', val)
  }
  showToast(val ? '已开启深色模式' : '已关闭深色模式')
}

// ---- Member level tiers (会员成长值等级) ----
// Thresholds are cumulative growth points. Badges use the requested colors.
const levels = [
  { name: '青铜', min: 0,    color: '#a0522d', cls: 'bronze' },
  { name: '白银', min: 100,  color: '#9e9e9e', cls: 'silver' },
  { name: '黄金', min: 500,  color: '#f5a623', cls: 'gold' },
  { name: '铂金', min: 1500, color: '#00bcd4', cls: 'platinum' },
  { name: '钻石', min: 3000, color: '#9c27b0', cls: 'diamond' },
  { name: '王者', min: 6000, color: '#e1251b', cls: 'king' },
]

// Current level + progress to next level.
const currentLevel = computed(() => {
  let cur = levels[0]
  for (const lv of levels) {
    if (growthPoints.value >= lv.min) cur = lv
  }
  return cur
})
const nextLevel = computed(() => {
  for (const lv of levels) {
    if (lv.min > currentLevel.value.min) return lv
  }
  return null // already at max level
})
const progressPct = computed(() => {
  if (!nextLevel.value) return 100 // max level
  const span = nextLevel.value.min - currentLevel.value.min
  const got = growthPoints.value - currentLevel.value.min
  return Math.min(100, Math.max(0, Math.round((got / span) * 100)))
})
const pointsToNext = computed(() => {
  if (!nextLevel.value) return 0
  return Math.max(0, nextLevel.value.min - growthPoints.value)
})

async function load() {
  loggedIn.value = !!localStorage.getItem('jd_token')
  if (!loggedIn.value) return
  try {
    const res = await getProfile()
    user.value = res.user
    cartCount.value = res.cart_count || 0
    // Growth value = check-in points (simplified per the spec).
    try {
      const ci = await getCheckInStatus()
      growthPoints.value = ci.total_points || 0
    } catch (_) {
      growthPoints.value = 0
    }
    // Quick stats: fetch counts in parallel so a slow endpoint doesn't block.
    loadStats()
  } catch (e) {
    loggedIn.value = false
  }
}

// Fetch the four quick-stat counts. Failures leave the stat at 0 rather than
// breaking the page. Also captures the carbon-footprint inputs (check-in streak
// and completed-order count).
async function loadStats() {
  try {
    const [orders, favs, mine, ci] = await Promise.all([
      getOrders().catch(() => []),
      listFavorites().catch(() => []),
      getMyCoupons().catch(() => []),
      getCheckInStatus().catch(() => ({})),
    ])
    const orderList = orders || []
    stats.value = {
      orders: orderList.length,
      favorites: (favs || []).length,
      // "Unused" coupon count = coupons not yet redeemed/expired.
      coupons: (mine || []).filter((c) => c && !c.used).length,
      points: ci.total_points || 0,
    }
    // Carbon-footprint inputs: check-in streak + completed orders.
    checkInStreak.value = (ci.last && Number(ci.last.streak)) || 0
    completedOrders.value = orderList.filter(
      (o) => o && o.status === 'completed'
    ).length
  } catch (_) {
    // keep zeros
  }
}
onMounted(() => {
  load()
  refreshNotif()
})
onActivated(() => {
  load()
  refreshNotif()
})

function logout() {
  localStorage.removeItem('jd_token')
  localStorage.removeItem('jd_user')
  loggedIn.value = false
  user.value = null
  showToast('已退出登录')
}
</script>

<template>
  <div class="mine-page">
    <!-- Header -->
    <div class="mine-header">
      <div v-if="loggedIn && user" class="user-info">
        <van-image round width="60" height="60" :src="user.avatar || 'https://via.placeholder.com/60'" />
        <div class="u-text">
          <div class="u-name">{{ user.nickname || user.username }}</div>
          <div class="u-id">用户名: {{ user.username }}</div>
        </div>
      </div>
      <div v-else class="user-info" @click="router.push('/login')">
        <van-image round width="60" height="60" src="https://via.placeholder.com/60" />
        <div class="u-text">
          <div class="u-name">登录/注册</div>
          <div class="u-id">点击登录享受更多优惠</div>
        </div>
      </div>
    </div>

    <!-- Quick stats dashboard (个人中心速览) -->
    <div class="quick-stats">
      <div class="qs-card" @click="goStat('/orders')">
        <van-icon name="orders-o" size="24" color="#e1251b" />
        <div class="qs-num">{{ stats.orders }}</div>
        <div class="qs-label">订单数</div>
      </div>
      <div class="qs-card" @click="goStat('/favorites')">
        <van-icon name="star-o" size="24" color="#ff976a" />
        <div class="qs-num">{{ stats.favorites }}</div>
        <div class="qs-label">收藏数</div>
      </div>
      <div class="qs-card" @click="goStat('/coupons')">
        <van-icon name="coupon-o" size="24" color="#f5a623" />
        <div class="qs-num">{{ stats.coupons }}</div>
        <div class="qs-label">优惠券数</div>
      </div>
      <div class="qs-card" @click="goStat('/checkin')">
        <van-icon name="gem-o" size="24" color="#1989fa" />
        <div class="qs-num">{{ stats.points }}</div>
        <div class="qs-label">积分数</div>
      </div>
    </div>

    <!-- Member growth section (会员成长值) -->
    <div class="growth-card">
      <div class="gc-top">
        <div class="gc-head">
          <span class="gc-title">成长值</span>
          <span class="gc-cur-badge" :class="currentLevel.cls">
            <span class="badge-dot"></span>{{ currentLevel.name }}会员
          </span>
        </div>
        <div class="gc-points">{{ growthPoints }}</div>
      </div>

      <div class="gc-bar">
        <div class="gc-fill" :class="currentLevel.cls" :style="{ width: progressPct + '%' }"></div>
      </div>
      <div class="gc-progress-text">
        <span>{{ currentLevel.name }}</span>
        <span v-if="nextLevel">{{ nextLevel.name }}</span>
        <span v-else>已达最高等级</span>
      </div>
      <div class="gc-next" v-if="nextLevel">
        距下一等级还需 <b>{{ pointsToNext }}</b> 积分
      </div>
      <div class="gc-next gc-max" v-else>
        恭喜！您已达最高等级
      </div>

      <!-- Level badge ladder -->
      <div class="gc-ladder">
        <div
          v-for="lv in levels"
          :key="lv.name"
          class="lc-item"
          :class="lv.cls"
          :style="{ opacity: growthPoints >= lv.min ? 1 : 0.45 }"
        >
          <div class="lc-badge">{{ lv.name }}</div>
          <div class="lc-min">{{ lv.min }}+</div>
        </div>
      </div>
    </div>

    <!-- Feature: 个人碳足迹 (Mine Carbon Footprint) — earth-green themed card
         with a tree emoji animation. Shows green points, CO2 savings derived
         from completed orders, and the equivalent trees planted. -->
    <div class="carbon-card">
      <div class="carbon-head">
        <span class="carbon-title">🌱 碳足迹</span>
        <span class="carbon-points">绿色积分 {{ greenPoints }}</span>
      </div>
      <div class="carbon-trees">
        <span class="carbon-tree t1">🌳</span>
        <span class="carbon-tree t2">🌲</span>
        <span class="carbon-tree t3">🌴</span>
      </div>
      <div class="carbon-main">
        您的绿色消费已减少 <b>{{ co2Saved }}</b> kg碳排放
      </div>
      <div class="carbon-sub">相当于种了 <b>{{ treesEquivalent }}</b> 棵树</div>
    </div>

    <!-- Order entries -->
    <van-cell-group inset title="我的订单">
      <div class="order-entries">
        <div class="oe-item" @click="router.push('/orders')">
          <van-icon name="balance-pay-o" size="28" color="#ff976a" />
          <span>待付款</span>
        </div>
        <div class="oe-item" @click="router.push('/orders')">
          <van-icon name="logistics" size="28" color="#07c160" />
          <span>待发货</span>
        </div>
        <div class="oe-item" @click="router.push('/orders')">
          <van-icon name="gift-o" size="28" color="#e1251b" />
          <span>待收货</span>
        </div>
        <div class="oe-item" @click="router.push('/orders')">
          <van-icon name="comment-o" size="28" color="#1989fa" />
          <span>待评价</span>
        </div>
      </div>
    </van-cell-group>

    <van-cell-group inset title="常用功能">
      <van-cell title="购物车" :value="cartCount + '件'" is-link @click="router.push('/cart')" icon="cart-o" />
      <van-cell title="我的订单" is-link @click="router.push('/orders')" icon="orders-o" />
      <van-cell title="我的收藏" is-link @click="router.push('/favorites')" icon="star-o" />
      <van-cell title="浏览历史" is-link @click="router.push('/history')" icon="clock-o" />
      <van-cell title="每日签到" is-link @click="router.push('/checkin')" icon="calendar-o" />
      <van-cell title="积分商城" is-link @click="router.push('/points-shop')" icon="gold-coin-o" />
      <van-cell title="积分抽奖" is-link @click="router.push('/lottery')" icon="gem-o" />
      <van-cell title="超值拼团" is-link @click="router.push('/group-buy')" icon="friends-o" />
      <van-cell title="超值套餐" is-link @click="router.push('/bundles')" icon="gift-o" />
      <van-cell title="预售专区" is-link @click="router.push('/presale')" icon="underway-o" />
      <van-cell title="售后服务" is-link @click="goNotif('/refunds', 'refunds')" icon="after-sale">
        <template #value><span v-if="notif.refunds" class="notif-dot"></span></template>
      </van-cell>
      <van-cell title="优惠券" is-link @click="goNotif('/coupons', 'coupons')" icon="coupon-o">
        <template #value><span v-if="notif.coupons" class="notif-dot"></span></template>
      </van-cell>
      <van-cell title="礼品卡" is-link @click="router.push('/gift-card')" icon="card" />
      <van-cell title="收货地址" is-link icon="location-o" @click="goNotif('/addresses', 'addresses')">
        <template #value><span v-if="notif.addresses" class="notif-dot"></span></template>
      </van-cell>
      <van-cell title="编辑资料" is-link icon="edit" @click="router.push('/profile')" />
      <van-cell title="PLUS会员" is-link icon="diamond-o" @click="showToast('演示功能')" />
      <!-- Dark mode toggle (深色模式) -->
      <van-cell title="深色模式" icon="closed-eye">
        <template #right-icon>
          <van-switch :model-value="darkMode" @update:model-value="toggleDark" size="22px" />
        </template>
      </van-cell>
      <van-cell title="管理后台" is-link @click="router.push('/admin')" icon="setting-o" />
    </van-cell-group>

    <div v-if="loggedIn" style="margin: 20px">
      <van-button block plain type="danger" @click="logout">退出登录</van-button>
    </div>
  </div>
</template>

<style scoped>
.mine-page { min-height: 100vh; padding-bottom: 20px; }
.mine-header { background: linear-gradient(135deg, #e1251b, #f5574d); padding: 30px 20px; color: #fff; }
.user-info { display: flex; align-items: center; gap: 14px; }
.u-name { font-size: 18px; font-weight: bold; }
.u-id { font-size: 12px; opacity: 0.8; margin-top: 4px; }
.order-entries { display: flex; padding: 16px 0; }
.oe-item { flex: 1; text-align: center; font-size: 12px; color: #666; }
.oe-item span { display: block; margin-top: 4px; }

/* ---- Quick stats dashboard (个人中心速览) ---- */
.quick-stats {
  display: flex;
  margin: -16px 12px 12px;
  background: #fff;
  border-radius: 12px;
  padding: 14px 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  position: relative;
  z-index: 2;
}
.qs-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}
.qs-card:active { opacity: 0.6; }
.qs-num {
  font-size: 18px;
  font-weight: bold;
  color: #333;
  font-family: 'Courier New', monospace;
}
.qs-label { font-size: 11px; color: #999; }

/* ---- Member growth section (会员成长值) ---- */
.growth-card {
  margin: -12px 16px 12px;
  background: #fff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  position: relative;
  z-index: 2;
}
.gc-top { display: flex; justify-content: space-between; align-items: flex-start; }
.gc-head { display: flex; align-items: center; gap: 8px; }
.gc-title { font-size: 16px; font-weight: bold; color: #333; }
.gc-points { font-size: 22px; font-weight: bold; color: #e1251b; font-family: 'Courier New', monospace; }

/* current level badge */
.gc-cur-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #fff;
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: bold;
}
.gc-cur-badge .badge-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: #fff;
  box-shadow: 0 0 0 1px rgba(255,255,255,0.5);
}

/* progress bar */
.gc-bar {
  height: 10px;
  background: #f0f0f0;
  border-radius: 6px;
  overflow: hidden;
  margin: 12px 0 6px;
}
.gc-fill {
  height: 100%;
  border-radius: 6px;
  transition: width 0.4s ease;
}
.gc-progress-text {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #999;
}
.gc-next { font-size: 12px; color: #666; margin-top: 8px; }
.gc-next b { color: #e1251b; font-size: 14px; }
.gc-next.gc-max { color: #e1251b; font-weight: bold; }

/* badge ladder */
.gc-ladder {
  display: flex;
  justify-content: space-between;
  margin-top: 14px;
  gap: 4px;
}
.lc-item {
  flex: 1;
  text-align: center;
}
.lc-badge {
  font-size: 11px;
  color: #fff;
  padding: 4px 0;
  border-radius: 6px;
  font-weight: bold;
}
.lc-min { font-size: 10px; color: #aaa; margin-top: 4px; }

/* tier colors */
.bronze   { background: linear-gradient(135deg, #c0855a, #8b5a2b); }
.silver   { background: linear-gradient(135deg, #c9c9c9, #8e8e8e); }
.gold     { background: linear-gradient(135deg, #ffd700, #f5a623); }
.platinum { background: linear-gradient(135deg, #4dd0e1, #00bcd4); }
.diamond  { background: linear-gradient(135deg, #ba68c8, #9c27b0); }
.king     { background: linear-gradient(135deg, #ff5858, #e1251b 60%, #b71c1c); }

/* make the current-badge use a solid tier color (no gradient via .cls bg override) */
.gc-cur-badge.bronze   { background: #8b5a2b; }
.gc-cur-badge.silver   { background: #9e9e9e; }
.gc-cur-badge.gold     { background: #f5a623; }
.gc-cur-badge.platinum { background: #00bcd4; }
.gc-cur-badge.diamond  { background: #9c27b0; }
.gc-cur-badge.king     { background: linear-gradient(135deg, #ff5858, #e1251b); }

/* keep fill bar colors vivid per tier */
.gc-fill.bronze   { background: linear-gradient(90deg, #c0855a, #8b5a2b); }
.gc-fill.silver   { background: linear-gradient(90deg, #c9c9c9, #8e8e8e); }
.gc-fill.gold     { background: linear-gradient(90deg, #ffe082, #f5a623); }
.gc-fill.platinum { background: linear-gradient(90deg, #80deea, #00bcd4); }
.gc-fill.diamond  { background: linear-gradient(90deg, #ce93d8, #9c27b0); }
.gc-fill.king     { background: linear-gradient(90deg, #ff8a80, #e1251b); }

/* ---- Notification dot (个人中心消息红点) ---- */
.notif-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #e1251b;
  vertical-align: middle;
}

/* ---- Feature: 个人碳足迹 (Mine Carbon Footprint) ---- */
/* Fun earth-green themed card with a tree emoji animation. */
.carbon-card {
  margin: -8px 16px 12px;
  background: linear-gradient(135deg, #e8f5e9 0%, #d0eee2 50%, #f0fff4 100%);
  border: 1px solid #b7eb8f;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(7, 193, 96, 0.12);
  position: relative;
  overflow: hidden;
}
.carbon-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.carbon-title {
  font-size: 16px;
  font-weight: bold;
  color: #1b5e20;
}
.carbon-points {
  font-size: 12px;
  color: #fff;
  background: #07c160;
  padding: 3px 10px;
  border-radius: 12px;
  font-weight: 600;
}
.carbon-trees {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin: 4px 0 10px;
}
.carbon-tree {
  font-size: 26px;
  display: inline-block;
  transform-origin: bottom center;
}
.carbon-tree.t1 { animation: carbon-sway 2.4s ease-in-out infinite; }
.carbon-tree.t2 { animation: carbon-sway 2.4s ease-in-out infinite 0.4s; }
.carbon-tree.t3 { animation: carbon-sway 2.4s ease-in-out infinite 0.8s; }
@keyframes carbon-sway {
  0%, 100% { transform: rotate(-6deg); }
  50% { transform: rotate(6deg); }
}
.carbon-main {
  font-size: 14px;
  color: #2e7d32;
  line-height: 22px;
  text-align: center;
}
.carbon-main b {
  color: #07c160;
  font-size: 18px;
  font-weight: bold;
}
.carbon-sub {
  font-size: 13px;
  color: #388e3c;
  text-align: center;
  margin-top: 4px;
}
.carbon-sub b {
  color: #07c160;
  font-size: 16px;
  font-weight: bold;
}
</style>
