<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast, showSuccessToast } from 'vant'
import { getProfile, updateProfile, uploadImage } from '../api'

const router = useRouter()
const form = ref({ nickname: '', avatar: '', phone: '' })
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await getProfile()
    const u = res.user
    form.value = { nickname: u.nickname || '', avatar: u.avatar || '', phone: u.phone || '' }
  } catch (e) {
    showToast('加载失败')
  } finally {
    loading.value = false
  }
})

async function onUploadAvatar(item) {
  try {
    const res = await uploadImage(item.file)
    form.value.avatar = res.url
    showSuccessToast('头像已更新')
  } catch (e) {
    showToast('上传失败')
  }
}

async function save() {
  try {
    await updateProfile(form.value)
    showSuccessToast('资料已更新')
    router.back()
  } catch (e) {
    showToast(e.response?.data?.error || '保存失败')
  }
}
</script>

<template>
  <div class="profile-page">
    <van-nav-bar title="编辑资料" left-arrow @click-left="router.back()" fixed placeholder />
    <div v-if="loading" class="loading"><van-loading /></div>
    <div v-else>
      <van-cell-group inset>
        <div class="avatar-cell">
          <span>头像</span>
          <div class="avatar-right">
            <van-image round width="56" height="56" :src="form.avatar || 'https://via.placeholder.com/56'" fit="cover" />
            <van-uploader :after-read="onUploadAvatar" accept="image/*" :preview-image="false">
              <van-button size="small" plain round>更换</van-button>
            </van-uploader>
          </div>
        </div>
        <van-field v-model="form.nickname" label="昵称" placeholder="请输入昵称" />
        <van-field v-model="form.phone" label="手机号" placeholder="请输入手机号" type="tel" />
      </van-cell-group>
      <div class="save-btn">
        <van-button block type="danger" round @click="save">保存</van-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.profile-page { min-height: 100vh; }
.loading { text-align: center; padding: 80px; }
.avatar-cell { display: flex; justify-content: space-between; align-items: center; padding: 14px 16px; }
.avatar-right { display: flex; align-items: center; gap: 12px; }
.save-btn { padding: 24px 16px; }
</style>
