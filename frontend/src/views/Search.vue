<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { showToast } from 'vant'
import { ftsSearch, ftsSuggest } from '../api'

const router = useRouter()
const route = useRoute()
const keyword = ref('')
const results = ref([])
const history = ref(JSON.parse(localStorage.getItem('jd_search_history') || '[]'))
const searched = ref(false)
const suggestions = ref([])
const focused = ref(false)

// Popular search terms (hot searches). Used as clickable tags above results.
const hotSearches = ['iPhone 15', '华为', '空调', '笔记本', '球鞋', '扫地机器人', '蓝牙耳机', '冰箱']

// The discover panel (history + hot searches) shows when the input is focused
// or the user is typing and hasn't committed a search yet.
const showDiscover = computed(() => focused.value && !searched.value)

// Debounced real-time suggestions: 300ms after the user stops typing.
let suggestTimer = null
function debounceSuggest(val) {
  if (suggestTimer) clearTimeout(suggestTimer)
  if (!val || !val.trim()) {
    suggestions.value = []
    return
  }
  suggestTimer = setTimeout(async () => {
    try {
      suggestions.value = await ftsSuggest(val)
    } catch (e) {
      suggestions.value = []
    }
  }, 300)
}

function onInput(val) {
  debounceSuggest(val)
}

function onFocus() {
  focused.value = true
}

function onBlur() {
  // Delay so a click on a suggestion/tag can register before we collapse.
  setTimeout(() => {
    focused.value = false
  }, 200)
}

async function doSearch(kw) {
  if (!kw || !kw.trim()) return
  keyword.value = kw
  suggestions.value = []
  focused.value = false
  if (suggestTimer) clearTimeout(suggestTimer)
  // Persist to localStorage history (last 10, newest first, dedup).
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

function backToDiscover() {
  // Return to the suggestion/history panel without dropping the current text.
  searched.value = false
  focused.value = true
  results.value = []
}

onMounted(() => {
  // Support deep-links with ?q=<keyword> (e.g. from the 热门标签 feed on
  // the home page): pre-fill and run the search immediately.
  const q = route.query.q
  if (q && String(q).trim()) {
    doSearch(String(q).trim())
  }
})
onUnmounted(() => {
  if (suggestTimer) clearTimeout(suggestTimer)
})

function fmt(n) {
  return Number(n).toFixed(2)
}
</script>

<template>
  <div class="search-page">
    <van-sticky>
      <van-search
        v-model="keyword"
        placeholder="搜索京东商品(全文搜索)"
        shape="round"
        show-action
        @search="doSearch(keyword)"
        @update:model-value="onInput"
        @focus="onFocus"
        @blur="onBlur"
      >
        <template #action>
          <span @click="doSearch(keyword)">搜索</span>
        </template>
      </van-search>
      <!-- Auto-complete suggestions (real-time, debounced) -->
      <div v-if="suggestions.length" class="suggest-list">
        <div v-for="s in suggestions" :key="s" class="suggest-item" @mousedown.prevent="doSearch(s)">
          <van-icon name="search" size="14" /> {{ s }}
        </div>
      </div>
    </van-sticky>

    <!-- Discover panel: shown when focused and no committed search yet. -->
    <div v-if="showDiscover">
      <div v-if="history.length" class="history">
        <div class="h-head">
          <span>搜索历史</span>
          <van-button size="mini" plain hairline @click="clearHistory">清空历史</van-button>
        </div>
        <div class="h-tags">
          <van-tag v-for="h in history" :key="h" plain round size="medium" @mousedown.prevent="doSearch(h)">{{ h }}</van-tag>
        </div>
      </div>
      <div class="hot">
        <div class="h-head">热门搜索</div>
        <div class="h-tags">
          <van-tag v-for="h in hotSearches" :key="h" round type="primary" plain size="medium" @mousedown.prevent="doSearch(h)">
            {{ h }}
          </van-tag>
        </div>
      </div>
    </div>

    <!-- Search results -->
    <div v-else-if="searched">
      <div class="results-bar">
        <span class="rb-count">共 {{ results.length }} 件相关商品</span>
        <van-button size="mini" plain hairline @click="backToDiscover">返回搜索</van-button>
      </div>
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

    <!-- Default landing (not focused, no search yet) -->
    <div v-else>
      <div v-if="history.length" class="history">
        <div class="h-head">
          <span>搜索历史</span>
          <van-button size="mini" plain hairline @click="clearHistory">清空历史</van-button>
        </div>
        <div class="h-tags">
          <van-tag v-for="h in history" :key="h" plain round size="medium" @click="doSearch(h)">{{ h }}</van-tag>
        </div>
      </div>
      <div class="hot">
        <div class="h-head">热门搜索</div>
        <div class="h-tags">
          <van-tag v-for="h in hotSearches" :key="h" round type="primary" plain size="medium" @click="doSearch(h)">{{ h }}</van-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-page { min-height: 100vh; }
.suggest-list { background: #fff; border-top: 1px solid #eee; }
.suggest-item {
  padding: 10px 16px;
  font-size: 14px;
  color: #333;
  display: flex;
  align-items: center;
  gap: 6px;
  border-bottom: 1px solid #f5f5f5;
}
.suggest-item:active { background: #f5f5f5; }
.history, .hot { padding: 16px; background: #fff; margin-bottom: 8px; }
.h-head {
  font-size: 14px;
  color: #666;
  margin-bottom: 10px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.h-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.results-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: #fff;
  font-size: 13px;
  color: #666;
  border-bottom: 1px solid #f5f5f5;
}
.rb-count { color: #e1251b; font-weight: bold; }
.res-item { display: flex; gap: 10px; padding: 10px; background: #fff; border-bottom: 1px solid #f5f5f5; }
.ri-info { flex: 1; }
.ri-name { font-size: 13px; line-height: 18px; }
.ri-price { color: #e1251b; font-size: 16px; font-weight: bold; margin-top: 6px; }
.ri-sales { color: #999; font-size: 11px; }
</style>
