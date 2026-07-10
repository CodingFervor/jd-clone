<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getCart, updateCart, deleteCart, createOrder, getProducts, addToCart } from '../api'

const router = useRouter()
const items = ref([])
const selectedTotal = ref(0)
const allSelected = ref(true)
const loading = ref(true)
// Per-item selected count ("已选N件"), derived from the client cart items so
// the label stays accurate even before the server round-trip in load() settles.
const selectedCount = computed(() => items.value.filter((i) => i.selected === 1).length)
// "猜你喜欢" recommended products, fetched as best-sellers from catalog.
const recommendations = ref([])

async function loadRecommendations() {
  try {
    const res = await getProducts({ page: 1, page_size: 4 })
    // Support both { data: { data, list } } and { data: [...] } shapes.
    const list = res.data?.data || res.data?.list || res.data || []
    recommendations.value = (Array.isArray(list) ? list : []).slice(0, 4)
  } catch (e) {
    recommendations.value = []
  }
}

async function load() {
  loading.value = true
  try {
    const res = await getCart()
    items.value = res.data || []
    selectedTotal.value = res.selected_total || 0
    allSelected.value = items.value.length > 0 && items.value.every((i) => i.selected === 1)
    // Load recommendations after the cart is loaded so the "猜你喜欢" section
    // is populated while / above the cart items.
    await loadRecommendations()
  } catch (e) {
    if (e.response?.status === 401) router.replace('/login')
  } finally {
    loading.value = false
  }
}
onMounted(load)
onActivated(load)

async function quickAdd(p) {
  try {
    await addToCart(p.id, 1)
    showSuccessToast('已加入购物车')
    await load()
  } catch (e) {
    showToast(e.response?.data?.error || '加入失败')
  }
}

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
// 反选 (invert selection): toggle every item's selected state.
async function invertSelection() {
  for (const it of items.value) {
    const target = it.selected === 1 ? 0 : 1
    await updateCart(it.id, it.quantity, target)
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

// A "降价" tag for items whose original price is significantly higher than the
// current price (indicating an active discount / price drop).
function isPriceDrop(it) {
  return Number(it.original_price) > Number(it.price) * 1.1
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
      <!-- 猜你喜欢: horizontally scrollable recommended products -->
      <div v-if="recommendations.length" class="rec-section">
        <div class="rec-title">猜你喜欢</div>
        <div class="rec-scroll">
          <div
            v-for="p in recommendations"
            :key="p.id"
            class="rec-card"
            @click="router.push('/product/' + p.id)"
          >
            <van-image
              width="100"
              height="100"
              radius="6"
              :src="p.image"
              fit="cover"
            />
            <div class="rec-name van-multi-ellipsis--l2">{{ p.name }}</div>
            <div class="rec-bottom">
              <span class="rec-price">¥{{ fmt(p.price) }}</span>
              <van-button
                size="mini"
                type="danger"
                round
                @click.stop="quickAdd(p)"
              >加入购物车</van-button>
            </div>
          </div>
        </div>
      </div>

      <div v-for="it in items" :key="it.id" class="cart-item">
          <van-checkbox :model-value="it.selected === 1" @click="toggleSelect(it)" />
          <van-image width="80" height="80" radius="6" :src="it.product_image" fit="cover" @click="router.push('/product/' + it.product_id)" />
          <div class="ci-info">
            <div class="ci-name van-multi-ellipsis--l2">
              <van-tag v-if="isPriceDrop(it)" type="danger" size="mini" class="drop-tag">降价</van-tag>
              {{ it.product_name }}
            </div>
            <div class="ci-bottom">
              <span class="price">¥{{ fmt(it.price) }}</span>
              <van-stepper v-model="it.quantity" :min="1" :max="it.stock" @change="(v) => changeQty(it, v)" />
              <van-icon name="delete-o" size="20" @click="removeItem(it)" />
            </div>
          </div>
        </div>

      <van-submit-bar :price="selectedTotal * 100" button-text="结算" @submit="checkout">
        <van-checkbox :model-value="allSelected" @click="toggleAll">全选</van-checkbox>
        <span class="invert-btn" @click="invertSelection">反选</span>
        <span class="selected-count">已选{{ selectedCount }}件</span>
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
.ci-name .drop-tag { vertical-align: middle; margin-right: 4px; }
.ci-bottom { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
.ci-bottom .price { font-size: 16px; flex: 1; }
/* 猜你喜欢 recommended products section */
.rec-section { background: #fff; margin-bottom: 8px; padding: 12px 12px 8px; }
.rec-title { font-size: 15px; font-weight: 600; margin-bottom: 8px; }
.rec-scroll { display: flex; gap: 10px; overflow-x: auto; padding-bottom: 4px; }
.rec-scroll::-webkit-scrollbar { display: none; }
.rec-card { flex: 0 0 auto; width: 110px; }
.rec-name { font-size: 12px; line-height: 16px; height: 32px; margin-top: 6px; }
.rec-bottom { display: flex; flex-direction: column; align-items: flex-start; gap: 4px; margin-top: 4px; }
.rec-price { color: #e1251b; font-size: 14px; font-weight: 600; }
/* 反选 button + 已选N件 count inside the submit bar */
.invert-btn { margin-left: 10px; padding: 3px 10px; font-size: 12px; color: #e1251b; border: 1px solid #e1251b; border-radius: 12px; cursor: pointer; white-space: nowrap; }
.selected-count { margin-left: 10px; font-size: 12px; color: #999; white-space: nowrap; }
</style>
