<script setup lang="ts">
import { CirclePlus, Close, Delete, Document, Edit, Refresh, SwitchButton, VideoPlay, View } from '@element-plus/icons-vue'
import {
  bwsItemTagText,
  bwsReserveTimeLabel,
  bwsReserveTypeLabel,
  bwsTaskModeLabel,
  formatUnixSeconds,
  proxyGroupOptionLabel,
  statusLabel,
  taskProxySummary,
  taskStatusTagType,
} from '../app/format'
import { useAppStore } from '../app/useAppStore'

const {
  loading,
  accounts,
  bwsQueryForm,
  bwsTaskForm,
  bwsItems,
  bwsMyReservations,
  bwsTasks,
  bwsHideEnded,
  bwsHideReserved,
  selectedBWSItemId,
  editingBWSTaskId,
  filteredBWSItems,
  selectableProxyGroups,
  canSaveBWSTask,
  fetchBWSInfo,
  fetchBWSMyReservations,
  selectBWSItem,
  resetBWSTaskForm,
  saveBWSTask,
  normalizeBWSTaskProxyFields,
  editBWSTask,
  dispatchBWSTask,
  pauseBWSTask,
  confirmDeleteBWSTask,
  selectBWSTaskLog,
} = useAppStore()
</script>

<template>
  <section class="content-grid bws-layout">
    <div class="stack-column">
      <el-form class="panel form-panel bws-info-panel" label-position="top" @submit.prevent="fetchBWSInfo">
        <div class="panel-heading">
          <h3>BWS 预约信息</h3>
          <span class="muted">{{ filteredBWSItems.length }}/{{ bwsItems.length }} 个条目</span>
        </div>
        <el-form-item required label="账号">
          <el-select v-model="bwsQueryForm.accountId" placeholder="选择账号" filterable>
            <el-option v-for="account in accounts" :key="account.id" :value="account.id" :label="account.name" />
          </el-select>
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :sm="10">
            <el-form-item required label="年份">
              <el-input v-model="bwsQueryForm.year" placeholder="202601" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="14">
            <el-form-item required label="预约日期">
              <el-input v-model="bwsQueryForm.reserveDates" placeholder="20260710,20260711,20260712" clearable />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item required label="预约类型">
          <el-radio-group v-model="bwsQueryForm.reserveType">
            <el-radio-button label="activity">活动预约</el-radio-button>
            <el-radio-button label="goods">商品购买资格</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <div class="table-actions">
          <el-checkbox v-model="bwsHideEnded">隐藏已结束</el-checkbox>
          <el-checkbox v-model="bwsHideReserved">隐藏已预约</el-checkbox>
        </div>
        <div class="form-actions">
          <el-button native-type="submit" type="primary" :icon="Refresh" :loading="loading" :disabled="bwsQueryForm.accountId <= 0">
            拉取列表
          </el-button>
          <el-button :icon="View" :loading="loading" :disabled="bwsQueryForm.accountId <= 0" @click="fetchBWSMyReservations">
            我的预约
          </el-button>
        </div>
      </el-form>

      <el-form class="panel form-panel" label-position="top" @submit.prevent="saveBWSTask">
        <div class="panel-heading">
          <h3>{{ editingBWSTaskId ? '编辑 BWS 任务' : '新增 BWS 任务' }}</h3>
          <el-button :icon="Close" text @click="resetBWSTaskForm">清空</el-button>
        </div>
        <el-form-item required label="任务名称">
          <el-input v-model="bwsTaskForm.name" placeholder="选择条目后自动填充" clearable />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :xs="24" :sm="12">
            <el-form-item required label="账号">
              <el-select v-model="bwsTaskForm.accountId" filterable>
                <el-option v-for="account in accounts" :key="account.id" :value="account.id" :label="account.name" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="时间同步策略">
              <el-select v-model="bwsTaskForm.timeSyncStrategy">
                <el-option value="bilibili" label="哔哩哔哩时间" />
                <el-option value="local" label="本地时间" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="执行模式">
              <el-radio-group v-model="bwsTaskForm.mode" @change="normalizeBWSTaskProxyFields">
                <el-radio-button value="scheduled">等待开票</el-radio-button>
                <el-radio-button value="immediate">立刻抢</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :xs="24" :sm="12">
            <el-form-item required label="循环间隔（ms）">
              <el-input-number v-model="bwsTaskForm.pollIntervalMillis" :min="1" controls-position="right" class="full-input" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="代理组">
              <el-select v-model="bwsTaskForm.proxyGroupId" placeholder="不使用代理" filterable clearable @change="normalizeBWSTaskProxyFields">
                <el-option :value="0" label="不使用代理" />
                <el-option
                  v-for="group in selectableProxyGroups"
                  :key="group.id"
                  :value="group.id"
                  :label="proxyGroupOptionLabel(group)"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col v-if="Number(bwsTaskForm.proxyGroupId || 0) > 0" :xs="24" :sm="12">
            <el-form-item label="代理模式">
              <el-radio-group v-model="bwsTaskForm.proxyMode" @change="normalizeBWSTaskProxyFields">
                <el-radio-button value="round_robin">循环代理</el-radio-button>
                <el-radio-button value="concurrent">并发代理</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col v-if="Number(bwsTaskForm.proxyGroupId || 0) > 0" :xs="24">
            <p class="field-hint">代理只用于 BWS 预约执行请求；活动列表和我的预约仍直连。</p>
          </el-col>
        </el-row>
        <div class="ticket-summary" v-if="bwsTaskForm.reserveId">
          <strong>{{ bwsTaskForm.title }}</strong>
          <span>{{ bwsReserveTypeLabel(bwsTaskForm.reserveType) }} · {{ bwsTaskForm.reserveDate }}</span>
          <span>预约 ID {{ bwsTaskForm.reserveId }} · 票号 {{ bwsTaskForm.ticketNo }}</span>
          <span>预约开始 {{ bwsReserveTimeLabel(bwsTaskForm) }}</span>
        </div>
        <el-button native-type="submit" type="primary" :icon="Document" :disabled="!canSaveBWSTask" :loading="loading">
          保存 BWS 任务
        </el-button>
      </el-form>

      <section class="panel list-panel">
        <div class="panel-heading">
          <h3>我的预约</h3>
          <span class="muted">{{ bwsMyReservations.length }} 条</span>
        </div>
        <article v-for="item in bwsMyReservations" :key="`${item.date}-${item.reserveId}-${item.reserveNo}`" class="item-card">
          <div>
            <h4>{{ item.title }}</h4>
            <p>{{ item.date }} · {{ item.reserveLocation || '-' }}</p>
            <small>#{{ item.reserveNo || '-' }} · {{ formatUnixSeconds(item.actBeginTime) }} - {{ formatUnixSeconds(item.actEndTime) }}</small>
          </div>
          <div class="actions">
            <el-tag type="success">{{ item.status || '预约成功' }}</el-tag>
          </div>
        </article>
        <el-empty v-if="bwsMyReservations.length === 0" description="暂无我的预约" />
      </section>
    </div>

    <div class="stack-column">
      <section class="panel list-panel">
        <div class="panel-heading">
          <h3>BWS 条目</h3>
          <span class="muted">{{ bwsReserveTypeLabel(bwsQueryForm.reserveType) }}</span>
        </div>
        <article
          v-for="item in filteredBWSItems"
          :key="`${item.date}-${item.reserveId}`"
          class="item-card"
          :class="{ selected: selectedBWSItemId === item.reserveId }"
        >
          <div>
            <h4>{{ item.title }}</h4>
            <p>{{ item.date }} · {{ formatUnixSeconds(item.actBeginTime) }} · {{ item.reserveLocation || '-' }}</p>
            <small>预约开始：{{ bwsReserveTimeLabel(item) }} · 票号：{{ item.ticketNo || '-' }}</small>
            <small v-if="item.describeInfo">{{ item.describeInfo }}</small>
          </div>
          <div class="actions">
            <el-tag v-if="bwsItemTagText(item)" type="warning">{{ bwsItemTagText(item) }}</el-tag>
            <el-button type="primary" :icon="CirclePlus" :disabled="!item.ticketNo" @click="selectBWSItem(item)">生成任务</el-button>
          </div>
        </article>
        <el-empty v-if="filteredBWSItems.length === 0" description="暂无 BWS 条目" />
      </section>

      <section class="panel list-panel">
        <div class="panel-heading">
          <h3>BWS 任务</h3>
          <span class="muted">{{ bwsTasks.length }} 个任务</span>
        </div>
        <article v-for="task in bwsTasks" :key="task.id" class="item-card">
          <div>
            <h4>{{ task.name }}</h4>
            <p>{{ task.title }} · {{ task.accountName || '-' }}</p>
            <small>{{ bwsReserveTypeLabel(task.reserveType) }} · {{ task.reserveDate }}</small>
            <small>{{ bwsTaskModeLabel(task.mode) }} · 预约开始：{{ bwsReserveTimeLabel(task) }} · 间隔 {{ task.pollIntervalMillis }}ms</small>
            <small v-if="task.proxyGroupName">{{ taskProxySummary(task) }}</small>
            <small>{{ task.lastMessage || '-' }}</small>
          </div>
          <div class="actions">
            <el-tag :type="taskStatusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
            <el-button :icon="View" @click="selectBWSTaskLog(task)">日志</el-button>
            <el-button type="primary" :icon="VideoPlay" @click="dispatchBWSTask(task.id)">启动</el-button>
            <el-button :icon="SwitchButton" @click="pauseBWSTask(task.id)">停止</el-button>
            <el-button :icon="Edit" :disabled="task.status === 'running' || task.status === 'waiting_start'" @click="editBWSTask(task)">编辑</el-button>
            <el-button type="danger" plain :icon="Delete" @click="confirmDeleteBWSTask(task)">删除</el-button>
          </div>
        </article>
        <el-empty v-if="bwsTasks.length === 0" description="暂无 BWS 任务" />
      </section>
    </div>
  </section>
</template>
