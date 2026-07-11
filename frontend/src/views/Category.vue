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

// Product compare: track up to 2 products for side-by-side comparison.
const compareList = ref([])
const showCompare = ref(false)

function toggleCompare(p) {
  const i = compareList.value.findIndex((x) => x.id === p.id)
  if (i >= 0) {
    compareList.value.splice(i, 1)
    return
  }
  if (compareList.value.length >= 2) {
    showToast('最多对比 2 件商品')
    return
  }
  compareList.value.push(p)
}

function inCompare(p) {
  return compareList.value.some((x) => x.id === p.id)
}

function clearCompare() {
  compareList.value = []
  showCompare.value = false
}

// Which of the two compared products has the cheaper price (for red highlight).
function cheaperIs(idx) {
  if (compareList.value.length < 2) return false
  const a = compareList.value[0]
  const b = compareList.value[1]
  if (idx === 0) return a.price <= b.price
  return b.price < a.price
}

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

// ---- New product & seckill badges (新品限时标签) ----
// A product is "NEW" if created within the last 7 days.
const WEEK_MS = 7 * 24 * 60 * 60 * 1000
function isNew(p) {
  if (!p || !p.created_at) return false
  const created = new Date(p.created_at).getTime()
  if (isNaN(created)) return false
  return Date.now() - created < WEEK_MS
}
function isSeckill(p) {
  return !!(p && p.is_seckill)
}

// ---- Category promo banner carousel (分类页广告条) ----
// 3 demo promo banners shown in a van-swipe at the top of the category content
// area. Each carries a picsum background image plus an overlay with promo text.
// Auto-rotates every 3s; tapping a banner toasts "活动详情".
const promoBanners = [
  { img: 'https://picsum.photos/seed/jdcat1/600/200', tag: '限时特惠', title: '精选好物 5折起', sub: '每日上新 抢先购' },
  { img: 'https://picsum.photos/seed/jdcat2/600/200', tag: '新人专享', title: '领券立减 100元', sub: '首单包邮 限时领' },
  { img: 'https://picsum.photos/seed/jdcat3/600/200', tag: '品牌狂欢', title: '大牌直降 不止5折', sub: '满199减50' },
]
function onBannerTap() {
  showToast('活动详情')
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
        <!-- Category promo banner carousel (分类页广告条) -->
        <van-swipe class="cat-banner" :autoplay="3000" indicator-color="#e1251b" :height="100">
          <van-swipe-item v-for="(b, i) in promoBanners" :key="i" @click="onBannerTap">
            <div class="cb-slide" :style="{ backgroundImage: 'url(' + b.img + ')' }">
              <div class="cb-overlay">
                <span class="cb-tag">{{ b.tag }}</span>
                <div class="cb-title">{{ b.title }}</div>
                <div class="cb-sub">{{ b.sub }}</div>
              </div>
            </div>
          </van-swipe-item>
        </van-swipe>
        <!-- Sort/filter bar -->
        <div class="sort-bar">
          <span :class="{ active: sortBy === 'default' }" @click="sortBy = 'default'">综合</span>
          <span :class="{ active: sortBy === 'sales' }" @click="sortBy = 'sales'">销量</span>
          <span :class="{ active: sortBy === 'price_asc' }" @click="sortBy = 'price_asc'">价格↑</span>
          <span :class="{ active: sortBy === 'price_desc' }" @click="sortBy = 'price_desc'">价格↓</span>
        </div>
        <div v-if="loading" class="loading"><van-loading /></div>
        <div v-else>
          <div
            v-for="p in filteredProducts"
            :key="p.id"
            class="prod-row"
            :class="{ 'prod-comparing': inCompare(p) }"
            @click="goProduct(p.id)"
          >
            <div class="prod-img-wrap">
              <van-image width="100" height="100" radius="6" :src="p.image" fit="cover" />
              <span v-if="isNew(p)" class="prod-badge prod-badge-new">NEW</span>
              <span v-if="isSeckill(p)" class="prod-badge prod-badge-seckill">限时</span>
            </div>
            <div class="prod-info">
              <div class="prod-name van-multi-ellipsis--l2">{{ p.name }}</div>
              <div class="prod-sub van-ellipsis">{{ p.subtitle }}</div>
              <div class="prod-bottom">
                <span class="prod-price">¥{{ fmt(p.price) }}</span>
                <span class="prod-sales">{{ p.sales }}人付款</span>
              </div>
            </div>
            <div class="compare-btn" @click.stop="toggleCompare(p)">
              <span class="cb-box" :class="{ checked: inCompare(p) }">{{ inCompare(p) ? '✓' : '' }}</span>
              <span class="cb-text">对比</span>
            </div>
          </div>
          <van-empty v-if="!filteredProducts.length" description="暂无商品" />
        </div>
      </div>
    </div>

    <!-- Floating compare button: appears once at least one product is picked. -->
    <transition name="cmp-fade">
      <div v-if="compareList.length" class="cmp-fab" @click="showCompare = true">
        <span class="cmp-fab-icon">⇄</span>
        <span class="cmp-fab-text">对比 {{ compareList.length }}/2</span>
      </div>
    </transition>

    <!-- Side-by-side compare dialog. -->
    <van-dialog v-model:show="showCompare" :show-confirm-button="false" class="cmp-dialog">
      <div class="cmp-head">
        <span class="cmp-title">商品对比</span>
        <van-icon name="cross" class="cmp-close" @click="showCompare = false" />
      </div>
      <div class="cmp-table">
        <div class="cmp-col cmp-field-col">
          <div class="cmp-cell cmp-cell-img"></div>
          <div class="cmp-cell cmp-cell-field">商品</div>
          <div class="cmp-cell cmp-cell-field">价格</div>
          <div class="cmp-cell cmp-cell-field">原价</div>
          <div class="cmp-cell cmp-cell-field">店铺</div>
          <div class="cmp-cell cmp-cell-field">销量</div>
          <div class="cmp-cell cmp-cell-field">库存</div>
        </div>
        <div v-for="(p, idx) in compareList" :key="p.id" class="cmp-col">
          <div class="cmp-cell cmp-cell-img">
            <van-image width="70" height="70" radius="6" :src="p.image" fit="cover" />
          </div>
          <div class="cmp-cell cmp-cell-val cmp-cell-name van-multi-ellipsis--l2">{{ p.name }}</div>
          <div class="cmp-cell cmp-cell-val" :class="{ 'cmp-cheaper': cheaperIs(idx) }">¥{{ fmt(p.price) }}</div>
          <div class="cmp-cell cmp-cell-val cmp-orig">¥{{ fmt(p.original_price) }}</div>
          <div class="cmp-cell cmp-cell-val">{{ p.shop || '-' }}</div>
          <div class="cmp-cell cmp-cell-val">{{ p.sales }}人付款</div>
          <div class="cmp-cell cmp-cell-val">{{ p.stock }}件</div>
        </div>
        <!-- Placeholder column when only one product is selected. -->
        <div v-if="compareList.length < 2" class="cmp-col cmp-empty">
          <div class="cmp-cell cmp-cell-img cmp-placeholder">+</div>
          <div class="cmp-cell cmp-cell-val cmp-placeholder-text">添加第二件</div>
        </div>
      </div>
      <div class="cmp-actions">
        <van-button plain block round @click="clearCompare">清空</van-button>
        <van-button type="danger" block round @click="showCompare = false">关闭</van-button>
      </div>
    </van-dialog>
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

/* ---- Category promo banner carousel (分类页广告条) ---- */
.cat-banner { margin: 8px; border-radius: 10px; overflow: hidden; }
.cb-slide {
  width: 100%;
  height: 100px;
  background-size: cover;
  background-position: center;
  position: relative;
  cursor: pointer;
}
.cb-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, rgba(0, 0, 0, 0.55) 0%, rgba(0, 0, 0, 0.25) 60%, rgba(0, 0, 0, 0.1) 100%);
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  color: #fff;
}
.cb-tag {
  align-self: flex-start;
  background: #e1251b;
  color: #fff;
  font-size: 10px;
  font-weight: bold;
  padding: 1px 7px;
  border-radius: 8px;
  line-height: 1.6;
}
.cb-title { font-size: 17px; font-weight: bold; text-shadow: 0 1px 3px rgba(0, 0, 0, 0.5); }
.cb-sub { font-size: 12px; opacity: 0.92; text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5); }
.sort-bar { display: flex; gap: 0; background: #fff; border-bottom: 1px solid #f0f0f0; position: sticky; top: 0; z-index: 5; }
.sort-bar span { flex: 1; text-align: center; padding: 10px 0; font-size: 13px; color: #666; }
.sort-bar span.active { color: #e1251b; font-weight: bold; }
.prod-row { display: flex; gap: 10px; padding: 10px; border-bottom: 1px solid #f5f5f5; }
.prod-info { flex: 1; display: flex; flex-direction: column; justify-content: space-between; }
.prod-img-wrap { position: relative; flex-shrink: 0; }
/* New product & seckill badges (新品限时标签) */
.prod-badge {
  position: absolute;
  left: 6px;
  z-index: 2;
  color: #fff;
  font-size: 11px;
  font-weight: bold;
  padding: 2px 7px;
  border-radius: 20px;
  line-height: 1.4;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.2);
}
.prod-badge-new { top: 6px; background: #e1251b; }
.prod-badge-seckill { top: 6px; background: #ff7a18; }
/* When both badges are present, drop the second one below the first. */
.prod-badge-new ~ .prod-badge-seckill { top: 30px; }
.prod-name { font-size: 13px; line-height: 18px; }
.prod-sub { font-size: 11px; color: #999; }
.prod-bottom { display: flex; align-items: baseline; gap: 8px; }
.prod-price { color: #e1251b; font-weight: bold; font-size: 16px; }
.prod-sales { font-size: 11px; color: #999; }
.loading { text-align: center; padding: 40px; }

/* ---- Product compare ---- */
.prod-row { position: relative; }
.compare-btn { position: absolute; right: 8px; top: 8px; display: flex; align-items: center; gap: 3px; font-size: 11px; color: #888; padding: 3px 6px; border-radius: 10px; background: rgba(255,255,255,0.85); z-index: 2; }
.cb-box { width: 14px; height: 14px; border: 1px solid #ccc; border-radius: 3px; display: inline-flex; align-items: center; justify-content: center; font-size: 11px; color: #fff; line-height: 1; }
.cb-box.checked { background: #e1251b; border-color: #e1251b; }
.prod-comparing { border-left: 3px solid #e1251b; box-shadow: inset 2px 0 0 #e1251b; }
.prod-comparing .compare-btn { color: #e1251b; }

/* Floating compare button */
.cmp-fab { position: fixed; left: 50%; transform: translateX(-50%); bottom: 70px; background: #e1251b; color: #fff; border-radius: 999px; padding: 10px 22px; display: flex; align-items: center; gap: 6px; font-size: 14px; font-weight: 600; box-shadow: 0 4px 14px rgba(225,37,27,0.4); z-index: 100; }
.cmp-fab-icon { font-size: 18px; }
.cmp-fade-enter-active, .cmp-fade-leave-active { transition: opacity 0.2s, transform 0.2s; }
.cmp-fade-enter-from, .cmp-fade-leave-to { opacity: 0; transform: translateX(-50%) translateY(10px); }

/* Compare dialog */
.cmp-dialog { width: 94%; max-width: 420px; }
.cmp-head { display: flex; justify-content: space-between; align-items: center; padding: 14px 16px 6px; }
.cmp-title { font-size: 16px; font-weight: 600; }
.cmp-close { font-size: 18px; color: #999; }
.cmp-table { display: flex; overflow-x: auto; padding: 4px 12px 12px; }
.cmp-col { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.cmp-field-col { flex: 0 0 52px; }
.cmp-cell { padding: 6px 4px; font-size: 12px; min-height: 40px; display: flex; align-items: center; }
.cmp-cell-img { min-height: 82px; }
.cmp-cell-field { color: #999; justify-content: flex-start; }
.cmp-cell-val { color: #333; word-break: break-all; }
.cmp-cell-name { line-height: 16px; }
.cmp-orig { color: #999; text-decoration: line-through; }
.cmp-cheaper { color: #e1251b; font-weight: 700; font-size: 15px; }
.cmp-placeholder { width: 70px; height: 70px; border: 1px dashed #ddd; border-radius: 6px; color: #ccc; font-size: 28px; display: flex; align-items: center; justify-content: center; }
.cmp-placeholder-text { color: #ccc; }
.cmp-actions { display: flex; gap: 10px; padding: 0 16px 16px; }
</style>
