<template>
  <div class="login-container">
    <div class="login-box">
      <div class="logo-area">
        <div class="logo-text">418 CRM</div>
        <div class="logo-subtitle">数据中台 · 游戏推广管理</div>
      </div>

      <n-form
        ref="formRef"
        :model="form"
        :rules="rules"
        size="large"
        label-placement="left"
        label-width="80"
      >
        <n-form-item label="账号" path="username">
          <n-input v-model:value="form.username" placeholder="请输入账号" clearable />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="请输入密码"
            @keyup.enter="handleLogin"
          />
        </n-form-item>

        <!-- Cloudflare Turnstile 占位（开发环境跳过） -->
        <n-form-item v-if="needTurnstile" label="验证">
          <div class="turnstile-placeholder">
            <n-tag>Cloudflare Turnstile</n-tag>
            <span class="text-muted" style="margin-left: 8px">
              生产环境配置 TURNSTILE_SECRET 后启用
            </span>
          </div>
        </n-form-item>

        <n-button
          type="primary"
          block
          :loading="loading"
          @click="handleLogin"
        >
          登录
        </n-button>
      </n-form>

      <div class="tips">
        <p>默认账号：admin / admin123</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NForm, NFormItem, NInput, NButton, NTag, useMessage, type FormRules } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const userStore = useUserStore()

const formRef = ref<InstanceType<typeof NForm> | null>(null)
const loading = ref(false)
const needTurnstile = computed(() => false) // 开发环境始终 false

const form = reactive({
  username: '',
  password: ''
})

const rules: FormRules = {
  username: { required: true, message: '请输入账号', trigger: 'blur' },
  password: { required: true, message: '请输入密码', trigger: 'blur' }
}

async function handleLogin() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }

  loading.value = true
  try {
    await userStore.login(form.username, form.password, '')
    message.success('登录成功')
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.push(redirect)
  } catch (err: any) {
    message.error(err?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #18a058 0%, #0c7a43 100%);
}

.login-box {
  background: white;
  border-radius: 12px;
  padding: 40px;
  width: 420px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
}

.logo-area {
  text-align: center;
  margin-bottom: 32px;
}

.logo-text {
  font-size: 32px;
  font-weight: 800;
  color: #18a058;
  letter-spacing: 2px;
}

.logo-subtitle {
  font-size: 13px;
  color: #6b7280;
  margin-top: 8px;
}

.turnstile-placeholder {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.tips {
  margin-top: 16px;
  text-align: center;
  color: #9ca3af;
  font-size: 12px;
}
</style>