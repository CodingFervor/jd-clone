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
function progress(d) {
  return Math.min(100, Math.round((d.joined / d.required) * 100))
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
          <div class="gc-progress">
            <div class="gp-bar"><div class="gp-fill" :style="{ width: progress(d) + '%' }"></div></div>
            <span class="gp-text">{{ d.joined }}/{{ d.required }}人</span>
          </div>
          <div class="gc-bottom">
            <span class="gc-countdown">⏰ {{ fmtRemain(remainMs(d)) }}</span>
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
.gc-progress { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.gp-bar { flex: 1; height: 12px; background: #ffe0e0; border-radius: 6px; overflow: hidden; }
.gp-fill { height: 100%; background: linear-gradient(90deg, #ff9800, #e1251b); transition: width 0.3s; }
.gp-text { font-size: 11px; color: #e1251b; white-space: nowrap; }
.gc-bottom { display: flex; justify-content: space-between; align-items: center; margin-top: auto; }
.gc-countdown { color: #e1251b; font-size: 14px; font-weight: bold; font-variant-numeric: tabular-nums; }
</style>
