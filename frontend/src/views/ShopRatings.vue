<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getShopRatings, createShopRating } from '../api'

const route = useRoute()
const router = useRouter()
const shopName = ref(decodeURIComponent(route.params.name))
const summary = ref(null)
const ratings = ref([])
const loading = ref(true)
const showForm = ref(false)
const form = ref({ description_score: 5, logistics_score: 5, service_score: 5, comment: '' })

async function load() {
  loading.value = true
  try {
    const res = await getShopRatings(shopName.value)
    summary.value = res.summary
    ratings.value = res.ratings || []
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function submit() {
  try {
    await createShopRating(shopName.value, form.value)
    showSuccessToast('评价成功')
    showForm.value = false
    form.value = { description_score: 5, logistics_score: 5, service_score: 5, comment: '' }
    await load()
  } catch (e) {
    showToast('请先登录')
    router.push('/login')
  }
}
function fmt(n) { return Number(n).toFixed(1) }
</script>

<template>
  <div class="sr-page">
    <van-nav-bar :title="shopName" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <template v-else>
      <div v-if="summary" class="summary-card">
        <div class="sc-overall">
          <div class="sc-score">{{ fmt(summary.overall) }}</div>
          <div class="sc-label">综合评分</div>
          <van-rate :model-value="summary.overall" readonly allow-half size="14" color="#e1251b" />
        </div>
        <div class="sc-detail">
          <div class="sc-row"><span>描述相符</span><van-rate :model-value="summary.description_avg" readonly allow-half size="12" color="#e1251b" /><b>{{ fmt(summary.description_avg) }}</b></div>
          <div class="sc-row"><span>物流服务</span><van-rate :model-value="summary.logistics_avg" readonly allow-half size="12" color="#e1251b" /><b>{{ fmt(summary.logistics_avg) }}</b></div>
          <div class="sc-row"><span>服务态度</span><van-rate :model-value="summary.service_avg" readonly allow-half size="12" color="#e1251b" /><b>{{ fmt(summary.service_avg) }}</b></div>
          <div class="sc-count">{{ summary.count }}人评价</div>
        </div>
      </div>
      <div class="section-head">
        <span>用户评价</span>
        <van-button size="mini" type="danger" plain round @click="showForm = true">我要评价</van-button>
      </div>
      <div v-if="!ratings.length" class="empty">暂无评价</div>
      <div v-for="r in ratings" :key="r.id" class="rating-item">
        <div class="ri-head"><span>{{ r.username || '匿名用户' }}</span><van-rate :model-value="r.description_score" readonly size="10" color="#e1251b" /></div>
        <div v-if="r.comment" class="ri-comment">{{ r.comment }}</div>
      </div>
    </template>

    <van-popup v-model:show="showForm" position="bottom" round closeable>
      <div class="rating-form">
        <h3>评价店铺</h3>
        <div class="rf-row"><span>描述相符</span><van-rate v-model="form.description_score" color="#e1251b" /></div>
        <div class="rf-row"><span>物流服务</span><van-rate v-model="form.logistics_score" color="#e1251b" /></div>
        <div class="rf-row"><span>服务态度</span><van-rate v-model="form.service_score" color="#e1251b" /></div>
        <van-field v-model="form.comment" type="textarea" placeholder="说说你的购物体验（选填）" rows="2" />
        <van-button type="danger" block @click="submit" style="margin-top:12px">提交评价</van-button>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.sr-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.summary-card { display: flex; background: linear-gradient(135deg, #e1251b, #f5574d); color: #fff; padding: 20px; }
.sc-overall { text-align: center; padding-right: 20px; border-right: 1px solid rgba(255,255,255,0.3); }
.sc-score { font-size: 36px; font-weight: bold; }
.sc-label { font-size: 12px; opacity: 0.9; margin: 4px 0; }
.sc-detail { flex: 1; padding-left: 20px; }
.sc-row { display: flex; align-items: center; gap: 8px; font-size: 13px; padding: 3px 0; }
.sc-row span { width: 64px; }
.sc-row b { margin-left: auto; }
.sc-count { font-size: 11px; opacity: 0.8; margin-top: 6px; }
.section-head { display: flex; justify-content: space-between; align-items: center; padding: 16px; font-size: 15px; font-weight: bold; }
.empty { text-align: center; color: #999; padding: 30px; }
.rating-item { background: #fff; margin: 0 8px 8px; border-radius: 8px; padding: 12px; }
.ri-head { display: flex; justify-content: space-between; align-items: center; font-size: 13px; color: #666; }
.ri-comment { font-size: 13px; color: #333; margin-top: 6px; line-height: 18px; }
.rating-form { padding: 20px; }
.rating-form h3 { text-align: center; margin-bottom: 16px; }
.rf-row { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; font-size: 14px; }
</style>
