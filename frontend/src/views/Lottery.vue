<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getPrizes, spinLottery } from '../api'

const router = useRouter()
const prizes = ref([])
const points = ref(0)
const loading = ref(true)
const spinning = ref(false)
// Cumulative rotation applied to the wheel (keeps increasing so it always spins forward).
const rotation = ref(0)
const showResult = ref(false)
const result = ref(null)
const history = ref([])

// Each wheel segment spans 60° (360 / 6). We alternate two background colors.
const segmentAngle = computed(() => (prizes.value.length ? 360 / prizes.value.length : 60))

// Build the conic-gradient background and place the prize labels around the wheel.
const wheelStyle = computed(() => {
  const colors = ['#fff5f0', '#ffe0d6']
  const stops = []
  const n = prizes.value.length
  for (let i = 0; i < n; i++) {
    const from = i * (360 / n)
    const to = (i + 1) * (360 / n)
    stops.push(`${colors[i % 2]} ${from}deg ${to}deg`)
  }
  return {
    background: `conic-gradient(${stops.join(',')})`,
    transform: `rotate(${rotation.value}deg)`,
  }
})

// Position each prize label at the center of its segment.
function labelStyle(i) {
  const n = prizes.value.length || 6
  const angle = i * (360 / n) + (360 / n) / 2 // center of segment
  return { transform: `rotate(${angle}deg) translate(50%, 0) rotate(90deg)` }
}

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await getPrizes()
    prizes.value = res.data || []
    points.value = res.points || 0
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}

function checkLogin() {
  if (!localStorage.getItem('jd_token')) {
    showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login'))
    return false
  }
  return true
}

async function spin() {
  if (!checkLogin()) return
  if (spinning.value || !prizes.value.length) return
  if (points.value < 50) {
    showToast('积分不足，签到或兑换可获取积分')
    return
  }
  spinning.value = true
  try {
    const res = await spinLottery()
    const won = res.data
    points.value = res.points || points.value
    // Compute the target rotation so the winning segment lands under the top pointer.
    const idx = prizes.value.findIndex((p) => p.id === won.id)
    const n = prizes.value.length || 6
    const segCenter = idx * (360 / n) + (360 / n) / 2
    // The pointer is at the top (0°). We want segCenter to end at 0°, so rotate
    // by (-segCenter) plus full extra turns so the wheel visibly spins forward.
    const extraTurns = 6
    // Normalize the previous rotation to a 360 base, then add turns + offset.
    const base = Math.floor(rotation.value / 360) * 360
    const target = base + extraTurns * 360 + (360 - segCenter)
    rotation.value = target
    // Wait for the CSS transition to finish before showing the result.
    setTimeout(() => {
      spinning.value = false
      result.value = won
      showResult.value = true
      history.value.unshift(won)
      if (history.value.length > 10) history.value.pop()
    }, 4500)
  } catch (e) {
    spinning.value = false
    showToast(e.response?.data?.error || '抽奖失败')
  }
}
</script>

<template>
  <div class="lottery-page">
    <van-nav-bar title="积分大转盘" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <div v-else class="ly-body">
      <!-- Points banner -->
      <div class="points-banner">
        <div class="pb-label">我的积分</div>
        <div class="pb-value">{{ points }}</div>
      </div>

      <!-- Wheel -->
      <div class="wheel-wrap">
        <!-- Top pointer -->
        <div class="wheel-pointer">▼</div>
        <div class="wheel">
          <div class="wheel-inner" :style="wheelStyle" :class="{ spinning }">
            <div v-for="(p, i) in prizes" :key="p.id" class="seg-label" :style="labelStyle(i)">
              <div class="seg-icon">{{ p.icon }}</div>
              <div class="seg-name">{{ p.name }}</div>
            </div>
          </div>
          <!-- Center hub / spin button -->
          <div class="wheel-center" :class="{ disabled: spinning }" @click="spin">
            <span class="wc-text">{{ spinning ? '抽奖中' : '开始\n抽奖' }}</span>
            <span class="wc-cost">消耗50积分</span>
          </div>
        </div>
      </div>

      <div class="spin-btn">
        <van-button block type="danger" round size="large" :loading="spinning" @click="spin">
          抽奖(消耗50积分)
        </van-button>
      </div>

      <!-- Prize list -->
      <div class="prize-list">
        <div class="pl-head">🎁 奖品池</div>
        <div class="pl-grid">
          <div v-for="p in prizes" :key="p.id" class="pl-item">
            <div class="pl-icon">{{ p.icon }}</div>
            <div class="pl-name">{{ p.name }}</div>
          </div>
        </div>
      </div>

      <!-- Rules -->
      <div class="rules">
        <h3>活动规则</h3>
        <p>· 每次抽奖消耗 50 积分</p>
        <p>· 中奖结果随机，不同奖品概率不同</p>
        <p>· 积分可通过每日签到获取</p>
      </div>

      <!-- History -->
      <div v-if="history.length" class="history">
        <div class="pl-head">📜 抽奖记录</div>
        <div v-for="(h, i) in history" :key="i" class="hist-item">
          <span class="hist-icon">{{ h.icon }}</span>
          <span class="hist-name">{{ h.name }}</span>
        </div>
      </div>
    </div>

    <!-- Result popup -->
    <van-dialog v-model:show="showResult" :show-confirm-button="true" confirm-button-text="太棒了" confirm-button-color="#e1251b" title="抽奖结果">
      <div class="result-box">
        <div class="result-icon">{{ result?.icon }}</div>
        <div class="result-name">{{ result?.name }}</div>
        <div v-if="result?.name === '谢谢参与'" class="result-tip">再接再厉，下次一定中！</div>
        <div v-else class="result-tip">恭喜中奖，奖品已发放~</div>
      </div>
    </van-dialog>
  </div>
</template>

<style scoped>
.lottery-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.ly-body { padding: 16px; }
.points-banner {
  background: linear-gradient(135deg, #e1251b, #f5574d);
  border-radius: 12px;
  padding: 24px 20px;
  text-align: center;
  color: #fff;
}
.pb-label { font-size: 14px; opacity: 0.9; }
.pb-value { font-size: 38px; font-weight: bold; margin-top: 6px; }

.wheel-wrap { display: flex; justify-content: center; position: relative; margin: 32px 0 12px; }
.wheel-pointer {
  position: absolute;
  top: -14px;
  left: 50%;
  transform: translateX(-50%);
  color: #e1251b;
  font-size: 30px;
  z-index: 5;
  filter: drop-shadow(0 2px 2px rgba(0, 0, 0, 0.25));
  line-height: 1;
}
.wheel {
  position: relative;
  width: 300px;
  height: 300px;
  border-radius: 50%;
  border: 8px solid #e1251b;
  box-shadow: 0 6px 18px rgba(225, 37, 27, 0.25);
  overflow: hidden;
}
.wheel-inner {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  transition: transform 4.5s cubic-bezier(0.17, 0.67, 0.12, 0.99);
}
.wheel-inner.spinning { /* transition declared above */ }
.seg-label {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 0;
  height: 0;
  transform-origin: 0 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}
.seg-icon { font-size: 26px; margin-bottom: 2px; }
.seg-name { font-size: 11px; color: #e1251b; font-weight: bold; white-space: nowrap; transform: translate(-44px, -10px); }
.seg-label .seg-icon { transform: translate(-40px, -34px); }

.wheel-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 86px;
  height: 86px;
  border-radius: 50%;
  background: linear-gradient(135deg, #e1251b, #f5574d);
  color: #fff;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 0 5px #fff, 0 4px 12px rgba(0, 0, 0, 0.2);
  cursor: pointer;
  z-index: 4;
  user-select: none;
}
.wheel-center.disabled { cursor: default; opacity: 0.85; }
.wc-text { font-size: 15px; font-weight: bold; line-height: 18px; white-space: pre-line; text-align: center; }
.wc-cost { font-size: 10px; opacity: 0.9; margin-top: 2px; }

.spin-btn { margin: 20px 0 8px; }

.prize-list { background: #fff; border-radius: 12px; padding: 16px; margin-top: 16px; }
.pl-head { font-size: 15px; font-weight: bold; margin-bottom: 12px; }
.pl-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.pl-item { display: flex; flex-direction: column; align-items: center; gap: 4px; }
.pl-icon { font-size: 28px; }
.pl-name { font-size: 12px; color: #666; }

.rules { background: #fff; border-radius: 12px; padding: 16px; margin-top: 16px; }
.rules h3 { font-size: 15px; margin-bottom: 8px; }
.rules p { font-size: 13px; color: #666; line-height: 22px; }

.history { background: #fff; border-radius: 12px; padding: 16px; margin-top: 16px; }
.hist-item { display: flex; align-items: center; gap: 8px; padding: 8px 0; border-top: 1px solid #f5f5f5; font-size: 13px; color: #333; }
.hist-item:first-of-type { border-top: none; }
.hist-icon { font-size: 18px; }

.result-box { text-align: center; padding: 24px 16px 8px; }
.result-icon { font-size: 56px; }
.result-name { font-size: 22px; font-weight: bold; color: #e1251b; margin-top: 8px; }
.result-tip { font-size: 13px; color: #999; margin-top: 6px; }
</style>
