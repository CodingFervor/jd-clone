<script setup>
import { ref, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { getProfile } from '../api'

const router = useRouter()
const user = ref(null)
const cartCount = ref(0)
const loggedIn = ref(false)

async function load() {
  loggedIn.value = !!localStorage.getItem('jd_token')
  if (!loggedIn.value) return
  try {
    const res = await getProfile()
    user.value = res.user
    cartCount.value = res.cart_count || 0
  } catch (e) {
    loggedIn.value = false
  }
}
onMounted(load)
onActivated(load)

function logout() {
  localStorage.removeItem('jd_token')
  localStorage.removeItem('jd_user')
  loggedIn.value = false
  user.value = null
  showToast('已退出登录')
}
</script>

<template>
  <div class="mine-page">
    <!-- Header -->
    <div class="mine-header">
      <div v-if="loggedIn && user" class="user-info">
        <van-image round width="60" height="60" :src="user.avatar || 'https://via.placeholder.com/60'" />
        <div class="u-text">
          <div class="u-name">{{ user.nickname || user.username }}</div>
          <div class="u-id">用户名: {{ user.username }}</div>
        </div>
      </div>
      <div v-else class="user-info" @click="router.push('/login')">
        <van-image round width="60" height="60" src="https://via.placeholder.com/60" />
        <div class="u-text">
          <div class="u-name">登录/注册</div>
          <div class="u-id">点击登录享受更多优惠</div>
        </div>
      </div>
    </div>

    <!-- Order entries -->
    <van-cell-group inset title="我的订单">
      <div class="order-entries">
        <div class="oe-item" @click="router.push('/orders')">
          <van-icon name="balance-pay-o" size="28" color="#ff976a" />
          <span>待付款</span>
        </div>
        <div class="oe-item" @click="router.push('/orders')">
          <van-icon name="logistics" size="28" color="#07c160" />
          <span>待发货</span>
        </div>
        <div class="oe-item" @click="router.push('/orders')">
          <van-icon name="gift-o" size="28" color="#e1251b" />
          <span>待收货</span>
        </div>
        <div class="oe-item" @click="router.push('/orders')">
          <van-icon name="comment-o" size="28" color="#1989fa" />
          <span>待评价</span>
        </div>
      </div>
    </van-cell-group>

    <van-cell-group inset title="常用功能">
      <van-cell title="购物车" :value="cartCount + '件'" is-link @click="router.push('/cart')" icon="cart-o" />
      <van-cell title="我的订单" is-link @click="router.push('/orders')" icon="orders-o" />
      <van-cell title="售后服务" is-link @click="router.push('/refunds')" icon="after-sale" />
      <van-cell title="优惠券" is-link @click="router.push('/coupons')" icon="coupon-o" />
      <van-cell title="收货地址" is-link icon="location-o" @click="showToast('演示功能')" />
      <van-cell title="PLUS会员" is-link icon="diamond-o" @click="showToast('演示功能')" />
      <van-cell title="管理后台" is-link @click="router.push('/admin')" icon="setting-o" />
    </van-cell-group>

    <div v-if="loggedIn" style="margin: 20px">
      <van-button block plain type="danger" @click="logout">退出登录</van-button>
    </div>
  </div>
</template>

<style scoped>
.mine-page { min-height: 100vh; padding-bottom: 20px; }
.mine-header { background: linear-gradient(135deg, #e1251b, #f5574d); padding: 30px 20px; color: #fff; }
.user-info { display: flex; align-items: center; gap: 14px; }
.u-name { font-size: 18px; font-weight: bold; }
.u-id { font-size: 12px; opacity: 0.8; margin-top: 4px; }
.order-entries { display: flex; padding: 16px 0; }
.oe-item { flex: 1; text-align: center; font-size: 12px; color: #666; }
.oe-item span { display: block; margin-top: 4px; }
</style>
