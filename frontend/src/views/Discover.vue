<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProducts } from '../api'

const router = useRouter()
const products = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await getProducts({ page: 1, page_size: 30 })
    products.value = res.data
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})
function fmt(n) { return Number(n).toFixed(2) }
</script>

<template>
  <div class="discover">
    <van-nav-bar title="发现好货" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <div class="masonry">
      <div v-for="p in products" :key="p.id" class="m-card" @click="router.push('/product/' + p.id)">
        <van-image width="100%" height="160" :src="p.image" fit="cover" radius="8" />
        <div class="m-name van-multi-ellipsis--l2">{{ p.name }}</div>
        <div class="m-price">¥{{ fmt(p.price) }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.discover { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.masonry { columns: 2; column-gap: 8px; padding: 8px; }
.m-card { break-inside: avoid; margin-bottom: 8px; background: #fff; border-radius: 8px; padding: 6px; }
.m-name { font-size: 13px; line-height: 18px; padding: 4px; height: 36px; }
.m-price { color: #e1251b; font-weight: bold; padding: 0 4px 4px; }
</style>
