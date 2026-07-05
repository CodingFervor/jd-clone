<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { trackOrder, shipOrder, advanceShipment } from '../api'

const route = useRoute()
const router = useRouter()
const orderId = ref(Number(route.query.order_id) || 0)
const shipment = ref(null)
const loading = ref(true)

onMounted(load)

async function load() {
  loading.value = true
  try {
    shipment.value = await trackOrder(orderId.value)
  } catch (e) {
    // no shipment yet — that's fine
  } finally {
    loading.value = false
  }
}

async function doShip() {
  try {
    const res = await shipOrder(orderId.value)
    shipment.value = res.shipment
    showSuccessToast('已发货')
  } catch (e) {
    showToast(e.response?.data?.error || '发货失败')
  }
}

async function doAdvance() {
  try {
    shipment.value = await advanceShipment(orderId.value)
    showSuccessToast('物流已更新')
  } catch (e) {
    showToast(e.response?.data?.error || '更新失败')
  }
}

function statusText(s) {
  return { shipped: '已发货', in_transit: '运输中', delivered: '已送达', pending: '待发货' }[s] || s || '暂无物流'
}
// Map progress: 0-100% based on how many tracks have passed.
function mapProgress() {
  if (!shipment.value || !shipment.value.tracks) return 0
  const stages = { shipped: 25, in_transit: 60, delivered: 100, pending: 0 }
  return stages[shipment.value.status] || 25
}
const routePoints = [
  { name: '仓库', pos: 5 },
  { name: '中转站', pos: 38 },
  { name: '派送中', pos: 72 },
  { name: '已签收', pos: 95 },
]
function activePoint() {
  const p = mapProgress()
  return routePoints.filter((rp) => rp.pos <= p).length - 1
}
</script>

<template>
  <div class="logi-page">
    <van-nav-bar title="物流跟踪" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <template v-else-if="shipment">
      <div class="ship-head">
        <div class="carrier">{{ shipment.carrier }}</div>
        <div class="track-no">运单号: {{ shipment.tracking_no }}</div>
        <van-tag :type="shipment.status === 'delivered' ? 'success' : 'primary'" size="large">
          {{ statusText(shipment.status) }}
        </van-tag>
      </div>
      <div class="track-list">
        <div v-for="(t, i) in shipment.tracks" :key="t.id" class="track-item" :class="{ first: i === 0 }">
          <div class="track-dot" :class="{ active: i === 0 }"></div>
          <div class="track-content">
            <div class="track-desc">{{ t.description }}</div>
            <div class="track-loc" v-if="t.location">{{ t.location }}</div>
            <div class="track-time">{{ new Date(t.occurred_at).toLocaleString('zh-CN') }}</div>
          </div>
        </div>
      </div>
      <!-- Logistics route map (物流地图) -->
      <div class="map-section">
        <div class="map-title">📦 物流路线图</div>
        <div class="map-bar">
          <div class="map-fill" :style="{ width: mapProgress() + '%' }"></div>
        </div>
        <div class="map-points">
          <div v-for="(p, i) in routePoints" :key="i" class="mp-item" :class="{ active: i <= activePoint() }">
            <div class="mp-dot"></div>
            <span class="mp-name">{{ p.name }}</span>
          </div>
        </div>
      </div>
      <div v-if="shipment.status !== 'delivered'" style="margin: 20px">
        <van-button type="danger" block round @click="doAdvance">模拟物流更新（运输→送达）</van-button>
      </div>
    </template>
    <template v-else>
      <van-empty description="该订单尚未发货">
        <van-button type="danger" round @click="doShip">模拟发货（生成运单+轨迹）</van-button>
      </van-empty>
    </template>
  </div>
</template>

<style scoped>
.logi-page { min-height: 100vh; background: #f5f5f5; }
.loading { text-align: center; padding: 80px; }
.ship-head { background: #fff; padding: 20px; text-align: center; margin-bottom: 10px; }
.carrier { font-size: 16px; font-weight: bold; color: #333; }
.track-no { color: #999; font-size: 13px; margin: 6px 0 10px; }
.track-list { background: #fff; padding: 16px; margin: 0 10px; border-radius: 8px; }
.track-item { display: flex; gap: 14px; padding-bottom: 20px; position: relative; }
.track-item:not(:last-child)::before { content: ''; position: absolute; left: 5px; top: 14px; bottom: 0; width: 1px; background: #eee; }
.track-dot { width: 10px; height: 10px; border-radius: 50%; background: #ccc; flex-shrink: 0; margin-top: 4px; }
.track-dot.active { background: #e1251b; box-shadow: 0 0 0 4px #ffe0e0; }
.track-content { flex: 1; }
.track-desc { font-size: 14px; color: #333; line-height: 20px; }
.track-item.first .track-desc { color: #e1251b; font-weight: bold; }
.track-loc { font-size: 12px; color: #999; margin-top: 3px; }
.track-time { font-size: 11px; color: #ccc; margin-top: 3px; }
.map-section { background: #fff; margin: 10px; border-radius: 8px; padding: 16px; }
.map-title { font-size: 15px; font-weight: bold; margin-bottom: 16px; }
.map-bar { height: 6px; background: #eee; border-radius: 3px; overflow: hidden; margin: 0 8px; }
.map-fill { height: 100%; background: linear-gradient(90deg, #ff9800, #e1251b); transition: width 0.5s; }
.map-points { display: flex; justify-content: space-between; margin-top: 12px; padding: 0 4px; }
.mp-item { display: flex; flex-direction: column; align-items: center; gap: 4px; }
.mp-dot { width: 12px; height: 12px; border-radius: 50%; background: #ddd; }
.mp-item.active .mp-dot { background: #e1251b; }
.mp-name { font-size: 11px; color: #999; }
.mp-item.active .mp-name { color: #e1251b; font-weight: bold; }
</style>
