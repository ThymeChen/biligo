<script setup lang="ts">
import { Refresh, Menu as MenuIcon, SwitchButton } from '@element-plus/icons-vue'
import { sections } from './app/constants'
import { useAppStore } from './app/useAppStore'
import AuthPanel from './components/AuthPanel.vue'
import AccountsSection from './components/AccountsSection.vue'
import NotificationsSection from './components/NotificationsSection.vue'
import ProxiesSection from './components/ProxiesSection.vue'
import TaskConfigSection from './components/TaskConfigSection.vue'
import BwsSection from './components/BwsSection.vue'
import TaskStatusSection from './components/TaskStatusSection.vue'

const {
  userAuth,
  loading,
  error,
  notice,
  health,
  activeSection,
  mobileNavOpen,
  setActiveSection,
  sectionIcon,
  loadAll,
  logoutUser,
} = useAppStore()
</script>

<template>
  <AuthPanel v-if="!userAuth.authenticated" />

  <el-container v-else class="app-shell">
    <div v-if="loading" class="top-progress" aria-label="页面加载中"></div>

    <el-aside width="260px" class="sidebar desktop-sidebar">
      <div>
        <p class="eyebrow">Biligo</p>
        <h1>票务控制台</h1>
      </div>
      <nav class="nav-list">
        <el-button
          v-for="section in sections"
          :key="section.key"
          :icon="sectionIcon(section.key)"
          text
          :class="{ active: activeSection === section.key }"
          @click="setActiveSection(section.key)"
        >
          {{ section.label }}
        </el-button>
      </nav>
      <div class="system-strip">
        <span :class="['dot', health?.status === 'ok' ? 'ok' : 'bad']"></span>
        <span>API {{ health?.status ?? '未连接' }}</span>
      </div>
    </el-aside>

    <el-drawer v-model="mobileNavOpen" title="票务控制台" direction="ltr" size="280px" class="mobile-drawer">
      <nav class="nav-list drawer-nav">
        <el-button
          v-for="section in sections"
          :key="section.key"
          :icon="sectionIcon(section.key)"
          text
          :class="{ active: activeSection === section.key }"
          @click="setActiveSection(section.key)"
        >
          {{ section.label }}
        </el-button>
      </nav>
      <div class="system-strip drawer-system">
        <span :class="['dot', health?.status === 'ok' ? 'ok' : 'bad']"></span>
        <span>API {{ health?.status ?? '未连接' }}</span>
      </div>
    </el-drawer>

    <el-main class="workspace">
      <header class="topbar">
        <el-button class="mobile-menu-button" :icon="MenuIcon" circle @click="mobileNavOpen = true" />
        <div class="topbar-title">
          <p class="eyebrow">{{ userAuth.username || '已登录' }}</p>
          <h2>{{ sections.find((section) => section.key === activeSection)?.label }}</h2>
        </div>
        <div class="topbar-actions">
          <el-button :icon="Refresh" :loading="loading" @click="loadAll">刷新</el-button>
          <el-button :icon="SwitchButton" plain @click="logoutUser">退出</el-button>
        </div>
      </header>

      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" class="global-message-alert" />
      <el-alert v-if="notice" :title="notice" type="success" show-icon :closable="false" class="global-message-alert" />

      <AccountsSection v-if="activeSection === 'accounts'" />
      <NotificationsSection v-if="activeSection === 'notifications'" />
      <ProxiesSection v-if="activeSection === 'proxies'" />
      <TaskConfigSection v-if="activeSection === 'taskConfig'" />
      <BwsSection v-if="activeSection === 'bws'" />
      <TaskStatusSection v-if="activeSection === 'taskStatus'" />
    </el-main>
  </el-container>
</template>
