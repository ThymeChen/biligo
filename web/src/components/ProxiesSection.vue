<script setup lang="ts">
import { Check, Close, Delete, Document, Edit, Refresh, View } from '@element-plus/icons-vue'
import { proxyGroupTypeLabel, proxyProtocolLabel, proxyTestTagType } from '../app/format'
import { useAppStore } from '../app/useAppStore'

const {
  loading,
  proxyGroups,
  proxyNodes,
  proxyGroupForm,
  proxyNodeForm,
  editingProxyGroupId,
  editingProxyNodeId,
  selectedProxyGroupId,
  proxyNodeFormLocked,
  canSaveProxyGroup,
  canSaveProxyNode,
  resetProxyGroupForm,
  resetProxyNodeForm,
  saveProxyGroup,
  saveProxyNode,
  handleProxyGroupSelection,
  selectProxyGroup,
  editProxyGroup,
  confirmDeleteProxyGroup,
  testProxyGroup,
  pullProxyGroup,
  editProxyNode,
  deleteProxyNode,
} = useAppStore()
</script>

<template>
  <section class="content-grid">
    <div class="stack-column">
      <el-form class="panel form-panel proxy-form-panel" label-position="top" @submit.prevent="saveProxyGroup">
        <div class="panel-heading">
          <h3>{{ editingProxyGroupId ? '编辑代理组' : '新增代理组' }}</h3>
          <el-button :icon="Close" text @click="resetProxyGroupForm">清空</el-button>
        </div>
        <el-form-item required label="代理组名称">
          <el-input v-model="proxyGroupForm.name" placeholder="例如：代理组1" clearable />
        </el-form-item>
        <el-form-item required label="代理组类型">
          <el-radio-group v-model="proxyGroupForm.type">
            <el-radio-button label="static">普通代理组</el-radio-button>
            <el-radio-button label="api">API 代理组</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <template v-if="proxyGroupForm.type === 'api'">
          <el-form-item required label="接口类型">
            <el-select v-model="proxyGroupForm.apiProvider">
              <el-option value="kuaidaili_dps" label="快代理私密代理" />
            </el-select>
          </el-form-item>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12">
              <el-form-item required label="SecretId">
                <el-input v-model="proxyGroupForm.apiConfig.secretId" placeholder="快代理 SecretId" clearable />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item required label="SecretKey">
                <el-input v-model="proxyGroupForm.apiConfig.secretKey" type="password" show-password placeholder="快代理 SecretKey" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12">
              <el-form-item label="鉴权方式">
                <el-select v-model="proxyGroupForm.apiConfig.signType">
                  <el-option value="hmacsha1" label="HMAC-SHA1" />
                  <el-option value="token" label="Token" />
                  <el-option value="simple" label="SecretKey" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item label="提前提取（分钟）">
                <el-input v-model="proxyGroupForm.apiConfig.pullBeforeMinutes" placeholder="5" clearable />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12">
              <el-form-item label="提取数量">
                <el-input v-model="proxyGroupForm.apiConfig.num" placeholder="5" clearable />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item label="提取次数">
                <el-input v-model="proxyGroupForm.apiConfig.pullTimes" placeholder="1" clearable />
              </el-form-item>
            </el-col>
          </el-row>
          <el-alert
            type="info"
            show-icon
            :closable="false"
            class="form-tip-alert"
            title="提取次数只影响任务执行时的代理准备；手动拉取仍只提取 1 次。"
          />
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12">
              <el-form-item label="代理协议">
                <el-select v-model="proxyGroupForm.apiConfig.proxyProtocol">
                  <el-option value="http" label="HTTP" />
                  <el-option value="https" label="HTTPS" />
                  <el-option value="socks5" label="SOCKS5" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item label="地区参数">
                <el-input v-model="proxyGroupForm.apiConfig.area" placeholder="可选，例如：北京,上海" clearable />
              </el-form-item>
            </el-col>
          </el-row>
        </template>
        <el-button native-type="submit" type="primary" :icon="Document" :disabled="!canSaveProxyGroup" :loading="loading">
          保存代理组
        </el-button>
      </el-form>

      <el-form class="panel form-panel proxy-form-panel" label-position="top" @submit.prevent="saveProxyNode">
        <div class="panel-heading">
          <h3>{{ editingProxyNodeId ? '编辑代理节点' : '新增代理节点' }}</h3>
          <el-button :icon="Close" text @click="resetProxyNodeForm">清空</el-button>
        </div>
        <el-form-item required label="所属代理组">
          <el-select v-model="selectedProxyGroupId" placeholder="选择代理组" @change="handleProxyGroupSelection">
            <el-option v-for="group in proxyGroups" :key="group.id" :value="group.id" :label="group.name" />
          </el-select>
        </el-form-item>
        <el-alert v-if="proxyNodeFormLocked" type="info" :closable="false" title="API 代理组请使用拉取生成节点。" class="form-tip-alert" />
        <template v-else>
          <el-form-item label="节点名称">
            <el-input v-model="proxyNodeForm.name" placeholder="默认使用 host:port" clearable />
          </el-form-item>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="8">
              <el-form-item required label="协议">
                <el-select v-model="proxyNodeForm.protocol">
                  <el-option value="http" label="HTTP" />
                  <el-option value="https" label="HTTPS" />
                  <el-option value="socks5" label="SOCKS5" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="10">
              <el-form-item required label="地址">
                <el-input v-model="proxyNodeForm.host" placeholder="127.0.0.1" clearable />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="6">
              <el-form-item required label="端口">
                <el-input-number v-model="proxyNodeForm.port" :min="1" :max="65535" controls-position="right" class="full-input" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12">
              <el-form-item label="账号">
                <el-input v-model="proxyNodeForm.username" clearable />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item label="密码">
                <el-input v-model="proxyNodeForm.password" type="password" show-password />
              </el-form-item>
            </el-col>
          </el-row>
          <el-button native-type="submit" type="primary" :icon="Document" :disabled="!canSaveProxyNode" :loading="loading">
            保存代理节点
          </el-button>
        </template>
      </el-form>
    </div>

    <section class="panel list-panel">
      <div class="panel-heading">
        <h3>代理组</h3>
        <span class="muted">{{ proxyGroups.length }} 个代理组</span>
      </div>
      <article v-for="group in proxyGroups" :key="group.id" class="item-card">
        <div>
          <h4>{{ group.name }}</h4>
          <p>{{ proxyGroupTypeLabel(group.type) }} · {{ group.availableNodeCount }}/{{ group.nodeCount }} 可用</p>
          <small v-if="group.lastPullStatus">最近拉取：{{ group.lastPullMessage || '-' }} {{ group.lastPulledAt ? `· ${group.lastPulledAt}` : '' }}</small>
          <small v-if="group.lastTestStatus">最近检测：{{ group.lastTestMessage || '-' }} {{ group.lastTestedAt ? `· ${group.lastTestedAt}` : '' }}</small>
        </div>
        <div class="actions">
          <el-tag :type="group.inUse ? 'warning' : 'info'">{{ group.inUse ? '运行中占用' : '可编辑' }}</el-tag>
          <el-tag :type="proxyTestTagType(group.lastTestStatus)">{{ group.lastTestStatus || '未检测' }}</el-tag>
          <el-button :icon="View" @click="selectProxyGroup(group)">节点</el-button>
          <el-button :icon="Check" :disabled="group.inUse || group.nodeCount === 0" @click="testProxyGroup(group)">检测</el-button>
          <el-button v-if="group.type === 'api'" :icon="Refresh" :disabled="group.inUse" @click="pullProxyGroup(group)">拉取</el-button>
          <el-button :icon="Edit" :disabled="group.inUse" @click="editProxyGroup(group)">编辑</el-button>
          <el-button type="danger" plain :icon="Delete" :disabled="group.inUse" @click="confirmDeleteProxyGroup(group)">删除</el-button>
        </div>
      </article>
      <el-empty v-if="proxyGroups.length === 0" description="暂无代理组" />

      <div class="panel-heading node-heading">
        <h3>代理节点</h3>
        <span class="muted">{{ selectedProxyGroupId ? `${proxyNodes.length} 个节点` : '未选择代理组' }}</span>
      </div>
      <article v-for="node in proxyNodes" :key="node.id" class="item-card">
        <div>
          <h4>{{ node.name || `${node.host}:${node.port}` }}</h4>
          <p>{{ proxyProtocolLabel(node.protocol) }} · {{ node.host }}:{{ node.port }} · {{ node.source === 'api' ? 'API 拉取' : '手动添加' }}</p>
          <small v-if="node.username">账号：{{ node.username }}</small>
          <small v-if="node.lastTestStatus">最近检测：{{ node.lastTestMessage || '-' }} {{ node.lastTestedAt ? `· ${node.lastTestedAt}` : '' }}</small>
          <small v-if="node.lastTestLatencyMillis || node.lastTestIpLocation">
            延时：{{ node.lastTestLatencyMillis ? `${node.lastTestLatencyMillis}ms` : '-' }} · IP归属地：{{ node.lastTestIpLocation || '-' }}
          </small>
        </div>
        <div class="actions">
          <el-tag :type="proxyTestTagType(node.lastTestStatus)">{{ node.lastTestStatus || '未检测' }}</el-tag>
          <el-button :icon="Edit" @click="editProxyNode(node)">编辑</el-button>
          <el-button type="danger" plain :icon="Delete" @click="deleteProxyNode(node)">删除</el-button>
        </div>
      </article>
      <el-empty v-if="selectedProxyGroupId && proxyNodes.length === 0" description="该代理组暂无节点" />
    </section>
  </section>
</template>
