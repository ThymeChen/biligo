<script setup lang="ts">
import { Close, Delete, Document, Edit, Search, SwitchButton, User, VideoPlay } from '@element-plus/icons-vue'
import { superModeBaseOptions } from '../app/constants'
import {
  addressLabel,
  buyerLabel,
  formatMoney,
  lastCheckedSummary,
  proxyGroupOptionLabel,
  statusLabel,
  taskBuyerSummary,
  taskModeLabel,
  taskModeTagType,
  taskProxySummary,
  taskStatusTagType,
  taskTicketSummary,
} from '../app/format'
import { useAppStore } from '../app/useAppStore'

const {
  loading,
  accounts,
  taskForm,
  editingTaskId,
  ticketProjectInput,
  fetchedTicketProject,
  ticketOptions,
  ticketBuyers,
  ticketAddresses,
  selectedTicketValue,
  selectedTicketValues,
  selectedBuyerIndexes,
  selectedAddressId,
  selectedTicketOption,
  selectedTicketOptions,
  hasFetchedTicketInfo,
  isRestockTaskForm,
  isHybridTaskForm,
  isRestockUnlimitedTaskForm,
  hasRushTaskSection,
  hasRestockTaskSection,
  selectableProxyGroups,
  canSaveTask,
  pendingTasks,
  issuedTasks,
  resetTaskForm,
  saveTask,
  handleTaskModeChange,
  handleDurationModeChange,
  handleTaskAccountChange,
  handleSaleStartChange,
  normalizeTaskModeFields,
  ticketProjectHistorySuggestions,
  selectTicketProjectHistorySuggestion,
  applyTicketProjectHistory,
  fetchTicketProject,
  fetchTicketAccountContext,
  selectTicketOption,
  selectRestockTickets,
  updateSelectedBuyers,
  updateSelectedAddress,
  editTask,
  dispatchTask,
  stopTask,
  confirmDeleteTask,
} = useAppStore()
</script>

<template>
  <section class="content-grid">
    <el-form class="panel form-panel" label-position="top" @submit.prevent="saveTask">
      <div class="panel-heading">
        <h3>{{ editingTaskId ? '编辑任务' : '新增任务' }}</h3>
        <el-button :icon="Close" text @click="resetTaskForm">清空</el-button>
      </div>
      <el-form-item required label="任务名称">
        <el-input v-model="taskForm.name" placeholder="例如：上海场 2 张" clearable />
      </el-form-item>
      <el-form-item required label="任务模式">
        <el-radio-group v-model="taskForm.taskMode" @change="handleTaskModeChange">
          <el-radio-button label="rush">抢票模式</el-radio-button>
          <el-radio-button label="restock">回流捡漏模式</el-radio-button>
          <el-radio-button label="rush_restock">抢票+回流捡漏模式</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="isRestockTaskForm" required label="回流捡漏时长">
        <el-radio-group v-model="taskForm.durationMode" @change="handleDurationModeChange">
          <el-radio-button label="limited">有限模式</el-radio-button>
          <el-radio-button label="unlimited">无限模式</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item required label="抢票项目 ID">
        <div class="input-action-row">
          <el-autocomplete
            v-model="ticketProjectInput"
            :fetch-suggestions="ticketProjectHistorySuggestions"
            value-key="value"
            placeholder="项目 ID 或详情页链接"
            clearable
            @change="applyTicketProjectHistory"
            @select="selectTicketProjectHistorySuggestion"
          >
            <template #default="{ item }">
              <div class="history-suggestion">
                <strong>{{ item.projectName || '未命名项目' }}</strong>
                <span>ID {{ item.projectId }}</span>
                <small>{{ [item.venueName, item.venueAddress].filter(Boolean).join(' ') || '暂无场馆信息' }}</small>
              </div>
            </template>
          </el-autocomplete>
          <el-button :icon="Search" :loading="loading" :disabled="!ticketProjectInput.trim()" @click="fetchTicketProject">
            获取信息
          </el-button>
        </div>
      </el-form-item>
      <div v-if="fetchedTicketProject" class="ticket-summary">
        <strong>{{ fetchedTicketProject.projectName || '未命名项目' }}</strong>
        <span>ID {{ fetchedTicketProject.projectId }}</span>
        <span v-if="fetchedTicketProject.venueName || fetchedTicketProject.venueAddress">
          {{ [fetchedTicketProject.venueName, fetchedTicketProject.venueAddress].filter(Boolean).join(' ') }}
        </span>
        <span v-if="fetchedTicketProject.startAt || fetchedTicketProject.endAt">
          {{ fetchedTicketProject.startAt || '-' }} 至 {{ fetchedTicketProject.endAt || '-' }}
        </span>
      </div>

      <template v-if="taskForm.taskMode === 'rush'">
        <el-form-item required label="票信息">
          <el-select
            v-model="selectedTicketValue"
            :disabled="ticketOptions.length === 0"
            placeholder="选择票信息"
            filterable
            @change="selectTicketOption"
          >
            <el-option v-for="ticket in ticketOptions" :key="ticket.value" :value="ticket.value" :label="ticket.display" />
          </el-select>
        </el-form-item>
        <div v-if="selectedTicketOption" class="ticket-detail-grid">
          <span>{{ selectedTicketOption.screenName }}</span>
          <span>{{ selectedTicketOption.ticketLevel }}</span>
          <span>{{ selectedTicketOption.priceText }}</span>
          <span>{{ selectedTicketOption.saleStatus }}</span>
        </div>
      </template>

      <template v-if="isRestockTaskForm">
        <el-form-item required label="回流票种">
          <el-select
            v-model="selectedTicketValues"
            :disabled="ticketOptions.length === 0"
            placeholder="选择一个或多个票种"
            filterable
            multiple
            collapse-tags
            collapse-tags-tooltip
            @change="selectRestockTickets"
          >
            <el-option v-for="ticket in ticketOptions" :key="ticket.value" :value="ticket.value" :label="ticket.display" />
          </el-select>
        </el-form-item>
        <el-alert type="info" show-icon :closable="false" class="form-tip-alert">
          <template #title>
            若选择多个票种，则按每次检测中第一个可购买的票种下单。
          </template>
        </el-alert>
        <div v-if="selectedTicketOptions.length > 0" class="ticket-detail-grid">
          <span>已选 {{ selectedTicketOptions.length }} 个票种</span>
          <span v-for="ticket in selectedTicketOptions" :key="ticket.value">{{ ticket.display }}</span>
        </div>
      </template>

      <section v-if="isHybridTaskForm" class="task-mode-section">
        <div class="subsection-heading">
          <h4>抢票阶段</h4>
          <span>等待起售后直接提交订单</span>
        </div>
        <el-form-item required label="抢票票种">
          <el-select
            v-model="selectedTicketValue"
            :disabled="ticketOptions.length === 0"
            placeholder="选择抢票票种"
            filterable
            @change="selectTicketOption"
          >
            <el-option v-for="ticket in ticketOptions" :key="ticket.value" :value="ticket.value" :label="ticket.display" />
          </el-select>
        </el-form-item>
        <div v-if="selectedTicketOption" class="ticket-detail-grid">
          <span>{{ selectedTicketOption.screenName }}</span>
          <span>{{ selectedTicketOption.ticketLevel }}</span>
          <span>{{ selectedTicketOption.priceText }}</span>
          <span>{{ selectedTicketOption.saleStatus }}</span>
        </div>
        <el-row :gutter="12">
          <el-col :xs="24" :sm="12">
            <el-form-item label="时间同步策略">
              <el-select v-model="taskForm.timeSyncStrategy">
                <el-option value="bilibili" label="哔哩哔哩时间" />
                <el-option value="local" label="本地时间" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item required label="起售时间">
              <el-date-picker
                v-model="taskForm.saleStart"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm:ss"
                placeholder="选择起售时间"
                class="full-input"
                @change="handleSaleStartChange"
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-alert type="info" show-icon :closable="false" class="form-tip-alert">
          <template #title>
            <strong>时间同步提示</strong>
          </template>
          <div class="tip-copy">
            <span>默认在任务下发时请求哔哩哔哩时间接口同步。</span>
            <span>建议距离开票时间大于 1 分钟时使用，时间过近可切换本地时间。</span>
          </div>
        </el-alert>
      </section>

      <section v-if="isHybridTaskForm" class="task-mode-section switch-section">
        <el-form-item required label="模式切换">
          <div class="inline-number-row">
            <span>开始抢票</span>
            <el-input-number v-model="taskForm.rushDurationSeconds" :min="1" controls-position="right" />
            <span>秒后变成回流捡漏</span>
          </div>
        </el-form-item>
      </section>

      <section v-if="isHybridTaskForm" class="task-mode-section">
        <div class="subsection-heading">
          <h4>回流捡漏阶段</h4>
          <span>每轮检测票种 clickable 状态</span>
        </div>
        <el-form-item required label="回流票种">
          <el-select
            v-model="selectedTicketValues"
            :disabled="ticketOptions.length === 0"
            placeholder="选择一个或多个票种"
            filterable
            multiple
            collapse-tags
            collapse-tags-tooltip
            @change="selectRestockTickets"
          >
            <el-option v-for="ticket in ticketOptions" :key="ticket.value" :value="ticket.value" :label="ticket.display" />
          </el-select>
        </el-form-item>
        <el-alert
          type="info"
          show-icon
          :closable="false"
          class="form-tip-alert"
        >
          <template #title>
            若选择多个票种，则按每次检测中第一个可购买的票种下单。
          </template>
        </el-alert>
        <div v-if="selectedTicketOptions.length > 0" class="ticket-detail-grid">
          <span>已选 {{ selectedTicketOptions.length }} 个票种</span>
          <span v-for="ticket in selectedTicketOptions" :key="ticket.value">{{ ticket.display }}</span>
        </div>
      </section>
      <el-form-item required label="账号">
        <div class="input-action-row">
          <el-select v-model="taskForm.accountId" placeholder="未选择" @change="handleTaskAccountChange">
            <el-option :value="0" label="未选择" />
            <el-option v-for="account in accounts" :key="account.id" :value="account.id" :label="account.name" />
          </el-select>
          <el-button
            :icon="User"
            :loading="loading"
            :disabled="taskForm.accountId <= 0 || !hasFetchedTicketInfo"
            @click="fetchTicketAccountContext"
          >
            获取信息
          </el-button>
        </div>
      </el-form-item>
      <section v-if="hasRushTaskSection" class="proxy-setting-section">
        <div class="subsection-heading">
          <h4>代理设置</h4>
          <span>仅抢票阶段生效</span>
        </div>
        <el-form-item label="代理组">
          <el-select v-model="taskForm.proxyGroupId" placeholder="不使用代理" filterable clearable @change="normalizeTaskModeFields">
            <el-option :value="0" label="不使用代理" />
            <el-option
              v-for="group in selectableProxyGroups"
              :key="group.id"
              :value="group.id"
              :label="proxyGroupOptionLabel(group)"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="Number(taskForm.proxyGroupId || 0) > 0" label="代理模式">
          <el-radio-group v-model="taskForm.proxyMode">
            <el-radio-button label="round_robin">循环代理</el-radio-button>
            <el-radio-button label="concurrent">并发代理</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <p v-if="Number(taskForm.proxyGroupId || 0) > 0 && taskForm.proxyMode === 'concurrent'" class="field-hint">
          每个可用代理节点启动一个抢票线程，任一线程成功后停止其他线程。
        </p>
      </section>
      <el-form-item required label="实名购票人">
        <div class="buyer-check-list" :class="{ disabled: ticketBuyers.length === 0 }">
          <el-checkbox-group v-model="selectedBuyerIndexes" @change="updateSelectedBuyers">
            <el-checkbox v-for="(buyer, index) in ticketBuyers" :key="`${buyer.id ?? buyer.name}-${index}`" :label="index">
              {{ buyerLabel(buyer) }}
            </el-checkbox>
          </el-checkbox-group>
          <span v-if="ticketBuyers.length === 0" class="muted">请先在账号行获取信息</span>
        </div>
      </el-form-item>
      <el-form-item required label="收货地址">
        <el-select v-model="selectedAddressId" :disabled="ticketAddresses.length === 0" placeholder="未选择" filterable @change="updateSelectedAddress">
          <el-option :value="0" label="未选择" />
          <el-option v-for="address in ticketAddresses" :key="address.id" :value="address.id" :label="addressLabel(address)" />
        </el-select>
      </el-form-item>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12">
          <el-form-item required label="联系人姓名">
            <el-input v-model="taskForm.buyer" placeholder="用于订单联系人" clearable />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item required label="联系人电话">
            <el-input v-model="taskForm.tel" placeholder="用于订单联系人" clearable />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="支付手机号">
        <el-input v-model="taskForm.phone" placeholder="可选" clearable />
      </el-form-item>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12">
          <el-form-item label="张数">
            <el-input :model-value="String(taskForm.buyerInfo.length || 0)" disabled />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item label="预计金额">
            <el-input :model-value="formatMoney(taskForm.payMoney)" disabled />
          </el-form-item>
        </el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col v-if="!isHybridTaskForm" :xs="24" :sm="12">
          <el-form-item required label="重试间隔（ms）">
            <el-input-number v-model="taskForm.pollIntervalMillis" :min="1" controls-position="right" class="full-input" />
          </el-form-item>
        </el-col>
        <template v-else>
          <el-col :xs="24" :sm="12">
            <el-form-item required label="抢票阶段重试间隔（ms）">
              <el-input-number v-model="taskForm.rushPollIntervalMillis" :min="1" controls-position="right" class="full-input" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item required label="回流阶段重试间隔（ms）">
              <el-input-number v-model="taskForm.restockPollIntervalMillis" :min="1" controls-position="right" class="full-input" />
            </el-form-item>
          </el-col>
        </template>
        <el-col v-if="taskForm.taskMode === 'rush'" :xs="24" :sm="12">
          <el-form-item label="时间同步策略">
            <el-select v-model="taskForm.timeSyncStrategy">
              <el-option value="bilibili" label="哔哩哔哩时间" />
              <el-option value="local" label="本地时间" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
      <el-alert v-if="taskForm.taskMode === 'rush'" type="info" show-icon :closable="false" class="form-tip-alert">
        <template #title>
          <strong>时间同步提示</strong>
        </template>
        <div class="tip-copy">
          <span>默认在任务下发时请求哔哩哔哩时间接口同步。</span>
          <span>建议距离开票时间大于 1 分钟时使用，时间过近可切换本地时间。</span>
        </div>
      </el-alert>
      <el-row v-if="hasRushTaskSection || hasRestockTaskSection" :gutter="12">
        <el-col :xs="24" :sm="12">
          <el-form-item>
            <template #label>
              <el-tooltip
                placement="top"
                content="开启后，抢票或回流下单阶段遇到 412 或 3 会自动切换域名，尝试降低 412 出现概率。"
              >
                <span class="tooltip-label">SuperMode</span>
              </el-tooltip>
            </template>
            <el-switch v-model="taskForm.superMode" active-text="开启" inactive-text="关闭" />
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item label="首选域名">
            <el-select v-model="taskForm.superModeBase" :disabled="!taskForm.superMode">
              <el-option
                v-for="option in superModeBaseOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="订单类型">
        <el-input :model-value="String(taskForm.orderType)" disabled />
      </el-form-item>
      <el-form-item v-if="isHybridTaskForm" required label="回流捡漏时长">
        <el-radio-group v-model="taskForm.durationMode" @change="handleDurationModeChange">
          <el-radio-button label="limited">有限模式</el-radio-button>
          <el-radio-button label="unlimited">无限模式</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-row v-if="!isRestockUnlimitedTaskForm" :gutter="12">
        <el-col :xs="24" :sm="12">
          <el-form-item :required="hasRestockTaskSection" :label="hasRestockTaskSection ? '截止时间' : '结束时间'">
            <el-date-picker
              v-model="taskForm.endAt"
              type="datetime"
              value-format="YYYY-MM-DDTHH:mm"
              format="YYYY-MM-DD HH:mm"
              placeholder=""
              class="full-input"
            />
          </el-form-item>
        </el-col>
        <el-col v-if="taskForm.taskMode === 'rush'" :xs="24" :sm="12">
          <el-form-item required label="起售时间">
            <el-date-picker
              v-model="taskForm.saleStart"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              format="YYYY-MM-DD HH:mm:ss"
              placeholder="选择起售时间"
              class="full-input"
              @change="handleSaleStartChange"
            />
          </el-form-item>
        </el-col>
      </el-row>
      <p v-if="!hasRestockTaskSection" class="field-hint">不填写则默认开票后 10 分钟停止。</p>
      <p v-else-if="isRestockTaskForm && taskForm.durationMode === 'limited'" class="field-hint">有限模式达到截止时间后停止检测。</p>
      <p v-else-if="taskForm.durationMode === 'limited'" class="field-hint">有限模式达到截止时间后停止回流检测。</p>
      <p v-else class="field-hint">无限模式不会按时间自动停止，请在任务管理中手动停止。</p>
      <el-button native-type="submit" type="primary" :icon="Document" :disabled="!canSaveTask" :loading="loading">
        保存任务
      </el-button>
    </el-form>

    <div class="stack-column">
      <section class="panel list-panel">
        <div class="panel-heading">
          <h3>待下发任务</h3>
          <span class="muted">{{ pendingTasks.length }} 个任务</span>
        </div>
        <article v-for="task in pendingTasks" :key="task.id" class="item-card">
          <div>
            <h4>{{ task.name }}</h4>
            <p>{{ task.projectName || '未选择项目' }} · {{ task.accountName || '未选择账号' }}</p>
            <small v-if="task.proxyGroupName">{{ taskProxySummary(task) }}</small>
            <small>{{ taskTicketSummary(task) }}</small>
            <small>{{ taskBuyerSummary(task) }}</small>
            <small>{{ lastCheckedSummary(task) }}</small>
          </div>
          <div class="actions">
            <el-tag :type="taskModeTagType(task)">{{ taskModeLabel(task) }}</el-tag>
            <el-tag :type="taskStatusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
            <el-button :icon="Edit" @click="editTask(task)">编辑</el-button>
            <el-button type="primary" :icon="VideoPlay" @click="dispatchTask(task.id)">下发</el-button>
            <el-button type="danger" plain :icon="Delete" @click="confirmDeleteTask(task)">删除</el-button>
          </div>
        </article>
        <el-empty v-if="pendingTasks.length === 0" description="暂无待下发任务" />
      </section>

      <section class="panel list-panel">
        <div class="panel-heading">
          <h3>已下发任务</h3>
          <span class="muted">{{ issuedTasks.length }} 个任务</span>
        </div>
        <article v-for="task in issuedTasks" :key="task.id" class="item-card">
          <div>
            <h4>{{ task.name }}</h4>
            <p>{{ task.projectName || '未选择项目' }} · {{ task.accountName || '未选择账号' }}</p>
            <small v-if="task.proxyGroupName">{{ taskProxySummary(task) }}</small>
            <small>{{ taskTicketSummary(task) }}</small>
            <small>{{ taskBuyerSummary(task) }}</small>
            <small>{{ lastCheckedSummary(task) }}</small>
          </div>
          <div class="actions">
            <el-tag :type="taskModeTagType(task)">{{ taskModeLabel(task) }}</el-tag>
            <el-tag :type="taskStatusTagType(task.status)">{{ statusLabel(task.status) }}</el-tag>
            <el-button :icon="Edit" disabled title="请先停止任务后再编辑">编辑</el-button>
            <el-button :icon="SwitchButton" @click="stopTask(task.id)">停止</el-button>
            <el-button type="danger" plain :icon="Delete" @click="confirmDeleteTask(task)">删除</el-button>
          </div>
        </article>
        <el-empty v-if="issuedTasks.length === 0" description="暂无已下发任务" />
      </section>
    </div>
  </section>
</template>
