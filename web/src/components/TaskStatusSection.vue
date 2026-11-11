<script setup lang="ts">
import { CopyDocument, Delete, Document, SwitchButton, VideoPlay, View } from '@element-plus/icons-vue'
import {
  lastCheckedSummary,
  lastCheckedText,
  logLevelTagType,
  managedTaskAccountName,
  managedTaskDetail,
  managedTaskLastCheckedAt,
  managedTaskLastMessage,
  managedTaskModeTagType,
  managedTaskModeText,
  managedTaskName,
  managedTaskSourceLabel,
  managedTaskStatus,
  managedTaskTimeSummary,
  statusLabel,
  taskProxySummary,
  taskStatusTagType,
} from '../app/format'
import { useAppStore } from '../app/useAppStore'

const {
  sseStatus,
  managedTasks,
  managedLogs,
  mobileLogMode,
  selectedManagedTask,
  selectedLogSubtitle,
  managedTaskTimeText,
  selectManagedTaskLog,
  startManagedTask,
  stopManagedTask,
  confirmDeleteManagedTask,
  copyPaymentUrl,
  showAllLogs,
  showAllMobileLogs,
} = useAppStore()
</script>

<template>
  <section class="status-layout">
    <section class="panel">
      <div class="panel-heading">
        <h3>任务状态</h3>
        <span class="muted">SSE {{ sseStatus }} · {{ selectedManagedTask ? managedTaskName(selectedManagedTask) : '未选择任务' }}</span>
      </div>
      <el-table :data="managedTasks" class="desktop-task-table" row-key="key" empty-text="暂无任务">
        <el-table-column label="任务" min-width="220">
          <template #default="{ row }">
            <strong>{{ managedTaskName(row) }}</strong>
            <small>
              <span>模式：{{ managedTaskModeText(row) }}</span>
            </small>
            <small v-if="row.kind === 'ticket' && row.task.proxyGroupName">{{ taskProxySummary(row.task) }}</small>
            <small v-if="row.kind === 'bws' && row.bwsTask.proxyGroupName">{{ taskProxySummary(row.bwsTask) }}</small>
            <small>{{ managedTaskDetail(row) }}</small>
          </template>
        </el-table-column>
        <el-table-column label="账号" min-width="130">
          <template #default="{ row }">
            {{ managedTaskAccountName(row) || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="taskStatusTagType(managedTaskStatus(row))">{{ statusLabel(managedTaskStatus(row)) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="倒计时/时间源" min-width="170">
          <template #default="{ row }">
            <strong>{{ managedTaskTimeText(row) }}</strong>
            <small>{{ managedTaskTimeSummary(row) }}</small>
          </template>
        </el-table-column>
        <el-table-column label="最近消息" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            {{ managedTaskLastMessage(row) || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="最近检测" min-width="170">
          <template #default="{ row }">
            <span>{{ lastCheckedText(managedTaskLastCheckedAt(row)) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="支付" min-width="130">
          <template #default="{ row }">
            <div v-if="row.kind === 'ticket' && row.task.status === 'waiting_payment' && row.task.paymentUrl" class="payment-cell">
              <img v-if="row.task.paymentQrImageDataUrl" :src="row.task.paymentQrImageDataUrl" alt="支付二维码" />
              <el-button size="small" :icon="CopyDocument" @click="copyPaymentUrl(row.task)">复制链接</el-button>
            </div>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button size="small" :icon="View" @click="selectManagedTaskLog(row)">日志</el-button>
              <el-button size="small" type="primary" :icon="VideoPlay" @click="startManagedTask(row)">启动</el-button>
              <el-button size="small" :icon="SwitchButton" @click="stopManagedTask(row)">停止</el-button>
              <el-button size="small" type="danger" plain :icon="Delete" @click="confirmDeleteManagedTask(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div class="mobile-task-list">
        <article v-for="row in managedTasks" :key="row.key" class="item-card">
          <div class="mobile-task-main">
            <h4>{{ managedTaskName(row) }}</h4>
            <p>{{ managedTaskAccountName(row) || '-' }} · {{ managedTaskTimeText(row) }}</p>
            <small>
              {{ managedTaskSourceLabel(row) }} · {{ managedTaskModeText(row) }}
            </small>
            <small v-if="row.kind === 'ticket' && row.task.proxyGroupName">{{ taskProxySummary(row.task) }}</small>
            <small v-if="row.kind === 'bws' && row.bwsTask.proxyGroupName">{{ taskProxySummary(row.bwsTask) }}</small>
            <small>{{ managedTaskDetail(row) }}</small>
            <small>{{ managedTaskTimeSummary(row) }}</small>
            <small>{{ lastCheckedSummary({ lastCheckedAt: managedTaskLastCheckedAt(row) }) }}</small>
            <small class="mobile-task-message">{{ managedTaskLastMessage(row) || '-' }}</small>
          </div>
          <div class="actions">
            <el-tag :type="managedTaskModeTagType(row)">{{ managedTaskModeText(row) }}</el-tag>
            <el-tag :type="taskStatusTagType(managedTaskStatus(row))">{{ statusLabel(managedTaskStatus(row)) }}</el-tag>
            <el-button :icon="View" @click="selectManagedTaskLog(row)">日志</el-button>
            <el-button type="primary" :icon="VideoPlay" @click="startManagedTask(row)">启动</el-button>
            <el-button :icon="SwitchButton" @click="stopManagedTask(row)">停止</el-button>
            <el-button type="danger" plain :icon="Delete" @click="confirmDeleteManagedTask(row)">删除</el-button>
          </div>
        </article>
        <el-empty v-if="managedTasks.length === 0" description="暂无任务" />
        <section class="mobile-log-panel">
          <div class="mobile-log-toolbar">
            <div class="log-heading-title">
              <h3>{{ mobileLogMode === 'all' ? '全部日志' : mobileLogMode === 'task' ? '运行日志' : '日志显示区' }}</h3>
              <small v-if="mobileLogMode === 'task' && selectedLogSubtitle" class="muted log-subtitle">
                {{ selectedLogSubtitle }}
              </small>
            </div>
            <el-button text :icon="Document" @click="showAllMobileLogs">全部日志</el-button>
          </div>
          <template v-if="mobileLogMode !== 'empty'">
            <article v-for="log in managedLogs" :key="log.key" class="mobile-log-line">
              <div class="mobile-log-meta">
                <el-tag :type="logLevelTagType(log.level)" size="small">{{ log.level }}</el-tag>
                <time>{{ log.createdAt }}</time>
              </div>
              <p>{{ log.message }}</p>
            </article>
            <el-empty v-if="managedLogs.length === 0" description="暂无日志" />
          </template>
          <el-empty v-else description="请选择任务查看日志，或查看全部日志" />
        </section>
      </div>
    </section>

    <section class="panel log-panel">
      <div class="panel-heading">
        <div class="log-heading-title">
          <h3>运行日志</h3>
          <small v-if="selectedLogSubtitle" class="muted log-subtitle">{{ selectedLogSubtitle }}</small>
        </div>
        <el-button text :icon="Document" @click="showAllLogs">全部</el-button>
      </div>
      <article v-for="log in managedLogs" :key="log.key" class="log-line">
        <el-tag :type="logLevelTagType(log.level)" size="small">{{ log.level }}</el-tag>
        <p>{{ log.message }}</p>
        <time>{{ log.createdAt }}</time>
      </article>
      <el-empty v-if="managedLogs.length === 0" description="暂无日志" />
    </section>
  </section>
</template>
