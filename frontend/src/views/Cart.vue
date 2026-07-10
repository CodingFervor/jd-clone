<script setup>
import { ref, computed, onMounted, onActivated, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showConfirmDialog } from 'vant'
import { getCart, updateCart, deleteCart, createOrder, getProducts, addToCart, toggleFavorite, getTieredDiscounts } from '../api'

// ---- Group buy invite (好友拼单邀请) ----
// A festive popup that generates shareable invite text for the current cart,
// lets the user copy it or use the Web Share API, and tracks a demo invite
// count in localStorage.
const showInvite = ref(false)
// Persisted demo counter of "已邀请 N 人" (number of invites accepted).
const invitedCount = ref(Number(localStorage.getItem('jd_group_invite_count') || 0))

// Invite text reflects the current cart contents (item count + total price).
const inviteText = computed(() => {
  const count = items.value.reduce((s, i) => s + (i.quantity || 1), 0)
  const total = Number(selectedTotal.value || 0).toFixed(2)
  return `我在京东挑了${count}件好物，总价¥${total}，快来一起拼单享优惠！链接：${window.location.origin}`
})

function openInvite() {
  showInvite.value = true
}

async function copyInvite() {
  try {
    await navigator.clipboard.writeText(inviteText.value)
    showSuccessToast('已复制邀请文案')
  } catch (e) {
    // Fallback for non-secure contexts / older browsers.
    const ta = document.createElement('textarea')
    ta.value = inviteText.value
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      showSuccessToast('已复制邀请文案')
    } catch (_) {
      showToast('复制失败，请手动复制')
    }
    document.body.removeChild(ta)
  }
}

async function shareInvite() {
  if (navigator.share) {
    try {
      await navigator.share({ title: '好友拼单邀请', text: inviteText.value, url: window.location.origin })
      bumpInvite()
    } catch (e) {
      // User cancelled the share sheet — do nothing.
    }
  } else {
    // No Web Share API: fall back to copying the invite text.
    await copyInvite()
    bumpInvite()
  }
}

// Increment the demo invite counter when an invite is actually sent.
function bumpInvite() {
  invitedCount.value += 1
  localStorage.setItem('jd_group_invite_count', String(invitedCount.value))
}

const router = useRouter()
const items = ref([])
const selectedTotal = ref(0)
const allSelected = ref(true)
const loading = ref(true)
// Per-item selected count ("已选N件"), derived from the client cart items so
// the label stays accurate even before the server round-trip in load() settles.
const selectedCount = computed(() => items.value.filter((i) => i.selected === 1).length)
// "猜你喜欢" recommended products, fetched as best-sellers from catalog.
const recommendations = ref([])

async function loadRecommendations() {
  try {
    const res = await getProducts({ page: 1, page_size: 4 })
    // Support both { data: { data, list } } and { data: [...] } shapes.
    const list = res.data?.data || res.data?.list || res.data || []
    recommendations.value = (Array.isArray(list) ? list : []).slice(0, 4)
  } catch (e) {
    recommendations.value = []
  }
}

async function load() {
  loading.value = true
  try {
    const res = await getCart()
    items.value = res.data || []
    selectedTotal.value = res.selected_total || 0
    allSelected.value = items.value.length > 0 && items.value.every((i) => i.selected === 1)
    // Load recommendations after the cart is loaded so the "猜你喜欢" section
    // is populated while / above the cart items.
    await loadRecommendations()
    // Load tiered-discount tiers (best-effort) to power the smart quantity hints.
    loadTiers()
  } catch (e) {
    if (e.response?.status === 401) router.replace('/login')
  } finally {
    loading.value = false
  }
}
onMounted(load)
onActivated(load)

async function quickAdd(p) {
  try {
    await addToCart(p.id, 1)
    showSuccessToast('已加入购物车')
    await load()
  } catch (e) {
    showToast(e.response?.data?.error || '加入失败')
  }
}

async function toggleSelect(item) {
  const newSel = item.selected === 1 ? 0 : 1
  try {
    await updateCart(item.id, item.quantity, newSel)
    item.selected = newSel
    await load()
  } catch (e) {}
}
async function changeQty(item, qty) {
  if (qty < 1) return
  try {
    await updateCart(item.id, qty, item.selected)
    item.quantity = qty
    await load()
  } catch (e) {}
}
async function removeItem(item) {
  try {
    await deleteCart(item.id)
    await load()
    showSuccessToast('已删除')
  } catch (e) {}
}

// ---- Swipe cell actions (购物车滑动操作) ----
// Right swipe reveals 收藏 (favorite) + 删除 (delete).
// Left swipe reveals 移至收藏夹 (favorite then remove from cart).
async function swipeFavorite(it) {
  try {
    await toggleFavorite(it.product_id)
    showSuccessToast('已加入收藏')
  } catch (e) {
    showToast(e.response?.data?.error || '收藏失败')
  }
}
// Move to favorites: favorite the product, then delete it from the cart.
async function moveToFavorite(it) {
  try {
    await toggleFavorite(it.product_id)
    await deleteCart(it.id)
    await load()
    showSuccessToast('已移至收藏夹')
  } catch (e) {
    showToast(e.response?.data?.error || '操作失败')
  }
}
async function toggleAll() {
  const target = allSelected.value ? 0 : 1
  for (const it of items.value) {
    if (it.selected !== target) await updateCart(it.id, it.quantity, target)
  }
  await load()
}
// 反选 (invert selection): toggle every item's selected state.
async function invertSelection() {
  for (const it of items.value) {
    const target = it.selected === 1 ? 0 : 1
    await updateCart(it.id, it.quantity, target)
  }
  await load()
}
async function checkout() {
  if (selectedTotal.value <= 0) {
    showToast('请选择商品')
    return
  }
  const selected = items.value.filter((i) => i.selected === 1).map((i) => ({ product_id: i.product_id, quantity: i.quantity }))
  try {
    await createOrder({ items: selected, address: '' })
    showSuccessToast('下单成功')
    router.push('/orders')
  } catch (e) {
    showToast(e.response?.data?.error || '下单失败')
  }
}
function fmt(n) {
  return Number(n).toFixed(2)
}

// A "降价" tag for items whose original price is significantly higher than the
// current price (indicating an active discount / price drop).
function isPriceDrop(it) {
  return Number(it.original_price) > Number(it.price) * 1.1
}

// ---- Smart Quantity Selector (智能数量选择) ----
// Store-wide spend-X-get-Y-off tiers loaded from the API. Each tier is
// { threshold, discount }. When tiers are missing/unloaded we gracefully fall
// back to the plain quantity stepper (tiersLoaded stays false).
const tiers = ref([])
const tiersLoaded = ref(false)

// Selected cart total in yuan (sum of selected items' price * quantity).
const selectedSum = computed(() =>
  items.value
    .filter((i) => i.selected === 1)
    .reduce((s, i) => s + Number(i.price) * Number(i.quantity || 1), 0)
)

// Sort tiers ascending by threshold so we can scan in order.
const sortedTiers = computed(() =>
  [...tiers.value].sort((a, b) => Number(a.threshold) - Number(b.threshold))
)

// The next tier the selected total has not yet reached. null if none or
// the total already meets the highest tier.
const nextTier = computed(() => {
  if (!tiersLoaded.value || !sortedTiers.value.length) return null
  return sortedTiers.value.find((t) => selectedSum.value < Number(t.threshold)) || null
})

// The best (highest) tier the selected total currently meets, if any.
const metTier = computed(() => {
  if (!tiersLoaded.value || !sortedTiers.value.length) return null
  let met = null
  for (const t of sortedTiers.value) {
    if (selectedSum.value >= Number(t.threshold)) met = t
  }
  return met
})

// Whether the current selected total already satisfies at least one tier.
const hasMetTier = computed(() => !!metTier.value)

// For a given cart line, decide whether to surface a "买2件更划算" style hint.
// The hint appears when adding a small number of this item (default 2) would
// push the selected total into the next tier, i.e. the gap is within this
// item's line value. Returns an object describing the suggestion or null.
function itemTierHint(it) {
  if (!tiersLoaded.value || !nextTier.value || it.selected !== 1) return null
  const gap = Number(nextTier.value.threshold) - selectedSum.value
  if (gap <= 0) return null
  const unit = Number(it.price)
  if (unit <= 0) return null
  const addQty = Math.ceil(gap / unit)
  // Only nudge when the add is small (1-3 units) so it's actually "更划算".
  if (addQty < 1 || addQty > 3) return null
  const targetQty = Number(it.quantity || 1) + addQty
  if (targetQty > Number(it.stock || 99)) return null
  return { addQty, targetQty, nextTier: nextTier.value }
}

// "凑单" quick-add: bump this line's quantity by the suggested amount to
// reach the next discount tier.
async function quickAddTier(it) {
  const hint = itemTierHint(it)
  if (!hint) return
  await changeQty(it, hint.targetQty)
  showSuccessToast(`已凑单至满¥${hint.nextTier.threshold}减¥${hint.nextTier.discount}`)
}

async function loadTiers() {
  try {
    const data = await getTieredDiscounts()
    tiers.value = Array.isArray(data) ? data : []
    tiersLoaded.value = true
  } catch {
    // Best-effort: without tiers we just show the basic stepper.
    tiersLoaded.value = false
  }
}

// ---- Shake to Undo (摇一摇撤销) ----
// Detects a phone shake via DeviceMotionEvent. When a shake above the
// threshold is registered, prompt the user to re-fetch the cart (restoring
// any recently deleted items). On iOS 13+, the permission must be requested
// first via a user gesture; otherwise the motion data is never delivered.
const SHAKE_THRESHOLD = 25 // m/s^2 on any axis (重力加速度阈值)
const SHAKE_INTERVAL = 1000 // min ms between two shake triggers
let lastShakeAt = 0

function onDeviceMotion(e) {
  const acc = e.accelerationIncludingGravity
  if (!acc) return
  // accelerationIncludingGravity may report null values on some browsers;
  // bail out if the axis data isn't usable.
  if (acc.x == null && acc.y == null && acc.z == null) return
  const mag = Math.max(Math.abs(acc.x || 0), Math.abs(acc.y || 0), Math.abs(acc.z || 0))
  if (mag < SHAKE_THRESHOLD) return
  const now = Date.now()
  if (now - lastShakeAt < SHAKE_INTERVAL) return
  lastShakeAt = now
  triggerShakeUndo()
}

// Re-fetch the cart data to restore deleted items (server round-trip).
function restoreCart() {
  load()
  showSuccessToast('已恢复购物车')
}

function triggerShakeUndo() {
  showConfirmDialog({
    title: '摇一摇撤销',
    message: '摇一摇撤销删除？',
    confirmButtonText: '撤销',
    cancelButtonText: '取消',
    confirmButtonColor: '#e1251b',
  })
    .then(() => restoreCart())
    .catch(() => {})
}

// Begin listening for motion events. On iOS 13+, requestPermission must be
// called from a user gesture, so the listener is wired inside the resolver.
async function enableShake() {
  try {
    if (typeof DeviceMotionEvent !== 'undefined' && typeof DeviceMotionEvent.requestPermission === 'function') {
      // iOS 13+: request permission via a user gesture.
      const res = await DeviceMotionEvent.requestPermission()
      if (res === 'granted') {
        window.addEventListener('devicemotion', onDeviceMotion)
        showSuccessToast('摇一摇已开启')
      } else {
        showToast('摇一摇功能需要授权')
      }
    } else if (typeof DeviceMotionEvent !== 'undefined') {
      // Android / other: just attach.
      window.addEventListener('devicemotion', onDeviceMotion)
      showSuccessToast('摇一摇已开启')
    } else {
      showToast('摇一摇功能需要授权')
    }
  } catch (e) {
    showToast('摇一摇功能需要授权')
  }
}

onUnmounted(() => {
  window.removeEventListener('devicemotion', onDeviceMotion)
})
</script>

<template>
  <div class="cart-page">
    <van-nav-bar title="购物车" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!items.length" description="购物车是空的">
      <van-button type="danger" round @click="router.push('/home')">去逛逛</van-button>
    </van-empty>
    <div v-else>
      <!-- 猜你喜欢: horizontally scrollable recommended products -->
      <div v-if="recommendations.length" class="rec-section">
        <div class="rec-title">猜你喜欢</div>
        <div class="rec-scroll">
          <div
            v-for="p in recommendations"
            :key="p.id"
            class="rec-card"
            @click="router.push('/product/' + p.id)"
          >
            <van-image
              width="100"
              height="100"
              radius="6"
              :src="p.image"
              fit="cover"
            />
            <div class="rec-name van-multi-ellipsis--l2">{{ p.name }}</div>
            <div class="rec-bottom">
              <span class="rec-price">¥{{ fmt(p.price) }}</span>
              <van-button
                size="mini"
                type="danger"
                round
                @click.stop="quickAdd(p)"
              >加入购物车</van-button>
            </div>
          </div>
        </div>
      </div>

      <van-swipe-cell v-for="it in items" :key="it.id">
        <template #left>
          <div class="swipe-left-btn" @click="moveToFavorite(it)">移至收藏夹</div>
        </template>
        <div class="cart-item">
          <van-checkbox :model-value="it.selected === 1" @click="toggleSelect(it)" />
          <van-image width="80" height="80" radius="6" :src="it.product_image" fit="cover" @click="router.push('/product/' + it.product_id)" />
          <div class="ci-info">
            <div class="ci-name van-multi-ellipsis--l2">
              <van-tag v-if="isPriceDrop(it)" type="danger" size="mini" class="drop-tag">降价</van-tag>
              {{ it.product_name }}
            </div>
            <div class="ci-bottom">
              <span class="price">¥{{ fmt(it.price) }}</span>
              <van-stepper v-model="it.quantity" :min="1" :max="it.stock" @change="(v) => changeQty(it, v)" />
              <van-icon name="delete-o" size="20" @click="removeItem(it)" />
            </div>
            <!-- Smart Quantity Selector (智能数量选择): tier-aware hints -->
            <div v-if="tiersLoaded" class="qty-hints">
              <span
                v-if="itemTierHint(it)"
                class="qty-tip"
                @click="quickAddTier(it)"
              >
                买{{ itemTierHint(it).targetQty }}件更划算，凑单满¥{{ itemTierHint(it).nextTier.threshold }}减¥{{ itemTierHint(it).nextTier.discount }}
                <span class="qty-add-btn">凑单 +{{ itemTierHint(it).addQty }}</span>
              </span>
            </div>
          </div>
        </div>
        <template #right>
          <div class="swipe-right-fav" @click="swipeFavorite(it)">收藏</div>
          <div class="swipe-right-del" @click="removeItem(it)">删除</div>
        </template>
      </van-swipe-cell>

      <!-- Smart Quantity Selector (智能数量选择): tier status banner -->
      <div v-if="tiersLoaded" class="tier-banner">
        <span v-if="hasMetTier" class="tier-met">
          <span class="tier-check">✓</span> 已达满减：满¥{{ metTier.threshold }}减¥{{ metTier.discount }}
          <span v-if="nextTier" class="tier-next">距满¥{{ nextTier.threshold }}减¥{{ nextTier.discount }}还差¥{{ (nextTier.threshold - selectedSum).toFixed(2) }}</span>
        </span>
        <span v-else-if="nextTier" class="tier-progress">
          再买¥{{ (nextTier.threshold - selectedSum).toFixed(2) }}可享满¥{{ nextTier.threshold }}减¥{{ nextTier.discount }}
        </span>
      </div>

      <div class="invite-row">
        <van-button class="invite-btn" round size="small" icon="friends-o" @click="openInvite">
          👥 邀请拼单
        </van-button>
      </div>

      <van-submit-bar :price="selectedTotal * 100" button-text="结算" @submit="checkout">
        <van-checkbox :model-value="allSelected" @click="toggleAll">全选</van-checkbox>
        <span class="invert-btn" @click="invertSelection">反选</span>
        <span class="selected-count">已选{{ selectedCount }}件</span>
      </van-submit-bar>
    </div>

    <!-- Group buy invite popup (好友拼单邀请) -->
    <van-popup v-model:show="showInvite" round closeable position="bottom" :style="{ height: '52%' }">
      <div class="invite-popup">
        <div class="invite-header">
          <div class="invite-title">邀请好友一起拼单</div>
          <div class="invite-sub">拼着买，更优惠</div>
        </div>
        <div class="invite-body">
          <div class="invite-counter">
            <span class="ic-icon">🎉</span>
            <span class="ic-text">已邀请 <b>{{ invitedCount }}</b> 人</span>
          </div>
          <div class="invite-text">{{ inviteText }}</div>
          <div class="invite-actions">
            <van-button class="ia-btn" round block type="danger" icon="records" @click="copyInvite">复制邀请</van-button>
            <van-button class="ia-btn" round block plain type="danger" icon="share-o" @click="shareInvite">系统分享</van-button>
          </div>
        </div>
      </div>
    </van-popup>

    <!-- Shake to Undo hint (摇一摇撤销) -->
    <div class="shake-hint" @click="enableShake">
      <span class="shake-icon">📳</span>
      <span class="shake-text">摇一摇撤销</span>
    </div>
  </div>
</template>

<style scoped>
.cart-page { padding-bottom: 60px; }
.loading { text-align: center; padding: 80px; }
.cart-item { display: flex; align-items: center; gap: 10px; padding: 12px; background: #fff; border-bottom: 1px solid #f5f5f5; }
/* ---- Swipe cell action buttons (购物车滑动操作) ---- */
.swipe-left-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 0 18px;
  font-size: 13px;
  color: #fff;
  background: linear-gradient(135deg, #ffb74d, #ff9800);
  white-space: nowrap;
}
.swipe-left-btn:active { opacity: 0.85; }
.swipe-right-fav {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 0 18px;
  font-size: 13px;
  color: #fff;
  background: #ff9800;
  white-space: nowrap;
}
.swipe-right-fav:active { opacity: 0.85; }
.swipe-right-del {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 0 18px;
  font-size: 13px;
  color: #fff;
  background: #e1251b;
  white-space: nowrap;
}
.swipe-right-del:active { opacity: 0.85; }
.ci-info { flex: 1; }
.ci-name { font-size: 13px; line-height: 18px; height: 36px; }
.ci-name .drop-tag { vertical-align: middle; margin-right: 4px; }
.ci-bottom { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
.ci-bottom .price { font-size: 16px; flex: 1; }
/* Smart Quantity Selector (智能数量选择) */
.qty-hints { margin-top: 6px; }
.qty-tip {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 12px;
  color: #ff7a18;
  background: #fff7e6;
  border: 1px solid #ffd591;
  border-radius: 6px;
  padding: 4px 8px;
  cursor: pointer;
  line-height: 1.5;
}
.qty-add-btn {
  display: inline-flex;
  align-items: center;
  background: #ff7a18;
  color: #fff;
  font-weight: 600;
  padding: 1px 8px;
  border-radius: 10px;
  white-space: nowrap;
}
.qty-add-btn:active { opacity: 0.85; }
.tier-banner {
  margin: 8px 12px 0;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.5;
}
.tier-banner .tier-met {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  color: #07c160;
  background: #f0fff4;
  border: 1px solid #b7eb8f;
  border-radius: 8px;
  padding: 8px 12px;
}
.tier-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  background: #07c160;
  color: #fff;
  border-radius: 50%;
  font-size: 11px;
  font-weight: bold;
}
.tier-next {
  color: #ff7a18;
  font-size: 11px;
}
.tier-progress {
  display: block;
  color: #ff7a18;
  background: #fff7e6;
  border: 1px solid #ffd591;
  border-radius: 8px;
  padding: 8px 12px;
}
/* 猜你喜欢 recommended products section */
.rec-section { background: #fff; margin-bottom: 8px; padding: 12px 12px 8px; }
.rec-title { font-size: 15px; font-weight: 600; margin-bottom: 8px; }
.rec-scroll { display: flex; gap: 10px; overflow-x: auto; padding-bottom: 4px; }
.rec-scroll::-webkit-scrollbar { display: none; }
.rec-card { flex: 0 0 auto; width: 110px; }
.rec-name { font-size: 12px; line-height: 16px; height: 32px; margin-top: 6px; }
.rec-bottom { display: flex; flex-direction: column; align-items: flex-start; gap: 4px; margin-top: 4px; }
.rec-price { color: #e1251b; font-size: 14px; font-weight: 600; }
/* 反选 button + 已选N件 count inside the submit bar */
.invert-btn { margin-left: 10px; padding: 3px 10px; font-size: 12px; color: #e1251b; border: 1px solid #e1251b; border-radius: 12px; cursor: pointer; white-space: nowrap; }
.selected-count { margin-left: 10px; font-size: 12px; color: #999; white-space: nowrap; }

/* Group buy invite (好友拼单邀请) */
.invite-row {
  display: flex;
  justify-content: center;
  padding: 8px 12px 4px;
}
.invite-btn {
  background: linear-gradient(90deg, #e1251b 0%, #ff7a18 100%);
  color: #fff;
  border: none;
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(225, 37, 27, 0.3);
}
.invite-popup { display: flex; flex-direction: column; height: 100%; }
.invite-header {
  background: linear-gradient(135deg, #e1251b 0%, #ff4d4f 50%, #ff7a45 100%);
  color: #fff;
  padding: 24px 20px 18px;
  border-radius: 16px 16px 0 0;
  text-align: center;
}
.invite-title { font-size: 20px; font-weight: bold; letter-spacing: 1px; }
.invite-sub { font-size: 13px; opacity: 0.92; margin-top: 4px; }
.invite-body { flex: 1; padding: 18px 20px 20px; display: flex; flex-direction: column; gap: 16px; }
.invite-counter {
  align-self: center;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #fff5f5;
  border: 1px solid #ffd6d6;
  color: #e1251b;
  font-size: 14px;
  padding: 6px 16px;
  border-radius: 20px;
}
.invite-counter .ic-icon { font-size: 16px; }
.invite-counter b { font-size: 16px; margin: 0 2px; }
.invite-text {
  background: #f7f8fa;
  border-radius: 10px;
  padding: 14px;
  font-size: 14px;
  line-height: 22px;
  color: #333;
  word-break: break-all;
}
.invite-actions { display: flex; flex-direction: column; gap: 10px; }
.ia-btn { font-weight: 600; }

/* Shake to Undo hint (摇一摇撤销) */
.shake-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 12px auto 4px;
  padding: 6px 16px;
  align-self: center;
  font-size: 12px;
  color: #999;
  background: #f7f8fa;
  border: 1px dashed #dcdfe6;
  border-radius: 20px;
  cursor: pointer;
  width: fit-content;
  transition: color 0.2s ease, border-color 0.2s ease;
}
.shake-hint:active { color: #e1251b; border-color: #e1251b; }
.shake-icon {
  font-size: 14px;
  display: inline-block;
  animation: shake-wobble 2.5s ease-in-out infinite;
  transform-origin: center;
}
@keyframes shake-wobble {
  0%, 80%, 100% { transform: rotate(0deg); }
  85% { transform: rotate(-18deg); }
  90% { transform: rotate(18deg); }
  95% { transform: rotate(-12deg); }
}
</style>
