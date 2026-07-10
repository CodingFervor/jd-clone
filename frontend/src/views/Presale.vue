<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getPresales, payPresaleDeposit } from '../api'

const router = useRouter()
const deals = ref([])
const loading = ref(true)
const now = ref(Date.now())
let timer = null

onMounted(async () => {
  try { deals.value = await getPresales() } catch (e) { showToast('加载失败') } finally { loading.value = false }
  timer = setInterval(() => { now.value = Date.now() }, 1000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })

function remainMs(d) {
  return Math.max(0, new Date(d.deposit_end).getTime() - now.value)
}
function fmtRemain(ms) {
  const h = Math.floor(ms / 3600000)
  const m = Math.floor((ms % 3600000) / 60000)
  return `${h}时${m}分`
}
// Feature 4: 定金期倒计时格式 X天X时X分
function fmtDayHourMin(ms) {
  const d = Math.floor(ms / 86400000)
  const h = Math.floor((ms % 86400000) / 3600000)
  const m = Math.floor((ms % 3600000) / 60000)
  return `${d}天${h}时${m}分`
}
// 尾款开始时间格式 X月X日
function fmtBalanceDate(d) {
  const dt = new Date(d.balance_start)
  return `${dt.getMonth() + 1}月${dt.getDate()}日`
}
// Feature 4: 阶段判定 — deposit(定金期) / gap(间隔期) / balance(尾款期) / done(已完成)
const DEPOSIT_WINDOW_DAYS = 3 // demo: 假设定金期为 3 天
function currentStage(d) {
  const t = now.value
  const depEnd = new Date(d.deposit_end).getTime()
  const balStart = new Date(d.balance_start).getTime()
  if (d.status && d.status !== 'active') return 'done'
  if (t >= balStart) return 'balance'
  if (t >= depEnd) return 'gap'
  return 'deposit'
}
// 定金期是否在前 24 小时内 (demo 逻辑: depositStart = depositEnd - 定金期)
function isDoubleDeposit(d) {
  const depEnd = new Date(d.deposit_end).getTime()
  const depStart = depEnd - DEPOSIT_WINDOW_DAYS * 86400000
  const t = now.value
  return t >= depStart && t < depStart + 86400000 && t < depEnd
}
const STAGES = [
  { key: 'deposit', label: '定金期' },
  { key: 'gap', label: '间隔期' },
  { key: 'balance', label: '尾款期' },
  { key: 'done', label: '已完成' }
]
function savePct(d) {
  if (!d.original_price || d.original_price <= d.final_price) return 0
  return Math.round((1 - d.final_price / d.original_price) * 100)
}
async function payDeposit(d) {
  try {
    await payPresaleDeposit(d.id)
    d.sold++
    showSuccessToast('定金支付成功')
  } catch (e) {
    showToast(e.response?.data?.error || '支付失败')
  }
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="ps-page">
    <van-nav-bar title="预售专区" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!deals.length" description="暂无预售活动" />
    <div v-else class="ps-list">
      <div class="banner">🎁 定金预售 · 提前锁定好价</div>
      <div v-for="d in deals" :key="d.id" class="ps-card">
        <van-image width="100" height="100" radius="8" :src="d.product_image" fit="cover" @click="router.push('/product/' + d.product_id)" />
        <div class="pc-info">
          <div class="pc-name van-multi-ellipsis--l2" @click="router.push('/product/' + d.product_id)">{{ d.product_name }}</div>
          <div class="pc-price-row">
            <span class="pc-final">¥{{ fmt(d.final_price) }}</span>
            <span class="pc-origin">¥{{ fmt(d.original_price) }}</span>
            <van-tag type="danger" round v-if="savePct(d) > 0">省{{ savePct(d) }}%</van-tag>
          </div>
          <div class="pc-deposit">定金 ¥{{ fmt(d.deposit) }} · 尾款 ¥{{ fmt(d.balance) }}</div>
          <!-- Feature 4: 阶段指示器 -->
          <div class="pc-stages">
            <template v-for="(stg, idx) in STAGES" :key="stg.key">
              <span class="ps-stage" :class="{ 'ps-active': currentStage(d) === stg.key }">{{ stg.label }}</span>
              <span v-if="idx < STAGES.length - 1" class="ps-arrow">→</span>
            </template>
          </div>
          <div class="pc-bottom">
            <div class="pc-countdown-wrap">
              <span v-if="currentStage(d) === 'deposit'" class="pc-countdown">⏰ 定金还剩 {{ fmtDayHourMin(remainMs(d)) }}</span>
              <span v-else-if="currentStage(d) === 'gap'" class="pc-countdown pc-countdown--wait">尾款开始 {{ fmtBalanceDate(d) }}</span>
              <span v-else-if="currentStage(d) === 'balance'" class="pc-countdown pc-countdown--balance">尾款支付中 · {{ fmtBalanceDate(d) }}</span>
              <span v-else class="pc-countdown pc-countdown--done">活动已结束</span>
              <span v-if="currentStage(d) !== 'deposit' && currentStage(d) !== 'done'" class="pc-balance-date">尾款开始 {{ fmtBalanceDate(d) }}</span>
              <van-tag v-if="isDoubleDeposit(d)" type="warning" round class="pc-double">定金翻倍</van-tag>
            </div>
            <van-button size="small" type="danger" round :disabled="currentStage(d) !== 'deposit'" @click="payDeposit(d)">付定金</van-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ps-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.banner { background: linear-gradient(135deg, #e1251b, #f5574d); color: #fff; text-align: center; padding: 14px; font-size: 16px; font-weight: bold; }
.ps-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 10px; padding: 12px; }
.pc-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.pc-name { font-size: 14px; line-height: 20px; flex: 1; }
.pc-price-row { display: flex; align-items: baseline; gap: 8px; margin: 6px 0; }
.pc-final { color: #e1251b; font-size: 20px; font-weight: bold; }
.pc-origin { color: #999; font-size: 12px; text-decoration: line-through; }
.pc-deposit { font-size: 12px; color: #666; margin-bottom: 6px; }
/* Feature 4: 阶段指示器 */
.pc-stages { display: flex; align-items: center; flex-wrap: wrap; gap: 2px; margin-bottom: 8px; }
.ps-stage { font-size: 10px; padding: 2px 6px; border-radius: 8px; background: #f2f3f5; color: #999; }
.ps-stage.ps-active { background: #e1251b; color: #fff; font-weight: bold; }
.ps-arrow { font-size: 10px; color: #ccc; margin: 0 1px; }
.pc-bottom { display: flex; justify-content: space-between; align-items: center; margin-top: auto; }
.pc-countdown-wrap { display: flex; flex-direction: column; gap: 2px; }
.pc-countdown { color: #e1251b; font-size: 12px; font-weight: bold; }
.pc-countdown--wait { color: #ff9800; }
.pc-countdown--balance { color: #07c160; }
.pc-countdown--done { color: #999; }
.pc-balance-date { font-size: 11px; color: #999; }
.pc-double { margin-top: 2px; align-self: flex-start; }
</style>
