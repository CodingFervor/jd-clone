<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, updateCart, deleteCart, createOrder } from '../api'

const router = useRouter()
const items = ref([])
const selectedTotal = ref(0)
const allSelected = ref(true)
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    const res = await getCart()
    items.value = res.data || []
    selectedTotal.value = res.selected_total || 0
    allSelected.value = items.value.length > 0 && items.value.every((i) => i.selected === 1)
  } catch (e) {
    if (e.response?.status === 401) router.replace('/login')
  } finally {
    loading.value = false
  }
}
onMounted(load)
onActivated(load)

async function toggleSelect(item) {
  const newSel = item.selected === 1 ? 0 : 1
  try {
    await updateCart(item.id, item.quantity, newSel)
    item.selected = newSel
    await load()
  } catch (e) {}
}
async function changeQty(item, qty) {
  if (qty < 1) return
  try {
    await updateCart(item.id, qty, item.selected)
    item.quantity = qty
    await load()
  } catch (e) {}
}
async function removeItem(item) {
  try {
    await deleteCart(item.id)
    await load()
    showSuccessToast('已删除')
  } catch (e) {}
}
async function toggleAll() {
  const target = allSelected.value ? 0 : 1
  for (const it of items.value) {
    if (it.selected !== target) await updateCart(it.id, it.quantity, target)
  }
  await load()
}
async function checkout() {
  if (selectedTotal.value <= 0) {
    showToast('请选择商品')
    return
  }
  const selected = items.value.filter((i) => i.selected === 1).map((i) => ({ product_id: i.product_id, quantity: i.quantity }))
  try {
    await createOrder({ items: selected, address: '' })
    showSuccessToast('下单成功')
    router.push('/orders')
  } catch (e) {
    showToast(e.response?.data?.error || '下单失败')
  }
}
function fmt(n) {
  return Number(n).toFixed(2)
}
</script>

<template>
  <div class="cart-page">
    <van-nav-bar title="购物车" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!items.length" description="购物车是空的">
      <van-button type="danger" round @click="router.push('/home')">去逛逛</van-button>
    </van-empty>
    <div v-else>
      <van-checkbox-group v-model="dummy">
        <div v-for="it in items" :key="it.id" class="cart-item">
          <van-checkbox :model-value="it.selected === 1" @click="toggleSelect(it)" />
          <van-image width="80" height="80" radius="6" :src="it.product_image" fit="cover" @click="router.push('/product/' + it.product_id)" />
          <div class="ci-info">
            <div class="ci-name van-multi-ellipsis--l2">{{ it.product_name }}</div>
            <div class="ci-bottom">
              <span class="price">¥{{ fmt(it.price) }}</span>
              <van-stepper v-model="it.quantity" :min="1" :max="it.stock" @change="(v) => changeQty(it, v)" />
              <van-icon name="delete-o" size="20" @click="removeItem(it)" />
            </div>
          </div>
        </div>
      </van-checkbox-group>

      <van-submit-bar :price="selectedTotal * 100" button-text="结算" @submit="checkout">
        <van-checkbox :model-value="allSelected" @click="toggleAll">全选</van-checkbox>
      </van-submit-bar>
    </div>
  </div>
</template>

<style scoped>
.cart-page { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.cart-item { display: flex; align-items: center; gap: 10px; padding: 12px; background: #fff; border-bottom: 1px solid #f5f5f5; }
.ci-info { flex: 1; }
.ci-name { font-size: 13px; line-height: 18px; height: 36px; }
.ci-bottom { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
.ci-bottom .price { font-size: 16px; flex: 1; }
</style>
