<script setup>
import { ref, onMounted } from 'vue'
import { showToast, showSuccessToast, showDialog } from 'vant'
import { getProducts, adminCreateProduct, adminUpdateProduct, adminDeleteProduct, getCategories, uploadImage } from '../api'

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
const loading = ref(true)
const showForm = ref(false)
const editingId = ref(null)
const form = ref(emptyForm())

function emptyForm() {
  return { name: '', subtitle: '', price: 0, original_price: 0, image: '', category: '', category_id: 0, shop: '', stock: 999, sales: 0, description: '', tags: '', is_seckill: 0 }
}

onMounted(async () => {
  await loadProducts()
  try { categories.value = await getCategories() } catch (e) {}
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
    <van-cell-group v-else inset>
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
</style>
