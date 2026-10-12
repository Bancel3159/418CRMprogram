<template>
  <div class="login-container">
    <!-- 顶部品牌 -->
    <div class="brand">
      <div class="brand-mark">418</div>
      <div class="brand-text">
        <div class="brand-title">CRM 维护客户端</div>
        <div class="brand-sub">Maintenance Console</div>
      </div>
    </div>

    <!-- 登录卡片(蓝色阴影 + 圆角,与 lead 端风格统一) -->
    <div class="login-card">
      <div class="card-head">
        <div class="card-title">欢迎登录</div>
        <div class="card-sub">后端 · 维护跟进端</div>
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

      <div class="hint">
        <div class="hint-title">测试账号</div>
        <div class="hint-row"><span>普通员工</span><code>b_staff_01</code><code>pass123</code></div>
        <div class="hint-row"><span>主管</span><code>b_manager_01</code><code>pass123</code></div>
      </div>
    </div>

    <!-- 底部版本 -->
    <div class="footer">v0.1.0 · 维护端</div>
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
    message.success('登录成功')
    router.push((route.query.redirect as string) || '/pool')
  } catch (err: any) {
    message.error(err?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* 整页:浅蓝渐变(与 lead 端统一) */
.login-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 18px;
  padding: 24px;
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
  border-radius: 16px;
  padding: 22px 22px 16px;
  box-shadow:
    0 1px 0 rgba(37, 99, 235, 0.06),
    0 12px 32px rgba(37, 99, 235, 0.18);
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

/* 测试账号提示框 */
.hint {
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: 10px;
  background: #f1f5fb;
  border: 1px dashed #c7d8f7;
  font-size: 11px;
  color: #475569;
}
.hint-title {
  font-weight: 700;
  font-size: 11px;
  color: #1d4ed8;
  letter-spacing: 1px;
  margin-bottom: 6px;
}
.hint-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 3px 0;
}
.hint-row > span {
  width: 56px;
  color: #64748b;
}
.hint-row > code {
  background: #ffffff;
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid #e2e8f0;
  font-family: "SF Mono", Consolas, "Microsoft YaHei", monospace;
  font-size: 10.5px;
  color: #1e3a8a;
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
