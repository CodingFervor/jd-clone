<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { listFavorites, toggleFavorite, addToCart } from '../api'

const router = useRouter()
const list = ref([])
const loading = ref(true)
// Wishlist share card (收藏夹分享)
const showShare = ref(false)

// Top 3 favorited products (first three of the newest-first list).
const topThree = computed(() => list.value.slice(0, 3))
// Sum of all favorited product prices.
const totalValue = computed(() =>
  list.value.reduce((sum, f) => sum + Number(f.price || 0), 0)
)
// Plain-text share summary.
const shareText = computed(() => {
  const n = list.value.length
  if (!n) return ''
  const lines = list.value.map((f, i) => `${i + 1}. ${f.product_name} ¥${fmt(f.price)}`)
  return `我的京东收藏清单（${n}件好物，总价¥${fmt(totalValue.value)}）：${lines.join(' ')}`
})

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
async function copyShareText() {
  if (!shareText.value) return
  try {
    await navigator.clipboard.writeText(shareText.value)
    showSuccessToast('分享文本已复制')
  } catch (e) {
    showToast('复制失败，请手动复制')
  }
}
async function copyShareLink() {
  try {
    await navigator.clipboard.writeText(window.location.origin + '/#/favorites')
    showSuccessToast('链接已复制')
  } catch (e) {
    showToast('复制失败，请手动复制')
  }
}
</script>

<template>
  <div class="fav-page">
    <van-nav-bar title="我的收藏" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!list.length" description="还没有收藏商品" />
    <div v-else>
      <div class="share-bar">
        <van-button type="danger" round block icon="share-o" @click="showShare = true">分享收藏</van-button>
      </div>
      <div class="fav-list">
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

      <!-- Wishlist share card popup (收藏夹分享) -->
      <van-popup v-model:show="showShare" round closeable position="bottom" :style="{ width: '92%' }">
        <div class="share-wrap">
          <div class="share-card">
            <div class="sc-header">
              <div class="sc-title">我的收藏清单</div>
              <div class="sc-count">{{ list.length }} 件好物</div>
            </div>
            <div class="sc-body">
              <div v-for="(f, i) in topThree" :key="f.id" class="sc-item">
                <div class="sc-rank">{{ i + 1 }}</div>
                <van-image width="56" height="56" radius="6" :src="f.product_image" fit="cover" />
                <div class="sc-info">
                  <div class="sc-name van-multi-ellipsis--l2">{{ f.product_name }}</div>
                  <div class="sc-price">¥{{ fmt(f.price) }}</div>
                </div>
              </div>
              <div v-if="list.length > 3" class="sc-more">还有 {{ list.length - 3 }} 件好物</div>
            </div>
            <div class="sc-footer">
              <div class="sc-total-label">收藏总价</div>
              <div class="sc-total-value">¥{{ fmt(totalValue) }}</div>
            </div>
            <div class="sc-brand">京东 JD.COM</div>
          </div>
          <div class="share-actions">
            <van-button block round type="danger" icon="description" @click="copyShareText">复制分享文本</van-button>
            <van-button block round plain type="danger" icon="link-o" @click="copyShareLink" style="margin-top:10px">复制链接</van-button>
          </div>
        </div>
      </van-popup>
    </div>
  </div>
</template>

<style scoped>
.fav-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.share-bar { padding: 10px 12px 0; }
.fav-card { display: flex; gap: 12px; background: #fff; margin: 8px; border-radius: 8px; padding: 12px; }
.fc-info { flex: 1; display: flex; flex-direction: column; }
.fc-name { font-size: 14px; line-height: 20px; flex: 1; }
.fc-price { color: #e1251b; font-size: 16px; font-weight: bold; margin: 8px 0; }
.fc-actions { display: flex; gap: 8px; }
/* Wishlist share card */
.share-wrap { padding: 20px; }
.share-card { background: #fff; border-radius: 14px; overflow: hidden; box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08); }
.sc-header {
  background: linear-gradient(135deg, #e1251b, #ff6a5c);
  color: #fff;
  padding: 18px 16px;
}
.sc-title { font-size: 18px; font-weight: bold; }
.sc-count { font-size: 13px; opacity: 0.9; margin-top: 4px; }
.sc-body { padding: 12px 16px; }
.sc-item { display: flex; align-items: center; gap: 10px; padding: 8px 0; border-bottom: 1px solid #f5f5f5; }
.sc-item:last-child { border-bottom: none; }
.sc-rank {
  width: 22px; height: 22px; flex-shrink: 0; border-radius: 50%;
  background: #e1251b; color: #fff; font-size: 12px; font-weight: bold;
  display: flex; align-items: center; justify-content: center;
}
.sc-info { flex: 1; min-width: 0; }
.sc-name { font-size: 13px; line-height: 18px; color: #333; }
.sc-price { color: #e1251b; font-size: 14px; font-weight: bold; margin-top: 2px; }
.sc-more { text-align: center; color: #999; font-size: 12px; padding: 10px 0 4px; }
.sc-footer {
  display: flex; justify-content: space-between; align-items: baseline;
  padding: 12px 16px; background: #fff7f6; border-top: 1px dashed #ffd4d0;
}
.sc-total-label { color: #666; font-size: 13px; }
.sc-total-value { color: #e1251b; font-size: 22px; font-weight: bold; }
.sc-brand { text-align: center; color: #e1251b; font-size: 13px; font-weight: bold; padding: 10px 0 16px; }
.share-actions { margin-top: 16px; }
</style>
