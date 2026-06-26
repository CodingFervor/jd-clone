<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showConfirmDialog } from 'vant'
import { getHistory, clearHistory, addToCart } from '../api'

const router = useRouter()
const list = ref([])
const loading = ref(true)

async function load() {
  loading.value = true
  try { list.value = await getHistory(50) } catch (e) { showToast('加载失败') } finally { loading.value = false }
}
onMounted(load)

async function clear() {
  try {
    await showConfirmDialog({ title: '提示', message: '确定清除全部浏览历史？' })
    await clearHistory()
    list.value = []
    showSuccessToast('已清除')
  } catch (e) { /* cancelled */ }
}
async function addCart(h) {
  try { await addToCart(h.product_id, 1); showSuccessToast('已加入购物车') } catch (e) { showToast('请先登录') }
}
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="hist-page">
    <van-nav-bar title="浏览历史" left-arrow @click-left="router.back()" fixed placeholder>
      <template #right>
        <span v-if="list.length" style="color: #e1251b; font-size: 13px" @click="clear">清除</span>
      </template>
    </van-nav-bar>
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!list.length" description="还没有浏览记录" />
    <div v-else class="hist-list">
      <div v-for="h in list" :key="h.id" class="hist-card">
        <van-image width="80" height="80" radius="8" :src="h.product_image" fit="cover" @click="router.push('/product/' + h.product_id)" />
        <div class="hc-info">
          <div class="hc-name van-multi-ellipsis--l2" @click="router.push('/product/' + h.product_id)">{{ h.product_name }}</div>
          <div class="hc-price">¥{{ fmt(h.price) }}</div>
          <van-button size="small" type="danger" plain round @click="addCart(h)">加入购物车</van-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hist-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.hist-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.hc-info { flex: 1; display: flex; flex-direction: column; }
.hc-name { font-size: 14px; line-height: 20px; flex: 1; }
.hc-price { color: #e1251b; font-size: 16px; font-weight: bold; margin: 8px 0; }
</style>
