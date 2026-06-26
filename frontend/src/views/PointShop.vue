<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showConfirmDialog } from 'vant'
import { getPointShop, redeemPoints, getRedemptions } from '../api'

const router = useRouter()
const products = ref([])
const points = ref(0)
const loading = ref(true)
const redemptions = ref([])

async function load() {
  loading.value = true
  try {
    const res = await getPointShop()
    products.value = res.data || []
    points.value = res.points || 0
    redemptions.value = await getRedemptions()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function redeem(p) {
  try {
    await showConfirmDialog({ title: '确认兑换', message: `兑换「${p.name}」将扣除 ${p.points} 积分？` })
    const res = await redeemPoints(p.id)
    points.value = res.points
    showSuccessToast('兑换成功')
    await load()
  } catch (e) {
    if (e && e.message !== 'cancel' && e !== 'cancel') {
      showToast(e.response?.data?.error || '兑换失败')
    }
  }
}
</script>

<template>
  <div class="shop-page">
    <van-nav-bar title="积分商城" left-arrow @click-left="router.back()" fixed placeholder />
    <div class="points-banner">
      <div class="pb-label">我的可用积分</div>
      <div class="pb-value">{{ points }}</div>
      <van-button size="mini" plain round color="#fff" @click="router.push('/checkin')">去签到赚积分</van-button>
    </div>

    <div v-if="loading" class="loading"><van-loading /></div>
    <template v-else>
      <div class="section-title">积分好礼</div>
      <div class="product-grid">
        <div v-for="p in products" :key="p.id" class="product-card">
          <van-image width="100%" height="100" radius="8" :src="p.image" fit="cover" />
          <div class="pc-name van-multi-ellipsis--l2">{{ p.name }}</div>
          <div class="pc-points">{{ p.points }} 积分</div>
          <div class="pc-stock">剩余 {{ p.stock }} 件</div>
          <van-button size="small" type="danger" round block :disabled="points < p.points" @click="redeem(p)">
            {{ points < p.points ? '积分不足' : '立即兑换' }}
          </van-button>
        </div>
      </div>

      <div v-if="redemptions.length" class="section-title">兑换记录</div>
      <div v-if="redemptions.length" class="record-list">
        <div v-for="r in redemptions" :key="r.id" class="record-item">
          <div class="ri-info">
            <div class="ri-name">{{ r.product_name }}</div>
            <div class="ri-time">{{ r.created_at?.slice(0, 16).replace('T', ' ') }}</div>
          </div>
          <div class="ri-cost">-{{ r.points_cost }} 积分</div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.shop-page { min-height: 100vh; padding-bottom: 20px; }
.loading { text-align: center; padding: 80px; }
.points-banner { background: linear-gradient(135deg, #e1251b, #f5574d); padding: 24px 20px; text-align: center; color: #fff; }
.pb-label { font-size: 13px; opacity: 0.9; }
.pb-value { font-size: 36px; font-weight: bold; margin: 6px 0 10px; }
.section-title { font-size: 15px; font-weight: bold; padding: 16px 16px 8px; }
.product-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; padding: 0 12px; }
.product-card { background: #fff; border-radius: 10px; padding: 10px; }
.pc-name { font-size: 13px; line-height: 18px; height: 36px; margin-top: 8px; }
.pc-points { color: #e1251b; font-size: 16px; font-weight: bold; margin-top: 4px; }
.pc-stock { color: #999; font-size: 11px; margin: 4px 0 8px; }
.record-list { margin: 0 12px; background: #fff; border-radius: 10px; padding: 4px 12px; }
.record-item { display: flex; justify-content: space-between; align-items: center; padding: 12px 0; border-bottom: 1px solid #f5f5f5; }
.record-item:last-child { border-bottom: none; }
.ri-name { font-size: 14px; }
.ri-time { font-size: 11px; color: #999; margin-top: 2px; }
.ri-cost { color: #e1251b; font-size: 14px; font-weight: bold; }
</style>
