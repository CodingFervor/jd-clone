<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showSuccessToast, showToast, showDialog } from 'vant'
import { getProduct, addToCart, createOrder, createReview } from '../api'

const route = useRoute()
const router = useRouter()
const product = ref(null)
const reviews = ref([])
const loading = ref(true)
const showReview = ref(false)
const reviewRating = ref(5)
const reviewContent = ref('')

onMounted(async () => {
  try {
    const res = await getProduct(route.params.id)
    product.value = res.data
    reviews.value = res.reviews || []
  } catch (e) {
    showToast('商品不存在')
  } finally {
    loading.value = false
  }
})

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
    const rv = await createReview({ product_id: product.value.id, rating: reviewRating.value, content: reviewContent.value })
    reviews.value.unshift(rv)
    showReview.value = false
    reviewContent.value = ''
    showSuccessToast('评价成功')
  } catch (e) {
    showToast('请先登录')
  }
}
function fmt(n) {
  return Number(n).toFixed(2)
}
</script>

<template>
  <div v-if="loading" class="loading"><van-loading /></div>
  <div v-else-if="product" class="detail">
    <van-nav-bar title="商品详情" left-arrow @click-left="router.back()" fixed placeholder />
    <van-image width="100%" height="375" :src="product.image" fit="cover" />
    <div class="price-block">
      <span class="big-price">¥{{ fmt(product.price) }}</span>
      <span class="origin">¥{{ fmt(product.original_price) }}</span>
    </div>
    <div class="title-block">
      <h2 class="p-title">{{ product.name }}</h2>
      <p class="p-sub">{{ product.subtitle }}</p>
    </div>
    <van-cell-group inset>
      <van-cell title="店铺" :value="product.shop" />
      <van-cell title="销量" :value="product.sales + '人付款'" />
      <van-cell title="标签" :value="product.tags || '京东自营'" />
    </van-cell-group>
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
        </div>
        <div class="rev-content">{{ r.content }}</div>
      </div>
      <van-empty v-if="!reviews.length" description="暂无评价" />
    </div>

    <!-- Bottom action bar -->
    <van-action-bar>
      <van-action-bar-icon icon="chat-o" text="客服" @click="showToast('客服功能为演示')" />
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
        <van-button type="danger" block @click="submitReview">提交评价</van-button>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.detail { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.price-block { padding: 12px 16px; background: #fff; }
.big-price { color: #e1251b; font-size: 28px; font-weight: bold; }
.origin { color: #999; text-decoration: line-through; margin-left: 10px; font-size: 14px; }
.title-block { padding: 0 16px 12px; background: #fff; }
.p-title { font-size: 17px; line-height: 24px; }
.p-sub { color: #999; font-size: 13px; margin-top: 4px; }
.desc, .reviews { background: #fff; margin-top: 8px; padding: 12px 16px; }
.desc h3, .rev-head { font-size: 15px; margin-bottom: 8px; display: flex; justify-content: space-between; align-items: center; }
.rev-item { padding: 10px 0; border-top: 1px solid #f5f5f5; }
.rev-user { display: flex; gap: 8px; align-items: center; font-size: 13px; color: #666; }
.rev-content { font-size: 13px; margin-top: 4px; line-height: 18px; }
.rev-form { padding: 20px; }
.rev-form h3 { text-align: center; margin-bottom: 16px; }
.rev-form .van-field { margin: 12px 0; border: 1px solid #eee; }
</style>
