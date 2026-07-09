<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getRefunds } from '../api'

const router = useRouter()
const refunds = ref([])
const loading = ref(true)
const expanded = ref({}) // refund id -> bool (timeline expanded)

onMounted(async () => {
  try { refunds.value = await getRefunds() }
  catch (e) { if (e.response?.status === 401) router.replace('/login') }
  finally { loading.value = false }
})

function statusText(s) {
  return { pending: '审核中', approved: '已通过', rejected: '已拒绝', completed: '已完成' }[s] || s
}
function statusColor(s) {
  return { pending: 'warning', approved: 'primary', rejected: 'danger', completed: 'success' }[s] || 'default'
}
function typeText(t) {
  return { refund_only: '仅退款', return_refund: '退货退款', exchange: '换货' }[t] || '退款'
}
function fmtTime(t) {
  if (!t) return ''
  return String(t).slice(0, 16).replace('T', ' ')
}
function toggleTimeline(id) {
  expanded.value[id] = !expanded.value[id]
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="refunds-page">
    <van-nav-bar title="售后服务" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!refunds.length" description="暂无售后申请" />
    <div v-else>
      <div v-for="r in refunds" :key="r.id" class="refund-card">
        <div class="r-head">
          <span>订单号: {{ r.order_id }}</span>
          <van-tag :type="statusColor(r.status)">{{ statusText(r.status) }}</van-tag>
        </div>
        <div class="r-type">{{ typeText(r.type) }} · ¥{{ fmt(r.amount) }}</div>
        <div class="r-reason">原因: {{ r.reason }}</div>
        <div v-if="r.admin_note" class="r-note">客服回复: {{ r.admin_note }}</div>
        <div v-if="r.tracks && r.tracks.length" class="r-track-toggle" @click="toggleTimeline(r.id)">
          {{ expanded[r.id] ? '收起进度 ▲' : '查看售后进度 ▼' }}
        </div>
        <!-- Timeline -->
        <div v-if="expanded[r.id] && r.tracks && r.tracks.length" class="timeline">
          <div v-for="(t, i) in r.tracks" :key="t.id" class="tl-item">
            <div class="tl-dot" :class="{ done: i < r.tracks.length, current: i === r.tracks.length - 1 }"></div>
            <div class="tl-content">
              <div class="tl-status">{{ statusText(t.status) }}</div>
              <div class="tl-note">{{ t.note }}</div>
              <div class="tl-time">{{ fmtTime(t.created_at) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.refunds-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.refund-card { background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.r-head { display: flex; justify-content: space-between; font-size: 13px; color: #999; }
.r-type { font-size: 16px; font-weight: bold; color: #e1251b; margin: 8px 0; }
.r-reason { font-size: 13px; color: #666; }
.r-note { font-size: 12px; color: #1989fa; margin-top: 6px; padding: 6px; background: #f0f7ff; border-radius: 4px; }
.r-track-toggle { color: #e1251b; font-size: 13px; margin-top: 10px; padding-top: 8px; border-top: 1px solid #f5f5f5; }
.timeline { margin-top: 12px; padding-left: 4px; }
.tl-item { display: flex; gap: 10px; padding-bottom: 16px; position: relative; }
.tl-item:not(:last-child)::before { content: ''; position: absolute; left: 5px; top: 14px; bottom: 0; width: 2px; background: #e1251b; opacity: 0.3; }
.tl-dot { width: 12px; height: 12px; border-radius: 50%; background: #ddd; margin-top: 3px; flex-shrink: 0; }
.tl-dot.done { background: #e1251b; }
.tl-dot.current { background: #e1251b; box-shadow: 0 0 0 4px rgba(225,37,27,0.15); }
.tl-content { flex: 1; }
.tl-status { font-size: 14px; font-weight: bold; color: #333; }
.tl-note { font-size: 12px; color: #666; margin-top: 2px; }
.tl-time { font-size: 11px; color: #999; margin-top: 2px; }
</style>
