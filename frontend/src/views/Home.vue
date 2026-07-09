<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getSeckill, getProducts, getCategories } from '../api'

const router = useRouter()
const seckill = ref([])
const products = ref([])
const categories = ref([])
const loading = ref(false)

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

function goSearch() {
  router.push('/search')
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

// Split countdown into 3 monospace blocks for styling.
const timeBlocks = computed(() => countdown.value.split(':'))

onUnmounted(() => {
  if (flashTimer) {
    clearInterval(flashTimer)
    flashTimer = null
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

    <!-- Seckill floor -->
    <div class="section">      <div class="section-head" @click="router.push('/seckill')" style="cursor:pointer">
        <span class="jd-red"><van-icon name="clock-o" /> 京东秒杀</span>
        <span class="more">更多 ›</span>
      </div>
      <div class="seckill-scroll">
        <div v-for="p in seckill" :key="p.id" class="seckill-card" @click="goProduct(p.id)">
          <van-image width="90" height="90" radius="6" :src="p.image" fit="cover" />
          <div class="price">¥{{ fmt(p.price) }}</div>
          <div class="origin">¥{{ fmt(p.original_price) }}</div>
        </div>
      </div>
    </div>

    <!-- Product list (waterfall) -->
    <div class="section">
      <div class="section-head"><span>为你推荐</span></div>
      <div class="product-grid">
        <div v-for="p in products" :key="p.id" class="product-card" @click="goProduct(p.id)">
          <van-image width="100%" height="170" :src="p.image" fit="cover" radius="6" />
          <div class="p-name van-multi-ellipsis--l2">{{ p.name }}</div>
          <div class="p-shop">{{ p.shop }}</div>
          <div class="p-bottom">
            <span class="price">¥{{ fmt(p.price) }}</span>
            <span class="sales">{{ p.sales }}人付款</span>
          </div>
        </div>
      </div>
    </div>
    <div v-if="loading" class="loading"><van-loading /></div>
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
</style>
