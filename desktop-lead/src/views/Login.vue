<template>
  <div class="login-container">
    <!-- 顶部品牌 -->
    <div class="brand">
      <div class="brand-mark">海外</div>
      <div class="brand-text">
        <div class="brand-title">418前端注册提交</div>
        <div class="brand-sub">Registration Submission</div>
      </div>
    </div>

    <!-- 登录卡片(蓝色阴影 + 圆角) -->
    <div class="login-card">
      <div class="card-head">
        <div class="card-title">欢迎登录</div>
        <div class="card-sub"></div>
      </div>

      <n-form :model="form" :rules="rules" size="large" label-placement="top" ref="formRef">
        <n-form-item label="账号" path="username">
          <n-input
            v-model:value="form.username"
            placeholder="请输入登录账号"
            autofocus
            @keyup.enter="handleLogin"
          />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="请输入登录密码"
            @keyup.enter="handleLogin"
          />
        </n-form-item>

        <n-button
          type="primary"
          block
          size="large"
          :loading="loading"
          @click="handleLogin"
          class="login-btn"
        >
          登 录
        </n-button>
      </n-form>
    </div>

    <!-- 底部版本 -->
    <div class="footer">v0.1.0 · 前端 · Developed by 418</div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NForm, NFormItem, NInput, NButton, useMessage, type FormRules } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const userStore = useUserStore()

const formRef = ref<any>(null)
const loading = ref(false)
const form = reactive({ username: '', password: '' })

// 通用账号密码校验
const rules: FormRules = {
  username: { required: true, message: '请输入账号', trigger: 'blur' },
  password: { required: true, message: '请输入密码', trigger: 'blur' }
}

async function handleLogin() {
  if (!formRef.value) return
  try { await formRef.value.validate() } catch { return }
  loading.value = true
  try {
    await userStore.login(form.username, form.password)
    // 窗口尺寸已固定 400×560 默认 + 用户可自由拉缩,登录后不再改尺寸
    message.success('登录成功')
    router.push((route.query.redirect as string) || '/leads/new')
  } catch (err: any) {
    message.error(err?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* 整页:浅蓝渐变背景,垂直居中 */
.login-container {
  min-height: calc(100vh - 28px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 18px;
  padding: 16px 24px 8px;
  background: linear-gradient(160deg, #f0f6ff 0%, #dbe7fb 60%, #c7d8f7 100%);
}

/* 顶部品牌 */
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand-mark {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-weight: 800;
  font-size: 17px;
  display: flex;
  align-items: center;
  justify-content: center;
  letter-spacing: 0.5px;
  box-shadow: 0 6px 16px rgba(37, 99, 235, 0.35);
}
.brand-title {
  font-size: 17px;
  font-weight: 700;
  color: #1e3a8a;
  letter-spacing: 0.3px;
}
.brand-sub {
  font-size: 11px;
  color: #64748b;
  letter-spacing: 0.6px;
  margin-top: 2px;
  text-transform: uppercase;
}

/* 登录卡片 */
.login-card {
  width: 100%;
  max-width: 360px;
  background: #ffffff;
  border-radius: var(--r-xl, 18px);
  padding: 22px 22px 16px;
  box-shadow: 0 1px 0 rgba(37, 99, 235, 0.06), 0 12px 32px rgba(37, 99, 235, 0.18);
  transition: box-shadow 200ms ease, transform 200ms ease;
}
.login-card:hover {
  box-shadow: 0 1px 0 rgba(37, 99, 235, 0.10), 0 16px 40px rgba(37, 99, 235, 0.22);
  transform: translateY(-1px);
}

/* 卡片头部 */
.card-head {
  margin-bottom: 14px;
  text-align: left;
}
.card-title {
  font-size: 20px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: 0.3px;
}
.card-sub {
  font-size: 12px;
  color: #2563eb;
  margin-top: 4px;
  letter-spacing: 0.5px;
}

/* 表单 label 在输入框上方 */
:deep(.n-form-item-label) {
  font-size: 12px !important;
  color: #475569 !important;
  font-weight: 600 !important;
  padding: 0 0 4px 0 !important;
  text-align: left !important;
}
:deep(.n-form-item) {
  margin-bottom: 12px;
}
:deep(.n-input .n-input__wrapper) {
  border-radius: 10px !important;
}
.login-btn {
  margin-top: 4px;
  border-radius: 10px !important;
  font-weight: 600;
  letter-spacing: 4px;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.25);
}

/* 底部版本 */
.footer {
  font-size: 10px;
  color: #94a3b8;
  letter-spacing: 0.5px;
  text-align: center;
  margin-top: -8px;
}
</style>
