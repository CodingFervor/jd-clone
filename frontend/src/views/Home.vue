<script setup>
import { ref, onMounted, onUnmounted, onActivated, computed } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getSeckill, getProducts, getCategories, getProduct } from '../api'

const router = useRouter()
const seckill = ref([])
const products = ref([])
const categories = ref([])
const loading = ref(false)
// Skeleton placeholders: a fixed set of dummy cards rendered while the real
// data is loading, so the layout doesn't jump when content arrives.
const skeletonSeckill = Array.from({ length: 4 })
const skeletonProducts = Array.from({ length: 3 })

// ---- Flash sale countdown (首页限时抢购倒计时) ----
// Counts down to the next top-of-hour. When it hits 0, briefly shows
// "正在抢购中!" then resets the target to the following hour.
const countdown = ref('00:00:00')
const flashLive = ref(false) // true -> sale is "live" right now
let flashTimer = null

// Target is the next top-of-hour (e.g. 14:00:00).
function nextTopOfHour(from = new Date()) {
  const t = new Date(from)
  t.setMinutes(0, 0, 0)
  t.setHours(t.getHours() + 1)
  return t.getTime()
}

let flashTarget = nextTopOfHour()

function pad(n) {
  return String(n).padStart(2, '0')
}

function tick() {
  const diff = flashTarget - Date.now()
  if (diff <= 0) {
    // Sale just went live: show live state briefly, then reset target.
    flashLive.value = true
    countdown.value = '00:00:00'
    setTimeout(() => {
      flashLive.value = false
      flashTarget = nextTopOfHour()
      updateCountdown()
    }, 5000)
    return
  }
  flashLive.value = false
  updateCountdown(diff)
}

function updateCountdown(diff = flashTarget - Date.now()) {
  const d = Math.max(0, diff)
  const h = Math.floor(d / 3600000)
  const m = Math.floor((d % 3600000) / 60000)
  const s = Math.floor((d % 60000) / 1000)
  countdown.value = `${pad(h)}:${pad(m)}:${pad(s)}`
}

onMounted(async () => {
  // Start the flash sale countdown ticker immediately.
  tick()
  flashTimer = setInterval(tick, 1000)
  // Load the last-viewed product for the price-tracking floating ball and
  // refresh its live price every 30s so the ↑/↓ badge stays accurate.
  loadLastViewed()
  refreshLivePrice()
  priceBallTimer = setInterval(refreshLivePrice, 30000)
  loading.value = true
  try {
    const [sk, pl, cats] = await Promise.all([getSeckill(), getProducts({ page: 1, page_size: 20 }), getCategories()])
    seckill.value = sk
    products.value = pl.data
    categories.value = (cats || []).slice(0, 10)
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})

// When returning to Home (e.g. after viewing a product), reload the tracked
// product so the floating ball reflects the most recently viewed item. This
// runs under keep-alive as well as on a fresh mount.
onActivated(() => {
  loadLastViewed()
  refreshLivePrice()
})

function goSearch() {
  router.push('/search')
}

// ---- Product comparison floating widget (比价悬浮球) ----
// Tracks the last-viewed product (saved by ProductDetail.vue to localStorage)
// and surfaces a small floating ball bottom-right, above the tabbar. It shows
// the product thumbnail + name + price and an ↑涨价 / ↓降价 badge when the
// live price differs from the price the user saw. Tapping navigates back to
// the product; long-pressing dismisses it.
const lastViewed = ref(null) // { id, name, image, price, original_price }
const livePrice = ref(null) // current price fetched from the backend
let priceBallTimer = null
let priceBallPressTimer = null

// Price-change badge comparing the stored price to the live price.
const priceDelta = computed(() => {
  if (!lastViewed.value || livePrice.value == null) return null
  const stored = Number(lastViewed.value.price) || 0
  const live = Number(livePrice.value) || 0
  if (!stored || !live) return null
  if (live > stored) return 'up' // ↑涨价
  if (live < stored) return 'down' // ↓降价
  return null
})

// Refresh the live price for the tracked product so the badge stays current.
async function refreshLivePrice() {
  if (!lastViewed.value) return
  try {
    const res = await getProduct(lastViewed.value.id)
    const p = res && res.data ? res.data : null
    if (p) livePrice.value = Number(p.price) || null
  } catch (_) {
    // Network/backend may be unavailable; just keep the stored price.
  }
}

function loadLastViewed() {
  try {
    const raw = localStorage.getItem('jd_last_viewed')
    if (!raw) return
    const obj = JSON.parse(raw)
    if (obj && obj.id) lastViewed.value = obj
  } catch (_) {
    // ignore malformed entries
  }
}

function goLastViewed() {
  if (priceBallPressTimer) {
    clearTimeout(priceBallPressTimer)
    priceBallPressTimer = null
  }
  if (lastViewed.value && lastViewed.value.id) {
    router.push('/product/' + lastViewed.value.id)
  }
}

// Long-press (~600ms) dismisses the widget and forgets the product.
function onBallTouchStart() {
  priceBallPressTimer = setTimeout(() => {
    priceBallPressTimer = null
    dismissBall()
  }, 600)
}
function onBallTouchEnd() {
  if (priceBallPressTimer) {
    clearTimeout(priceBallPressTimer)
    priceBallPressTimer = null
  }
}
function dismissBall() {
  lastViewed.value = null
  livePrice.value = null
  try {
    localStorage.removeItem('jd_last_viewed')
  } catch (_) {}
  showToast('已移除比价')
}
function goCategory(id) {
  router.push({ name: 'category', query: { id } })
}
function goProduct(id) {
  router.push('/product/' + id)
}
function fmt(n) {
  return Number(n).toFixed(2)
}

// ---- New product & seckill badges (新品限时标签) ----
// A product is "NEW" if created within the last 7 days.
const WEEK_MS = 7 * 24 * 60 * 60 * 1000
function isNew(p) {
  if (!p || !p.created_at) return false
  const created = new Date(p.created_at).getTime()
  if (isNaN(created)) return false
  return Date.now() - created < WEEK_MS
}
function isSeckill(p) {
  return !!(p && p.is_seckill)
}

// ---- Price trend tag (价格趋势图标) ----
// Returns the inline trend tag for a product, shown next to its price.
// Priority: seckill (⚡秒杀, orange) > price drop (📉降价, green) > none.
// A "drop" only shows when the original price is more than 10% above the
// current price; when original_price <= price (or unset) nothing renders.
function priceTrendTag(p) {
  if (!p) return null
  if (isSeckill(p)) return 'seckill'
  const orig = Number(p.original_price)
  const cur = Number(p.price)
  if (!orig || orig <= cur) return null
  if (orig > cur * 1.1) return 'drop'
  return null
}

// Split countdown into 3 monospace blocks for styling.
const timeBlocks = computed(() => countdown.value.split(':'))

// ---- Popular tags feed (热门标签信息流) ----
// Hardcoded popular discovery tags. Clicking a tag navigates to the search
// page pre-filled with the tag text, reusing the existing search flow.
const hotTags = [
  '新品上市',
  '限时特惠',
  '品质好物',
  '夏日必备',
  '居家优选',
  '数码达人',
  '美妆护肤',
  '食品生鲜',
]
// Rotate through 5 festive pill colors for visual variety.
const tagColorClass = (i) => `tag-c${(i % 5) + 1}`

function goTag(tag) {
  router.push({ path: '/search', query: { q: tag } })
}

// ---- Feature: 首页天气小组件 (Home Weather Widget) ----
// Deterministic weather (sunny/rainy/cloudy) + temperature derived from a hash
// of today's date, so the same day always shows the same weather. Includes a
// "适合购物" shopping tip.
function dateHash() {
  const d = new Date()
  const s = `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`
  let h = 0
  for (let i = 0; i < s.length; i++) {
    h = (h << 5) - h + s.charCodeAt(i)
    h |= 0
  }
  return Math.abs(h)
}
const WEATHER_STATES = [
  { icon: '☀️', label: '晴', tip: '阳光明媚，适合购物', color: '#ff9800' },
  { icon: '☁️', label: '多云', tip: '天气舒适，适合购物', color: '#7cb342' },
  { icon: '🌧️', label: '小雨', tip: '雨天宅家，适合购物', color: '#42a5f5' },
]
const weather = WEATHER_STATES[dateHash() % WEATHER_STATES.length]
// Deterministic temperature in [18, 32].
const weatherTemp = computed(() => 18 + (dateHash() % 15))
const weatherDateStr = computed(() => {
  const d = new Date()
  const wk = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][d.getDay()]
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}月${d.getDate()}日 ${wk}`
})

// ---- Auto-scroll promo marquee (首页跑马灯) ----
// Five promotional messages scrolled horizontally below the flash banner.
// Each entry carries the display text (without the emoji prefix) and a longer
// detail string shown in a toast when the bar is tapped.
const marqueeMessages = [
  { text: '满199减15', detail: '🎉 满199减15：下单满199元立减15元，部分商品专享' },
  { text: '全国包邮', detail: '📦 全国包邮：所有实物商品全国包邮（偏远地区除外）' },
  { text: '限时秒杀进行中', detail: '⚡ 限时秒杀进行中：整点开抢，低至5折，手慢无！' },
  { text: '新人专享礼包', detail: '🎁 新人专享礼包：新用户登录即领188元大礼包' },
  { text: 'PLUS会员95折', detail: '💎 PLUS会员95折：PLUS会员专享全场自营商品95折' },
]
// The continuously scrolling track content: messages joined by • dots.
const marqueeTrack = marqueeMessages.map((m) => m.text).join('  •  ')

// Tapping the marquee shows the detail of a (rotating) message in a toast.
const marqueeIdx = ref(0)
function onMarqueeTap() {
  const m = marqueeMessages[marqueeIdx.value % marqueeMessages.length]
  marqueeIdx.value += 1
  showToast(m.detail)
}

onUnmounted(() => {
  if (flashTimer) {
    clearInterval(flashTimer)
    flashTimer = null
  }
  if (priceBallTimer) {
    clearInterval(priceBallTimer)
    priceBallTimer = null
  }
  if (priceBallPressTimer) {
    clearTimeout(priceBallPressTimer)
    priceBallPressTimer = null
  }
})
</script>

<template>
  <div class="home">
    <!-- Top search bar -->
    <van-sticky>
      <div class="topbar">
        <span class="logo">JD</span>
        <van-search class="search" placeholder="搜索京东商品" shape="round" readonly @click="goSearch" />
        <van-icon name="login" size="22" @click="router.push('/login')" />
      </div>
    </van-sticky>

    <!-- Banner -->
    <div class="banner">
      <van-image
        fit="cover"
        width="100%"
        height="160"
        src="https://img12.360buyimg.com/babel/s1180x270_jfs/t1/banner-jd.jpg"
      />
    </div>

    <!-- Feature: 首页天气小组件 (Home Weather Widget) — deterministic weather
         + temperature card with a "适合购物" shopping tip. -->
    <div class="weather-card">
      <div class="wc-icon" :style="{ color: weather.color }">{{ weather.icon }}</div>
      <div class="wc-info">
        <div class="wc-top">
          <span class="wc-temp">{{ weatherTemp }}°</span>
          <span class="wc-label">{{ weather.label }}</span>
        </div>
        <div class="wc-date">{{ weatherDateStr }}</div>
      </div>
      <div class="wc-tip">
        <span class="wc-tip-icon">🛒</span>
        <span class="wc-tip-text">{{ weather.tip }}</span>
      </div>
    </div>

    <!-- Category grid -->
    <div class="cat-grid">
      <div v-for="c in categories" :key="c.id" class="cat-item" @click="goCategory(c.id)">
        <div class="cat-icon">{{ c.icon }}</div>
        <div class="cat-name">{{ c.name }}</div>
      </div>
    </div>

    <!-- Flash sale countdown banner (首页限时抢购倒计时) -->
    <div class="flash-banner" :class="{ live: flashLive }">
      <div class="fb-text">
        <span class="fb-icon">⚡</span>
        <span class="fb-title">限时秒杀</span>
        <span class="fb-sub">{{ flashLive ? '正在抢购中!' : '距开抢还有' }}</span>
      </div>
      <div v-if="!flashLive" class="fb-countdown">
        <span class="fb-block">{{ timeBlocks[0] }}</span>
        <span class="fb-colon">:</span>
        <span class="fb-block">{{ timeBlocks[1] }}</span>
        <span class="fb-colon">:</span>
        <span class="fb-block">{{ timeBlocks[2] }}</span>
      </div>
      <div v-else class="fb-live-tag">GO</div>
    </div>

    <!-- Auto-scroll promo marquee (首页跑马灯) -->
    <div class="marquee-bar" @click="onMarqueeTap">
      <span class="mb-icon">📣</span>
      <div class="mb-viewport">
        <span class="mb-track">{{ marqueeTrack }}</span>
      </div>
    </div>

    <!-- Seckill floor -->
    <div class="section">      <div class="section-head" @click="router.push('/seckill')" style="cursor:pointer">
        <span class="jd-red"><van-icon name="clock-o" /> 京东秒杀</span>
        <span class="more">更多 ›</span>
      </div>
      <!-- Skeleton seckill cards while loading -->
      <div v-if="loading" class="seckill-scroll">
        <div v-for="(_, i) in skeletonSeckill" :key="'sk-' + i" class="seckill-card">
          <div class="skel skel-seckill-img"></div>
          <div class="skel skel-seckill-price"></div>
          <div class="skel skel-seckill-origin"></div>
        </div>
      </div>
      <!-- Real content (fade in once loaded) -->
      <div v-else class="seckill-scroll" :class="{ 'content-fade-in': !loading }">
        <div v-for="p in seckill" :key="p.id" class="seckill-card" @click="goProduct(p.id)">
          <van-image width="90" height="90" radius="6" :src="p.image" fit="cover" />
          <div class="price">¥{{ fmt(p.price) }}</div>
          <div class="origin">¥{{ fmt(p.original_price) }}</div>
        </div>
      </div>
    </div>

    <!-- Popular tags feed (热门标签) -->
    <div class="tag-section">
      <div class="tag-section-head">
        <span class="tsh-title"><van-icon name="hot-o" /> 热门标签</span>
        <span class="tsh-sub">点击发现好物</span>
      </div>
      <div class="tag-scroll">
        <span
          v-for="(t, i) in hotTags"
          :key="t"
          class="tag-chip"
          :class="tagColorClass(i)"
          @click="goTag(t)"
        >{{ t }}</span>
      </div>
    </div>

    <!-- Product list (waterfall) -->
    <div class="section">
      <div class="section-head"><span>为你推荐</span></div>
      <!-- Skeleton product cards while loading -->
      <div v-if="loading" class="product-grid">
        <div v-for="(_, i) in skeletonProducts" :key="'sp-' + i" class="product-card">
          <div class="skel skel-product-img"></div>
          <div class="skel skel-product-line"></div>
          <div class="skel skel-product-line short"></div>
        </div>
      </div>
      <!-- Real content (fade in once loaded) -->
      <div v-else class="product-grid" :class="{ 'content-fade-in': !loading }">
        <div v-for="p in products" :key="p.id" class="product-card" @click="goProduct(p.id)">
          <div class="p-img-wrap">
            <van-image width="100%" height="170" :src="p.image" fit="cover" radius="6" />
            <span v-if="isNew(p)" class="p-badge p-badge-new">NEW</span>
            <span v-if="isSeckill(p)" class="p-badge p-badge-seckill">限时</span>
          </div>
          <div class="p-name van-multi-ellipsis--l2">{{ p.name }}</div>
          <div class="p-shop">{{ p.shop }}</div>
          <div class="p-bottom">
            <span class="price-wrap">
              <span class="price">¥{{ fmt(p.price) }}</span>
              <span v-if="priceTrendTag(p) === 'seckill'" class="trend-tag trend-seckill">⚡秒杀</span>
              <span v-else-if="priceTrendTag(p) === 'drop'" class="trend-tag trend-drop">📉降价</span>
            </span>
            <span class="sales">{{ p.sales }}人付款</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Product comparison floating widget (比价悬浮球) -->
    <transition name="ball-pop">
      <div
        v-if="lastViewed"
        class="price-ball"
        @click="goLastViewed"
        @touchstart.passive="onBallTouchStart"
        @touchend="onBallTouchEnd"
        @touchcancel="onBallTouchEnd"
        @mousedown="onBallTouchStart"
        @mouseup="onBallTouchEnd"
        @mouseleave="onBallTouchEnd"
      >
        <van-image round width="32" height="32" :src="lastViewed.image" fit="cover" class="pb-thumb" />
        <span class="pb-badge" :class="priceDelta" v-if="priceDelta">{{ priceDelta === 'up' ? '↑涨价' : '↓降价' }}</span>
        <span class="pb-info">
          <span class="pb-name van-ellipsis">{{ lastViewed.name }}</span>
          <span class="pb-price">¥{{ fmt(livePrice != null ? livePrice : lastViewed.price) }}</span>
        </span>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #fff;
}
.logo {
  color: #e1251b;
  font-weight: bold;
  font-size: 22px;
}
.search {
  flex: 1;
  padding: 0;
}
.banner {
  margin: 8px;
  border-radius: 8px;
  overflow: hidden;
}

/* Feature: 首页天气小组件 (Home Weather Widget) */
.weather-card {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 8px 8px;
  padding: 12px 14px;
  border-radius: 10px;
  background: linear-gradient(135deg, #e3f2fd 0%, #e8f5e9 60%, #fff8e1 100%);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}
.wc-icon {
  font-size: 34px;
  line-height: 1;
  flex-shrink: 0;
}
.wc-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex-shrink: 0;
}
.wc-top {
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.wc-temp {
  font-size: 24px;
  font-weight: bold;
  color: #333;
  font-family: 'Courier New', monospace;
}
.wc-label {
  font-size: 13px;
  color: #666;
}
.wc-date {
  font-size: 11px;
  color: #999;
}
.wc-tip {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: rgba(225, 37, 27, 0.9);
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  padding: 5px 12px;
  border-radius: 999px;
  white-space: nowrap;
}
.wc-tip-icon {
  font-size: 13px;
}
.cat-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 12px 0;
  padding: 16px 8px;
  background: #fff;
  margin: 0 8px 8px;
  border-radius: 8px;
}
.cat-item {
  text-align: center;
}
.cat-icon {
  font-size: 28px;
}
.cat-name {
  font-size: 12px;
  color: #666;
  margin-top: 4px;
}
.section {
  background: #fff;
  margin: 0 8px 8px;
  border-radius: 8px;
  padding: 12px;
}
.section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 16px;
  font-weight: bold;
  margin-bottom: 10px;
}
.more {
  font-size: 12px;
  color: #999;
  font-weight: normal;
}
.seckill-scroll {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 4px;
}
.seckill-card {
  flex-shrink: 0;
  width: 90px;
  text-align: center;
}
.origin {
  font-size: 11px;
  color: #999;
  text-decoration: line-through;
}

/* Popular tags feed (热门标签) */
.tag-section {
  margin: 0 8px 8px;
  border-radius: 10px;
  padding: 14px 12px;
  background: linear-gradient(135deg, #fff5f5 0%, #fff0f6 40%, #fff7e6 70%, #f0f5ff 100%);
  box-shadow: 0 2px 8px rgba(225, 37, 27, 0.06);
}
.tag-section-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 12px;
}
.tsh-title {
  font-size: 16px;
  font-weight: bold;
  color: #e1251b;
}
.tsh-sub {
  font-size: 12px;
  color: #999;
}
.tag-scroll {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.tag-chip {
  display: inline-block;
  padding: 7px 16px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  color: #fff;
  cursor: pointer;
  white-space: nowrap;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.tag-chip:active {
  transform: scale(0.95);
}
.tag-c1 { background: linear-gradient(135deg, #e1251b, #ff4d4f); box-shadow: 0 2px 6px rgba(225, 37, 27, 0.3); }
.tag-c2 { background: linear-gradient(135deg, #ff7a18, #ffb84d); box-shadow: 0 2px 6px rgba(255, 122, 24, 0.3); }
.tag-c3 { background: linear-gradient(135deg, #fa2c6e, #ff6fae); box-shadow: 0 2px 6px rgba(250, 44, 110, 0.3); }
.tag-c4 { background: linear-gradient(135deg, #13c2c2, #36cfc9); box-shadow: 0 2px 6px rgba(19, 194, 194, 0.3); }
.tag-c5 { background: linear-gradient(135deg, #722ed1, #9254de); box-shadow: 0 2px 6px rgba(114, 46, 209, 0.3); }
.product-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.product-card {
  background: #fafafa;
  border-radius: 8px;
  overflow: hidden;
  padding-bottom: 6px;
}
.p-img-wrap {
  position: relative;
}
/* New product & seckill badges (新品限时标签) */
.p-badge {
  position: absolute;
  left: 6px;
  z-index: 2;
  color: #fff;
  font-size: 11px;
  font-weight: bold;
  padding: 2px 7px;
  border-radius: 20px;
  line-height: 1.4;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.2);
}
.p-badge-new {
  top: 6px;
  background: #e1251b;
}
.p-badge-seckill {
  top: 6px;
  background: #ff7a18;
}
/* When both badges are present, drop the second one below the first. */
.p-badge-new ~ .p-badge-seckill {
  top: 30px;
}
.p-name {
  font-size: 13px;
  line-height: 18px;
  padding: 4px 6px 0;
  height: 36px;
}
.p-shop {
  font-size: 11px;
  color: #e1251b;
  padding: 0 6px;
}
.p-bottom {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 2px 6px;
}
.p-bottom .price {
  font-size: 16px;
}
/* Price trend tags (价格趋势图标) — small inline tags next to the price */
.price-wrap {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}
.trend-tag {
  display: inline-flex;
  align-items: center;
  font-size: 10px;
  font-weight: 600;
  line-height: 1;
  padding: 2px 5px;
  border-radius: 4px;
  white-space: nowrap;
}
.trend-drop {
  color: #07c160;
  background: #f0fff4;
  border: 1px solid #b7eb8f;
}
.trend-seckill {
  color: #ff7a18;
  background: #fff7e6;
  border: 1px solid #ffd591;
}
.sales {
  font-size: 11px;
  color: #999;
}
.loading {
  text-align: center;
  padding: 20px;
}

/* Flash sale countdown banner */
.flash-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 0 8px 8px;
  padding: 14px 16px;
  border-radius: 10px;
  background: linear-gradient(90deg, #e1251b 0%, #ff4d4f 60%, #ff7a45 100%);
  color: #fff;
  box-shadow: 0 4px 12px rgba(225, 37, 27, 0.3);
  overflow: hidden;
  position: relative;
}
.flash-banner::after {
  content: '';
  position: absolute;
  inset: 0;
  background: repeating-linear-gradient(
    45deg,
    rgba(255, 255, 255, 0.06) 0,
    rgba(255, 255, 255, 0.06) 12px,
    transparent 12px,
    transparent 24px
  );
  pointer-events: none;
}
.flash-banner.live {
  background: linear-gradient(90deg, #ff4d4f 0%, #e1251b 100%);
}
.fb-text {
  display: flex;
  align-items: baseline;
  gap: 6px;
  position: relative;
  z-index: 1;
}
.fb-icon {
  font-size: 20px;
}
.fb-title {
  font-size: 18px;
  font-weight: bold;
  letter-spacing: 1px;
}
.fb-sub {
  font-size: 12px;
  opacity: 0.92;
  margin-left: 4px;
}
.fb-countdown {
  display: flex;
  align-items: center;
  gap: 4px;
  position: relative;
  z-index: 1;
}
.fb-block {
  display: inline-block;
  min-width: 34px;
  text-align: center;
  padding: 6px 4px;
  background: rgba(0, 0, 0, 0.28);
  border-radius: 6px;
  font-family: 'Courier New', Consolas, monospace;
  font-size: 22px;
  font-weight: bold;
  line-height: 1;
}
.fb-colon {
  font-family: 'Courier New', Consolas, monospace;
  font-size: 22px;
  font-weight: bold;
}
.fb-live-tag {
  font-size: 16px;
  font-weight: bold;
  letter-spacing: 2px;
  background: #fff;
  color: #e1251b;
  padding: 6px 14px;
  border-radius: 20px;
  position: relative;
  z-index: 1;
}

/* Auto-scroll promo marquee (首页跑马灯) */
.marquee-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 8px 8px;
  padding: 8px 12px;
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  cursor: pointer;
}
.mb-icon { font-size: 16px; flex-shrink: 0; }
.mb-viewport {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
}
.mb-track {
  display: inline-block;
  padding-left: 100%;
  font-size: 13px;
  color: #e1251b;
  font-weight: 500;
  animation: marquee-scroll 16s linear infinite;
}
@keyframes marquee-scroll {
  0% { transform: translateX(0); }
  100% { transform: translateX(-100%); }
}

/* ---- Skeleton loading (首页骨架屏) ---- */
.skel {
  background: #ececec;
  border-radius: 6px;
  animation: skeleton-pulse 1.4s ease-in-out infinite;
}
@keyframes skeleton-pulse {
  0% { background: #ececec; }
  50% { background: #f5f5f5; }
  100% { background: #ececec; }
}
/* Skeleton seckill card pieces */
.skel-seckill-img {
  width: 90px;
  height: 90px;
  margin: 0 auto;
}
.skel-seckill-price {
  width: 60px;
  height: 16px;
  margin: 8px auto 0;
}
.skel-seckill-origin {
  width: 44px;
  height: 12px;
  margin: 6px auto 0;
}
/* Skeleton product card pieces */
.skel-product-img {
  width: 100%;
  height: 170px;
  border-radius: 6px;
}
.skel-product-line {
  height: 14px;
  margin: 8px 6px 0;
}
.skel-product-line.short {
  width: 50%;
}
/* Fade real content in once loaded so it doesn't pop in abruptly. */
.content-fade-in {
  animation: content-fade-in 0.4s ease-out;
}
@keyframes content-fade-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ---- Product comparison floating widget (比价悬浮球) ---- */
.price-ball {
  position: fixed;
  right: 14px;
  bottom: 70px; /* sit just above the tabbar */
  z-index: 200;
  display: flex;
  align-items: center;
  gap: 6px;
  width: 48px;
  height: 48px;
  padding: 0 10px 0 8px;
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.18);
  overflow: hidden;
  cursor: pointer;
  transition: width 0.25s ease, box-shadow 0.2s ease;
  user-select: none;
}
.price-ball:hover {
  width: 168px;
  box-shadow: 0 6px 18px rgba(225, 37, 27, 0.28);
}
.pb-thumb {
  flex-shrink: 0;
  border: 2px solid #ffe3e0;
}
.pb-badge {
  position: absolute;
  top: -6px;
  left: 30px;
  z-index: 3;
  font-size: 9px;
  font-weight: bold;
  color: #fff;
  padding: 1px 5px;
  border-radius: 8px;
  white-space: nowrap;
  line-height: 1.5;
}
.pb-badge.up { background: #e1251b; }
.pb-badge.down { background: #07c160; }
.pb-info {
  display: none;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
}
.price-ball:hover .pb-info { display: flex; }
.pb-name {
  font-size: 11px;
  color: #333;
  max-width: 96px;
}
.pb-price {
  font-size: 13px;
  font-weight: bold;
  color: #e1251b;
}
.ball-pop-enter-active, .ball-pop-leave-active { transition: opacity 0.25s, transform 0.25s; }
.ball-pop-enter-from, .ball-pop-leave-to { opacity: 0; transform: scale(0.4) translateY(10px); }
</style>
