<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getSeckillDeals, grabSeckill } from '../api'

const router = useRouter()
const deals = ref([])
const loading = ref(true)
const now = ref(Date.now())
let timer = null

// ---- Feature: 限时抢购提醒 (Flash Deal Reminder) ----
// Reminded deal ids are persisted in localStorage as a JSON array under
// 'jd_seckill_reminder'. Toggling the 🔔 button adds/removes a deal id and
// flips the button to a "已设置提醒" state.
const remindedIds = ref([])

function loadReminders() {
  try {
    const raw = localStorage.getItem('jd_seckill_reminder')
    const arr = raw ? JSON.parse(raw) : []
    remindedIds.value = Array.isArray(arr) ? arr.map((x) => Number(x)).filter(Number.isFinite) : []
  } catch (_) {
    remindedIds.value = []
  }
}

function saveReminders() {
  try {
    localStorage.setItem('jd_seckill_reminder', JSON.stringify(remindedIds.value))
  } catch (_) {
    // localStorage may be unavailable; ignore.
  }
}

function isReminded(d) {
  return remindedIds.value.includes(Number(d.id))
}

// Toggle the reminder for a deal. Setting it on surfaces a toast simulating
// the push notification being scheduled; turning it off cancels it.
function toggleReminder(d) {
  const id = Number(d.id)
  if (isReminded(d)) {
    remindedIds.value = remindedIds.value.filter((x) => x !== id)
    saveReminders()
    showToast('已取消提醒')
  } else {
    remindedIds.value = [...remindedIds.value, id]
    saveReminders()
    showSuccessToast('已设置提醒，开始前通知您')
  }
}

// Time remaining (ms) until a deal's start_time. When the deal has already
// started (start in the past), this returns 0 so the badge degrades to the
// "进行中" state instead of showing a negative countdown.
function remainToStartMs(d) {
  if (!d.start_time) return 0
  return Math.max(0, new Date(d.start_time).getTime() - now.value)
}

// A deal is "即将开始" (about to start) when its start is within the next hour
// but has not yet passed. Used to flip the countdown badge to red urgency.
function isStartingSoon(d) {
  const ms = remainToStartMs(d)
  return ms > 0 && ms < 3600000
}

// Whether the deal is still upcoming (has not started yet). When false the
// start-countdown badge is hidden and the existing end-countdown is shown.
function isUpcoming(d) {
  return remainToStartMs(d) > 0
}

onMounted(async () => {
  loadReminders()
  try { deals.value = await getSeckillDeals() } catch (e) { showToast('加载失败') } finally { loading.value = false }
  timer = setInterval(() => { now.value = Date.now() }, 1000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })

function remainMs(d) {
  return Math.max(0, new Date(d.end_time).getTime() - now.value)
}
function fmtRemain(ms) {
  const h = Math.floor(ms / 3600000)
  const m = Math.floor((ms % 3600000) / 60000)
  const s = Math.floor((ms % 60000) / 1000)
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}
function progress(d) {
  if (d.stock <= 0) return 100
  return Math.min(100, Math.round((d.sold / d.stock) * 100))
}
// 库存剩余比例 (剩余可抢 / 总库存)
function stockRatio(d) {
  if (d.stock <= 0) return 0
  return Math.max(0, (d.stock - d.sold) / d.stock)
}
// Feature 2: 紧迫感 — 库存低于 20% 时进度条抖动
function isLowStock(d) {
  return stockRatio(d) < 0.2 && stockRatio(d) > 0
}
async function grab(d) {
  try {
    const res = await grabSeckill(d.id)
    d.sold = res.data.sold
    showSuccessToast('抢购成功！')
  } catch (e) {
    showToast(e.response?.data?.error || '抢购失败')
  }
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="seckill-page">
    <!-- Feature 2: 闪电闪屏覆盖层 (纯 CSS 每 5 秒闪一次) -->
    <div class="lightning-flash" aria-hidden="true"></div>
    <van-nav-bar title="京东秒杀" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!deals.length" description="暂无秒杀活动" />
    <div v-else class="deal-list">
      <div class="banner">⚡ 限时秒杀 · 抢完即止</div>
      <div v-for="d in deals" :key="d.id" class="deal-card">
        <van-image width="110" height="110" radius="8" :src="d.product_image" fit="cover" @click="router.push('/product/' + d.product_id)" />
        <div class="dc-info">
          <div class="dc-name van-multi-ellipsis--l2" @click="router.push('/product/' + d.product_id)">{{ d.product_name }}</div>
          <div class="dc-price-row">
            <span class="dc-sk-price">¥{{ fmt(d.seckill_price) }}</span>
            <span class="dc-origin">¥{{ fmt(d.original_price) }}</span>
          </div>
          <div class="dc-progress">
            <div class="dp-bar" :class="{ 'dp-bar--urgent': isLowStock(d) }"><div class="dp-fill" :style="{ width: progress(d) + '%' }"><span class="dp-shine" aria-hidden="true"></span></div></div>
            <span class="dp-text">已抢{{ progress(d) }}%</span>
          </div>
          <!-- Feature: 限时抢购提醒 — live start countdown badge. Turns red
               and shows "即将开始!" when under 1 hour remains. -->
          <div v-if="isUpcoming(d)" class="dc-start-badge" :class="{ 'starting-soon': isStartingSoon(d) }">
            <span v-if="isStartingSoon(d)" class="dsb-soon">🔥 即将开始!</span>
            <span v-else class="dsb-count">⏳ {{ fmtRemain(remainToStartMs(d)) }}后开始</span>
          </div>
          <div class="dc-bottom">
            <span class="dc-countdown">⏰ {{ fmtRemain(remainMs(d)) }}</span>
            <div class="dc-btns">
              <!-- Feature: 限时抢购提醒 — toggle reminder button -->
              <van-button
                class="remind-btn"
                :class="{ 'remind-btn--on': isReminded(d) }"
                size="small"
                plain
                round
                @click="toggleReminder(d)"
              >{{ isReminded(d) ? '🔔 已设置提醒' : '🔔 提醒我' }}</van-button>
              <van-button class="grab-btn" :class="{ 'grab-btn--pulse': progress(d) < 100 }" size="small" type="danger" round :disabled="progress(d) >= 100" @click="grab(d)">
                {{ progress(d) >= 100 ? '已抢光' : '马上抢' }}
              </van-button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.seckill-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
/* Feature 2: 闪电闪屏覆盖层 — 纯 CSS 每 5 秒闪一次 */
.lightning-flash {
  position: fixed;
  inset: 0;
  background: #fff;
  pointer-events: none;
  opacity: 0;
  z-index: 2000;
  animation: lightning-flash 5s infinite;
}
@keyframes lightning-flash {
  0%, 92%, 100% { opacity: 0; }
  93% { opacity: 0.55; }
  94% { opacity: 0.1; }
  95% { opacity: 0.7; }
  96% { opacity: 0; }
}
.banner { background: linear-gradient(135deg, #e1251b, #f5574d); color: #fff; text-align: center; padding: 14px; font-size: 16px; font-weight: bold; }
.deal-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 10px; padding: 12px; }
.dc-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.dc-name { font-size: 14px; line-height: 20px; flex: 1; }
.dc-price-row { display: flex; align-items: baseline; gap: 8px; margin: 6px 0; }
.dc-sk-price { color: #e1251b; font-size: 20px; font-weight: bold; }
.dc-origin { color: #999; font-size: 12px; text-decoration: line-through; }
.dc-progress { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.dp-bar { flex: 1; height: 12px; background: #ffe0e0; border-radius: 6px; overflow: hidden; }
.dp-fill { height: 100%; background: linear-gradient(90deg, #ff9800, #e1251b); transition: width 0.3s; position: relative; overflow: hidden; }
/* Feature: 库存进度条流光 (Seckill Stock Bar Shimmer) — a moving diagonal
   highlight sweeps across the fill like a loading bar shimmer. */
.dp-shine {
  position: absolute;
  top: 0;
  left: -60%;
  width: 60%;
  height: 100%;
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0) 0%,
    rgba(255, 255, 255, 0.5) 50%,
    rgba(255, 255, 255, 0) 100%
  );
  transform: skewX(-20deg);
  animation: dp-shine-sweep 1.6s linear infinite;
}
@keyframes dp-shine-sweep {
  0% { left: -60%; }
  100% { left: 100%; }
}
.dp-text { font-size: 11px; color: #e1251b; white-space: nowrap; }
.dc-bottom { display: flex; justify-content: space-between; align-items: center; margin-top: auto; }
.dc-countdown { color: #e1251b; font-size: 14px; font-weight: bold; font-variant-numeric: tabular-nums; }
/* Feature: 限时抢购提醒 — start countdown badge + reminder button */
.dc-btns { display: flex; gap: 6px; align-items: center; }
.dc-start-badge {
  align-self: flex-start;
  margin: 4px 0 6px;
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 10px;
  background: #fff7e6;
  color: #ff7a18;
  border: 1px solid #ffd591;
  font-variant-numeric: tabular-nums;
  line-height: 1.4;
}
.dc-start-badge.starting-soon {
  background: linear-gradient(90deg, #e1251b, #ff4d4f);
  color: #fff;
  border-color: #e1251b;
  font-weight: bold;
  animation: dsb-blink 0.8s steps(2, start) infinite;
}
.dsb-soon { white-space: nowrap; }
@keyframes dsb-blink { 50% { opacity: 0.55; } }
.remind-btn { flex-shrink: 0; }
.remind-btn--on {
  color: #e1251b !important;
  border-color: #e1251b !important;
  background: #fff5f5 !important;
}
/* Feature 2: 库存不足时进度条紧迫感抖动 */
.dp-bar--urgent { animation: urgent-shake 0.5s infinite; }
@keyframes urgent-shake {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-2px); }
  40% { transform: translateX(2px); }
  60% { transform: translateX(-1px); }
  80% { transform: translateX(1px); }
}
/* Feature 2: 马上抢按钮脉冲发光 (库存>0时) */
.grab-btn--pulse { animation: btn-pulse 1s infinite; }
@keyframes btn-pulse {
  0% { box-shadow: 0 0 0 0 rgba(225,37,27,0.55); }
  70% { box-shadow: 0 0 0 12px rgba(225,37,27,0); }
  100% { box-shadow: 0 0 0 0 rgba(225,37,27,0); }
}
</style>
