<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getBundles, addToCart } from '../api'

const router = useRouter()
const bundles = ref([])
const loading = ref(true)

onMounted(async () => {
  try { bundles.value = await getBundles() } catch (e) { showToast('加载失败') } finally { loading.value = false }
})

async function buyBundle(b) {
  try {
    for (const pid of b.product_ids.split(',')) {
      await addToCart(pid.trim(), 1)
    }
    showSuccessToast('套餐已加入购物车')
    router.push('/cart')
  } catch (e) { showToast('请先登录') }
}
function savePct(b) {
  if (!b.original_total || b.original_total <= b.bundle_price) return 0
  return Math.round((1 - b.bundle_price / b.original_total) * 100)
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="bd-page">
    <van-nav-bar title="超值套餐" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!bundles.length" description="暂无套餐" />
    <div v-else class="bd-list">
      <div class="bd-banner">🎁 组合套餐 · 一次买齐更划算</div>
      <div v-for="b in bundles" :key="b.id" class="bundle-card">
        <div class="bc-title">{{ b.title }} <van-tag type="danger" round>省{{ savePct(b) }}%</van-tag></div>
        <div class="bc-products">
          <div v-for="p in b.products" :key="p.id" class="bp-item" @click="router.push('/product/' + p.id)">
            <van-image width="70" height="70" radius="6" :src="p.image" fit="cover" />
            <div class="bp-name van-ellipsis">{{ p.name }}</div>
            <div class="bp-price">¥{{ fmt(p.price) }}</div>
          </div>
        </div>
        <div class="bc-bottom">
          <div class="bc-price-row">
            <span class="bc-bundle-price">套餐价 ¥{{ fmt(b.bundle_price) }}</span>
            <span class="bc-original">原价 ¥{{ fmt(b.original_total) }}</span>
          </div>
          <van-button size="small" type="danger" round @click="buyBundle(b)">一键购买</van-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bd-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.bd-banner { background: linear-gradient(135deg, #e1251b, #f5574d); color: #fff; text-align: center; padding: 14px; font-size: 16px; font-weight: bold; }
.bundle-card { background: #fff; margin: 8px; border-radius: 10px; padding: 12px; }
.bc-title { font-size: 15px; font-weight: bold; display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.bc-products { display: flex; gap: 10px; overflow-x: auto; }
.bp-item { flex-shrink: 0; width: 80px; }
.bp-name { font-size: 11px; color: #666; margin-top: 4px; }
.bp-price { font-size: 12px; color: #e1251b; }
.bc-bottom { display: flex; justify-content: space-between; align-items: center; margin-top: 12px; padding-top: 10px; border-top: 1px solid #f5f5f5; }
.bc-price-row { display: flex; align-items: baseline; gap: 8px; }
.bc-bundle-price { color: #e1251b; font-size: 20px; font-weight: bold; }
.bc-original { color: #999; font-size: 12px; text-decoration: line-through; }
</style>
