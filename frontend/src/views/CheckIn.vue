<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCheckInStatus, doCheckIn } from '../api'

const router = useRouter()
const last = ref(null)
const totalPoints = ref(0)
const loading = ref(true)
const checking = ref(false)
const todayChecked = ref(false)

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await getCheckInStatus()
    last.value = res.last
    totalPoints.value = res.total_points || 0
    // Today already checked in?
    const today = new Date().toISOString().slice(0, 10)
    todayChecked.value = !!(last.value && last.value.check_date === today)
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}

async function checkIn() {
  checking.value = true
  try {
    const res = await doCheckIn()
    if (!res.is_new) {
      showToast('今日已签到')
    } else {
      showSuccessToast('签到成功 +' + res.data.points + ' 积分')
    }
    await load()
  } catch (e) {
    showToast('签到失败')
  } finally {
    checking.value = false
  }
}

// Build a 7-day calendar strip highlighting the streak.
const weekDays = ['日', '一', '二', '三', '四', '五', '六']
</script>

<template>
  <div class="checkin-page">
    <van-nav-bar title="每日签到" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <div v-else class="ci-body">
      <div class="points-card">
        <div class="pc-label">我的积分</div>
        <div class="pc-value">{{ totalPoints }}</div>
        <div class="pc-streak" v-if="last">已连续签到 <b>{{ last.streak }}</b> 天</div>
      </div>

      <div class="week-strip">
        <div v-for="(d, i) in weekDays" :key="i" class="day-cell">
          <span class="day-label">{{ d }}</span>
          <div class="day-dot" :class="{ active: last && i < last.streak % 7 }">✓</div>
        </div>
      </div>

      <div class="rules">
        <h3>签到规则</h3>
        <p>· 每天可签到一次，连续签到积分递增（第N天+N积分）</p>
        <p>· 单日积分上限 30 分</p>
        <p>· 中断签到后连续天数将重新计算</p>
      </div>

      <div class="ci-btn">
        <van-button block type="danger" round size="large" :loading="checking" :disabled="todayChecked" @click="checkIn">
          {{ todayChecked ? '今日已签到' : '立即签到' }}
        </van-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.checkin-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.ci-body { padding: 16px; }
.points-card { background: linear-gradient(135deg, #e1251b, #f5574d); border-radius: 12px; padding: 28px 20px; text-align: center; color: #fff; }
.pc-label { font-size: 14px; opacity: 0.9; }
.pc-value { font-size: 42px; font-weight: bold; margin: 8px 0; }
.pc-streak { font-size: 13px; opacity: 0.9; }
.pc-streak b { font-size: 16px; }
.week-strip { display: flex; justify-content: space-between; background: #fff; border-radius: 12px; padding: 16px 12px; margin-top: 16px; }
.day-cell { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.day-label { font-size: 12px; color: #999; }
.day-dot { width: 30px; height: 30px; border-radius: 50%; background: #f5f5f5; display: flex; align-items: center; justify-content: center; color: #ccc; font-size: 14px; }
.day-dot.active { background: #e1251b; color: #fff; }
.rules { background: #fff; border-radius: 12px; padding: 16px; margin-top: 16px; }
.rules h3 { font-size: 15px; margin-bottom: 8px; }
.rules p { font-size: 13px; color: #666; line-height: 22px; }
.ci-btn { margin-top: 28px; }
</style>
