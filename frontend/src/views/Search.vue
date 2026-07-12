<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { showToast } from 'vant'
import { ftsSearch, ftsSuggest } from '../api'

const router = useRouter()
const route = useRoute()
const keyword = ref('')
const results = ref([])
// History is stored as a list of { text, count, last } objects so the tag
// cloud can scale font size by search count and color by recency. Old
// string-only entries (from prior versions) are migrated on load.
function loadHistory() {
  let raw = []
  try {
    raw = JSON.parse(localStorage.getItem('jd_search_history') || '[]')
  } catch {
    raw = []
  }
  // Migrate legacy string entries into objects.
  return raw.map((item) => {
    if (typeof item === 'string') return { text: item, count: 1, last: 0 }
    return item
  })
}
const history = ref(loadHistory())
const searched = ref(false)
const suggestions = ref([])
const focused = ref(false)

// ---- Feature: 搜索历史云图 (Smart Search History Cloud) ----
// Build the tag-cloud model from the history list. Each tag gets:
//   - fontSize scaled by its search count relative to the max count
//   - color: red for the most recent searches, gray for older ones
//   - overflow: only the top 15 tags render at normal size; the rest show in a
//     smaller font below so the cloud stays readable.
const MAX_TAGS = 15
const cloudTags = computed(() => {
  const list = history.value.slice()
  if (!list.length) return { primary: [], overflow: [] }
  const nowTs = Date.now()
  const maxCount = Math.max(1, ...list.map((h) => h.count || 1))
  // Sort by recency (most recent first) for stable ordering of the primary set.
  const sorted = list.slice().sort((a, b) => (b.last || 0) - (a.last || 0))
  return {
    primary: sorted.slice(0, MAX_TAGS).map((h, idx) => {
      const cnt = h.count || 1
      // Font size ranges from 13px (count=1) to 24px (count=max), scaled
      // linearly so frequent terms stand out.
      const size = 13 + Math.round((cnt / maxCount) * 11)
      // Recency window: searched within the last 3 days is "recent" (red).
      const ageMs = nowTs - (h.last || 0)
      const recent = h.last > 0 && ageMs < 3 * 24 * 60 * 60 * 1000
      return {
        text: h.text,
        count: cnt,
        size,
        // Top 5 most-recent are always red; otherwise based on the 3-day window.
        color: idx < 5 || recent ? '#e1251b' : '#999',
        weight: idx < 3 ? 'bold' : 'normal',
      }
    }),
    overflow: sorted.slice(MAX_TAGS).map((h) => ({
      text: h.text,
      count: h.count || 1,
      size: 11,
      color: '#bbb',
      weight: 'normal',
    })),
  }
})

// ---- Voice search (语音搜索) ----
// Uses the Web Speech API when available. While listening the mic icon pulses.
const listening = ref(false)
let recognition = null

function isSpeechSupported() {
  return typeof window !== 'undefined' &&
    !!(window.SpeechRecognition || window.webkitSpeechRecognition)
}

function startVoice() {
  if (!isSpeechSupported()) {
    showToast('当前浏览器不支持语音搜索')
    return
  }
  // If already listening, stop instead of starting a second session.
  if (listening.value && recognition) {
    recognition.stop()
    return
  }
  const SR = window.SpeechRecognition || window.webkitSpeechRecognition
  recognition = new SR()
  recognition.lang = 'zh-CN'
  recognition.interimResults = false
  recognition.maxAlternatives = 1
  recognition.onstart = () => {
    listening.value = true
  }
  recognition.onend = () => {
    listening.value = false
  }
  recognition.onerror = (ev) => {
    listening.value = false
    if (ev.error === 'not-allowed' || ev.error === 'service-not-allowed') {
      showToast('请允许麦克风权限')
    } else if (ev.error !== 'no-speech' && ev.error !== 'aborted') {
      showToast('语音识别失败')
    }
  }
  recognition.onresult = (ev) => {
    const text = ev.results?.[0]?.[0]?.transcript || ''
    if (text.trim()) {
      keyword.value = text.trim()
      // Auto-run the search with the recognized text.
      doSearch(keyword.value)
    }
  }
  try {
    recognition.start()
  } catch (e) {
    listening.value = false
    showToast('语音启动失败')
  }
}

onUnmounted(() => {
  if (suggestTimer) clearTimeout(suggestTimer)
  // Clean up any active recognition session when leaving the page.
  if (recognition) {
    try { recognition.stop() } catch (_) {}
    recognition = null
  }
})

// Popular search terms (hot searches). Used as clickable tags above results.
// Each entry carries a deterministic "热度" count so the hottest terms can show
// an animated flame icon.
const HOT_RAW = ['iPhone 15', '华为', '空调', '笔记本', '球鞋', '扫地机器人', '蓝牙耳机', '冰箱']
const hotSearches = HOT_RAW.map((text, i) => {
  // Deterministic hot count in [120, 9800] from a simple index-based hash.
  const seed = (i + 1) * 2654435761
  const count = 120 + (Math.abs(seed) % 9680)
  return { text, count }
})
// Threshold above which a hot term gets the flame icon.
const HOT_FLAME_THRESHOLD = 2000
function isHot(h) {
  return (h.count || 0) >= HOT_FLAME_THRESHOLD
}

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
  // Persist to localStorage history as { text, count, last } objects so the
  // tag cloud can scale font size by count and color by recency. Dedup by
  // text, increment the count, and bump the last-searched timestamp. Keep the
  // most recent 40 entries so the 15-tag cap + overflow have data to draw from.
  const h = history.value.slice()
  const idx = h.findIndex((x) => x.text === kw)
  if (idx >= 0) {
    h[idx] = { text: kw, count: (h[idx].count || 1) + 1, last: Date.now() }
  } else {
    h.unshift({ text: kw, count: 1, last: Date.now() })
  }
  history.value = h.slice(0, 40)
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
        <template #left-icon>
          <span
            class="voice-btn"
            :class="{ pulsing: listening }"
            @mousedown.prevent="startVoice"
            title="语音搜索"
          >🎤</span>
        </template>
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
          <van-button size="mini" plain hairline @click="clearHistory">清空</van-button>
        </div>
        <!-- Feature: 搜索历史云图 — varying-sized tag cloud, color by recency -->
        <div class="cloud">
          <span
            v-for="t in cloudTags.primary"
            :key="t.text"
            class="cloud-tag"
            :style="{ fontSize: t.size + 'px', color: t.color, fontWeight: t.weight }"
            @mousedown.prevent="doSearch(t.text)"
          >{{ t.text }}</span>
          <span
            v-for="t in cloudTags.overflow"
            :key="'o-' + t.text"
            class="cloud-tag cloud-overflow"
            :style="{ fontSize: t.size + 'px', color: t.color, fontWeight: t.weight }"
            @mousedown.prevent="doSearch(t.text)"
          >{{ t.text }}</span>
        </div>
      </div>
      <div class="hot">
        <div class="h-head">热门搜索</div>
        <div class="h-tags">
          <van-tag v-for="h in hotSearches" :key="h.text" round type="primary" plain size="medium" @mousedown.prevent="doSearch(h.text)">
            <span v-if="isHot(h)" class="hot-flame">🔥</span>{{ h.text }}
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
          <van-button size="mini" plain hairline @click="clearHistory">清空</van-button>
        </div>
        <!-- Feature: 搜索历史云图 — varying-sized tag cloud, color by recency -->
        <div class="cloud">
          <span
            v-for="t in cloudTags.primary"
            :key="t.text"
            class="cloud-tag"
            :style="{ fontSize: t.size + 'px', color: t.color, fontWeight: t.weight }"
            @click="doSearch(t.text)"
          >{{ t.text }}</span>
          <span
            v-for="t in cloudTags.overflow"
            :key="'o-' + t.text"
            class="cloud-tag cloud-overflow"
            :style="{ fontSize: t.size + 'px', color: t.color, fontWeight: t.weight }"
            @click="doSearch(t.text)"
          >{{ t.text }}</span>
        </div>
      </div>
      <div class="hot">
        <div class="h-head">热门搜索</div>
        <div class="h-tags">
          <van-tag v-for="h in hotSearches" :key="h.text" round type="primary" plain size="medium" @click="doSearch(h.text)"><span v-if="isHot(h)" class="hot-flame">🔥</span>{{ h.text }}</van-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-page { min-height: 100vh; }
/* ---- Voice search mic button (语音搜索) ---- */
.voice-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  user-select: none;
  padding: 0 4px;
  transition: transform 0.15s ease;
}
.voice-btn:active { transform: scale(1.2); }
/* Pulsing animation while listening. */
.voice-btn.pulsing {
  color: #e1251b;
  animation: voice-pulse 1s ease-in-out infinite;
}
@keyframes voice-pulse {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.4); opacity: 0.6; }
}
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
/* Feature: 搜索历史云图 (Smart Search History Cloud) */
.cloud {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 4px 10px;
  line-height: 1.8;
}
.cloud-tag {
  cursor: pointer;
  display: inline-block;
  padding: 2px 4px;
  border-radius: 4px;
  transition: background 0.15s ease, transform 0.15s ease;
  user-select: none;
}
.cloud-tag:active {
  background: #fff5f5;
  transform: scale(1.06);
}
.cloud-overflow { opacity: 0.75; }
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

/* Feature: 热门搜索火焰图标 (Search Trending Fire Icon) */
.hot-flame {
  display: inline-block;
  margin-right: 3px;
  font-size: 14px;
  line-height: 1;
  vertical-align: -2px;
  animation: hot-flame-flicker 0.9s ease-in-out infinite;
  transform-origin: bottom center;
}
@keyframes hot-flame-flicker {
  0%, 100% { transform: scale(1) rotate(-4deg); opacity: 1; }
  50% { transform: scale(1.25) rotate(4deg); opacity: 0.85; }
}
</style>
