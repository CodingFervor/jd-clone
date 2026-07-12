<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showLoadingToast, showDialog } from 'vant'
import { getOrders, payOrder, createRefund, confirmOrder, cancelOrder, addToCart, getRefunds } from '../api'

const router = useRouter()
const orders = ref([])
const loading = ref(true)
// ---- Feature: 订单退款状态徽章 (Order Refund Status Icon) ----
// The Order model itself has no refund_status field, so we fetch the user's
// refund list (best-effort) and build a map order_id -> refund status. A
// completed refund renders a green "✓已退款" badge; a pending/approved refund
// renders an orange "⏳退款中" badge. Rejected/unknown refunds render nothing.
const refundByOrder = ref({})
// Returns 'completed' | 'pending' | '' for an order's refund (if any).
// 'pending' here also covers 'approved' (still being processed/refunded).
function refundStateFor(o) {
  const r = refundByOrder.value[o.id]
  if (!r) return ''
  if (r === 'completed') return 'completed'
  // pending (审核中) and approved (已通过, refund being processed) → 退款中.
  if (r === 'pending' || r === 'approved') return 'pending'
  return ''
}
// Expanded order lifecycle timelines: a Set of order IDs whose timeline is shown.
// We reassign a new Set on toggle so Vue reliably re-renders.
const expandedOrders = ref(new Set())
// Numeric rank for each order status, used to decide which lifecycle stages
// are already reached. 'cancelled' is ranked below 'pending' so only the
// creation stage lights up for it.
const STATUS_RANK = { pending: 0, paid: 1, shipped: 2, completed: 3, cancelled: -1 }

// ---- Status filter tabs (订单按状态分组) ----
// Tabs: 全部 | 待付款 | 待发货 | 待收货 | 已完成. Each tab maps to one or more
// underlying order statuses. `activeStatus` drives the real-time filter.
const STATUS_TABS = [
  { key: 'all', label: '全部', statuses: null },
  { key: 'pending', label: '待付款', statuses: ['pending'] },
  { key: 'paid', label: '待发货', statuses: ['paid'] },
  { key: 'shipped', label: '待收货', statuses: ['shipped'] },
  { key: 'completed', label: '已完成', statuses: ['completed'] },
]
const activeStatus = ref('all')
// Count per tab, computed live from the orders list. 'all' excludes cancelled
// orders so the "全部" tab reflects actionable orders (mirrors typical JD UX).
const statusCounts = computed(() => {
  const counts = {}
  for (const t of STATUS_TABS) {
    if (t.statuses === null) {
      counts[t.key] = orders.value.filter((o) => o.status !== 'cancelled').length
    } else {
      counts[t.key] = orders.value.filter((o) => t.statuses.includes(o.status)).length
    }
  }
  return counts
})
// Orders filtered by the active tab, in real time.
const filteredOrders = computed(() => {
  const tab = STATUS_TABS.find((t) => t.key === activeStatus.value)
  if (!tab || tab.statuses === null) {
    return orders.value.filter((o) => o.status !== 'cancelled')
  }
  return orders.value.filter((o) => tab.statuses.includes(o.status))
})
function setStatus(key) {
  activeStatus.value = key
}

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
    // Best-effort load of the refund list to power the refund-status badges.
    // A failure (e.g. no refunds yet) just leaves the map empty.
    try {
      const refunds = await getRefunds()
      const map = {}
      for (const r of refunds || []) {
        // Keep the most-recent status if there are multiple refunds per order.
        map[r.order_id] = r.status
      }
      refundByOrder.value = map
    } catch (_) { /* ignore */ }
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
// One-click repurchase: re-add every item from a completed order to the cart
// and route the user to /cart. Missing products (deleted/discontinued) are
// skipped and reported so the repurchase is best-effort, not all-or-nothing.
async function repurchase(o) {
  if (!localStorage.getItem('jd_token')) {
    showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login'))
    return
  }
  const items = parseItems(o.items_json)
  const targets = items.filter((it) => it.product_id)
  if (!targets.length) {
    showToast('无可回购商品')
    return
  }
  showLoadingToast({ message: '正在加入购物车...', forbidClick: true, duration: 0 })
  let ok = 0
  let fail = 0
  await Promise.all(
    targets.map((it) =>
      addToCart(it.product_id, it.quantity).then(() => { ok++ }).catch(() => { fail++ })
    )
  )
  if (ok && !fail) {
    showSuccessToast('已加入购物车')
  } else if (ok && fail) {
    showSuccessToast(`已加入 ${ok} 件，${fail} 件商品已失效`)
  } else {
    showToast('商品已失效，无法购买')
    return
  }
  router.push('/cart')
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
// Order export (订单导出): build a text summary of every order and copy it to
// the clipboard. Each order's line items are joined into one "商品" cell.
async function exportOrders() {
  if (!orders.value.length) {
    showToast('暂无订单可导出')
    return
  }
  const header = '订单号 | 金额 | 状态 | 商品'
  const lines = [header]
  for (const o of orders.value) {
    const items = parseItems(o.items_json)
    const goods = items.length
      ? items.map((it) => `${it.name} x${it.quantity}`).join(', ')
      : '-'
    lines.push(`${o.order_no} | ¥${fmt(o.total)} | ${statusText(o.status)} | ${goods}`)
  }
  const text = lines.join('\n')
  try {
    await navigator.clipboard.writeText(text)
    showSuccessToast('订单已导出并复制到剪贴板')
  } catch (e) {
    // Fallback for non-secure contexts without the async clipboard API.
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      showSuccessToast('订单已导出并复制到剪贴板')
    } catch {
      showToast('复制失败，请手动复制')
    }
    document.body.removeChild(ta)
  }
}

// ---- Feature: 订单物流迷你跟踪 (Order Express Tracking Mini) ----
// For shipped orders we show a compact express-tracking row: a mini logo emoji
// (📦 / 🚚 / ✈️, chosen deterministically by order id so it's stable) and a
// truncated waybill number "运单: JD****1234". The full number is derived from
// the order id so the same order always shows the same waybill; the middle is
// masked to mimic a real carrier's truncated tracking number.
const EXPRESS_LOGOS = ['📦', '🚚', '✈️']
function expressLogo(o) {
  return EXPRESS_LOGOS[(Number(o.id) || 0) % EXPRESS_LOGOS.length]
}
// Build a deterministic 12-digit waybill and mask the middle: JD****<last4>.
function expressWaybill(o) {
  let h = 0
  const s = 'JD' + String(o.id || '') + String(o.order_no || '')
  for (let i = 0; i < s.length; i++) {
    h = (h << 5) - h + s.charCodeAt(i)
    h |= 0
  }
  const num = String(Math.abs(h)).slice(0, 10).padEnd(10, '0')
  const last4 = num.slice(-4)
  return `JD****${last4}`
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

// Toggle the expanded/collapsed state of an order's lifecycle timeline.
// A new Set is created so Vue detects the reactive change reliably.
function toggleExpand(o) {
  const next = new Set(expandedOrders.value)
  if (next.has(o.id)) {
    next.delete(o.id)
  } else {
    next.add(o.id)
  }
  expandedOrders.value = next
}
// Build the lifecycle stages for an order. The first stage (下单成功) always
// shows its created_at timestamp; later stages show "已完成" because we don't
// track per-stage timestamps. The highest reached stage is highlighted as the
// current (red) stage.
function orderStages(o) {
  const rank = STATUS_RANK[o.status] ?? 0
  // Format created_at as "YYYY-MM-DD HH:mm".
  const fmtTime = (raw) => {
    const d = new Date(raw)
    if (Number.isNaN(d.getTime())) return '已完成'
    const p = (n) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
  }
  const stages = [
    { icon: '📝', label: '下单成功', threshold: 0, time: fmtTime(o.created_at) },
    { icon: '💰', label: '已付款', threshold: 1, time: '已完成' },
    { icon: '📦', label: '已发货', threshold: 2, time: '已完成' },
    { icon: '✅', label: '已完成', threshold: 3, time: '已完成' },
  ]
  // The current stage is the highest threshold not exceeding the status rank.
  const currentThreshold = stages.map((s) => s.threshold).filter((t) => t <= rank).pop()
  return stages.map((s) => ({
    icon: s.icon,
    label: s.label,
    time: s.time,
    done: rank >= s.threshold,
    isCurrent: s.threshold === currentThreshold,
  }))
}

// ---- Feature: 评价提醒 (Order Review Reminder) ----
// Nudge users to review completed orders. For each completed order we check
// whether ALL of its items have been reviewed: an order is "reviewed" if every
// product_id in its line items appears in the localStorage reviewed-products
// set (populated when a review is submitted from ProductDetail). Reminders that
// the user dismisses with "稍后" are tracked per-order in jd_review_dismissed.
function reviewedProductIds() {
  try {
    return new Set(JSON.parse(localStorage.getItem('jd_reviewed_products') || '[]'))
  } catch {
    return new Set()
  }
}
function dismissedOrderIds() {
  try {
    return new Set(JSON.parse(localStorage.getItem('jd_review_dismissed') || '[]'))
  } catch {
    return new Set()
  }
}
// An order needs a review reminder when: it's completed, not dismissed, and at
// least one of its items has not been reviewed yet.
function needsReview(o) {
  if (o.status !== 'completed') return false
  if (dismissedOrderIds().has(o.id)) return false
  const items = parseItems(o.items_json)
  if (!items.length) return false
  const reviewed = reviewedProductIds()
  // Has at least one item with a product_id that hasn't been reviewed.
  return items.some((it) => it.product_id && !reviewed.has(it.product_id))
}
// Dismiss the reminder for a specific order (the "稍后" button).
function dismissReview(o) {
  const set = dismissedOrderIds()
  set.add(o.id)
  localStorage.setItem('jd_review_dismissed', JSON.stringify([...set]))
  // Trigger reactivity by bumping a refresh counter.
  reviewTick.value++
}
// Go review the first un-reviewed product in this order.
function goReview(o) {
  const items = parseItems(o.items_json)
  const reviewed = reviewedProductIds()
  const target = items.find((it) => it.product_id && !reviewed.has(it.product_id))
  if (target && target.product_id) {
    router.push('/product/' + target.product_id)
  } else if (items[0] && items[0].product_id) {
    router.push('/product/' + items[0].product_id)
  }
}
// A reactive tick so dismissing a reminder re-evaluates needsReview without
// needing to mutate the order objects themselves.
const reviewTick = ref(0)
// When a review is confirmed from the Orders page, record it locally so the
// reminder disappears immediately (the ProductDetail page also records it, but
// recording here too covers the case where the user navigated back directly).
function markReviewedFromOrder(o) {
  const items = parseItems(o.items_json)
  const set = reviewedProductIds()
  let changed = false
  for (const it of items) {
    if (it.product_id && !set.has(it.product_id)) {
      set.add(it.product_id)
      changed = true
    }
  }
  if (changed) {
    localStorage.setItem('jd_reviewed_products', JSON.stringify([...set]))
    reviewTick.value++
  }
}

// ---- Feature: 下载发票 (Order Invoice Download) ----
// Build a text-based invoice summary for a completed order and copy it to the
// clipboard. The invoice carries a deterministic invoice number derived from
// the order id, the buyer's username, the line items, totals and an issue date.
function invoiceNo(o) {
  // Deterministic 8-digit invoice number from the order id.
  let h = 0
  const s = String(o.id || '')
  for (let i = 0; i < s.length; i++) {
    h = (h << 5) - h + s.charCodeAt(i)
    h |= 0
  }
  return ('E' + (Math.abs(h) % 100000000)).padStart(9, '0')
}

function todayStr() {
  const d = new Date()
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

function buyerName() {
  try {
    const u = JSON.parse(localStorage.getItem('jd_user') || '{}')
    return (u && (u.nickname || u.username)) || '京东用户'
  } catch {
    return '京东用户'
  }
}

function buildInvoice(o) {
  const items = parseItems(o.items_json)
  const lines = items.map(
    (it) => `  ${it.name}  ${it.quantity} x ¥${fmt(it.price)} = ¥${fmt(it.price * it.quantity)}`
  )
  return [
    '================================',
    '         京东电子发票(普通发票)',
    '================================',
    `发票号码: ${invoiceNo(o)}`,
    `开票日期: ${todayStr()}`,
    `购方名称: ${buyerName()}`,
    `订单编号: ${o.order_no}`,
    '--------------------------------',
    '商品明细:',
    ...lines,
    '--------------------------------',
    `价税合计: ¥${fmt(o.total)}`,
    `开票状态: 已开具`,
    '================================',
    '本发票为电子发票，与纸质发票具有同等法律效力。',
  ].join('\n')
}

async function downloadInvoice(o) {
  const text = buildInvoice(o)
  try {
    await navigator.clipboard.writeText(text)
    showSuccessToast('发票已生成并复制到剪贴板')
  } catch (e) {
    // Fallback for non-secure contexts / older browsers.
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      showSuccessToast('发票已生成并复制到剪贴板')
    } catch {
      showToast('复制失败，请手动复制')
    }
    document.body.removeChild(ta)
  }
}
</script>

<template>
  <div class="orders-page">
    <van-nav-bar title="我的订单" left-arrow @click-left="router.back()" fixed placeholder>
      <template #right>
        <span class="export-btn" @click="exportOrders">📋 导出</span>
      </template>
    </van-nav-bar>
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!orders.length" description="暂无订单" />
    <div v-else>
      <!-- Status filter tabs (订单按状态分组) -->
      <div class="status-tabs">
        <span
          v-for="t in STATUS_TABS"
          :key="t.key"
          class="status-tab"
          :class="{ active: activeStatus === t.key }"
          @click="setStatus(t.key)"
        >
          {{ t.label }}
          <span v-if="statusCounts[t.key]" class="status-count">{{ statusCounts[t.key] }}</span>
        </span>
      </div>

      <div v-if="!filteredOrders.length" class="filter-empty">
        <van-empty :description="`暂无${STATUS_TABS.find((t) => t.key === activeStatus).label}订单`" />
      </div>
      <transition-group v-else name="order-fade" tag="div" class="order-list">
        <div v-for="o in filteredOrders" :key="o.id" class="order-card">
        <div class="o-head">
          <span class="o-no">订单号: {{ o.order_no }}</span>
          <span class="o-status">
            {{ statusText(o.status) }}
            <span v-if="o.status === 'pending'" class="o-countdown">（{{ countdownText(o) }}）</span>
          </span>
        </div>
        <!-- Feature: 订单退款状态徽章 (Order Refund Status Icon) — green "✓已退款"
             for completed refunds, orange "⏳退款中" for pending/approved refunds. -->
        <div v-if="refundStateFor(o)" class="refund-badge-row">
          <span v-if="refundStateFor(o) === 'completed'" class="refund-badge refund-done">✓已退款</span>
          <span v-else class="refund-badge refund-pending">⏳退款中</span>
        </div>
        <!-- Feature: 评价提醒 — pulsing badge nudging review of unreviewed completed orders -->
        <div v-if="needsReview(o)" :key="'rv-' + o.id + '-' + reviewTick" class="review-reminder">
          <div class="rr-badge">
            <span class="rr-icon">📝 写评价赚积分</span>
            <span class="rr-incentive">+5积分</span>
          </div>
          <div class="rr-actions">
            <van-button size="mini" type="danger" round @click="goReview(o)">去评价</van-button>
            <van-button size="mini" plain round @click="dismissReview(o)">稍后</van-button>
          </div>
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
        <!-- Feature: 订单物流迷你跟踪 — for shipped orders show a mini express
             logo + truncated waybill number "运单: JD****1234". -->
        <div v-if="o.status === 'shipped'" class="express-mini">
          <span class="em-logo">{{ expressLogo(o) }}</span>
          <span class="em-info">
            <span class="em-label">运输中</span>
            <span class="em-waybill">运单: {{ expressWaybill(o) }}</span>
          </span>
          <van-icon name="logistics" class="em-icon" />
        </div>
        <!-- Feature: 订单时间轴迷你图 (Order Timeline Mini Map) — a tiny 4-step
             horizontal progress bar (下单→付款→发货→收货) with the current step
             highlighted. Shown on every order card; cancelled orders freeze at 0. -->
        <div class="o-mini-timeline">
          <div
            v-for="(st, si) in orderStages(o)"
            :key="'mt-' + si"
            class="mt-step"
            :class="{ 'is-done': st.done, 'is-current': st.isCurrent }"
          >
            <div class="mt-node">
              <span class="mt-node-circle">{{ st.done ? '✓' : (si + 1) }}</span>
              <span v-if="si < orderStages(o).length - 1" class="mt-node-line"></span>
            </div>
            <span class="mt-label">{{ ['下单','付款','发货','收货'][si] }}</span>
          </div>
        </div>
        <div class="o-foot">
          <span>共 {{ parseItems(o.items_json).length }} 件 合计: <b class="price">¥{{ fmt(o.total) }}</b></span>
          <div class="o-actions">
            <van-button v-if="o.status === 'pending'" size="small" type="danger" round @click="pay(o)">去支付</van-button>
            <van-button v-if="o.status === 'pending'" size="small" plain round @click="cancel(o)">取消订单</van-button>
            <van-button v-if="o.status === 'completed'" size="small" type="danger" round @click="repurchase(o)">再次购买</van-button>
            <van-button v-if="o.status === 'completed'" size="small" plain round @click="downloadInvoice(o)">📥下载发票</van-button>
            <van-button v-if="['shipped','completed'].includes(o.status)" size="small" type="danger" round @click="confirm(o)">确认收货</van-button>
            <van-button v-if="['paid','shipped','completed'].includes(o.status)" size="small" plain type="danger" round @click="viewLogistics(o)">查看物流</van-button>
            <van-button v-if="['paid','shipped','completed'].includes(o.status)" size="small" plain @click="applyRefund(o)">申请售后</van-button>
            <van-button size="small" plain round @click="toggleExpand(o)">{{ expandedOrders.has(o.id) ? '收起进度' : '查看进度' }}</van-button>
          </div>
        </div>
        <div v-if="expandedOrders.has(o.id)" class="o-timeline">
          <div
            v-for="(st, si) in orderStages(o)"
            :key="si"
            class="ot-item"
            :class="{ 'is-current': st.isCurrent, 'is-done': st.done && !st.isCurrent }"
          >
            <div class="ot-dot">{{ st.icon }}</div>
            <div v-if="si < orderStages(o).length - 1" class="ot-line"></div>
            <div class="ot-body">
              <div class="ot-label">{{ st.label }}</div>
              <div class="ot-time">{{ st.time }}</div>
            </div>
          </div>
        </div>
      </div>
      </transition-group>
    </div>
  </div>
</template>

<style scoped>
.orders-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.export-btn { font-size: 14px; color: #e1251b; font-weight: 500; }
/* Status filter tabs (订单按状态分组) */
.status-tabs {
  display: flex;
  background: #fff;
  margin: 8px;
  border-radius: 8px;
  padding: 4px 4px 0;
  position: sticky;
  top: 46px;
  z-index: 10;
}
.status-tab {
  flex: 1;
  text-align: center;
  padding: 12px 2px;
  font-size: 13px;
  color: #666;
  cursor: pointer;
  position: relative;
  transition: color 0.25s ease;
  white-space: nowrap;
}
.status-tab.active {
  color: #e1251b;
  font-weight: 600;
}
.status-tab.active::after {
  content: '';
  position: absolute;
  left: 50%;
  bottom: 0;
  transform: translateX(-50%);
  width: 22px;
  height: 3px;
  border-radius: 2px;
  background: #e1251b;
}
.status-count {
  display: inline-block;
  min-width: 16px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  margin-left: 2px;
  font-size: 11px;
  font-weight: 500;
  color: #999;
  background: #f0f0f0;
  border-radius: 8px;
  vertical-align: 1px;
}
.status-tab.active .status-count {
  color: #fff;
  background: #e1251b;
}
.filter-empty { background: #fff; margin: 0 8px 8px; border-radius: 8px; }
/* Smooth transition when switching status tabs */
.order-fade-enter-active,
.order-fade-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}
.order-fade-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.order-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
.order-fade-move {
  transition: transform 0.25s ease;
}
.order-card { background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.o-head { display: flex; justify-content: space-between; font-size: 12px; color: #999; margin-bottom: 8px; }
.o-status { color: #e1251b; }
.o-countdown { color: #e1251b; font-weight: 600; }
/* Feature: 订单退款状态徽章 (Order Refund Status Icon) */
.refund-badge-row { margin-bottom: 8px; }
.refund-badge {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 10px;
  border-radius: 10px;
  line-height: 1.6;
}
.refund-done {
  color: #fff;
  background: #07c160;
}
.refund-pending {
  color: #fff;
  background: #ff9800;
  animation: refund-pending-blink 1.4s ease-in-out infinite;
}
@keyframes refund-pending-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}
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
/* Feature: 订单物流迷你跟踪 (Order Express Tracking Mini) */
.express-mini {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
  padding: 8px 12px;
  background: linear-gradient(90deg, #fff7e6, #fff);
  border: 1px solid #ffe7ba;
  border-radius: 8px;
}
.em-logo {
  font-size: 22px;
  line-height: 1;
  animation: em-bob 1.8s ease-in-out infinite;
}
@keyframes em-bob {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-3px); }
}
.em-info { flex: 1; display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.em-label { font-size: 12px; color: #ff7a18; font-weight: 600; }
.em-waybill {
  font-size: 13px;
  color: #333;
  font-family: 'Courier New', monospace;
  letter-spacing: 0.5px;
}
.em-icon { font-size: 18px; color: #ff9800; }
/* Feature: 订单时间轴迷你图 (Order Timeline Mini Map) — compact 4-step bar */
.o-mini-timeline {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 10px 4px 6px;
  margin-top: 8px;
  background: #fafafa;
  border-radius: 8px;
}
.mt-step {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  min-width: 0;
}
.mt-node {
  display: flex;
  align-items: center;
  width: 100%;
  justify-content: center;
  position: relative;
  height: 18px;
}
.mt-node-circle {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #e0e0e0;
  color: #fff;
  font-size: 11px;
  font-weight: bold;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  z-index: 1;
  transition: background 0.25s ease, transform 0.25s ease;
}
.mt-node-line {
  position: absolute;
  left: 50%;
  width: 100%;
  top: 50%;
  height: 2px;
  background: #e0e0e0;
  transform: translateY(-50%);
  z-index: 0;
}
.mt-label {
  margin-top: 6px;
  font-size: 11px;
  color: #999;
  white-space: nowrap;
}
/* Done steps: red node + red connector line. */
.mt-step.is-done .mt-node-circle {
  background: #e1251b;
}
.mt-step.is-done .mt-node-line {
  background: #e1251b;
}
/* Current step: enlarged red node with a subtle ring. */
.mt-step.is-current .mt-node-circle {
  background: #e1251b;
  transform: scale(1.18);
  box-shadow: 0 0 0 3px rgba(225, 37, 27, 0.18);
}
.mt-step.is-current .mt-label {
  color: #e1251b;
  font-weight: 600;
}
.o-foot { display: flex; justify-content: space-between; align-items: center; padding-top: 8px; border-top: 1px solid #f5f5f5; margin-top: 8px; font-size: 13px; }
.o-actions { display: flex; gap: 8px; }
/* Feature: 评价提醒 (Order Review Reminder) */
.review-reminder {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
  padding: 8px 12px;
  background: linear-gradient(90deg, #fff5f5, #ffe9e8);
  border: 1px solid #ffd6d4;
  border-radius: 8px;
}
.rr-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  animation: rr-pulse 1.6s ease-in-out infinite;
}
.rr-icon {
  font-size: 13px;
  font-weight: 600;
  color: #e1251b;
}
.rr-incentive {
  font-size: 11px;
  font-weight: bold;
  color: #fff;
  background: #e1251b;
  padding: 1px 8px;
  border-radius: 10px;
}
.rr-actions { display: flex; gap: 6px; flex-shrink: 0; }
@keyframes rr-pulse {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.04); opacity: 0.85; }
}
/* Order lifecycle timeline (订单全流程时间线) */
.o-timeline { margin-top: 12px; padding: 12px 8px 4px; background: #fafafa; border-radius: 8px; }
.ot-item { position: relative; display: flex; gap: 12px; padding-bottom: 16px; }
.ot-item:last-child { padding-bottom: 0; }
.ot-dot { position: relative; z-index: 1; width: 26px; height: 26px; border-radius: 50%; background: #fff; border: 2px solid #ddd; display: flex; align-items: center; justify-content: center; font-size: 13px; flex-shrink: 0; }
.ot-item.is-done .ot-dot { border-color: #07c160; background: #f0fff4; }
.ot-item.is-current .ot-dot { border-color: #e1251b; background: #fff5f5; box-shadow: 0 0 0 4px rgba(225, 37, 27, 0.12); }
.ot-line { position: absolute; left: 12px; top: 26px; width: 2px; height: calc(100% - 14px); background: #ddd; transform: translateX(-1px); }
.ot-item.is-done .ot-line { background: #07c160; }
.ot-body { padding-top: 2px; }
.ot-label { font-size: 14px; color: #999; }
.ot-item.is-done .ot-label { color: #333; }
.ot-item.is-current .ot-label { color: #e1251b; font-weight: 600; }
.ot-time { font-size: 12px; color: #bbb; margin-top: 2px; }
.ot-item.is-current .ot-time { color: #e1251b; }
</style>
