<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getRefunds } from '../api'

const router = useRouter()
const refunds = ref([])
const loading = ref(true)

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
  return { refund_only: '仅退款', return_refund: '退货退款' }[t] || '退款'
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
</style>
