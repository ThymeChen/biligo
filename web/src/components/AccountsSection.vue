<script setup lang="ts">
import { Check, CopyDocument, Delete, Document, Download, Edit } from '@element-plus/icons-vue'
import { accountStatusLabel, accountStatusTagType } from '../app/format'
import { useAppStore } from '../app/useAppStore'

const {
  loading,
  session,
  accounts,
  accountForm,
  accountExportDialogVisible,
  accountEditDialogVisible,
  selectedAccountExportIds,
  selectedAccountExportCount,
  openAccountExportDialog,
  selectAllAccountExports,
  clearAccountExportSelection,
  exportSelectedAccounts,
  editAccount,
  saveAccount,
  verifyAccount,
  copyAccountCookie,
  deleteAccount,
} = useAppStore()
</script>

<template>
  <section class="stack-column">
    <section class="panel list-panel">
      <div class="panel-heading">
        <div>
          <h3>账号列表</h3>
          <span class="muted">{{ session?.message }}</span>
        </div>
        <div class="table-actions">
          <el-button :icon="Download" :disabled="accounts.length === 0 || loading" @click="openAccountExportDialog">
            批量导出
          </el-button>
        </div>
      </div>
      <div class="summary-row">
        <el-tag>账号 {{ session?.accountCount ?? 0 }}</el-tag>
        <el-tag type="warning">已配置 {{ session?.configuredAccounts ?? 0 }}</el-tag>
        <el-tag type="success">已验证 {{ session?.verifiedAccounts ?? 0 }}</el-tag>
      </div>
      <article v-for="account in accounts" :key="account.id" class="item-card">
        <div>
          <h4>{{ account.name }}</h4>
          <p>{{ account.cookiePreview || '未保存 Cookie' }}</p>
          <small>{{ account.note || '无备注' }}</small>
        </div>
        <div class="actions">
          <el-tag :type="accountStatusTagType(account)">{{ accountStatusLabel(account) }}</el-tag>
          <el-button :icon="Check" :disabled="!account.hasCookie || loading" @click="verifyAccount(account.id)">验证</el-button>
          <el-button :icon="CopyDocument" :disabled="!account.hasCookie || loading" @click="copyAccountCookie(account.id)">
            复制 Cookie
          </el-button>
          <el-button :icon="Edit" @click="editAccount(account)">编辑</el-button>
          <el-button type="danger" plain :icon="Delete" @click="deleteAccount(account.id)">删除</el-button>
        </div>
      </article>
      <el-empty v-if="accounts.length === 0" description="暂无账号" />
    </section>
  </section>

  <el-dialog
    v-model="accountExportDialogVisible"
    title="批量导出账号"
    width="min(520px, calc(100vw - 32px))"
    class="account-export-dialog"
  >
    <div class="account-export-toolbar">
      <div class="summary-row">
        <el-tag>账号 {{ accounts.length }}</el-tag>
        <el-tag type="info">已选 {{ selectedAccountExportCount }}</el-tag>
      </div>
      <div class="table-actions">
        <el-button text :disabled="accounts.length === 0 || loading" @click="selectAllAccountExports">全选</el-button>
        <el-button text :disabled="selectedAccountExportCount === 0 || loading" @click="clearAccountExportSelection">清空</el-button>
      </div>
    </div>
    <el-checkbox-group v-model="selectedAccountExportIds" class="account-export-list">
      <el-checkbox
        v-for="account in accounts"
        :key="account.id"
        :value="account.id"
        :disabled="loading"
        class="account-export-option"
      >
        <span>{{ account.name }}</span>
        <small>{{ account.cookiePreview || '未保存 Cookie' }}</small>
      </el-checkbox>
    </el-checkbox-group>
    <el-empty v-if="accounts.length === 0" description="暂无账号" />
    <template #footer>
      <div class="dialog-footer-actions">
        <el-button :disabled="loading" @click="accountExportDialogVisible = false">取消</el-button>
        <el-button type="primary" :icon="Download" :loading="loading" :disabled="selectedAccountExportCount === 0" @click="exportSelectedAccounts">
          导出选中
        </el-button>
      </div>
    </template>
  </el-dialog>

  <el-dialog
    v-model="accountEditDialogVisible"
    title="编辑账号"
    width="min(520px, calc(100vw - 32px))"
    class="account-edit-dialog"
  >
    <el-form label-position="top" @submit.prevent="saveAccount">
      <el-form-item required label="账号名称">
        <el-input v-model="accountForm.name" placeholder="例如：主账号" clearable />
      </el-form-item>
      <el-form-item label="Cookie">
        <el-input v-model="accountForm.cookie" type="textarea" :rows="5" placeholder="仅保存在本地 SQLite" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="accountForm.note" placeholder="可填写实名人或用途备注" clearable />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer-actions">
        <el-button :disabled="loading" @click="accountEditDialogVisible = false">取消</el-button>
        <el-button type="primary" :icon="Document" :loading="loading" @click="saveAccount">保存账号</el-button>
      </div>
    </template>
  </el-dialog>
</template>
