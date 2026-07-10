<script setup>
import { ref, onMounted, onActivated, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCoupons, claimCoupon, getMyCoupons } from '../api'

const router = useRouter()
const coupons = ref([])
const myCoupons = ref([])
const loading = ref(true)
const loggedIn = ref(false)
// Reactive "now" tick for the live expiry countdowns. Updated every second
// only when at least one coupon is expiring within 24h.
const now = ref(Date.now())
let timer = null

function startTimer() {
  stopTimer()
  timer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
}
function stopTimer() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}
// Parse an "YYYY-MM-DD" end_date string to a deadline timestamp (end of that
// day). Returns null when the date can't be parsed.
function endDateMs(endDate) {
  if (!endDate) return null
  // Treat the day as expiring at 23:59:59 of that date.
  const d = new Date(String(endDate) + 'T23:59:59')
  if (Number.isNaN(d.getTime())) return null
  return d.getTime()
}
// Compute the expiry countdown descriptor for a coupon. Returns an object with
// a `kind` (urgent/near/far), color and display text. The urgent label carries
// a live HH:MM:SS countdown; near/far labels show whole days.
function expiryInfo(endDate) {
  const ms = endDateMs(endDate)
  if (ms == null) return { kind: 'far', color: '#999', text: '' }
  const diff = ms - now.value
  if (diff <= 0) return { kind: 'urgent', color: '#e1251b', text: '已过期' }
  const totalSec = Math.floor(diff / 1000)
  const days = Math.floor(totalSec / 86400)
  if (diff < 24 * 60 * 60 * 1000) {
    // Within 24h: live HH:MM:SS countdown.
    const h = String(Math.floor(totalSec / 3600)).padStart(2, '0')
    const m = String(Math.floor((totalSec % 3600) / 60)).padStart(2, '0')
    const s = String(totalSec % 60).padStart(2, '0')
    return { kind: 'urgent', color: '#e1251b', text: `即将过期 ${h}:${m}:${s}` }
  }
  if (days < 3) {
    return { kind: 'near', color: '#ff976a', text: `${days + 1}天后过期` }
  }
  return { kind: 'far', color: '#999', text: `${days + 1}天后过期` }
}
// Whether any displayed coupon is urgent (within 24h). Drives the 1s timer.
function hasUrgent(list) {
  return list.some((c) => expiryInfo(c.end_date).kind === 'urgent')
}
onUnmounted(stopTimer)

async function load() {
  loggedIn.value = !!localStorage.getItem('jd_token')
  loading.value = true
  try {
    coupons.value = await getCoupons()
    if (loggedIn.value) myCoupons.value = await getMyCoupons()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
  // Start the live countdown timer if any displayed coupon is urgent.
  if (hasUrgent(coupons.value) || (myCoupons.value.length && hasUrgent(myCoupons.value.map((mc) => mc.coupon).filter(Boolean)))) {
    startTimer()
  }
  // Auto-claim when arriving via a shared ?claim=<id> link.
  const claimId = router.currentRoute.value.query.claim
  if (claimId) maybeClaimShared(Number(claimId))
}
onMounted(load)
onActivated(load)

// Claim a coupon referenced by a shared link, if still available.
async function maybeClaimShared(id) {
  const c = coupons.value.find((x) => x.id === id)
  if (!c) return
  if (!loggedIn.value) { router.push('/login'); return }
  if (c.is_claimed) return
  try {
    await claimCoupon(id)
    c.is_claimed = true
    myCoupons.value = await getMyCoupons()
    showSuccessToast('已通过分享链接领取')
  } catch (e) {
    showToast(e.response?.data?.error || '领取失败')
  }
}

async function claim(c) {
  if (!loggedIn.value) { router.push('/login'); return }
  try {
    await claimCoupon(c.id)
    c.is_claimed = true
    myCoupons.value = await getMyCoupons()
    showSuccessToast('领取成功')
  } catch (e) {
    showToast(e.response?.data?.error || '领取失败')
  }
}
async function share(c) {
  const link = window.location.origin + '/#/coupons?claim=' + c.id
  try {
    await navigator.clipboard.writeText(link)
    showSuccessToast('分享链接已复制')
  } catch (e) {
    // Fallback for non-secure contexts (e.g. plain HTTP) without clipboard API.
    const ta = document.createElement('textarea')
    ta.value = link
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try { document.execCommand('copy'); showSuccessToast('分享链接已复制') }
    catch { showToast('复制失败，请手动复制') }
    document.body.removeChild(ta)
  }
}
function couponValue(c) {
  if (c.coupon_type === 'discount') return (c.value * 10).toFixed(1) + '折'
  return '¥' + c.value
}
</script>

<template>
  <div class="coupons-page">
    <van-nav-bar title="领券中心" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <div v-else>
      <!-- Available coupons -->
      <div class="section-head">可领取的优惠券</div>
      <div v-for="c in coupons" :key="c.id" class="coupon-card" :class="{ claimed: c.is_claimed }">
        <div class="cc-left">
          <div class="cc-value">{{ couponValue(c) }}</div>
          <div class="cc-threshold" v-if="c.threshold > 0">满{{ c.threshold }}元可用</div>
          <div class="cc-threshold" v-else>无门槛</div>
        </div>
        <div class="cc-right">
          <div class="cc-title">{{ c.title }}</div>
          <div class="cc-date">{{ c.end_date }} 到期</div>
          <div class="cc-countdown" :style="{ color: expiryInfo(c.end_date).color }">{{ expiryInfo(c.end_date).text }}</div>
          <div class="cc-actions">
            <van-button v-if="!c.is_claimed" size="small" type="danger" round @click="claim(c)">立即领取</van-button>
            <van-tag v-else type="success">已领取</van-tag>
            <van-button size="small" plain type="danger" round @click="share(c)">分享</van-button>
          </div>
        </div>
      </div>

      <!-- My coupons -->
      <template v-if="loggedIn && myCoupons.length">
        <div class="section-head" style="margin-top: 16px">我的优惠券</div>
        <div v-for="mc in myCoupons" :key="mc.id" class="coupon-card" :class="{ used: mc.is_used }">
          <div class="cc-left">
            <div class="cc-value">{{ mc.coupon ? couponValue(mc.coupon) : '' }}</div>
            <div class="cc-threshold" v-if="mc.coupon && mc.coupon.threshold > 0">满{{ mc.coupon.threshold }}元可用</div>
          </div>
          <div class="cc-right">
            <div class="cc-title">{{ mc.coupon ? mc.coupon.title : '' }}</div>
            <div class="cc-date">{{ mc.coupon ? mc.coupon.end_date : '' }} 到期</div>
            <div v-if="mc.coupon" class="cc-countdown" :style="{ color: expiryInfo(mc.coupon.end_date).color }">{{ expiryInfo(mc.coupon.end_date).text }}</div>
            <van-tag :type="mc.is_used ? 'default' : 'success'">{{ mc.is_used ? '已使用' : '未使用' }}</van-tag>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.coupons-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.section-head { padding: 12px 16px; font-size: 14px; color: #666; font-weight: bold; }
.coupon-card { display: flex; margin: 8px; border-radius: 8px; overflow: hidden; background: #fff; }
.coupon-card.claimed { opacity: 0.6; }
.coupon-card.used { opacity: 0.5; }
.cc-left { background: linear-gradient(135deg, #e1251b, #ff5577); color: #fff; padding: 16px 12px; text-align: center; width: 100px; display: flex; flex-direction: column; justify-content: center; }
.cc-value { font-size: 22px; font-weight: bold; }
.cc-threshold { font-size: 11px; opacity: 0.9; margin-top: 4px; }
.cc-right { flex: 1; padding: 12px 16px; display: flex; flex-direction: column; justify-content: center; gap: 4px; }
.cc-title { font-size: 14px; font-weight: bold; }
.cc-date { font-size: 12px; color: #999; }
.cc-countdown { font-size: 12px; font-weight: 600; }
.cc-actions { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
</style>
