<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getOrders, payOrder, createRefund, confirmOrder, cancelOrder } from '../api'

const router = useRouter()
const orders = ref([])
const loading = ref(true)

// Order timeout countdown: pending orders expire 30 minutes after creation.
const TIMEOUT_MS = 30 * 60 * 1000
// Reactive "now" tick; updated every second to drive the countdowns.
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

onMounted(async () => {
  try {
    orders.value = await getOrders()
    if (orders.value.some((o) => o.status === 'pending')) {
      startTimer()
    }
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})
onUnmounted(stopTimer)

// Calculate remaining milliseconds for a pending order.
function remainingMs(o) {
  const created = new Date(o.created_at).getTime()
  if (Number.isNaN(created)) return 0
  const deadline = created + TIMEOUT_MS
  return Math.max(0, deadline - now.value)
}
// Format the remaining time as MM:SS, or "已超时" when expired.
function countdownText(o) {
  const ms = remainingMs(o)
  if (ms <= 0) return '已超时'
  const totalSec = Math.floor(ms / 1000)
  const m = String(Math.floor(totalSec / 60)).padStart(2, '0')
  const s = String(totalSec % 60).padStart(2, '0')
  return `剩余 ${m}:${s}`
}

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
async function cancel(o) {
  try {
    await cancelOrder(o.id)
    o.status = 'cancelled'
    showSuccessToast('订单已取消')
  } catch (e) {
    showToast(e.response?.data?.error || '操作失败')
  }
}
function viewLogistics(o) {
  router.push({ name: 'logistics', query: { order_id: o.id } })
}
async function applyRefund(o) {
  const type = pickRefundType()
  if (!type) return
  const promptLabel = type === 'exchange' ? '请输入换货原因' : '请输入退款原因'
  const reason = window.prompt(promptLabel)
  if (!reason || !reason.trim()) return
  try {
    await createRefund(o.id, reason, type)
    showSuccessToast(type === 'exchange' ? '换货申请已提交' : '退款申请已提交')
  } catch (e) {
    showToast(e.response?.data?.error || '申请失败')
  }
}

// Map a numeric choice (1/2/3) from the user to an after-sale type key.
// Returns '' if cancelled or the input is invalid.
function pickRefundType() {
  const choice = window.prompt('请选择售后类型，输入数字：\n1 仅退款\n2 退货退款\n3 换货')
  if (choice === null) return ''
  const map = { '1': 'refund_only', '2': 'return_refund', '3': 'exchange' }
  const key = map[String(choice).trim()]
  if (!key) showToast('已取消或选项无效')
  return key || ''
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

// Resolve a grouping key for an order line. Prefer an explicit `shop` field
// when the backend stores one; otherwise fall back to the brand/name prefix
// (first whitespace-delimited token of the product name, e.g. "Apple",
// "华为", "美的"). Returns '' for empty items.
function shopKey(it) {
  if (!it) return ''
  if (it.shop && String(it.shop).trim()) return String(it.shop).trim()
  const name = String(it.name || '').trim()
  if (!name) return ''
  return name.split(/\s+/)[0]
}

// Group an order's parsed items into "packages", one per shop/brand.
// Returns an array of { key, items } preserving first-seen order.
function groupPackages(items) {
  const groups = []
  const idx = new Map()
  for (const it of items) {
    const key = shopKey(it) || '默认包裹'
    if (idx.has(key)) {
      groups[idx.get(key)].items.push(it)
    } else {
      idx.set(key, groups.length)
      groups.push({ key, items: [it] })
    }
  }
  return groups
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
          <span class="o-status">
            {{ statusText(o.status) }}
            <span v-if="o.status === 'pending'" class="o-countdown">（{{ countdownText(o) }}）</span>
          </span>
        </div>
        <div class="o-packages">
          <div
            v-for="(pkg, pi) in groupPackages(parseItems(o.items_json))"
            :key="pi"
            class="o-package"
          >
            <div v-if="groupPackages(parseItems(o.items_json)).length > 1" class="o-pkg-head">
              📦 <span class="o-pkg-label">包裹{{ pi + 1 }}</span>
              <span class="o-pkg-shop van-ellipsis">{{ pkg.key }}</span>
            </div>
            <div v-for="(it, i) in pkg.items" :key="i" class="o-item">
              <van-image width="60" height="60" radius="6" :src="it.image" fit="cover" />
              <div class="oi-info">
                <div class="oi-name van-ellipsis">{{ it.name }}</div>
                <div class="oi-price">¥{{ fmt(it.price) }} × {{ it.quantity }}</div>
              </div>
            </div>
          </div>
        </div>
        <div class="o-foot">
          <span>共 {{ parseItems(o.items_json).length }} 件 合计: <b class="price">¥{{ fmt(o.total) }}</b></span>
          <div class="o-actions">
            <van-button v-if="o.status === 'pending'" size="small" type="danger" round @click="pay(o)">去支付</van-button>
            <van-button v-if="o.status === 'pending'" size="small" plain round @click="cancel(o)">取消订单</van-button>
            <van-button v-if="['shipped','completed'].includes(o.status)" size="small" type="danger" round @click="confirm(o)">确认收货</van-button>
            <van-button v-if="['paid','shipped','completed'].includes(o.status)" size="small" plain type="danger" round @click="viewLogistics(o)">查看物流</van-button>
            <van-button v-if="['paid','shipped','completed'].includes(o.status)" size="small" plain @click="applyRefund(o)">申请售后</van-button>
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
.o-countdown { color: #e1251b; font-weight: 600; }
.o-item { display: flex; gap: 10px; padding: 6px 0; }
.o-packages { display: flex; flex-direction: column; gap: 10px; }
.o-package { padding: 4px 0; }
.o-package + .o-package { border-top: 1px dashed #eee; padding-top: 10px; }
.o-pkg-head { display: flex; align-items: center; gap: 6px; font-size: 12px; color: #666; margin-bottom: 4px; }
.o-pkg-label { font-weight: 600; color: #333; }
.o-pkg-shop { color: #999; flex: 1; }
.oi-info { flex: 1; }
.oi-name { font-size: 13px; }
.oi-price { color: #999; font-size: 12px; margin-top: 4px; }
.o-foot { display: flex; justify-content: space-between; align-items: center; padding-top: 8px; border-top: 1px solid #f5f5f5; margin-top: 8px; font-size: 13px; }
.o-actions { display: flex; gap: 8px; }
</style>
