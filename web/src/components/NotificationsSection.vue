<script setup lang="ts">
import { Check, Close, Delete, Document, Edit, SwitchButton } from '@element-plus/icons-vue'
import { notificationProviderLabel, notificationTestTagType, notificationTokenLabel, notificationTokenPlaceholder } from '../app/format'
import { useAppStore } from '../app/useAppStore'

const {
  loading,
  notifications,
  notificationForm,
  editingNotificationId,
  canSaveNotification,
  resetNotificationForm,
  saveNotification,
  editNotification,
  confirmDeleteNotification,
  testNotification,
  setNotificationEnabled,
} = useAppStore()
</script>

<template>
  <section class="content-grid">
    <el-form class="panel form-panel" label-position="top" @submit.prevent="saveNotification">
      <div class="panel-heading">
        <h3>{{ editingNotificationId ? '编辑通知接口' : '新增通知接口' }}</h3>
        <el-button :icon="Close" text @click="resetNotificationForm">清空</el-button>
      </div>
      <el-form-item required label="接口名称">
        <el-input v-model="notificationForm.name" placeholder="例如：我的手机" clearable />
      </el-form-item>
      <el-form-item required label="接口类型">
        <el-radio-group v-model="notificationForm.provider">
          <el-radio-button label="pushplus">PushPlus</el-radio-button>
          <el-radio-button label="bark">Bark</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item required :label="notificationTokenLabel(notificationForm.provider)">
        <el-input
          v-model="notificationForm.config.token"
          type="textarea"
          :rows="3"
          :placeholder="notificationTokenPlaceholder(notificationForm.provider)"
        />
      </el-form-item>
      <el-alert type="info" show-icon :closable="false" class="form-tip-alert">
        <template #title>
          PushPlus 用 Token 推送；Bark 可填写 App 中的 Token，也可填写自建服务完整推送地址。
        </template>
      </el-alert>
      <el-button native-type="submit" type="primary" :icon="Document" :disabled="!canSaveNotification" :loading="loading">
        保存通知接口
      </el-button>
    </el-form>

    <section class="panel list-panel">
      <div class="panel-heading">
        <h3>通知接口</h3>
        <span class="muted">{{ notifications.length }} 个接口</span>
      </div>
      <article v-for="notification in notifications" :key="notification.id" class="item-card">
        <div>
          <h4>{{ notification.name }}</h4>
          <p>{{ notificationProviderLabel(notification.provider) }}</p>
          <small>{{ notification.config?.token || '未填写配置' }}</small>
          <small v-if="notification.lastTestStatus">
            最近检测：{{ notification.lastTestMessage || '-' }} {{ notification.lastTestedAt ? `· ${notification.lastTestedAt}` : '' }}
          </small>
        </div>
        <div class="actions">
          <el-tag :type="notification.enabled ? 'success' : 'info'">{{ notification.enabled ? '已启用' : '未启用' }}</el-tag>
          <el-tag :type="notificationTestTagType(notification.lastTestStatus)">
            {{ notification.lastTestStatus || '未检测' }}
          </el-tag>
          <el-button :icon="Check" :loading="loading" @click="testNotification(notification.id)">检测</el-button>
          <el-button
            :type="notification.enabled ? undefined : 'primary'"
            :icon="SwitchButton"
            @click="setNotificationEnabled(notification.id, !notification.enabled)"
          >
            {{ notification.enabled ? '停用' : '启用' }}
          </el-button>
          <el-button :icon="Edit" @click="editNotification(notification)">编辑</el-button>
          <el-button type="danger" plain :icon="Delete" @click="confirmDeleteNotification(notification)">删除</el-button>
        </div>
      </article>
      <el-empty v-if="notifications.length === 0" description="暂无通知接口" />
    </section>
  </section>
</template>
