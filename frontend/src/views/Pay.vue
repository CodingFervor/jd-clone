<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { createPayment, confirmPayment } from '../api'

const route = useRoute()
const router = useRouter()
const orderId = ref(Number(route.query.order_id) || 0)
const method = ref('wechat')
const payment = ref(null)
const loading = ref(false)
const confirming = ref(false)

onMounted(async () => {
  if (!orderId.value) {
    showToast('缺少订单参数')
    router.back()
    return
  }
  await initiate()
})

async function initiate() {
  loading.value = true
  try {
    const res = await createPayment(orderId.value, method.value)
    payment.value = res.payment
  } catch (e) {
    showToast(e.response?.data?.error || '发起支付失败')
  } finally {
    loading.value = false
  }
}

async function doConfirm() {
  if (!payment.value) return
  confirming.value = true
  try {
    await confirmPayment(payment.value.id)
    showSuccessToast('支付成功')
    setTimeout(() => router.replace('/orders'), 1200)
  } catch (e) {
    showToast(e.response?.data?.error || '支付确认失败')
  } finally {
    confirming.value = false
  }
}

function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="pay-page">
    <van-nav-bar title="京东收银台" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="amount-card">
      <div class="label">应付金额</div>
      <div class="amount">¥{{ fmt(payment ? payment.amount : 0) }}</div>
      <div class="order-no">订单号: {{ payment ? payment.transaction_no : '' }}</div>
    </div>
    <van-radio-group v-model="method" @change="initiate">
      <van-cell-group inset>
        <van-cell title="" clickable @click="method = 'wechat'; initiate()">
          <template #title>
            <div class="pay-method">
              <span class="pm-icon" style="background:#07c160">💚</span>
              <span>微信支付</span>
            </div>
          </template>
          <template #right-icon><van-radio name="wechat" /></template>
        </van-cell>
        <van-cell title="" clickable @click="method = 'alipay'; initiate()">
          <template #title>
            <div class="pay-method">
              <span class="pm-icon" style="background:#1677ff">💙</span>
              <span>支付宝</span>
            </div>
          </template>
          <template #right-icon><van-radio name="alipay" /></template>
        </van-cell>
        <van-cell title="" clickable @click="method = 'unionpay'; initiate()">
          <template #title>
            <div class="pay-method">
              <span class="pm-icon" style="background:#e1251b">❤️</span>
              <span>银联支付</span>
            </div>
          </template>
          <template #right-icon><van-radio name="unionpay" /></template>
        </van-cell>
      </van-cell-group>
    </van-radio-group>
    <div class="tip">
      <van-icon name="info-o" /> 沙箱模式：点击下方按钮模拟完成支付，订单将变为"已付款"
    </div>
    <div style="margin: 20px">
      <van-button type="danger" block round :loading="confirming" @click="doConfirm">
        确认支付 ¥{{ fmt(payment ? payment.amount : 0) }}
      </van-button>
    </div>
  </div>
</template>

<style scoped>
.pay-page { min-height: 100vh; background: #f5f5f5; }
.amount-card { text-align: center; padding: 36px 20px; background: #fff; margin-bottom: 10px; }
.label { color: #999; font-size: 14px; }
.amount { color: #e1251b; font-size: 36px; font-weight: bold; margin: 10px 0; }
.order-no { color: #ccc; font-size: 12px; }
.pay-method { display: flex; align-items: center; gap: 10px; }
.pm-icon { width: 28px; height: 28px; border-radius: 6px; display: inline-flex; align-items: center; justify-content: center; font-size: 14px; }
.tip { color: #999; font-size: 12px; padding: 12px 24px; text-align: center; line-height: 18px; }
</style>
