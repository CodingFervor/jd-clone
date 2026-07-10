<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProfile, getCheckInStatus } from '../api'

const router = useRouter()
const user = ref(null)
const cartCount = ref(0)
const loggedIn = ref(false)
const growthPoints = ref(0) // growth value derived from check-in points

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
  } catch (e) {
    loggedIn.value = false
  }
}
onMounted(load)
onActivated(load)

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
      <van-cell title="售后服务" is-link @click="router.push('/refunds')" icon="after-sale" />
      <van-cell title="优惠券" is-link @click="router.push('/coupons')" icon="coupon-o" />
      <van-cell title="礼品卡" is-link @click="router.push('/gift-card')" icon="card" />
      <van-cell title="收货地址" is-link icon="location-o" @click="router.push('/addresses')" />
      <van-cell title="编辑资料" is-link icon="edit" @click="router.push('/profile')" />
      <van-cell title="PLUS会员" is-link icon="diamond-o" @click="showToast('演示功能')" />
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
</style>
