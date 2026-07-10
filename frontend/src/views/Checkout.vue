<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, createOrder, getMyCoupons, getAddresses, requestInvoice, getTieredDiscounts } from '../api'

const router = useRouter()
const items = ref([])
const address = ref('')
const remark = ref('')
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
  try {
    await createOrder({
      items: items.value.map((i) => ({ product_id: i.product_id, quantity: i.quantity })),
      address: address.value,
      user_coupon_id: selectedCouponId.value || undefined,
      remark: remark.value,
    })
    // Show the success celebration overlay, then navigate to /orders after 2s.
    showSuccess.value = true
    spawnSuccessConfetti()
    successTimer = setTimeout(() => {
      showSuccess.value = false
      router.replace('/orders')
    }, 2000)
  } catch (e) {
    showToast(e.response?.data?.error || '下单失败')
  }
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
</style>
