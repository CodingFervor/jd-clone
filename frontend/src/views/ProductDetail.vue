<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showSuccessToast, showToast, showDialog } from 'vant'
import { getProduct, addToCart, createOrder, createReview, uploadImage, checkFavorite, toggleFavorite, replyReview, getPriceHistory } from '../api'

const route = useRoute()
const router = useRouter()
const product = ref(null)
const reviews = ref([])
const skus = ref([])
const selectedSKU = ref(null)
const recommendedSKU = ref(null)
// Price history (比价历史)
const priceHistory = ref([])
const priceStats = ref(null)
const showPriceHistory = ref(false)
const loading = ref(true)
const showReview = ref(false)
const reviewRating = ref(5)
const reviewContent = ref('')
const reviewImages = ref([])
const favorited = ref(false)
// Build the gallery list from the product's images field (comma-separated),
// falling back to the single main image.
const gallery = computed(() => {
  if (!product.value) return []
  const imgs = (product.value.images || '').split(',').map((s) => s.trim()).filter(Boolean)
  if (imgs.length) return imgs
  return product.value.image ? [product.value.image] : []
})
async function onUploadReviewImage(item) {
  try {
    const res = await uploadImage(item.file)
    reviewImages.value.push(res.url)
  } catch (e) {
    showToast('图片上传失败')
  }
}
function removeReviewImage(idx) {
  reviewImages.value.splice(idx, 1)
}

onMounted(async () => {
  try {
    const res = await getProduct(route.params.id)
    product.value = res.data
    reviews.value = res.reviews || []
    skus.value = res.skus || []
    // Auto-select the recommended (best-value) SKU.
    if (res.recommended_sku) {
      selectedSKU.value = res.recommended_sku
      recommendedSKU.value = res.recommended_sku
    }
    if (localStorage.getItem('jd_token')) {
      favorited.value = await checkFavorite(route.params.id)
    }
    // Load price history (best-effort).
    getPriceHistory(route.params.id).then((d) => { priceHistory.value = d.data || []; priceStats.value = d.stats }).catch(() => {})
  } catch (e) {
    showToast('商品不存在')
  } finally {
    loading.value = false
  }
})

async function doFavorite() {
  if (!checkLogin()) return
  try {
    const res = await toggleFavorite(product.value.id)
    favorited.value = res.favorited
    showSuccessToast(res.favorited ? '已收藏' : '已取消收藏')
  } catch (e) {
    showToast(e.response?.data?.error || '操作失败')
  }
}

function selectSKU(sku) {
  selectedSKU.value = sku
}
// Effective price: the chosen SKU's price, else the product's.
function currentPrice() {
  return selectedSKU.value ? selectedSKU.value.price : (product.value ? product.value.price : 0)
}

async function doAddCart() {
  if (!checkLogin()) return
  try {
    await addToCart(product.value.id, 1)
    showSuccessToast('已加入购物车')
  } catch (e) {
    showToast(e.response?.data?.error || '失败')
  }
}
async function buyNow() {
  if (!checkLogin()) return
  try {
    const order = await createOrder({ items: [{ product_id: product.value.id, quantity: 1 }], address: '' })
    showSuccessToast('下单成功')
    router.push('/orders')
  } catch (e) {
    showToast(e.response?.data?.error || '下单失败')
  }
}
function checkLogin() {
  if (!localStorage.getItem('jd_token')) {
    showDialog({ title: '提示', message: '请先登录' }).then(() => router.push('/login'))
    return false
  }
  return true
}
async function submitReview() {
  if (!reviewContent.value.trim()) {
    showToast('请输入评价内容')
    return
  }
  try {
    const rv = await createReview({ product_id: product.value.id, rating: reviewRating.value, content: reviewContent.value, images: reviewImages.value.join(',') })
    reviews.value.unshift(rv)
    showReview.value = false
    reviewContent.value = ''
    reviewImages.value = []
    showSuccessToast('评价成功')
  } catch (e) {
    showToast('请先登录')
  }
}
const replyingTo = ref(null)
const replyText = ref('')
function toggleReply(r) {
  replyingTo.value = replyingTo.value === r.id ? null : r.id
  replyText.value = ''
}
async function submitReply(r) {
  if (!replyText.value.trim()) { showToast('请输入回复内容'); return }
  try {
    const rep = await replyReview(r.id, replyText.value)
    r.reply = rep
    replyingTo.value = null
    replyText.value = ''
    showSuccessToast('回复成功')
  } catch (e) {
    showToast('请先登录')
  }
}
function fmt(n) {
  return Number(n).toFixed(2)
}
// Map price-history points to bar heights (0-100%) relative to the range.
function priceBars() {
  if (!priceHistory.value.length) return []
  const prices = priceHistory.value.map((p) => p.price)
  const min = Math.min(...prices)
  const max = Math.max(...prices)
  const range = max - min || 1
  return priceHistory.value.map((p) => ({
    price: p.price,
    height: 30 + Math.round(((p.price - min) / range) * 70), // 30-100%
    date: String(p.recorded_at).slice(5, 10), // MM-DD
  }))
}
function priceTrend() {
  if (!priceHistory.value.length || priceHistory.value.length < 2) return 'flat'
  const first = priceHistory.value[0].price
  const last = priceHistory.value[priceHistory.value.length - 1].price
  if (last < first) return 'down'
  if (last > first) return 'up'
  return 'flat'
}
</script>

<template>
  <div v-if="loading" class="loading"><van-loading /></div>
  <div v-else-if="product" class="detail">
    <van-nav-bar title="商品详情" left-arrow @click-left="router.back()" fixed placeholder />
    <!-- Product intro video (商品视频介绍) -->
    <div v-if="product.video_url" class="product-video">
      <video :src="product.video_url" controls preload="metadata" class="pv-player" poster=""></video>
    </div>
    <!-- Image gallery carousel: uses the multi-image field, falls back to the main image -->
    <van-swipe class="gallery" :autoplay="3000" indicator-color="#e1251b" v-if="gallery.length > 1">
      <van-swipe-item v-for="(img, i) in gallery" :key="i">
        <van-image width="100%" height="375" :src="img" fit="cover" />
      </van-swipe-item>
    </van-swipe>
    <van-image v-else width="100%" height="375" :src="product.image" fit="cover" />
    <div class="price-block">
      <span class="big-price">¥{{ fmt(currentPrice()) }}</span>
      <span class="origin">¥{{ fmt(product.original_price) }}</span>
    </div>
    <!-- SKU spec selector -->
    <div v-if="skus.length" class="sku-block">
      <div class="sku-title">
        已选：<b>{{ selectedSKU ? selectedSKU.spec_text : '请选择规格' }}</b>
        <van-tag v-if="recommendedSKU && selectedSKU && selectedSKU.id === recommendedSKU.id" type="danger" round size="mini" style="margin-left:6px">AI推荐·性价比</van-tag>
      </div>
      <div class="sku-tags">
        <span
          v-for="s in skus"
          :key="s.id"
          class="sku-tag"
          :class="{ active: selectedSKU && selectedSKU.id === s.id }"
          @click="selectSKU(s)"
        >{{ s.spec_text }} <small>¥{{ fmt(s.price) }}</small></span>
      </div>
    </div>
    <div class="title-block">
      <h2 class="p-title">{{ product.name }}</h2>
      <p class="p-sub">{{ product.subtitle }}</p>
    </div>
    <van-cell-group inset>
      <van-cell title="店铺" :value="product.shop" is-link @click="router.push('/shop/' + encodeURIComponent(product.shop))" />
      <van-cell title="销量" :value="product.sales + '人付款'" />
      <van-cell title="标签" :value="product.tags || '京东自营'" />
    </van-cell-group>
    <!-- Price history (比价历史) -->
    <div v-if="priceHistory.length" class="price-history">
      <div class="ph-head">
        <span>📈 比价历史</span>
        <span v-if="priceStats" class="ph-stats">
          最低 <b class="green">¥{{ fmt(priceStats.lowest) }}</b>
          <span v-if="priceTrend() === 'down'" class="trend down">↓降价</span>
          <span v-else-if="priceTrend() === 'up'" class="trend up">↑涨价</span>
          <span v-else class="trend flat">→平稳</span>
        </span>
      </div>
      <div class="ph-chart">
        <div v-for="(b, i) in priceBars()" :key="i" class="ph-bar-col">
          <div class="ph-bar" :style="{ height: b.height + '%' }"></div>
          <span class="ph-date">{{ b.date }}</span>
        </div>
      </div>
    </div>
    <div v-if="product.description" class="desc">
      <h3>商品详情</h3>
      <p>{{ product.description }}</p>
    </div>
    <div class="reviews">
      <div class="rev-head">
        <span>商品评价 ({{ reviews.length }})</span>
        <van-button size="mini" type="danger" plain @click="showReview = true">写评价</van-button>
      </div>
      <div v-for="r in reviews" :key="r.id" class="rev-item">
        <div class="rev-user">
          <span>{{ r.username }}</span>
          <van-rate v-model="r.rating" readonly size="12" />
          <span class="rev-reply-btn" @click="toggleReply(r)">回复</span>
        </div>
        <div class="rev-content">{{ r.content }}</div>
        <div v-if="r.images" class="rev-photos">
          <van-image v-for="(img, i) in r.images.split(',')" :key="i" width="72" height="72" radius="6" :src="img" fit="cover" />
        </div>
        <div v-if="r.reply" class="rev-reply">
          <span class="rev-reply-name">{{ r.reply.username }}：</span>{{ r.reply.content }}
        </div>
        <div v-if="replyingTo === r.id" class="rev-reply-box">
          <van-field v-model="replyText" placeholder="写下你的回复..." />
          <van-button size="small" type="danger" @click="submitReply(r)">发送</van-button>
        </div>
      </div>
      <van-empty v-if="!reviews.length" description="暂无评价" />
    </div>

    <!-- Bottom action bar -->
    <van-action-bar>
      <van-action-bar-icon icon="chat-o" text="客服" @click="showToast('客服功能为演示')" />
      <van-action-bar-icon :icon="favorited ? 'star' : 'star-o'" :text="favorited ? '已收藏' : '收藏'" :color="favorited ? '#e1251b' : '#323233'" @click="doFavorite" />
      <van-action-bar-icon icon="cart-o" text="购物车" @click="router.push('/cart')" />
      <van-action-bar-button color="#ff976a" type="warning" text="加入购物车" @click="doAddCart" />
      <van-action-bar-button color="#e1251b" type="danger" text="立即购买" @click="buyNow" />
    </van-action-bar>

    <!-- Review popup -->
    <van-popup v-model:show="showReview" position="bottom" round closeable>
      <div class="rev-form">
        <h3>写评价</h3>
        <van-rate v-model="reviewRating" />
        <van-field v-model="reviewContent" type="textarea" placeholder="说说你的使用感受" rows="3" />
        <div class="rev-upload">
          <van-uploader :after-read="onUploadReviewImage" accept="image/*" multiple :preview-image="false">
            <van-button icon="photo-o" size="small" plain round>添加晒图</van-button>
          </van-uploader>
          <div v-if="reviewImages.length" class="rev-imgs">
            <div v-for="(img, i) in reviewImages" :key="i" class="rev-img-wrap">
              <van-image width="60" height="60" radius="6" :src="img" fit="cover" />
              <van-icon name="cross" class="rev-img-del" @click="removeReviewImage(i)" />
            </div>
          </div>
        </div>
        <van-button type="danger" block @click="submitReview">提交评价</van-button>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.detail { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.product-video { background: #000; width: 100%; }
.pv-player { width: 100%; max-height: 280px; object-fit: contain; display: block; }
.price-block { padding: 12px 16px; background: #fff; }
.big-price { color: #e1251b; font-size: 28px; font-weight: bold; }
.origin { color: #999; text-decoration: line-through; margin-left: 10px; font-size: 14px; }
.sku-block { padding: 12px 16px; background: #fff; border-top: 1px solid #f5f5f5; }
.sku-title { font-size: 13px; color: #666; margin-bottom: 8px; }
.sku-title b { color: #333; }
.sku-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.sku-tag { padding: 6px 12px; background: #f7f7f7; border: 1px solid #eee; border-radius: 16px; font-size: 13px; color: #333; }
.sku-tag.active { background: #fff5f5; border-color: #e1251b; color: #e1251b; }
.sku-tag small { color: #e1251b; margin-left: 4px; }
.title-block { padding: 0 16px 12px; background: #fff; }
.p-title { font-size: 17px; line-height: 24px; }
.p-sub { color: #999; font-size: 13px; margin-top: 4px; }
.desc, .reviews { background: #fff; margin-top: 8px; padding: 12px 16px; }
.price-history { background: #fff; margin-top: 8px; padding: 12px 16px; }
.ph-head { display: flex; justify-content: space-between; align-items: center; font-size: 14px; font-weight: bold; margin-bottom: 10px; }
.ph-stats { font-size: 12px; color: #666; font-weight: normal; }
.ph-stats b.green { color: #07c160; }
.trend { margin-left: 6px; font-size: 11px; }
.trend.down { color: #07c160; }
.trend.up { color: #e1251b; }
.trend.flat { color: #999; }
.ph-chart { display: flex; align-items: flex-end; gap: 6px; height: 80px; }
.ph-bar-col { flex: 1; display: flex; flex-direction: column; align-items: center; height: 100%; justify-content: flex-end; }
.ph-bar { width: 60%; background: linear-gradient(180deg, #ff9800, #e1251b); border-radius: 3px 3px 0 0; min-height: 8px; }
.ph-date { font-size: 9px; color: #999; margin-top: 4px; }
.desc h3, .rev-head { font-size: 15px; margin-bottom: 8px; display: flex; justify-content: space-between; align-items: center; }
.rev-item { padding: 10px 0; border-top: 1px solid #f5f5f5; }
.rev-user { display: flex; gap: 8px; align-items: center; font-size: 13px; color: #666; }
.rev-content { font-size: 13px; margin-top: 4px; line-height: 18px; }
.rev-photos { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 6px; }
.rev-reply-btn { margin-left: auto; color: #e1251b; font-size: 12px; }
.rev-reply { background: #f7f7f7; border-radius: 6px; padding: 6px 10px; margin-top: 6px; font-size: 12px; color: #666; line-height: 18px; }
.rev-reply-name { color: #e1251b; }
.rev-reply-box { display: flex; gap: 8px; align-items: center; margin-top: 8px; }
.rev-reply-box .van-field { flex: 1; border: 1px solid #eee; border-radius: 6px; }
.rev-form { padding: 20px; }
.rev-form h3 { text-align: center; margin-bottom: 16px; }
.rev-form .van-field { margin: 12px 0; border: 1px solid #eee; }
.rev-upload { margin: 8px 0; }
.rev-imgs { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.rev-img-wrap { position: relative; }
.rev-img-del { position: absolute; top: -6px; right: -6px; background: #e1251b; color: #fff; border-radius: 50%; padding: 2px; font-size: 12px; }
</style>
