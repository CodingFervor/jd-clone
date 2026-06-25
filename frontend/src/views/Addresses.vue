<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast, showConfirmDialog } from 'vant'
import { getAddresses, createAddress, updateAddress, deleteAddress } from '../api'

const router = useRouter()
const list = ref([])
const loading = ref(true)
const showEdit = ref(false)
const editing = ref(null)
const form = ref({ name: '', phone: '', detail: '', is_default: 0 })

async function load() {
  loading.value = true
  try {
    list.value = await getAddresses()
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
}
onMounted(load)

function openAdd() {
  editing.value = null
  form.value = { name: '', phone: '', detail: '', is_default: 0 }
  showEdit.value = true
}
function openEdit(a) {
  editing.value = a
  form.value = { name: a.name, phone: a.phone, detail: a.detail, is_default: a.is_default }
  showEdit.value = true
}
async function save() {
  if (!form.value.name || !form.value.phone || !form.value.detail) {
    showToast('请填写完整信息')
    return
  }
  try {
    if (editing.value) {
      await updateAddress(editing.value.id, form.value)
    } else {
      await createAddress(form.value)
    }
    showSuccessToast(editing.value ? '已更新' : '已添加')
    showEdit.value = false
    await load()
  } catch (e) {
    showToast(e.response?.data?.error || '保存失败')
  }
}
async function remove(a) {
  try {
    await showConfirmDialog({ title: '提示', message: '确定删除该地址？' })
    await deleteAddress(a.id)
    showSuccessToast('已删除')
    await load()
  } catch (e) {
    // user cancelled or delete failed
  }
}
</script>

<template>
  <div class="addr-page">
    <van-nav-bar title="收货地址" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <van-empty v-else-if="!list.length" description="还没有收货地址">
      <van-button type="danger" round @click="openAdd">新增地址</van-button>
    </van-empty>
    <div v-else class="addr-list">
      <div v-for="a in list" :key="a.id" class="addr-card">
        <div class="ac-top">
          <span class="ac-name">{{ a.name }}</span>
          <span class="ac-phone">{{ a.phone }}</span>
          <van-tag v-if="a.is_default" type="danger" round>默认</van-tag>
        </div>
        <div class="ac-detail">{{ a.detail }}</div>
        <div class="ac-actions">
          <van-button size="small" plain icon="edit" @click="openEdit(a)">编辑</van-button>
          <van-button size="small" plain type="danger" icon="delete-o" @click="remove(a)">删除</van-button>
        </div>
      </div>
    </div>
    <div v-if="list.length" class="addr-add">
      <van-button block type="danger" round @click="openAdd">新增地址</van-button>
    </div>

    <van-popup v-model:show="showEdit" position="bottom" round closeable>
      <div class="edit-form">
        <h3>{{ editing ? '编辑地址' : '新增地址' }}</h3>
        <van-field v-model="form.name" label="收货人" placeholder="请输入姓名" />
        <van-field v-model="form.phone" label="手机号" placeholder="请输入手机号" type="tel" />
        <van-field v-model="form.detail" label="详细地址" type="textarea" placeholder="请输入详细地址" rows="2" />
        <div class="default-row">
          <span>设为默认地址</span>
          <van-switch v-model="form.is_default" :active-value="1" :inactive-value="0" />
        </div>
        <van-button type="danger" block @click="save">保存</van-button>
      </div>
    </van-popup>
  </div>
</template>

<style scoped>
.addr-page { min-height: 100vh; padding-bottom: 80px; }
.loading { text-align: center; padding: 80px; }
.addr-card { background: #fff; margin: 8px; border-radius: 8px; padding: 14px; }
.ac-top { display: flex; align-items: center; gap: 10px; margin-bottom: 6px; }
.ac-name { font-size: 16px; font-weight: bold; }
.ac-phone { font-size: 14px; color: #666; }
.ac-detail { font-size: 14px; color: #333; line-height: 20px; }
.ac-actions { display: flex; gap: 8px; margin-top: 10px; border-top: 1px solid #f5f5f5; padding-top: 10px; }
.addr-add { padding: 16px; }
.edit-form { padding: 20px; }
.edit-form h3 { text-align: center; margin-bottom: 16px; }
.edit-form .van-field { margin-bottom: 8px; border: 1px solid #eee; }
.default-row { display: flex; justify-content: space-between; align-items: center; padding: 12px 4px; }
.edit-form .van-button { margin-top: 12px; }
</style>
