<script setup>
import { ref, computed, onMounted } from 'vue'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getProducts, adminCreateProduct, adminUpdateProduct, adminDeleteProduct, getCategories, uploadImage, getOrders } from '../api'

const uploadingImg = ref(false)
async function onUploadMainImage(item) {
  const file = item.file
  uploadingImg.value = true
  try {
    const res = await uploadImage(file)
    form.value.image = res.url
    showToast('图片已上传')
  } catch (e) {
    showToast(e.response?.data?.error || '上传失败')
  } finally {
    uploadingImg.value = false
  }
}

const products = ref([])
const categories = ref([])
const orders = ref([])
const loading = ref(true)
const showForm = ref(false)
const editingId = ref(null)
const form = ref(emptyForm())

function emptyForm() {
  return { name: '', subtitle: '', price: 0, original_price: 0, image: '', category: '', category_id: 0, shop: '', stock: 999, sales: 0, description: '', tags: '', is_seckill: 0 }
}

// ---- Feature: 管理后台增强 (Admin Dashboard Enhancement) ----
// Stats overview derived from the loaded products + orders.
// Total revenue + avg order value exclude cancelled orders (no real revenue).
const totalProducts = computed(() => products.value.length)
const totalOrders = computed(() => orders.value.length)
const revenueOrders = computed(() =>
  orders.value.filter((o) => o && o.status !== 'cancelled')
)
const totalRevenue = computed(() =>
  revenueOrders.value.reduce((s, o) => s + Number(o.total || 0), 0)
)
const avgOrderValue = computed(() => {
  const n = revenueOrders.value.length
  if (!n) return 0
  return totalRevenue.value / n
})
// Latest 5 orders (newest first by created_at, then id as a fallback).
const recentOrders = computed(() => {
  const list = [...orders.value]
  list.sort((a, b) => {
    const ta = a.created_at ? new Date(a.created_at).getTime() : 0
    const tb = b.created_at ? new Date(b.created_at).getTime() : 0
    if (tb !== ta) return tb - ta
    return Number(b.id || 0) - Number(a.id || 0)
  })
  return list.slice(0, 5)
})
// Top-5 best sellers by sales count (descending).
const topProducts = computed(() => {
  return [...products.value]
    .sort((a, b) => Number(b.sales || 0) - Number(a.sales || 0))
    .slice(0, 5)
})
// Human-readable order status label + badge color.
function statusBadge(s) {
  return {
    pending: { text: '待付款', color: '#ff976a', bg: '#fff7e6' },
    paid: { text: '已付款', color: '#1989fa', bg: '#e8f3ff' },
    shipped: { text: '已发货', color: '#07c160', bg: '#e8faf0' },
    completed: { text: '已完成', color: '#07c160', bg: '#e8faf0' },
    cancelled: { text: '已取消', color: '#999', bg: '#f5f5f5' },
  }[s] || { text: s || '未知', color: '#999', bg: '#f5f5f5' }
}

onMounted(async () => {
  await loadProducts()
  try { categories.value = await getCategories() } catch (e) {}
  // Feature: 管理后台增强 — load orders for the stats overview.
  try { orders.value = (await getOrders()) || [] } catch (e) { orders.value = [] }
})

async function loadProducts() {
  loading.value = true
  try {
    const res = await getProducts({ page: 1, page_size: 100 })
    products.value = res.data
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
function openCreate() {
  editingId.value = null
  form.value = emptyForm()
  showForm.value = true
}
function openEdit(p) {
  editingId.value = p.id
  form.value = { name: p.name, subtitle: p.subtitle, price: p.price, original_price: p.original_price, image: p.image, category: p.category, category_id: p.category_id, shop: p.shop, stock: p.stock, sales: p.sales, description: p.description, tags: p.tags, is_seckill: p.is_seckill }
  showForm.value = true
}
async function save() {
  if (!form.value.name || !form.value.price) {
    showToast('商品名和价格必填')
    return
  }
  try {
    if (editingId.value) {
      await adminUpdateProduct(editingId.value, form.value)
      showSuccessToast('已更新')
    } else {
      await adminCreateProduct(form.value)
      showSuccessToast('已创建')
    }
    showForm.value = false
    await loadProducts()
  } catch (e) {
    showToast(e.response?.data?.error || '保存失败')
  }
}
async function remove(p) {
  try {
    await showDialog({ title: '确认删除', message: '删除商品「' + p.name + '」？' })
    await adminDeleteProduct(p.id)
    showSuccessToast('已删除')
    await loadProducts()
  } catch (e) {
    // user cancelled
  }
}
function fmt(n) { return Number(n).toFixed(2) }

// Parse an order's items_json into a readable "商品A x2, 商品B x1" summary.
function orderItemsSummary(o) {
  let items = []
  try {
    const parsed = typeof o.items_json === 'string' ? JSON.parse(o.items_json) : o.items_json
    if (Array.isArray(parsed)) items = parsed
  } catch (_) {
    items = []
  }
  if (!items.length) return '—'
  return items
    .map((it) => `${it.name || it.product_name || '商品'} x${it.quantity || 1}`)
    .join(', ')
}
</script>

<template>
  <div class="admin">
    <van-nav-bar title="商品管理后台" left-arrow @click-left="$router.back()" fixed placeholder>
      <template #right>
        <van-icon name="plus" size="20" @click="openCreate" />
      </template>
    </van-nav-bar>
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!products.length" description="暂无商品">
      <van-button type="danger" round @click="openCreate">添加商品</van-button>
    </van-empty>
    <div v-else>
      <!-- Feature: 管理后台增强 — stats overview cards -->
      <div class="stat-cards">
        <div class="stat-card sc-blue">
          <div class="sc-label">商品总数</div>
          <div class="sc-value">{{ totalProducts }}</div>
        </div>
        <div class="stat-card sc-orange">
          <div class="sc-label">订单总数</div>
          <div class="sc-value">{{ totalOrders }}</div>
        </div>
        <div class="stat-card sc-red">
          <div class="sc-label">总营业额</div>
          <div class="sc-value">¥{{ fmt(totalRevenue) }}</div>
        </div>
        <div class="stat-card sc-green">
          <div class="sc-label">客单价</div>
          <div class="sc-value">¥{{ fmt(avgOrderValue) }}</div>
        </div>
      </div>

      <!-- Feature: 管理后台增强 — latest 5 orders with status badges -->
      <div class="dashboard-section">
        <div class="ds-title">最近订单</div>
        <div v-if="!recentOrders.length" class="ds-empty">暂无订单</div>
        <div v-for="o in recentOrders" :key="o.id" class="recent-order">
          <div class="ro-top">
            <span class="ro-no">{{ o.order_no || ('#' + o.id) }}</span>
            <span
              class="ro-status"
              :style="{ color: statusBadge(o.status).color, background: statusBadge(o.status).bg }"
            >{{ statusBadge(o.status).text }}</span>
          </div>
          <div class="ro-items van-ellipsis">{{ orderItemsSummary(o) }}</div>
          <div class="ro-bottom">
            <span class="ro-time">{{ o.created_at ? String(o.created_at).slice(0, 16).replace('T', ' ') : '' }}</span>
            <span class="ro-total">¥{{ fmt(o.total) }}</span>
          </div>
        </div>
      </div>

      <!-- Feature: 管理后台增强 — top 5 best sellers -->
      <div class="dashboard-section">
        <div class="ds-title">热销商品TOP5</div>
        <div v-if="!topProducts.length" class="ds-empty">暂无商品</div>
        <div
          v-for="(p, idx) in topProducts"
          :key="p.id"
          class="top-product"
          @click="openEdit(p)"
        >
          <span class="tp-rank" :class="{ 'rank-top': idx < 3 }">{{ idx + 1 }}</span>
          <van-image width="40" height="40" radius="4" :src="p.image" fit="cover" />
          <div class="tp-info">
            <div class="van-ellipsis">{{ p.name }}</div>
            <div class="tp-sub">销量 {{ p.sales || 0 }} · 库存 {{ p.stock || 0 }}</div>
          </div>
          <span class="tp-price">¥{{ fmt(p.price) }}</span>
        </div>
      </div>

      <van-cell-group inset>
        <van-swipe-cell v-for="p in products" :key="p.id">
          <van-cell @click="openEdit(p)">
            <template #title>
              <div class="acell">
                <van-image width="50" height="50" radius="4" :src="p.image" fit="cover" />
                <div class="ac-info">
                  <div class="van-ellipsis">{{ p.name }}</div>
                  <div class="ac-price">¥{{ fmt(p.price) }} <small>库存{{ p.stock }}</small></div>
                </div>
              </div>
            </template>
            <template #right>
              <van-button square type="primary" text="编辑" @click="openEdit(p)" />
              <van-button square type="danger" text="删除" @click="remove(p)" />
            </template>
          </van-cell>
        </van-swipe-cell>
      </van-cell-group>
    </div>

    <!-- Create/edit popup -->
    <van-popup v-model:show="showForm" position="bottom" round :style="{ height: '80%' }" closeable>
      <div class="form">
        <h3>{{ editingId ? '编辑商品' : '新增商品' }}</h3>
        <van-cell-group inset>
          <van-field v-model="form.name" label="名称" placeholder="商品名称" />
          <van-field v-model="form.subtitle" label="副标题" placeholder="卖点" />
          <van-field v-model="form.price" type="number" label="价格" placeholder="0.00" />
          <van-field v-model="form.original_price" type="number" label="原价" placeholder="0.00" />
          <van-field label="商品主图" :loading="uploadingImg">
            <template #input>
              <van-uploader :after-read="onUploadMainImage" accept="image/*" max-count="1" :preview-image="false">
                <van-button icon="photo-o" size="small" round color="#e1251b">上传图片</van-button>
              </van-uploader>
              <van-image v-if="form.image" width="60" height="60" radius="6" :src="form.image" fit="cover" style="margin-left: 8px" />
            </template>
          </van-field>
          <van-field v-model="form.shop" label="店铺" placeholder="京东自营" />
          <van-field v-model="form.stock" type="digit" label="库存" placeholder="999" />
          <van-field v-model="form.tags" label="标签" placeholder="新品,自营" />
          <van-field v-model="form.description" type="textarea" label="描述" rows="2" />
          <van-cell title="秒杀商品">
            <template #right-icon>
              <van-switch v-model="form.is_seckill" :active-value="1" :inactive-value="0" />
            </template>
          </van-cell>
        </van-cell-group>
        <div style="margin: 16px">
          <van-button type="danger" block round @click="save">保 存</van-button>
        </div>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.admin { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.acell { display: flex; gap: 10px; align-items: center; }
.ac-info { flex: 1; font-size: 13px; }
.ac-price { color: #e1251b; margin-top: 4px; }
.ac-price small { color: #999; font-weight: normal; }
.form { padding: 16px 0; }
.form h3 { text-align: center; padding: 12px; }

/* Feature: 管理后台增强 — stats overview cards */
.stat-cards {
  display: flex;
  gap: 8px;
  padding: 12px;
  overflow-x: auto;
}
.stat-cards::-webkit-scrollbar { display: none; }
.stat-card {
  flex: 1 0 0;
  min-width: 76px;
  border-radius: 10px;
  padding: 12px 8px;
  color: #fff;
  text-align: center;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
}
.sc-label { font-size: 11px; opacity: 0.92; }
.sc-value {
  font-size: 18px;
  font-weight: bold;
  margin-top: 6px;
  font-family: 'Courier New', monospace;
  word-break: break-all;
}
.sc-blue   { background: linear-gradient(135deg, #4facfe, #00c6fb); }
.sc-orange { background: linear-gradient(135deg, #ffb74d, #ff9800); }
.sc-red    { background: linear-gradient(135deg, #ff5858, #e1251b); }
.sc-green  { background: linear-gradient(135deg, #43e97b, #07c160); }

/* Dashboard sections: recent orders + top sellers */
.dashboard-section {
  margin: 0 12px 12px;
  background: #fff;
  border-radius: 12px;
  padding: 12px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}
.ds-title {
  font-size: 15px;
  font-weight: bold;
  color: #333;
  margin-bottom: 10px;
}
.ds-empty { font-size: 13px; color: #999; padding: 8px 0; }
.recent-order {
  padding: 10px 0;
  border-top: 1px solid #f5f5f5;
}
.recent-order:first-of-type { border-top: none; }
.ro-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.ro-no { font-size: 13px; color: #333; font-weight: 600; }
.ro-status {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  white-space: nowrap;
}
.ro-items { font-size: 12px; color: #666; margin-top: 4px; }
.ro-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 4px;
}
.ro-time { font-size: 11px; color: #aaa; }
.ro-total { font-size: 14px; color: #e1251b; font-weight: 600; }

.top-product {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
  border-top: 1px solid #f5f5f5;
  cursor: pointer;
}
.top-product:first-of-type { border-top: none; }
.top-product:active { background: #fafafa; }
.tp-rank {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #f0f0f0;
  color: #999;
  font-size: 12px;
  font-weight: bold;
}
.tp-rank.rank-top {
  background: linear-gradient(135deg, #ff9800, #e1251b);
  color: #fff;
}
.tp-info { flex: 1; min-width: 0; font-size: 13px; color: #333; }
.tp-sub { font-size: 11px; color: #999; margin-top: 2px; }
.tp-price { color: #e1251b; font-size: 14px; font-weight: 600; flex-shrink: 0; }
</style>
