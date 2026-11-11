<script setup lang="ts">
import { Check, CirclePlus } from '@element-plus/icons-vue'
import { useAppStore } from '../app/useAppStore'

const {
  userAuth,
  loading,
  error,
  qrLogin,
  qrStatusLabel,
  qrStatusMessage,
  qrMessageAlertType,
  qrStatusTagType,
  startQRLogin,
  pollQRLogin,
} = useAppStore()
</script>

<template>
  <section v-if="userAuth.checking" class="auth-shell">
    <div class="auth-card">
      <p class="eyebrow">Biligo</p>
      <h1>票务控制台</h1>
      <p class="muted">正在校验登录状态...</p>
    </div>
  </section>

  <section v-else-if="!userAuth.authenticated" class="auth-shell">
    <div class="auth-card">
      <div>
        <p class="eyebrow">Biligo</p>
        <h1>票务控制台</h1>
        <p class="muted">使用哔哩哔哩 App 扫码登录。</p>
      </div>
      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" class="global-message-alert" />
      <div class="qr-status-row">
        <el-tag :type="qrStatusTagType()">{{ qrStatusLabel }}</el-tag>
        <span class="muted">{{ qrLogin.autoPolling ? '自动轮询中' : '自动轮询已停止' }}</span>
        <span v-if="qrLogin.lastCheckedAt" class="muted">上次检查 {{ qrLogin.lastCheckedAt }}</span>
      </div>
      <el-alert
        :title="qrStatusMessage"
        :type="qrMessageAlertType"
        show-icon
        :closable="false"
        class="qr-message-alert"
      />
      <div v-if="qrLogin.qrImageDataUrl" class="qr-preview">
        <img :src="qrLogin.qrImageDataUrl" alt="Bilibili 登录二维码" />
      </div>
      <div class="button-row">
        <el-button type="primary" :icon="CirclePlus" :loading="loading || qrLogin.polling" @click="startQRLogin">
          {{ qrLogin.qrImageDataUrl ? '刷新二维码' : '生成二维码' }}
        </el-button>
        <el-button :icon="Check" :disabled="loading || qrLogin.polling || !qrLogin.qrcodeKey" @click="pollQRLogin">
          检查登录
        </el-button>
      </div>
    </div>
  </section>
</template>
