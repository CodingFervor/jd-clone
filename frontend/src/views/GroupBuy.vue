<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getGroupBuys, joinGroupBuy } from '../api'

const router = useRouter()
const deals = ref([])
const loading = ref(true)
const now = ref(Date.now())
let timer = null

onMounted(async () => {
  try { deals.value = await getGroupBuys() } catch (e) { showToast('加载失败') } finally { loading.value = false }
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
// ---- Feature: 拼团倒计时圆环 (Group Buy Countdown Ring) ----
// SVG circle countdown. The total duration is derived from the deal's
// start_time → end_time window so the ring depletes proportionally. The ring's
// stroke-dashoffset grows as time runs out; the color shifts green→orange→red
// →gray based on remaining percentage, with HH:MM text in the center.
// Ring geometry constants.
const RING_R = 26          // circle radius
const RING_C = 2 * Math.PI * RING_R // circumference
// Total window for a deal. Falls back to a 24h span if start_time is missing
// so the ring still animates meaningfully.
function totalMs(d) {
  const start = d.start_time ? new Date(d.start_time).getTime() : (new Date(d.end_time).getTime() - 24 * 3600 * 1000)
  const end = new Date(d.end_time).getTime()
  return Math.max(1, end - start)
}
// Remaining fraction 0..1 (1 = just started, 0 = expired).
function remainFraction(d) {
  const ms = remainMs(d)
  const total = totalMs(d)
  return Math.max(0, Math.min(1, ms / total))
}
// stroke-dashoffset: 0 when full, circumference when empty. The ring depletes
// as time runs out (offset grows from 0 → C).
function ringOffset(d) {
  const frac = remainFraction(d)
  return RING_C * (1 - frac)
}
// Color by remaining percentage: green(>50%) → orange(<50%) → red(<25%) →
// gray(expired). Spec wants HH:MM center text, but we keep seconds precision
// in fmtRemain for the linear bar; the ring shows HH:MM per the spec.
function ringColor(d) {
  const ms = remainMs(d)
  if (ms <= 0) return '#bbb' // gray (expired)
  const frac = remainFraction(d)
  if (frac > 0.5) return '#07c160' // green
  if (frac > 0.25) return '#ff976a' // orange
  return '#e1251b' // red
}
// Center text in HH:MM format (seconds dropped per the spec).
function ringText(d) {
  const ms = remainMs(d)
  if (ms <= 0) return '已结束'
  const totalMin = Math.floor(ms / 60000)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}
function progress(d) {
  return Math.min(100, Math.round((d.joined / d.required) * 100))
}
// Feature 3: 为每个拼团生成 2-4 个演示成员头像 (dicebear API)
// 用 id 作种子保证同一拼团头像稳定，第一个为团长(金色边框)
function memberAvatars(d) {
  const count = 2 + (Math.abs(d.id) % 3) // 2~4 个
  const seedBase = 'gb-' + d.id + '-'
  const avatars = []
  for (let i = 0; i < count; i++) {
    avatars.push(`https://api.dicebear.com/7.x/adventurer/svg?seed=${seedBase}${i}`)
  }
  return avatars
}
async function join(d) {
  try {
    const res = await joinGroupBuy(d.id)
    d.joined = res.data.joined
    d.status = res.data.status
    showSuccessToast(res.message)
  } catch (e) {
    showToast(e.response?.data?.error || '参团失败')
  }
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="gb-page">
    <van-nav-bar title="超值拼团" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!deals.length" description="暂无拼团活动" />
    <div v-else class="gb-list">
      <div class="banner">👥 超值拼团 · 人多更便宜</div>
      <div v-for="d in deals" :key="d.id" class="gb-card">
        <van-image width="100" height="100" radius="8" :src="d.product_image" fit="cover" @click="router.push('/product/' + d.product_id)" />
        <div class="gc-info">
          <div class="gc-name van-multi-ellipsis--l2" @click="router.push('/product/' + d.product_id)">{{ d.product_name }}</div>
          <div class="gc-price-row">
            <span class="gc-group">¥{{ fmt(d.group_price) }}</span>
            <span class="gc-origin">¥{{ fmt(d.original_price) }}</span>
          </div>
          <!-- Feature 3: 拼团成员头像 -->
          <div class="gc-members">
            <div class="gm-stack">
              <img
                v-for="(a, i) in memberAvatars(d)"
                :key="i"
                :src="a"
                class="gm-avatar"
                :class="{ 'gm-leader': i === 0 }"
                :style="{ zIndex: 10 - i, marginLeft: i === 0 ? '0' : '-8px' }"
                alt="团员头像"
              />
            </div>
            <span class="gm-count">{{ d.joined }}人已参团</span>
          </div>
          <div class="gc-progress">
            <div class="gp-bar"><div class="gp-fill" :style="{ width: progress(d) + '%' }"></div></div>
            <span class="gp-text">{{ d.joined }}/{{ d.required }}人</span>
          </div>
          <div class="gc-bottom">
            <!-- Feature: 拼团倒计时圆环 — SVG circular progress ring with HH:MM center -->
            <div class="countdown-ring" :style="{ '--ring-color': ringColor(d) }">
              <svg width="64" height="64" viewBox="0 0 64 64" class="ring-svg">
                <!-- background track -->
                <circle cx="32" cy="32" :r="RING_R" fill="none" stroke="#f0f0f0" stroke-width="5" />
                <!-- depleting progress arc; stroke-dashoffset animates each second -->
                <circle
                  cx="32" cy="32" :r="RING_R" fill="none"
                  :stroke="ringColor(d)"
                  stroke-width="5"
                  stroke-linecap="round"
                  :stroke-dasharray="RING_C"
                  :stroke-dashoffset="ringOffset(d)"
                  :transform="'rotate(-90 32 32)'"
                  class="ring-progress"
                />
              </svg>
              <span class="ring-text" :style="{ color: ringColor(d) }">{{ ringText(d) }}</span>
            </div>
            <van-button size="small" type="danger" round :disabled="d.status !== 'active'" @click="join(d)">
              {{ d.status === 'success' ? '已成团' : '去拼团' }}
            </van-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gb-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.banner { background: linear-gradient(135deg, #e1251b, #f5574d); color: #fff; text-align: center; padding: 14px; font-size: 16px; font-weight: bold; }
.gb-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 10px; padding: 12px; }
.gc-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.gc-name { font-size: 14px; line-height: 20px; flex: 1; }
.gc-price-row { display: flex; align-items: baseline; gap: 8px; margin: 6px 0; }
.gc-group { color: #e1251b; font-size: 20px; font-weight: bold; }
.gc-origin { color: #999; font-size: 12px; text-decoration: line-through; }
/* Feature 3: 拼团成员头像堆叠 */
.gc-members { display: flex; align-items: center; gap: 8px; margin: 4px 0 6px; }
.gm-stack { display: flex; align-items: center; }
.gm-avatar {
  width: 26px; height: 26px; border-radius: 50%; border: 2px solid #fff;
  object-fit: cover; background: #f5f5f5; box-shadow: 0 1px 3px rgba(0,0,0,0.15);
}
.gm-leader { border-color: #ffc107; box-shadow: 0 0 0 1px #ffd54f, 0 1px 3px rgba(0,0,0,0.15); }
.gm-count { font-size: 11px; color: #e1251b; font-weight: bold; }
.gc-progress { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.gp-bar { flex: 1; height: 12px; background: #ffe0e0; border-radius: 6px; overflow: hidden; }
.gp-fill { height: 100%; background: linear-gradient(90deg, #ff9800, #e1251b); transition: width 0.3s; }
.gp-text { font-size: 11px; color: #e1251b; white-space: nowrap; }
.gc-bottom { display: flex; justify-content: space-between; align-items: center; margin-top: auto; }
.gc-countdown { color: #e1251b; font-size: 14px; font-weight: bold; font-variant-numeric: tabular-nums; }
/* Feature: 拼团倒计时圆环 (Group Buy Countdown Ring) */
.countdown-ring {
  position: relative;
  width: 64px;
  height: 64px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.ring-svg { display: block; }
/* Smooth per-second depletion of the progress arc. */
.ring-progress {
  transition: stroke-dashoffset 1s linear, stroke 0.5s ease;
}
.ring-text {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: bold;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.5px;
}
</style>
