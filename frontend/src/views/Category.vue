<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getCategories, getProducts } from '../api'

const route = useRoute()
const router = useRouter()
const cats = ref([])
const products = ref([])
const activeId = ref(0)
const loading = ref(false)
const sortBy = ref('default') // default, price_asc, price_desc, sales

onMounted(async () => {
  try {
    cats.value = await getCategories()
    if (cats.value.length) {
      activeId.value = route.query.id ? Number(route.query.id) : cats.value[0].id
    }
  } catch (e) {
    showToast('加载失败')
  }
})

watch(activeId, async (id) => {
  loading.value = true
  try {
    const res = await getProducts({ category_id: id, page: 1, page_size: 50 })
    products.value = res.data
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}, { immediate: true })

const filteredProducts = computed(() => {
  const list = [...products.value]
  if (sortBy.value === 'price_asc') return list.sort((a, b) => a.price - b.price)
  if (sortBy.value === 'price_desc') return list.sort((a, b) => b.price - a.price)
  if (sortBy.value === 'sales') return list.sort((a, b) => b.sales - a.sales)
  return list
})

function goProduct(id) {
  router.push('/product/' + id)
}
function fmt(n) {
  return Number(n).toFixed(2)
}
</script>

<template>
  <div class="cat-page">
    <van-sticky>
      <van-search placeholder="搜索京东商品" shape="round" readonly @click="router.push('/search')" />
    </van-sticky>
    <div class="cat-body">
      <div class="cat-sidebar">
        <div
          v-for="c in cats"
          :key="c.id"
          class="cat-side-item"
          :class="{ active: activeId === c.id }"
          @click="activeId = c.id"
        >
          {{ c.name }}
        </div>
      </div>
      <div class="cat-content">
        <!-- Sort/filter bar -->
        <div class="sort-bar">
          <span :class="{ active: sortBy === 'default' }" @click="sortBy = 'default'">综合</span>
          <span :class="{ active: sortBy === 'sales' }" @click="sortBy = 'sales'">销量</span>
          <span :class="{ active: sortBy === 'price_asc' }" @click="sortBy = 'price_asc'">价格↑</span>
          <span :class="{ active: sortBy === 'price_desc' }" @click="sortBy = 'price_desc'">价格↓</span>
        </div>
        <div v-if="loading" class="loading"><van-loading /></div>
        <div v-else>
          <div v-for="p in filteredProducts" :key="p.id" class="prod-row" @click="goProduct(p.id)">
            <van-image width="100" height="100" radius="6" :src="p.image" fit="cover" />
            <div class="prod-info">
              <div class="prod-name van-multi-ellipsis--l2">{{ p.name }}</div>
              <div class="prod-sub van-ellipsis">{{ p.subtitle }}</div>
              <div class="prod-bottom">
                <span class="prod-price">¥{{ fmt(p.price) }}</span>
                <span class="prod-sales">{{ p.sales }}人付款</span>
              </div>
            </div>
          </div>
          <van-empty v-if="!filteredProducts.length" description="暂无商品" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cat-page { display: flex; flex-direction: column; height: 100vh; }
.cat-body { display: flex; flex: 1; overflow: hidden; }
.cat-sidebar { width: 90px; background: #f7f7f7; overflow-y: auto; }
.cat-side-item { padding: 16px 8px; text-align: center; font-size: 13px; color: #333; position: relative; }
.cat-side-item.active { background: #fff; color: #e1251b; font-weight: bold; }
.cat-side-item.active::before { content: ''; position: absolute; left: 0; top: 50%; transform: translateY(-50%); width: 3px; height: 18px; background: #e1251b; }
.cat-content { flex: 1; background: #fff; overflow-y: auto; }
.sort-bar { display: flex; gap: 0; background: #fff; border-bottom: 1px solid #f0f0f0; position: sticky; top: 0; z-index: 5; }
.sort-bar span { flex: 1; text-align: center; padding: 10px 0; font-size: 13px; color: #666; }
.sort-bar span.active { color: #e1251b; font-weight: bold; }
.prod-row { display: flex; gap: 10px; padding: 10px; border-bottom: 1px solid #f5f5f5; }
.prod-info { flex: 1; display: flex; flex-direction: column; justify-content: space-between; }
.prod-name { font-size: 13px; line-height: 18px; }
.prod-sub { font-size: 11px; color: #999; }
.prod-bottom { display: flex; align-items: baseline; gap: 8px; }
.prod-price { color: #e1251b; font-weight: bold; font-size: 16px; }
.prod-sales { font-size: 11px; color: #999; }
.loading { text-align: center; padding: 40px; }
</style>
