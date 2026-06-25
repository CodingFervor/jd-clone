<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getOrders, payOrder, createRefund, confirmOrder } from '../api'

const router = useRouter()
const orders = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    orders.value = await getOrders()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})

async function pay(o) {
  // Navigate to the sandbox payment cashier.
  router.push({ name: 'pay', query: { order_id: o.id } })
}
async function confirm(o) {
  try {
    await confirmOrder(o.id)
    o.status = 'completed'
    showSuccessToast('确认收货成功')
  } catch (e) {
    showToast(e.response?.data?.error || '操作失败')
  }
}
function viewLogistics(o) {
  router.push({ name: 'logistics', query: { order_id: o.id } })
}
async function applyRefund(o) {
  const reason = window.prompt('请输入退款原因')
  if (!reason || !reason.trim()) return
  try {
    await createRefund(o.id, reason, 'refund_only')
    showSuccessToast('退款申请已提交')
  } catch (e) {
    showToast(e.response?.data?.error || '申请失败')
  }
}
function statusText(s) {
  return { pending: '待付款', paid: '已付款', shipped: '已发货', completed: '已完成', cancelled: '已取消' }[s] || s
}
function fmt(n) {
  return Number(n).toFixed(2)
}
function parseItems(json) {
  try {
    return JSON.parse(json)
  } catch {
    return []
  }
}
</script>

<template>
  <div class="orders-page">
    <van-nav-bar title="我的订单" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!orders.length" description="暂无订单" />
    <div v-else class="order-list">
      <div v-for="o in orders" :key="o.id" class="order-card">
        <div class="o-head">
          <span class="o-no">订单号: {{ o.order_no }}</span>
          <span class="o-status">{{ statusText(o.status) }}</span>
        </div>
        <div v-for="(it, i) in parseItems(o.items_json)" :key="i" class="o-item">
          <van-image width="60" height="60" radius="6" :src="it.image" fit="cover" />
          <div class="oi-info">
            <div class="oi-name van-ellipsis">{{ it.name }}</div>
            <div class="oi-price">¥{{ fmt(it.price) }} × {{ it.quantity }}</div>
          </div>
        </div>
        <div class="o-foot">
          <span>共 {{ parseItems(o.items_json).length }} 件 合计: <b class="price">¥{{ fmt(o.total) }}</b></span>
          <div class="o-actions">
            <van-button v-if="o.status === 'pending'" size="small" type="danger" round @click="pay(o)">去支付</van-button>
            <van-button v-if="['shipped','completed'].includes(o.status)" size="small" type="danger" round @click="confirm(o)">确认收货</van-button>
            <van-button v-if="['paid','shipped','completed'].includes(o.status)" size="small" plain type="danger" round @click="viewLogistics(o)">查看物流</van-button>
            <van-button v-if="['paid','shipped','completed'].includes(o.status)" size="small" plain @click="applyRefund(o)">申请退款</van-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.orders-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.order-card { background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.o-head { display: flex; justify-content: space-between; font-size: 12px; color: #999; margin-bottom: 8px; }
.o-status { color: #e1251b; }
.o-item { display: flex; gap: 10px; padding: 6px 0; }
.oi-info { flex: 1; }
.oi-name { font-size: 13px; }
.oi-price { color: #999; font-size: 12px; margin-top: 4px; }
.o-foot { display: flex; justify-content: space-between; align-items: center; padding-top: 8px; border-top: 1px solid #f5f5f5; margin-top: 8px; font-size: 13px; }
.o-actions { display: flex; gap: 8px; }
</style>
