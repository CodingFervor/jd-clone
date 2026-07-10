<script setup>
import { ref, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { generateGiftCard, redeemGiftCard, getMyGiftCards } from '../api'

const router = useRouter()
const redeemCode = ref('')
const genAmount = ref('')
const myCards = ref([])
const lastGenerated = ref(null) // the most recently generated demo card
const loggedIn = ref(false)

async function load() {
  loggedIn.value = !!localStorage.getItem('jd_token')
  if (!loggedIn.value) return
  try {
    myCards.value = await getMyGiftCards()
  } catch (e) {
    // ignore load errors (e.g. token expired)
  }
}
onMounted(load)
onActivated(load)

async function doRedeem() {
  const code = redeemCode.value.trim()
  if (!code) {
    showToast('请输入礼品卡卡号')
    return
  }
  if (!loggedIn.value) {
    router.push('/login')
    return
  }
  try {
    await redeemGiftCard(code)
    showSuccessToast('兑换成功')
    redeemCode.value = ''
    myCards.value = await getMyGiftCards()
  } catch (e) {
    showToast(e.response?.data?.error || '兑换失败')
  }
}

// Demo: generate a random card on the server and display its code so it can be
// copied / redeemed immediately. Mirrors an admin "issue card" flow.
async function doGenerate() {
  const amount = parseFloat(genAmount.value)
  if (!amount || amount <= 0) {
    showToast('请输入有效的面值')
    return
  }
  if (!loggedIn.value) {
    router.push('/login')
    return
  }
  try {
    const res = await generateGiftCard(amount)
    lastGenerated.value = res.data
    genAmount.value = ''
    showSuccessToast('已生成礼品卡')
  } catch (e) {
    showToast(e.response?.data?.error || '生成失败')
  }
}

async function copyCode(code) {
  try {
    await navigator.clipboard.writeText(code)
    showSuccessToast('卡号已复制')
  } catch (e) {
    const ta = document.createElement('textarea')
    ta.value = code
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      showSuccessToast('卡号已复制')
    } catch {
      showToast('复制失败，请手动复制')
    }
    document.body.removeChild(ta)
  }
}

function fmt(n) {
  return Number(n).toFixed(2)
}
</script>

<template>
  <div class="gift-page">
    <van-nav-bar title="礼品卡" left-arrow @click-left="router.back()" fixed placeholder />

    <!-- Redeem section -->
    <div class="gc-section">
      <div class="gc-section-head">兑换礼品卡</div>
      <van-cell-group inset>
        <van-field v-model="redeemCode" label="卡号" placeholder="请输入礼品卡卡号" clearable />
      </van-cell-group>
      <div style="padding: 12px 16px 0">
        <van-button block type="danger" round @click="doRedeem">立即兑换</van-button>
      </div>
    </div>

    <!-- Generate section (demo) -->
    <div class="gc-section">
      <div class="gc-section-head">生成礼品卡<span class="gc-tip">（演示）</span></div>
      <van-cell-group inset>
        <van-field v-model="genAmount" label="面值" type="number" placeholder="请输入面值，例如 100" clearable />
      </van-cell-group>
      <div style="padding: 12px 16px 0">
        <van-button block plain type="danger" round @click="doGenerate">生成礼品卡</van-button>
      </div>
      <div v-if="lastGenerated" class="gen-result">
        <div class="gr-line">
          <span class="gr-label">卡号：</span>
          <span class="gr-code">{{ lastGenerated.code }}</span>
          <van-button size="mini" plain type="danger" @click="copyCode(lastGenerated.code)">复制</van-button>
        </div>
        <div class="gr-line">
          <span class="gr-label">面值：</span>
          <span class="gr-amount">¥{{ fmt(lastGenerated.amount) }}</span>
        </div>
        <div class="gr-action">
          <van-button size="small" type="danger" round @click="redeemCode = lastGenerated.code">使用此卡号兑换</van-button>
        </div>
      </div>
    </div>

    <!-- My redeemed cards -->
    <div class="gc-section">
      <div class="gc-section-head">我的礼品卡</div>
      <div v-if="!loggedIn" class="gc-empty">
        <van-empty description="请先登录查看" />
      </div>
      <div v-else-if="!myCards.length" class="gc-empty">
        <van-empty description="暂无已兑换的礼品卡" />
      </div>
      <div v-else>
        <div v-for="c in myCards" :key="c.id" class="card-item">
          <div class="ci-left">
            <div class="ci-amount">¥{{ fmt(c.amount) }}</div>
            <div class="ci-status">已兑换</div>
          </div>
          <div class="ci-right">
            <div class="ci-code">{{ c.code }}</div>
            <div class="ci-date">{{ c.created_at }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gift-page { min-height: 100vh; padding-bottom: 30px; }
.gc-section { margin-top: 10px; }
.gc-section-head {
  padding: 14px 16px 8px;
  font-size: 14px;
  font-weight: bold;
  color: #333;
}
.gc-tip { font-size: 12px; color: #999; font-weight: normal; margin-left: 4px; }
.gen-result {
  margin: 12px 16px 0;
  background: #fff5f4;
  border: 1px dashed #e1251b;
  border-radius: 8px;
  padding: 12px;
}
.gr-line { display: flex; align-items: center; gap: 8px; font-size: 14px; color: #333; margin-bottom: 6px; }
.gr-label { color: #999; }
.gr-code { font-family: 'Courier New', monospace; font-weight: bold; letter-spacing: 1px; }
.gr-amount { color: #e1251b; font-weight: bold; }
.gr-action { margin-top: 8px; }
.gc-empty { background: #fff; }
.card-item {
  display: flex;
  margin: 8px;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}
.ci-left {
  background: linear-gradient(135deg, #e1251b, #ff5577);
  color: #fff;
  padding: 16px 12px;
  text-align: center;
  width: 100px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.ci-amount { font-size: 22px; font-weight: bold; }
.ci-status { font-size: 11px; opacity: 0.9; margin-top: 4px; }
.ci-right { flex: 1; padding: 12px 16px; display: flex; flex-direction: column; justify-content: center; gap: 6px; }
.ci-code { font-size: 14px; font-weight: bold; font-family: 'Courier New', monospace; letter-spacing: 1px; }
.ci-date { font-size: 12px; color: #999; }
</style>
