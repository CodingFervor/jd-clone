<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { listFavorites, toggleFavorite, addToCart } from '../api'

const router = useRouter()
const list = ref([])
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    list.value = await listFavorites()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function remove(f) {
  try {
    const res = await toggleFavorite(f.product_id)
    if (!res.favorited) {
      list.value = list.value.filter((x) => x.id !== f.id)
      showSuccessToast('已取消收藏')
    }
  } catch (e) {
    showToast('操作失败')
  }
}
async function addCart(f) {
  try {
    await addToCart(f.product_id, 1)
    showSuccessToast('已加入购物车')
  } catch (e) {
    showToast('请先登录')
  }
}
function fmt(n) {
  return Number(n).toFixed(2)
}
</script>

<template>
  <div class="fav-page">
    <van-nav-bar title="我的收藏" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!list.length" description="还没有收藏商品" />
    <div v-else class="fav-list">
      <div v-for="f in list" :key="f.id" class="fav-card">
        <van-image width="90" height="90" radius="8" :src="f.product_image" fit="cover" @click="router.push('/product/' + f.product_id)" />
        <div class="fc-info">
          <div class="fc-name van-multi-ellipsis--l2" @click="router.push('/product/' + f.product_id)">{{ f.product_name }}</div>
          <div class="fc-price">¥{{ fmt(f.price) }}</div>
          <div class="fc-actions">
            <van-button size="small" plain round @click="remove(f)">取消收藏</van-button>
            <van-button size="small" type="danger" round @click="addCart(f)">加入购物车</van-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fav-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.fav-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.fc-info { flex: 1; display: flex; flex-direction: column; }
.fc-name { font-size: 14px; line-height: 20px; flex: 1; }
.fc-price { color: #e1251b; font-size: 16px; font-weight: bold; margin: 8px 0; }
.fc-actions { display: flex; gap: 8px; }
</style>
