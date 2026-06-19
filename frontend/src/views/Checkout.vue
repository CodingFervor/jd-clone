<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, createOrder } from '../api'

const router = useRouter()
const items = ref([])
const total = ref(0)
const address = ref('')

onMounted(async () => {
  try {
    const res = await getCart()
    items.value = (res.data || []).filter((i) => i.selected === 1)
    total.value = items.value.reduce((s, i) => s + i.price * i.quantity, 0)
  } catch (e) {
    showToast('加载失败')
  }
})

async function submit() {
  try {
    await createOrder({ items: items.value.map((i) => ({ product_id: i.product_id, quantity: i.quantity })), address: address.value })
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
    <van-submit-bar :price="total * 100" button-text="提交订单" @submit="submit" />
  </div>
</template>

<style scoped>
.checkout { min-height: 100vh; padding-bottom: 60px; }
.ci { display: flex; gap: 10px; padding: 10px; background: #fff; }
.ci-info { flex: 1; font-size: 13px; }
.ci-p { color: #e1251b; margin-top: 4px; }
</style>
