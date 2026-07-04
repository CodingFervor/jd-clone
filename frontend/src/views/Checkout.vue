<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, createOrder, getMyCoupons } from '../api'

const router = useRouter()
const items = ref([])
const address = ref('')
const remark = ref('')
const coupons = ref([])
const selectedCouponId = ref(null) // user_coupon_id
const showCouponPicker = ref(false)

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

onMounted(async () => {
  try {
    const [cartRes, myCoupons] = await Promise.all([getCart(), getMyCoupons()])
    items.value = (cartRes.data || []).filter((i) => i.selected === 1)
    coupons.value = myCoupons || []
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
    showSuccessToast('下单成功')
    router.replace('/orders')
  } catch (e) {
    showToast(e.response?.data?.error || '下单失败')
  }
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="checkout">
    <van-nav-bar title="确认订单" left-arrow @click-left="router.back()" fixed placeholder />
    <van-cell-group inset title="收货信息">
      <van-field v-model="address" label="收货地址" placeholder="省市区 + 详细地址" rows="2" type="textarea" />
      <van-field v-model="remark" label="订单备注" placeholder="选填，如送货时间、发票等" />
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
  </div>
</template>

<style scoped>
.checkout { min-height: 100vh; padding-bottom: 60px; }
.ci { display: flex; gap: 10px; padding: 10px; background: #fff; }
.ci-info { flex: 1; font-size: 13px; }
.ci-p { color: #e1251b; margin-top: 4px; }
.price-detail { background: #fff; margin: 8px 16px; border-radius: 8px; padding: 12px 16px; }
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
</style>
