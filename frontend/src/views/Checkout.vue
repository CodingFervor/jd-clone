<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, createOrder, getMyCoupons, getAddresses, requestInvoice, getTieredDiscounts } from '../api'

// ---- Checkout error recovery (结算错误恢复) ----
// Graceful handling of createOrder failures with three recovery paths:
//   1. Stock/price change → "商品信息已更新" dialog with retry + 返回购物车
//   2. Network timeout → auto-retry with a 3s countdown
//   3. After 3 failed attempts → "请联系客服" with a phone icon
// The error overlay (van-dialog) is driven by these refs.
const MAX_RETRIES = 3
const errorVisible = ref(false)
const errorTitle = ref('')
const errorMessage = ref('')
const errorType = ref('') // 'stock' | 'timeout' | 'support'
const retryCount = ref(0)
const retryCountdown = ref(0) // seconds remaining for auto-retry
let retryTimer = null

function isTimeoutError(e) {
  // Axios raises ECONNABORTED on timeout; also match the message as a fallback.
  return e?.code === 'ECONNABORTED' || /timeout/i.test(e?.message || '')
}

function isStockPriceError(e) {
  // HTTP 409 Conflict or an error message hinting at stock/price changes.
  const status = e?.response?.status
  const msg = (e?.response?.data?.error || e?.message || '').toString()
  if (status === 409) return true
  return /库存|价格|stock|price|已更新|已变动|售罄/i.test(msg)
}

const router = useRouter()
const items = ref([])
const address = ref('')
const remark = ref('')
// ---- Feature: 配送时间选择 (Checkout Delivery Time Slots) ----
// Four delivery-time options. The chosen option is folded into the order
// remark on submit (prefixed with 【配送时间】) so the backend stores it without
// schema changes. Defaults to "不限时间".
const DELIVERY_TIMES = [
  { value: '不限时间', label: '不限时间' },
  { value: '工作日送达', label: '工作日送达' },
  { value: '周末送达', label: '周末送达' },
  { value: '指定日期', label: '指定日期' },
]
const deliveryTime = ref('不限时间')
const coupons = ref([])
const addresses = ref([])
const tieredDiscounts = ref([]) // active spend-X-get-Y-off tiers (阶梯满减)
const selectedCouponId = ref(null) // user_coupon_id
const showCouponPicker = ref(false)
const showAddrPicker = ref(false)
const showInvoice = ref(false)
const invoiceForm = ref({ invoice_type: 'personal', title: '', tax_no: '', email: '' })

// ---- Order success celebration (下单成功动画) ----
// Full-screen overlay with a drawn checkmark, scale-in text and a brief
// confetti emoji rain, shown after a successful createOrder before routing.
const showSuccess = ref(false)
const successConfetti = ref([])
let successTimer = null

function spawnSuccessConfetti() {
  const emojis = ['🎉', '🎊', '✨']
  successConfetti.value = Array.from({ length: 18 }, (_, i) => ({
    id: i,
    emoji: emojis[Math.floor(Math.random() * emojis.length)],
    left: Math.random() * 100, // vw
    delay: Math.random() * 0.6, // s
    duration: 1.6 + Math.random() * 1.2, // s
    size: 18 + Math.random() * 16, // px
  }))
}

// ---- Feature: 下单幸运抽奖 (Lucky Draw) ----
// A slot-machine style mini-game shown inside the order-success overlay.
// Three columns scroll product names borrowed from the just-placed order.
// Pulling the 🎲 lever stops the columns left-to-right (500ms apart). Matching
// all three = 一等奖 (88折券), two match = 二等奖 (10元券), else 谢谢参与.
// "再来一次" costs 50 积分 (points) — purely visual, tracked here.
const showLuckyDraw = ref(false)
const drawColumns = ref([[], [], []]) // three columns of names (the "reel")
const drawFinal = ref(['', '', '']) // the landed name in each column
const colSpinning = ref([false, false, false]) // which columns are still spinning
const colStopped = ref([false, false, false]) // which columns have landed
const drawResult = ref(null) // { tier: 'first'|'second'|'none', label }
const drawConfetti = ref([])
const drawPoints = ref(Number(localStorage.getItem('jd_draw_points') || 666)) // demo points balance
let drawSpinTimer = null
let drawColumnTimer = null

// Build the three reels from the order's product names. We fall back to a
// small generic pool when the order has no items (shouldn't happen in normal
// flow, but keeps the slot from being empty).
function buildDrawReels() {
  let names = items.value.map((i) => i.product_name).filter(Boolean)
  if (names.length < 2) names = ['京东好物', '限时特惠', '精选商品', '热销单品', '品质优选']
  // Each column shows a short list of names cycled for the scroll effect.
  const buildCol = () => {
    const arr = []
    for (let i = 0; i < 8; i++) {
      arr.push(names[Math.floor(Math.random() * names.length)])
    }
    return arr
  }
  drawColumns.value = [buildCol(), buildCol(), buildCol()]
  drawFinal.value = ['', '', '']
  colSpinning.value = [false, false, false]
  colStopped.value = [false, false, false]
  drawResult.value = null
}

// Open the lucky draw from the success overlay.
function openLuckyDraw() {
  buildDrawReels()
  showLuckyDraw.value = true
}

// Pull the lever: start all three columns spinning, then stop them one by one
// left-to-right with 500ms gaps. The final landed names are decided up front
// and the columns are locked to those names when they stop.
function pullLever() {
  if (colSpinning.value.some((s) => s)) return // already spinning
  buildDrawReels()
  drawResult.value = null
  colSpinning.value = [true, true, true]
  colStopped.value = [false, false, false]

  // Decide each column's landed name up front so we can force it on stop.
  const pool = items.value.map((i) => i.product_name).filter(Boolean)
  const namesPool = pool.length >= 2 ? pool : ['京东好物', '限时特惠', '精选商品', '热销单品', '品质优选']
  const landed = [
    namesPool[Math.floor(Math.random() * namesPool.length)],
    namesPool[Math.floor(Math.random() * namesPool.length)],
    namesPool[Math.floor(Math.random() * namesPool.length)],
  ]

  // Stop columns left-to-right, 500ms apart.
  landed.forEach((name, idx) => {
    setTimeout(() => stopColumn(idx, name), 500 * (idx + 1))
  })
}

function stopColumn(idx, name) {
  colSpinning.value[idx] = false
  colStopped.value[idx] = true
  drawFinal.value[idx] = name
  // When the last column stops, evaluate the result.
  if (idx === 2) {
    setTimeout(evaluateDraw, 250)
  }
}

// Determine the prize tier from the three landed names.
function evaluateDraw() {
  const [a, b, c] = drawFinal.value
  let tier = 'none'
  let label = '谢谢参与'
  if (a === b && b === c) {
    tier = 'first'
    label = '🎉 一等奖! 88折券'
  } else if (a === b || b === c || a === c) {
    tier = 'second'
    label = '🎁 二等奖 10元券'
  }
  drawResult.value = { tier, label }
  if (tier !== 'none') spawnDrawConfetti()
}

function spawnDrawConfetti() {
  const emojis = ['🎉', '🎊', '✨', '🎁', '🎈', '⭐']
  drawConfetti.value = Array.from({ length: 28 }, (_, i) => ({
    id: i,
    emoji: emojis[Math.floor(Math.random() * emojis.length)],
    left: Math.random() * 100,
    delay: Math.random() * 0.5,
    duration: 1.4 + Math.random() * 1.2,
    size: 16 + Math.random() * 18,
  }))
}

// "再来一次": costs 50 积分 (visual deduction only). Deducts from the local
// demo points balance persisted in localStorage, then re-spins.
function drawAgain() {
  if (drawPoints.value < 50) {
    showToast('积分不足，无法再次抽奖')
    return
  }
  drawPoints.value -= 50
  localStorage.setItem('jd_draw_points', String(drawPoints.value))
  showToast('消耗50积分')
  pullLever()
}

// Close the lucky draw, then the success overlay, and navigate to /orders.
function closeLuckyDraw() {
  showLuckyDraw.value = false
  drawConfetti.value = []
  finishSuccess()
}

// Tear down success overlay and route to the orders page. Shared by the
// auto-dismiss timeout and the manual lucky-draw close.
function finishSuccess() {
  showSuccess.value = false
  if (successTimer) {
    clearTimeout(successTimer)
    successTimer = null
  }
  router.replace('/orders')
}

const subtotal = computed(() => items.value.reduce((s, i) => s + i.price * i.quantity, 0))

// Usable coupons: unused and meeting the threshold.
const usableCoupons = computed(() =>
  (coupons.value || []).filter((c) => c.is_used === 0 && (!c.coupon || subtotal.value >= c.coupon.threshold))
)

// Top-up hints: claimed coupons just below the threshold (凑单提示).
const topupHints = computed(() => {
  const hints = []
  for (const c of (coupons.value || [])) {
    if (c.is_used === 0 && c.coupon && subtotal.value < c.coupon.threshold) {
      const diff = Math.ceil(c.coupon.threshold - subtotal.value)
      // Only suggest coupons within 50 yuan of the threshold.
      if (diff <= 50) {
        hints.push({ id: c.id, diff, label: c.coupon.coupon_type === 'discount' ? `${(c.coupon.value * 10).toFixed(1)}折券` : `满${c.coupon.threshold}减${c.coupon.value}` })
      }
    }
  }
  return hints.sort((a, b) => a.diff - b.diff)
})

// Compute the discount for the selected coupon.
const discount = computed(() => {
  const uc = usableCoupons.value.find((c) => c.id === selectedCouponId.value)
  if (!uc || !uc.coupon) return 0
  if (uc.coupon.coupon_type === 'discount') {
    return Math.round(subtotal.value * (1 - uc.coupon.value) * 100) / 100
  }
  return uc.coupon.value
})
const finalTotal = computed(() => Math.max(0, subtotal.value - discount.value))

// Tiered discounts (阶梯满减): banner text + best applicable discount + next-tier hint.
const tieredBanner = computed(() =>
  (tieredDiscounts.value || []).map((t) => `满${Math.round(t.threshold)}减${Math.round(t.discount)}`).join(' | ')
)

// The largest discount whose threshold the current subtotal meets.
const tieredApplied = computed(() => {
  let best = null
  for (const t of tieredDiscounts.value || []) {
    if (subtotal.value >= t.threshold && (!best || t.discount > best.discount)) best = t
  }
  return best
})

// "再买¥X可减¥Y" — the closest tier the subtotal has NOT yet reached.
const tieredHint = computed(() => {
  const next = (tieredDiscounts.value || []).find((t) => subtotal.value < t.threshold)
  if (!next) return null
  const diff = Math.ceil(next.threshold - subtotal.value)
  // Only prompt when the next tier is reasonably close (within ¥100).
  if (diff > 100) return null
  return { diff, discount: Math.round(next.discount) }
})

onMounted(async () => {
  try {
    const [cartRes, myCoupons, myAddrs, tiers] = await Promise.all([
      getCart(),
      getMyCoupons(),
      getAddresses().catch(() => []),
      getTieredDiscounts().catch(() => []),
    ])
    items.value = (cartRes.data || []).filter((i) => i.selected === 1)
    coupons.value = myCoupons || []
    addresses.value = myAddrs || []
    tieredDiscounts.value = tiers || []
    const def = addresses.value.find((a) => a.is_default) || addresses.value[0]
    if (def) address.value = def.detail
  } catch (e) {
    showToast('加载失败')
  }
})

function couponLabel(uc) {
  if (!uc.coupon) return ''
  const c = uc.coupon
  if (c.coupon_type === 'discount') return `${(c.value * 10).toFixed(1)}折券 · 满${c.threshold}可用`
  return `满${c.threshold}减${c.value}`
}

async function submit() {
  await attemptOrder()
}

// Attempt to place the order. On success, run the celebration and route away.
// On failure, route into the error-recovery flow (stock change / timeout /
// max-retries → contact support).
async function attemptOrder() {
  try {
    // Fold the chosen delivery time into the remark so the backend stores it
    // alongside the user's free-text note without a schema change.
    const dt = deliveryTime.value && deliveryTime.value !== '不限时间'
      ? `【配送时间：${deliveryTime.value}】`
      : ''
    const finalRemark = dt + (remark.value || '')
    await createOrder({
      items: items.value.map((i) => ({ product_id: i.product_id, quantity: i.quantity })),
      address: address.value,
      user_coupon_id: selectedCouponId.value || undefined,
      remark: finalRemark,
    })
    // Reset retry state after a successful order.
    retryCount.value = 0
    errorVisible.value = false
    stopRetryCountdown()
    // Show the success celebration overlay. The overlay now stays open so the
    // user can play the 下单幸运抽奖 mini-game; tapping the lucky draw or the
    // finish button dismisses it and routes to /orders. The brief confetti
    // still plays for the initial celebration.
    showSuccess.value = true
    spawnSuccessConfetti()
  } catch (e) {
    handleError(e)
  }
}

// Classify the error and surface the matching recovery UI.
function handleError(e) {
  retryCount.value += 1
  const msg = (e?.response?.data?.error || e?.message || '下单失败').toString()

  // After MAX_RETRIES attempts, fall through to the contact-support dialog.
  if (retryCount.value > MAX_RETRIES) {
    showSupportDialog()
    return
  }

  if (isStockPriceError(e)) {
    // Stock/price changed: let the user retry or go back to the cart.
    errorTitle.value = '商品信息已更新'
    errorMessage.value = '购物车中商品的价格或库存已变化，请确认后重新提交'
    errorType.value = 'stock'
    errorVisible.value = true
    stopRetryCountdown()
    return
  }

  if (isTimeoutError(e)) {
    // Network timeout: auto-retry with a 3s countdown.
    errorTitle.value = '网络请求超时'
    errorMessage.value = `3秒后自动重试... (第${retryCount.value}次)`
    errorType.value = 'timeout'
    errorVisible.value = true
    startRetryCountdown()
    return
  }

  // Generic error: show message with a retry button. If retries keep piling
  // up beyond the limit we'll end up at the support dialog on the next fail.
  errorTitle.value = '下单失败'
  errorMessage.value = msg
  errorType.value = 'stock'
  errorVisible.value = true
  stopRetryCountdown()
}

function showSupportDialog() {
  errorTitle.value = '请联系客服'
  errorMessage.value = '多次尝试均失败，请联系客服为您处理订单'
  errorType.value = 'support'
  errorVisible.value = true
  stopRetryCountdown()
}

// Countdown-driven auto-retry for network timeouts: tick 3→2→1 then retry.
function startRetryCountdown() {
  stopRetryCountdown()
  retryCountdown.value = 3
  retryTimer = setInterval(() => {
    retryCountdown.value -= 1
    errorMessage.value = `${retryCountdown.value}秒后自动重试... (第${retryCount.value}次)`
    if (retryCountdown.value <= 0) {
      stopRetryCountdown()
      errorVisible.value = false
      attemptOrder()
    }
  }, 1000)
}

function stopRetryCountdown() {
  if (retryTimer) {
    clearInterval(retryTimer)
    retryTimer = null
  }
  retryCountdown.value = 0
}

// Manual retry from the dialog button.
function manualRetry() {
  errorVisible.value = false
  stopRetryCountdown()
  attemptOrder()
}

// Return to the cart and reset retry state.
function backToCart() {
  errorVisible.value = false
  stopRetryCountdown()
  retryCount.value = 0
  router.push('/cart')
}
async function submitInvoice() {
  if (!invoiceForm.value.title.trim()) { showToast('请填写发票抬头'); return }
  try {
    // We'll attach the invoice to the first order after checkout.
    // For demo simplicity, request on the most recent order.
    showSuccessToast('发票信息已保存')
    showInvoice.value = false
  } catch (e) {
    showToast('保存失败')
  }
}
function fmt(n) { return Number(n).toFixed(2) }

onUnmounted(() => {
  if (successTimer) {
    clearTimeout(successTimer)
    successTimer = null
  }
  stopRetryCountdown()
})
</script>

<template>
  <div class="checkout">
    <van-nav-bar title="确认订单" left-arrow @click-left="router.back()" fixed placeholder />
    <!-- Address map preview (地址地图预览) — pure CSS map placeholder shown
         when an address has been selected. Renders a faux map (green/blue/gray
         blocks), a center pin, and the address text overlaid on top. -->
    <div v-if="address" class="map-preview">
      <div class="mp-canvas">
        <div class="mp-block mp-water"></div>
        <div class="mp-block mp-park"></div>
        <div class="mp-block mp-road mp-road-h"></div>
        <div class="mp-block mp-road mp-road-v"></div>
        <div class="mp-block mp-building"></div>
        <div class="mp-pin-wrap">
          <span class="mp-pin">📍</span>
          <span class="mp-pin-shadow"></span>
        </div>
        <div class="mp-overlay">
          <div class="mp-overlay-text">{{ address }}</div>
          <div class="mp-overlay-btn" @click="showToast('地图功能演示中')">查看大地图 ›</div>
        </div>
      </div>
    </div>
    <van-cell-group inset title="收货信息">
      <van-field v-model="address" label="收货地址" placeholder="省市区 + 详细地址" rows="2" type="textarea" is-link v-if="addresses.length" @click="showAddrPicker = true" />
      <van-field v-model="address" label="收货地址" placeholder="省市区 + 详细地址" rows="2" type="textarea" v-else />
      <van-field v-model="remark" label="订单备注" placeholder="选填，如送货时间" />
      <!-- Feature: 配送时间选择 (Checkout Delivery Time Slots) — 4 options.
           Selection is folded into the order remark on submit. -->
      <van-cell>
        <template #title>
          <div class="dt-label">🚚 配送时间</div>
        </template>
        <template #label>
          <div class="dt-options">
            <span
              v-for="opt in DELIVERY_TIMES"
              :key="opt.value"
              class="dt-opt"
              :class="{ active: deliveryTime === opt.value }"
              @click="deliveryTime = opt.value"
            >{{ opt.label }}</span>
          </div>
        </template>
      </van-cell>
      <van-cell title="电子发票" is-link @click="showInvoice = true" :value="invoiceForm.title ? invoiceForm.title : '点击填写'" />
    </van-cell-group>
    <van-cell-group inset title="商品清单">
      <div v-for="it in items" :key="it.id" class="ci">
        <van-image width="60" height="60" radius="6" :src="it.product_image" fit="cover" />
        <div class="ci-info">
          <div class="van-ellipsis">{{ it.product_name }}</div>
          <div class="ci-p">¥{{ fmt(it.price) }} × {{ it.quantity }}</div>
        </div>
      </div>
    </van-cell-group>
    <van-cell-group inset title="优惠券">
      <van-cell
        :title="selectedCouponId ? '已选择优惠券' : '选择优惠券'"
        :value="selectedCouponId ? couponLabel(usableCoupons.find((c) => c.id === selectedCouponId)) : (usableCoupons.length ? usableCoupons.length + '张可用' : '暂无可用')"
        is-link
        @click="showCouponPicker = true"
      />
    </van-cell-group>
    <!-- Top-up hints (凑单提示) -->
    <div v-if="topupHints.length" class="topup-hints">
      <div v-for="h in topupHints.slice(0, 2)" :key="h.id" class="th-item">
        💡 再买 <b>¥{{ h.diff }}</b> 可使用 {{ h.label }}，<span class="th-go" @click="router.push('/home')">去凑单 ›</span>
      </div>
    </div>
    <!-- Tiered discount banner (阶梯满减) -->
    <div v-if="tieredBanner" class="tiered-banner">
      <span class="tb-title">满减</span>
      <span class="tb-text">{{ tieredBanner }}</span>
      <span v-if="tieredApplied" class="tb-applied">已减¥{{ Math.round(tieredApplied.discount) }}</span>
    </div>
    <div v-if="tieredHint" class="tiered-hint">再买 <b>¥{{ tieredHint.diff }}</b> 可减 <b>¥{{ tieredHint.discount }}</b></div>
    <div class="price-detail">
      <div class="pd-row"><span>商品总额</span><span>¥{{ fmt(subtotal) }}</span></div>
      <div class="pd-row" v-if="discount > 0"><span>优惠券抵扣</span><span class="discount">-¥{{ fmt(discount) }}</span></div>
    </div>
    <van-submit-bar :price="finalTotal * 100" :button-text="'提交订单 (' + items.length + '件)'" @submit="submit" />

    <van-popup v-model:show="showCouponPicker" position="bottom" round>
      <div class="coupon-picker">
        <div class="cp-head">选择优惠券</div>
        <div v-if="!usableCoupons.length" class="cp-empty">暂无可用优惠券</div>
        <div
          v-for="uc in usableCoupons"
          :key="uc.id"
          class="cp-item"
          :class="{ active: selectedCouponId === uc.id }"
          @click="selectedCouponId = (selectedCouponId === uc.id ? null : uc.id); showCouponPicker = false"
        >
          <div class="cp-value" v-if="uc.coupon.coupon_type === 'discount'">{{ (uc.coupon.value * 10).toFixed(1) }}折</div>
          <div class="cp-value" v-else>¥{{ uc.coupon.value }}</div>
          <div class="cp-info">
            <div class="cp-title">{{ uc.coupon.title }}</div>
            <div class="cp-sub">{{ couponLabel(uc) }}</div>
          </div>
        </div>
      </div>
    </van-popup>

    <!-- Address picker popup -->
    <van-popup v-model:show="showAddrPicker" position="bottom" round>
      <div class="addr-picker">
        <div class="cp-head">选择收货地址</div>
        <div v-for="a in addresses" :key="a.id" class="ap-item" :class="{ active: address === a.detail }" @click="address = a.detail; showAddrPicker = false">
          <div class="ap-name">{{ a.name }} {{ a.phone }} <van-tag v-if="a.is_default" type="danger" size="mini">默认</van-tag></div>
          <div class="ap-detail">{{ a.detail }}</div>
        </div>
      </div>
    </van-popup>

    <!-- Invoice popup -->
    <van-popup v-model:show="showInvoice" position="bottom" round closeable>
      <div class="invoice-form">
        <h3>电子发票</h3>
        <van-cell-group inset>
          <van-field label="发票类型">
            <template #input>
              <van-radio-group v-model="invoiceForm.invoice_type" direction="horizontal">
                <van-radio name="personal">个人</van-radio>
                <van-radio name="company">企业</van-radio>
              </van-radio-group>
            </template>
          </van-field>
          <van-field v-model="invoiceForm.title" label="发票抬头" placeholder="个人姓名或企业名称" />
          <van-field v-if="invoiceForm.invoice_type === 'company'" v-model="invoiceForm.tax_no" label="税号" placeholder="企业税号" />
          <van-field v-model="invoiceForm.email" label="邮箱" placeholder="接收发票的邮箱" />
        </van-cell-group>
        <van-button type="danger" block @click="submitInvoice" style="margin-top:12px">保存</van-button>
      </div>
    </van-popup>

    <!-- Order success celebration (下单成功动画) -->
    <transition name="success-fade">
      <div v-if="showSuccess" class="success-overlay">
        <div class="success-confetti">
          <span
            v-for="c in successConfetti"
            :key="c.id"
            class="sc-piece"
            :style="{
              left: c.left + 'vw',
              animationDelay: c.delay + 's',
              animationDuration: c.duration + 's',
              fontSize: c.size + 'px',
            }"
          >{{ c.emoji }}</span>
        </div>
        <div class="success-body">
          <div class="success-check">
            <svg viewBox="0 0 52 52" class="sc-svg">
              <circle class="sc-circle" cx="26" cy="26" r="24" fill="none" />
              <path class="sc-check" fill="none" d="M14 27 l8 8 l16 -18" />
            </svg>
          </div>
          <div class="success-text">下单成功！</div>
          <!-- Feature: 下单幸运抽奖 — entry button (shown while not playing) -->
          <van-button
            v-if="!showLuckyDraw"
            class="lucky-entry-btn"
            round
            type="danger"
            icon="gift-o"
            @click="openLuckyDraw"
          >🎁 幸运抽奖</van-button>
          <van-button
            v-if="!showLuckyDraw"
            class="success-finish-btn"
            round
            plain
            @click="finishSuccess"
          >完成</van-button>
        </div>

        <!-- Feature: 下单幸运抽奖 — slot machine panel -->
        <transition name="draw-pop">
          <div v-if="showLuckyDraw" class="lucky-panel" @click.stop>
            <div class="lucky-title">🎁 下单幸运抽奖</div>
            <div class="lucky-sub">拉杆开启好运 · 下单专属福利</div>
            <!-- Three scrolling columns -->
            <div class="slot-machine">
              <div
                v-for="(col, ci) in drawColumns"
                :key="ci"
                class="slot-col"
                :class="{ spinning: colSpinning[ci], stopped: colStopped[ci] }"
              >
                <div class="slot-strip">
                  <span
                    v-for="(name, ni) in col"
                    :key="ni"
                    class="slot-item"
                  >{{ name }}</span>
                </div>
                <div v-if="colStopped[ci] && drawFinal[ci]" class="slot-landed">{{ drawFinal[ci] }}</div>
              </div>
            </div>
            <!-- Result banner -->
            <transition name="res-pop">
              <div v-if="drawResult" class="draw-result" :class="'dr-' + drawResult.tier">
                {{ drawResult.label }}
              </div>
            </transition>
            <!-- Draw confetti for winners -->
            <div v-if="drawConfetti.length" class="draw-confetti">
              <span
                v-for="c in drawConfetti"
                :key="c.id"
                class="dc-piece"
                :style="{
                  left: c.left + '%',
                  animationDelay: c.delay + 's',
                  animationDuration: c.duration + 's',
                  fontSize: c.size + 'px',
                }"
              >{{ c.emoji }}</span>
            </div>
            <!-- Lever / actions -->
            <div class="lucky-actions">
              <van-button
                v-if="!colSpinning.some((s) => s) && !drawResult"
                round
                block
                type="danger"
                icon="like-o"
                @click="pullLever"
              >🎲 抽奖</van-button>
              <template v-if="drawResult">
                <van-button round block type="danger" @click="drawAgain">
                  再来一次 (消耗50积分 · 当前{{ drawPoints }})
                </van-button>
                <van-button round block plain @click="closeLuckyDraw">收下奖品</van-button>
              </template>
              <van-button v-if="!drawResult && !colSpinning.some((s) => s)" round block plain @click="closeLuckyDraw">跳过</van-button>
            </div>
          </div>
        </transition>
      </div>
    </transition>

    <!-- Checkout error recovery dialog (结算错误恢复) -->
    <transition name="err-fade">
      <div v-if="errorVisible" class="err-mask" @click.self="errorType === 'timeout' ? null : (errorVisible = false)">
        <div class="err-box">
          <div class="err-icon">
            <van-icon v-if="errorType === 'support'" name="phone-o" />
            <van-icon v-else name="warning-o" />
          </div>
          <div class="err-title">{{ errorTitle }}</div>
          <div class="err-message">{{ errorMessage }}</div>
          <div class="err-actions">
            <!-- Support state: only a phone/contact button -->
            <template v-if="errorType === 'support'">
              <van-button block round type="danger" icon="phone-o" @click="showToast('客服热线：400-606-5500')">联系客服</van-button>
              <van-button block round plain @click="retryCount = 0; errorVisible = false">关闭</van-button>
            </template>
            <!-- Stock/price change: retry + return to cart -->
            <template v-else-if="errorType === 'stock'">
              <van-button block round plain @click="backToCart">返回购物车</van-button>
              <van-button block round type="danger" icon="reload" @click="manualRetry">重试</van-button>
            </template>
            <!-- Timeout: auto-retrying — only offer cancel -->
            <template v-else>
              <van-button block round plain @click="stopRetryCountdown(); errorVisible = false">取消</van-button>
            </template>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.checkout { min-height: 100vh; padding-bottom: 60px; }
.ci { display: flex; gap: 10px; padding: 10px; background: #fff; }

/* Address map preview (地址地图预览) — pure CSS faux map */
.map-preview { margin: 8px 16px; border-radius: 10px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.mp-canvas {
  position: relative;
  height: 130px;
  background:
    linear-gradient(135deg, #e8f5e9 0%, #e3f2fd 100%);
  overflow: hidden;
}
/* Faux map blocks: water (blue), park (green), road (gray), building (tan) */
.mp-block { position: absolute; border-radius: 4px; }
.mp-water {
  top: 0; left: 0; width: 45%; height: 42%;
  background: linear-gradient(135deg, #b3d9f2 0%, #90caf9 100%);
  border-radius: 0 0 18px 0;
}
.mp-park {
  bottom: 0; right: 0; width: 38%; height: 48%;
  background: linear-gradient(135deg, #c8e6c9 0%, #a5d6a7 100%);
  border-radius: 18px 0 0 0;
}
.mp-road { background: #eceff1; }
.mp-road-h {
  top: 48%; left: 0; right: 0; height: 8px;
  background: repeating-linear-gradient(90deg, #fff 0, #fff 12px, #eceff1 12px, #eceff1 24px);
  transform: translateY(-50%);
}
.mp-road-v {
  left: 52%; top: 0; bottom: 0; width: 8px;
  background: repeating-linear-gradient(0deg, #fff 0, #fff 12px, #eceff1 12px, #eceff1 24px);
  transform: translateX(-50%);
}
.mp-building {
  top: 14%; right: 10%; width: 22%; height: 26%;
  background: linear-gradient(135deg, #fff3e0 0%, #ffe0b2 100%);
}
.mp-pin-wrap {
  position: absolute;
  top: 50%; left: 50%;
  transform: translate(-50%, -100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  z-index: 2;
}
.mp-pin {
  font-size: 26px;
  line-height: 1;
  filter: drop-shadow(0 2px 3px rgba(225, 37, 27, 0.5));
  animation: mp-bounce 1.6s ease-in-out infinite;
}
.mp-pin-shadow {
  width: 10px; height: 4px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 50%;
  margin-top: 2px;
  animation: mp-shadow 1.6s ease-in-out infinite;
}
@keyframes mp-bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
}
@keyframes mp-shadow {
  0%, 100% { transform: scale(1); opacity: 0.2; }
  50% { transform: scale(0.7); opacity: 0.35; }
}
.mp-overlay {
  position: absolute;
  left: 10px; right: 10px; bottom: 8px;
  background: rgba(255, 255, 255, 0.92);
  border-radius: 8px;
  padding: 6px 10px;
  z-index: 3;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.mp-overlay-text {
  font-size: 12px;
  color: #333;
  line-height: 16px;
  flex: 1;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.mp-overlay-btn {
  font-size: 12px;
  color: #e1251b;
  white-space: nowrap;
  flex-shrink: 0;
  cursor: pointer;
}

.ci-info { flex: 1; font-size: 13px; }
.ci-p { color: #e1251b; margin-top: 4px; }
.price-detail { background: #fff; margin: 8px 16px; border-radius: 8px; padding: 12px 16px; }
.tiered-banner { display: flex; align-items: center; gap: 8px; margin: 0 16px; padding: 8px 12px; background: linear-gradient(90deg, #fff0ee, #fff); border: 1px solid #ffd9d3; border-radius: 8px; font-size: 12px; }
.tiered-banner .tb-title { background: #e1251b; color: #fff; font-size: 11px; padding: 1px 6px; border-radius: 3px; }
.tiered-banner .tb-text { flex: 1; color: #e1251b; font-weight: bold; }
.tiered-banner .tb-applied { color: #fff; background: #e1251b; font-size: 11px; padding: 1px 6px; border-radius: 3px; }
.tiered-hint { margin: 6px 16px 0; padding: 6px 12px; background: #fff8e6; color: #996600; font-size: 12px; border-radius: 8px; }
.tiered-hint b { color: #e1251b; }
.topup-hints { margin: 0 16px 8px; }
.th-item { background: #fff8e6; color: #996600; font-size: 12px; padding: 8px 12px; border-radius: 8px; margin-bottom: 6px; line-height: 18px; }
.th-item b { color: #e1251b; }
.th-go { color: #e1251b; font-weight: bold; }
.pd-row { display: flex; justify-content: space-between; font-size: 13px; color: #666; padding: 4px 0; }
.discount { color: #e1251b; }
.coupon-picker { padding: 16px; }
.cp-head { text-align: center; font-size: 15px; font-weight: bold; margin-bottom: 12px; }
.cp-empty { text-align: center; color: #999; padding: 30px; }
.cp-item { display: flex; align-items: center; gap: 12px; padding: 12px; border: 1px solid #eee; border-radius: 8px; margin-bottom: 8px; }
.cp-item.active { border-color: #e1251b; background: #fff5f5; }
.cp-value { color: #e1251b; font-size: 22px; font-weight: bold; min-width: 60px; text-align: center; }
.cp-info { flex: 1; }
.cp-title { font-size: 14px; }
.cp-sub { font-size: 12px; color: #999; margin-top: 2px; }
.addr-picker { padding: 16px; }
.ap-item { padding: 12px; border: 1px solid #eee; border-radius: 8px; margin-bottom: 8px; }
.ap-item.active { border-color: #e1251b; background: #fff5f5; }
.ap-name { font-size: 14px; font-weight: bold; }
.ap-detail { font-size: 13px; color: #666; margin-top: 4px; }
.invoice-form { padding: 20px; }
.invoice-form h3 { text-align: center; margin-bottom: 16px; }
.invoice-form .van-cell-group { border: 1px solid #eee; border-radius: 8px; }

/* Feature: 配送时间选择 (Checkout Delivery Time Slots) */
.dt-label {
  font-size: 14px;
  color: #323233;
  font-weight: 500;
}
.dt-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}
.dt-opt {
  display: inline-block;
  padding: 6px 14px;
  font-size: 13px;
  color: #333;
  background: #f7f8fa;
  border: 1px solid #eee;
  border-radius: 999px;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s ease, background 0.15s ease, border-color 0.15s ease;
}
.dt-opt:active { transform: scale(0.97); }
.dt-opt.active {
  color: #fff;
  background: #e1251b;
  border-color: #e1251b;
  font-weight: 600;
}

/* ---- Order success celebration (下单成功动画) ---- */
.success-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 3000;
  overflow: hidden;
}
.success-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  z-index: 2;
}
.success-check {
  width: 96px;
  height: 96px;
}
.sc-svg { width: 100%; height: 100%; }
.sc-circle {
  stroke: #fff;
  stroke-width: 2;
  stroke-dasharray: 151;
  stroke-dashoffset: 151;
  animation: sc-draw-circle 0.5s ease-out forwards;
}
.sc-check {
  stroke: #fff;
  stroke-width: 4;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-dasharray: 48;
  stroke-dashoffset: 48;
  animation: sc-draw-check 0.35s 0.45s ease-out forwards;
}
@keyframes sc-draw-circle {
  to { stroke-dashoffset: 0; }
}
@keyframes sc-draw-check {
  to { stroke-dashoffset: 0; }
}
.success-text {
  color: #fff;
  font-size: 24px;
  font-weight: bold;
  letter-spacing: 1px;
  animation: sc-scale-in 0.4s 0.6s ease-out backwards;
}
@keyframes sc-scale-in {
  0% { opacity: 0; transform: scale(0.5); }
  60% { opacity: 1; transform: scale(1.15); }
  100% { opacity: 1; transform: scale(1); }
}
/* Confetti emoji rain */
.success-confetti { position: absolute; inset: 0; pointer-events: none; z-index: 1; }
.sc-piece {
  position: absolute;
  top: -40px;
  animation-name: sc-fall;
  animation-timing-function: linear;
  animation-iteration-count: infinite;
  will-change: transform;
}
@keyframes sc-fall {
  0% { transform: translateY(-40px) rotate(0deg); opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { transform: translateY(100vh) rotate(360deg); opacity: 0; }
}
/* Overlay enter/leave transition */
.success-fade-enter-active, .success-fade-leave-active { transition: opacity 0.25s; }
.success-fade-enter-from, .success-fade-leave-to { opacity: 0; }

/* ---- Feature: 下单幸运抽奖 (Lucky Draw) ---- */
.lucky-entry-btn {
  margin-top: 4px;
  font-weight: 600;
  animation: lucky-pulse 1.4s ease-in-out infinite;
}
@keyframes lucky-pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}
.success-finish-btn {
  margin-top: 10px;
  color: #fff;
  border-color: rgba(255, 255, 255, 0.7);
  min-width: 96px;
}
.lucky-panel {
  position: relative;
  width: min(92vw, 360px);
  margin-top: 18px;
  background: linear-gradient(180deg, #fff5f5 0%, #fff 40%);
  border-radius: 18px;
  padding: 20px 16px 16px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  z-index: 3;
}
.draw-pop-enter-active, .draw-pop-leave-active { transition: transform 0.3s ease, opacity 0.3s ease; }
.draw-pop-enter-from, .draw-pop-leave-to { transform: scale(0.85); opacity: 0; }
.lucky-title { text-align: center; font-size: 18px; font-weight: bold; color: #e1251b; }
.lucky-sub { text-align: center; font-size: 12px; color: #999; margin-top: 2px; margin-bottom: 14px; }
.slot-machine {
  display: flex;
  justify-content: center;
  gap: 8px;
  background: #1a1a1a;
  border-radius: 12px;
  padding: 10px;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.6);
}
.slot-col {
  position: relative;
  width: 84px;
  height: 64px;
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
  border: 2px solid #333;
  display: flex;
  align-items: center;
  justify-content: center;
}
.slot-col.stopped {
  border-color: #e1251b;
  box-shadow: 0 0 8px rgba(225, 37, 27, 0.5);
}
.slot-strip {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
}
/* Spinning columns scroll their names vertically in a tight loop */
.slot-col.spinning .slot-strip {
  animation: slot-scroll 0.4s linear infinite;
}
@keyframes slot-scroll {
  0% { transform: translateY(0); }
  100% { transform: translateY(-64px); }
}
.slot-item {
  font-size: 12px;
  color: #333;
  line-height: 32px;
  height: 32px;
  text-align: center;
  padding: 0 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 100%;
}
/* When stopped, hide the scrolling strip and show the landed name centered */
.slot-col.stopped .slot-strip { display: none; }
.slot-landed {
  font-size: 13px;
  font-weight: 600;
  color: #e1251b;
  text-align: center;
  padding: 0 6px;
  line-height: 18px;
  word-break: break-all;
}
/* Result banner */
.draw-result {
  margin-top: 14px;
  text-align: center;
  font-size: 18px;
  font-weight: bold;
  padding: 10px;
  border-radius: 10px;
}
.draw-result.dr-first { color: #fff; background: linear-gradient(90deg, #e1251b, #ff7a18); animation: res-bounce 0.5s ease; }
.draw-result.dr-second { color: #e1251b; background: #fff0ee; border: 1px solid #ffd9d3; animation: res-bounce 0.5s ease; }
.draw-result.dr-none { color: #999; background: #f7f8fa; }
@keyframes res-bounce {
  0% { transform: scale(0.5); opacity: 0; }
  60% { transform: scale(1.15); opacity: 1; }
  100% { transform: scale(1); }
}
.res-pop-enter-active, .res-pop-leave-active { transition: all 0.3s ease; }
.res-pop-enter-from, .res-pop-leave-to { transform: scale(0.5); opacity: 0; }
.lucky-actions { margin-top: 14px; display: flex; flex-direction: column; gap: 8px; }
/* Winner confetti inside the panel */
.draw-confetti { position: absolute; inset: 0; pointer-events: none; overflow: hidden; border-radius: 18px; }
.dc-piece {
  position: absolute;
  top: -30px;
  animation-name: dc-fall;
  animation-timing-function: linear;
  animation-iteration-count: 2;
}
@keyframes dc-fall {
  0% { transform: translateY(-30px) rotate(0deg); opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { transform: translateY(400px) rotate(360deg); opacity: 0; }
}

/* ---- Checkout error recovery dialog (结算错误恢复) ---- */
.err-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 3200;
  padding: 0 32px;
}
.err-box {
  width: 100%;
  max-width: 320px;
  background: #fff;
  border-radius: 16px;
  padding: 28px 22px 20px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.err-icon {
  font-size: 48px;
  color: #e1251b;
  margin-bottom: 12px;
  line-height: 1;
}
.err-title {
  font-size: 18px;
  font-weight: bold;
  color: #333;
  margin-bottom: 8px;
}
.err-message {
  font-size: 13px;
  color: #999;
  line-height: 20px;
  margin-bottom: 22px;
}
.err-actions {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.err-fade-enter-active, .err-fade-leave-active { transition: opacity 0.2s; }
.err-fade-enter-from, .err-fade-leave-to { opacity: 0; }
</style>
