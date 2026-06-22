<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { ftsSearch, ftsSuggest } from '../api'

const router = useRouter()
const keyword = ref('')
const results = ref([])
const history = ref(JSON.parse(localStorage.getItem('jd_search_history') || '[]'))
const searched = ref(false)
const suggestions = ref([])

async function onInput(val) {
  if (!val.trim() || val.length < 1) { suggestions.value = []; return }
  try {
    suggestions.value = await ftsSuggest(val)
  } catch (e) {
    suggestions.value = []
  }
}

async function doSearch(kw) {
  if (!kw.trim()) return
  keyword.value = kw
  suggestions.value = []
  const h = history.value.filter((x) => x !== kw)
  h.unshift(kw)
  history.value = h.slice(0, 10)
  localStorage.setItem('jd_search_history', JSON.stringify(history.value))
  try {
    const res = await ftsSearch(kw)
    results.value = res.data || []
    searched.value = true
  } catch (e) {
    showToast('搜索失败')
  }
}
function clearHistory() {
  history.value = []
  localStorage.removeItem('jd_search_history')
}
function fmt(n) {
  return Number(n).toFixed(2)
}
</script>

<template>
  <div class="search-page">
    <van-sticky>
      <van-search v-model="keyword" placeholder="搜索京东商品(全文搜索)" shape="round" show-action @search="doSearch(keyword)" @update:model-value="onInput">
        <template #action>
          <span @click="doSearch(keyword)">搜索</span>
        </template>
      </van-search>
      <!-- Auto-complete suggestions -->
      <div v-if="suggestions.length" class="suggest-list">
        <div v-for="s in suggestions" :key="s" class="suggest-item" @click="doSearch(s)">
          <van-icon name="search" size="14" /> {{ s }}
        </div>
      </div>
    </van-sticky>

    <div v-if="!searched">
      <div v-if="history.length" class="history">
        <div class="h-head">搜索历史 <van-icon name="delete-o" @click="clearHistory" /></div>
        <div class="h-tags">
          <van-tag v-for="h in history" :key="h" plain round size="medium" @click="doSearch(h)">{{ h }}</van-tag>
        </div>
      </div>
      <div class="hot">
        <div class="h-head">热门搜索</div>
        <div class="h-tags">
          <van-tag v-for="h in ['iPhone 15', '华为', '空调', '笔记本', '球鞋']" :key="h" round type="primary" plain size="medium" @click="doSearch(h)">{{ h }}</van-tag>
        </div>
      </div>
    </div>

    <div v-else>
      <van-empty v-if="!results.length" description="没有找到相关商品" />
      <div class="res-list">
        <div v-for="p in results" :key="p.id" class="res-item" @click="router.push('/product/' + p.id)">
          <van-image width="100" height="100" radius="6" :src="p.image" fit="cover" />
          <div class="ri-info">
            <div class="ri-name van-multi-ellipsis--l2">{{ p.name }}</div>
            <div class="ri-price">¥{{ fmt(p.price) }}</div>
            <div class="ri-sales">{{ p.sales }}人付款</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-page { min-height: 100vh; }
.suggest-list { background: #fff; border-top: 1px solid #eee; }
.suggest-item { padding: 10px 16px; font-size: 14px; color: #333; display: flex; align-items: center; gap: 6px; border-bottom: 1px solid #f5f5f5; }
.suggest-item:active { background: #f5f5f5; }
.history, .hot { padding: 16px; background: #fff; margin-bottom: 8px; }
.h-head { font-size: 14px; color: #666; margin-bottom: 10px; display: flex; justify-content: space-between; }
.h-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.res-item { display: flex; gap: 10px; padding: 10px; background: #fff; border-bottom: 1px solid #f5f5f5; }
.ri-info { flex: 1; }
.ri-name { font-size: 13px; line-height: 18px; }
.ri-price { color: #e1251b; font-size: 16px; font-weight: bold; margin-top: 6px; }
.ri-sales { color: #999; font-size: 11px; }
</style>
