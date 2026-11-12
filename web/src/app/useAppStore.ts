import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import {
  Bell,
  Connection,
  Document,
  Monitor,
  Setting,
  User,
} from '@element-plus/icons-vue'
import type {
  Account,
  AccountInput,
  BWSInfoInput,
  BWSItem,
  BWSMyReservation,
  BWSReservationInfo,
  BWSTask,
  BWSTaskInput,
  BWSTaskLog,
  EventSnapshot,
  Health,
  Notification,
  NotificationInput,
  ProxyGroup,
  ProxyGroupInput,
  ProxyNode,
  ProxyNodeInput,
  SessionSummary,
  Task,
  TaskInput,
  TaskLog,
  TicketAccountContext,
  TicketAddress,
  TicketBuyer,
  TicketOption,
  TicketProject,
  TicketProjectHistory,
} from '../api'
import {
  API_BASE,
  api,
  clearUserAuthToken,
  getUserAuthExpiresAt,
  getUserAuthToken,
  setUserAuthToken,
  setUnauthorizedHandler,
} from '../api'
import { QR_AUTO_POLL_MS, defaultSuperModeBase } from './constants'
import type { ManagedLogRow, ManagedTaskKind, ManagedTaskRow, QRLoginStatus, SectionKey, TicketProjectHistorySuggestion } from './types'
import {
  bwsManagedSummary,
  cleanConfig,
  compareDateText,
  defaultEndAtFromSaleStart,
  formatDuration,
  managedTaskKey,
  managedTaskUpdatedAt,
  normalizeSaleStartInput,
  normalizeSuperModeBase,
  parseTaskTime,
  taskTicketSummary,
} from './format'

function createStore() {
  const activeSection = ref<SectionKey>('accounts')
  const loading = ref(false)
  const error = ref('')
  const notice = ref('')
  const health = ref<Health | null>(null)
  const session = ref<SessionSummary | null>(null)
  const mobileNavOpen = ref(false)
  const userAuth = reactive({
    checking: true,
    authenticated: false,
    username: '',
    expiresAt: getUserAuthExpiresAt(),
  })

  const accounts = ref<Account[]>([])
  const notifications = ref<Notification[]>([])
  const proxyGroups = ref<ProxyGroup[]>([])
  const proxyNodes = ref<ProxyNode[]>([])
  const ticketProjectHistories = ref<TicketProjectHistory[]>([])
  const tasks = ref<Task[]>([])
  const logs = ref<TaskLog[]>([])
  const bwsTasks = ref<BWSTask[]>([])
  const bwsLogs = ref<BWSTaskLog[]>([])

  const editingAccountId = ref<number | null>(null)
  const editingNotificationId = ref<number | null>(null)
  const editingProxyGroupId = ref<number | null>(null)
  const editingProxyNodeId = ref<number | null>(null)
  const editingTaskId = ref<number | null>(null)
  const editingBWSTaskId = ref<number | null>(null)

  const accountForm = reactive<AccountInput>({
    name: '',
    cookie: '',
    note: '',
  })
  const accountExportDialogVisible = ref(false)
  const accountEditDialogVisible = ref(false)
  const selectedAccountExportIds = ref<number[]>([])

  const notificationForm = reactive<NotificationInput>({
    name: '',
    provider: 'pushplus',
    config: {
      token: '',
    },
  })

  const proxyGroupForm = reactive<ProxyGroupInput>({
    name: '',
    type: 'static',
    apiProvider: 'kuaidaili_dps',
    apiConfig: {
      secretId: '',
      secretKey: '',
      signType: 'hmacsha1',
      num: '5',
      pullTimes: '1',
      pullBeforeMinutes: '5',
      proxyProtocol: 'http',
    },
  })

  const proxyNodeForm = reactive<ProxyNodeInput>({
    name: '',
    protocol: 'http',
    host: '',
    port: 0,
    username: '',
    password: '',
  })

  const qrLogin = reactive({
    qrcodeKey: '',
    qrImageDataUrl: '',
    message: '',
    status: 'idle' as QRLoginStatus,
    autoPolling: false,
    polling: false,
    lastCheckedAt: '',
  })

  let qrPollTimer: number | undefined

  const taskForm = reactive<TaskInput>({
    name: '',
    accountId: 0,
    proxyGroupId: 0,
    proxyMode: 'round_robin',
    superMode: false,
    superModeBase: defaultSuperModeBase,
    projectId: 0,
    projectName: '',
    screenId: 0,
    skuId: 0,
    sessionName: '',
    ticketLevel: '',
    ticketDisplay: '',
    ticketPrice: 0,
    saleStart: '',
    saleStatus: '',
    linkId: 0,
    isHotProject: false,
    taskMode: 'rush',
    durationMode: 'limited',
    selectedTickets: [],
    rushDurationSeconds: 600,
    orderType: 1,
    payMoney: 0,
    buyerInfo: [],
    buyer: '',
    tel: '',
    deliverInfo: undefined,
    phone: '',
    timeSyncStrategy: 'bilibili',
    quantity: 1,
    startAt: '',
    endAt: '',
    pollIntervalMillis: 1000,
    rushPollIntervalMillis: 1000,
    restockPollIntervalMillis: 1000,
  })

  const bwsQueryForm = reactive<BWSInfoInput>({
    accountId: 0,
    year: '202601',
    reserveDates: '20260710,20260711,20260712',
    reserveType: 'activity',
  })

  const bwsTaskForm = reactive<BWSTaskInput>({
    name: '',
    accountId: 0,
    proxyGroupId: 0,
    proxyMode: 'round_robin',
    reserveType: 'activity',
    year: '202601',
    reserveDate: '',
    ticketNo: '',
    reserveId: 0,
    title: '',
    reserveBeginTime: 0,
    mode: 'scheduled',
    timeSyncStrategy: 'bilibili',
    pollIntervalMillis: 1000,
  })

  const bwsInfo = ref<BWSReservationInfo | null>(null)
  const bwsItems = ref<BWSItem[]>([])
  const bwsMyReservations = ref<BWSMyReservation[]>([])
  const selectedBWSItemId = ref<number | null>(null)
  const selectedBWSTaskId = ref<number | null>(null)
  const bwsHideEnded = ref(true)
  const bwsHideReserved = ref(true)

  const selectedTaskId = ref<number | null>(null)
  const selectedProxyGroupId = ref<number | null>(null)
  const mobileLogMode = ref<'empty' | 'task' | 'all'>('empty')
  const selectedLogKind = ref<ManagedTaskKind | null>(null)
  const ticketProjectInput = ref('')
  const fetchedTicketProject = ref<TicketProject | null>(null)
  const ticketOptions = ref<TicketOption[]>([])
  const ticketBuyers = ref<TicketBuyer[]>([])
  const ticketAddresses = ref<TicketAddress[]>([])
  const selectedTicketValue = ref('')
  const selectedTicketValues = ref<string[]>([])
  const selectedBuyerIndexes = ref<number[]>([])
  const selectedAddressId = ref<number>(0)
  const nowMs = ref(Date.now())
  const sseStatus = ref('未连接')
  let eventSource: EventSource | null = null
  let clockTimer: number | undefined
  let userAuthExpiryTimer: number | undefined

  const managedTasks = computed<ManagedTaskRow[]>(() => {
    const rows: ManagedTaskRow[] = [
      ...tasks.value.map((task) => ({ key: managedTaskKey('ticket', task.id), kind: 'ticket' as const, id: task.id, task })),
      ...bwsTasks.value.map((task) => ({ key: managedTaskKey('bws', task.id), kind: 'bws' as const, id: task.id, bwsTask: task })),
    ]
    return rows.sort((a, b) => compareDateText(managedTaskUpdatedAt(b), managedTaskUpdatedAt(a)))
  })
  const selectedManagedTask = computed(() => {
    if (selectedLogKind.value === 'ticket' && selectedTaskId.value) {
      return managedTasks.value.find((task) => task.kind === 'ticket' && task.id === selectedTaskId.value) ?? null
    }
    if (selectedLogKind.value === 'bws' && selectedBWSTaskId.value) {
      return managedTasks.value.find((task) => task.kind === 'bws' && task.id === selectedBWSTaskId.value) ?? null
    }
    return null
  })
  const selectedLogSubtitle = computed(() => {
    const row = selectedManagedTask.value
    if (!row) {
      return ''
    }
    return row.kind === 'ticket' ? taskTicketSummary(row.task) : bwsManagedSummary(row.bwsTask)
  })
  const managedLogs = computed<ManagedLogRow[]>(() => {
    const ticketRows = selectedLogKind.value === 'bws'
      ? []
      : logs.value.map((log) => ({
          key: `ticket-${log.id}`,
          kind: 'ticket' as const,
          id: log.id,
          taskId: log.taskId,
          taskName: tasks.value.find((task) => task.id === log.taskId)?.name ?? '',
          level: log.level,
          message: log.message.replace(/^\[.*\/.*]\s/, ''),
          createdAt: log.createdAt,
        }))
    const bwsRows = selectedLogKind.value === 'ticket'
      ? []
      : bwsLogs.value.map((log) => ({
          key: `bws-${log.id}`,
          kind: 'bws' as const,
          id: log.id,
          taskId: log.taskId,
          taskName: bwsTasks.value.find((task) => task.id === log.taskId)?.name ?? '',
          level: log.level,
          message: log.message.replace(/^\[.*\/.*]\s/, ''),
          createdAt: log.createdAt,
        }))
    return [...ticketRows, ...bwsRows].sort((a, b) => compareDateText(b.createdAt, a.createdAt)).slice(0, 200)
  })
  const selectedAccountExportCount = computed(() => selectedAccountExportIds.value.length)
  const pendingTasks = computed(() => tasks.value.filter((task) => task.status === 'draft' || task.status === 'paused'))
  const issuedTasks = computed(() => tasks.value.filter((task) => task.status !== 'draft' && task.status !== 'paused'))
  const selectedTicketOption = computed(() =>
    ticketOptions.value.find((ticket) => ticket.value === selectedTicketValue.value),
  )
  const selectedTicketOptions = computed(() =>
    ticketOptions.value.filter((ticket) => selectedTicketValues.value.includes(ticket.value)),
  )
  const hasFetchedTicketInfo = computed(() => Boolean(fetchedTicketProject.value?.projectId && ticketOptions.value.length > 0))
  const isRestockTaskForm = computed(() => taskForm.taskMode === 'restock')
  const isHybridTaskForm = computed(() => taskForm.taskMode === 'rush_restock')
  const hasRushTaskSection = computed(() => taskForm.taskMode === 'rush' || taskForm.taskMode === 'rush_restock')
  const hasRestockTaskSection = computed(() => taskForm.taskMode === 'restock' || taskForm.taskMode === 'rush_restock')
  const isRestockUnlimitedTaskForm = computed(() => hasRestockTaskSection.value && taskForm.durationMode === 'unlimited')
  const selectableProxyGroups = computed(() => proxyGroups.value.filter((group) => (group.nodeCount > 0 || group.type === 'api') && !group.inUse))
  const canSaveTask = computed(
    () =>
      taskForm.name.trim() !== '' &&
      taskForm.accountId > 0 &&
      (!hasRushTaskSection.value || (taskForm.ticketDisplay.trim() !== '' && taskForm.skuId > 0 && taskForm.saleStart.trim() !== '')) &&
      (!hasRestockTaskSection.value || taskForm.selectedTickets.length > 0) &&
      (!isHybridTaskForm.value || taskForm.rushDurationSeconds > 0) &&
      (!hasRushTaskSection.value || taskForm.proxyMode !== 'concurrent' || Number(taskForm.proxyGroupId || 0) > 0) &&
      (!hasRestockTaskSection.value || taskForm.durationMode !== 'limited' || taskForm.endAt.trim() !== '') &&
      taskForm.buyerInfo.length > 0 &&
      Boolean(taskForm.deliverInfo?.id) &&
      taskForm.buyer.trim() !== '' &&
      taskForm.tel.trim() !== '' &&
      ((!isHybridTaskForm.value && taskForm.pollIntervalMillis > 0) ||
        (isHybridTaskForm.value && taskForm.rushPollIntervalMillis > 0 && taskForm.restockPollIntervalMillis > 0)),
  )
  const canSaveNotification = computed(
    () => notificationForm.name.trim() !== '' &&
      ['pushplus', 'bark'].includes(notificationForm.provider) &&
      (notificationForm.config.token ?? '').trim() !== '',
  )
  const canSaveProxyGroup = computed(
    () =>
      proxyGroupForm.name.trim() !== '' &&
      (proxyGroupForm.type !== 'api' ||
        (proxyGroupForm.apiProvider === 'kuaidaili_dps' &&
          (proxyGroupForm.apiConfig.secretId ?? '').trim() !== '' &&
          (proxyGroupForm.apiConfig.secretKey ?? '').trim() !== '')),
  )
  const selectedProxyGroup = computed(() => proxyGroups.value.find((group) => group.id === selectedProxyGroupId.value) ?? null)
  const selectedProxyGroupIsAPI = computed(() => selectedProxyGroup.value?.type === 'api')
  const proxyNodeFormLocked = computed(() => selectedProxyGroupIsAPI.value && editingProxyNodeId.value === null)
  const canSaveProxyNode = computed(
    () => selectedProxyGroupId.value !== null &&
      !proxyNodeFormLocked.value &&
      proxyNodeForm.host.trim() !== '' &&
      proxyNodeForm.port > 0 &&
      proxyNodeForm.port <= 65535,
  )
  const filteredBWSItems = computed(() =>
    bwsItems.value.filter((item) => {
      if (bwsHideEnded.value && item.state === 3) {
        return false
      }
      if (bwsHideReserved.value && item.reserved) {
        return false
      }
      return true
    }),
  )
  const canSaveBWSTask = computed(
    () =>
      bwsTaskForm.name.trim() !== '' &&
      bwsTaskForm.accountId > 0 &&
      bwsTaskForm.year.trim() !== '' &&
      bwsTaskForm.ticketNo.trim() !== '' &&
      bwsTaskForm.reserveId > 0 &&
      bwsTaskForm.title.trim() !== '' &&
      (bwsTaskForm.mode === 'immediate' || bwsTaskForm.reserveBeginTime > 0) &&
      bwsTaskForm.pollIntervalMillis > 0 &&
      (bwsTaskForm.proxyMode !== 'concurrent' || Number(bwsTaskForm.proxyGroupId || 0) > 0),
  )

  async function run(action: () => Promise<void>, success?: string) {
    loading.value = true
    error.value = ''
    notice.value = ''
    try {
      await action()
      if (success) {
        notice.value = success
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    } finally {
      loading.value = false
    }
  }

  async function loadAll() {
    await run(async () => {
      health.value = await api.health()

      const [sessionData, accountData, notificationData, proxyGroupData, historyData, taskData, logData, bwsTaskData, bwsLogData] = await Promise.all([
        api.session(),
        api.listAccounts(),
        api.listNotifications(),
        api.listProxyGroups(),
        api.listTicketProjectHistory(),
        api.listTasks(),
        api.listLogs(),
        api.listBWSTasks(),
        api.listBWSLogs(),
      ])
      session.value = sessionData
      accounts.value = accountData ?? []
      syncSelectedAccountExports()
      notifications.value = notificationData ?? []
      proxyGroups.value = proxyGroupData ?? []
      ticketProjectHistories.value = historyData ?? []
      tasks.value = taskData ?? []
      logs.value = logData ?? []
      bwsTasks.value = bwsTaskData ?? []
      bwsLogs.value = bwsLogData ?? []
      if (bwsTasks.value.length > 0 && selectedBWSTaskId.value === null) {
        selectedBWSTaskId.value = bwsTasks.value[0].id
      }
    })
  }

  async function initializeUserAuth() {
    setUnauthorizedHandler(handleUserUnauthorized)
    userAuth.checking = true
    try {
      if (!getUserAuthToken()) {
        userAuth.authenticated = false
        void startQRLogin()
        return
      }
      const session = await api.userSession()
      userAuth.expiresAt = session.expiresAt
      userAuth.username = session.user?.username ?? ''
      userAuth.authenticated = true
      scheduleUserAuthExpiry(session.expiresAt)
      await bootConsole()
    } catch {
      clearUserSession()
    } finally {
      userAuth.checking = false
    }
  }

  function completeUserLogin(token: string, expiresAt: string, username: string) {
    setUserAuthToken(token, expiresAt)
    userAuth.username = username
    userAuth.expiresAt = expiresAt
    userAuth.authenticated = true
    scheduleUserAuthExpiry(expiresAt)
    activeSection.value = 'accounts'
  }

  async function bootConsole() {
    await loadAll()
    resetTaskForm()
    resetBWSTaskForm()
    connectEvents()
    startClock()
  }

  async function logoutUser() {
    try {
      if (getUserAuthToken()) {
        await api.userLogout()
      }
    } catch {
      // Local logout should still work if the server token already expired.
    } finally {
      clearUserSession()
      notice.value = '已退出登录'
    }
  }

  function handleUserUnauthorized() {
    clearUserSession()
    error.value = '登录已失效，请重新登录。'
  }

  function clearUserSession() {
    clearUserAuthToken()
    userAuth.authenticated = false
    userAuth.username = ''
    userAuth.expiresAt = ''
    closeEvents()
    stopClock()
    clearUserAuthExpiry()
    resetQRLogin()
    accounts.value = []
    proxyGroups.value = []
    proxyNodes.value = []
    ticketProjectHistories.value = []
    tasks.value = []
    logs.value = []
    bwsTasks.value = []
    bwsLogs.value = []
    bwsInfo.value = null
    bwsItems.value = []
    bwsMyReservations.value = []
    selectedBWSItemId.value = null
    selectedBWSTaskId.value = null
    selectedTaskId.value = null
    selectedLogKind.value = null
    mobileLogMode.value = 'empty'
    session.value = null
    selectedAccountExportIds.value = []
  }

  function scheduleUserAuthExpiry(expiresAt: string) {
    clearUserAuthExpiry()
    const expiresAtMs = Date.parse(expiresAt)
    if (Number.isNaN(expiresAtMs)) {
      return
    }
    const delay = expiresAtMs - Date.now()
    if (delay <= 0) {
      handleUserUnauthorized()
      return
    }
    userAuthExpiryTimer = window.setTimeout(handleUserUnauthorized, delay)
  }

  function clearUserAuthExpiry() {
    if (userAuthExpiryTimer === undefined) {
      return
    }
    window.clearTimeout(userAuthExpiryTimer)
    userAuthExpiryTimer = undefined
  }

  function resetAccountForm() {
    editingAccountId.value = null
    accountForm.name = ''
    accountForm.cookie = ''
    accountForm.note = ''
  }

  function resetQRLogin() {
    stopQRAutoPoll()
    qrLogin.qrcodeKey = ''
    qrLogin.qrImageDataUrl = ''
    qrLogin.message = ''
    qrLogin.status = 'idle'
    qrLogin.polling = false
    qrLogin.lastCheckedAt = ''
  }

  async function refreshAccountsAndSession() {
    const [accountData, sessionData] = await Promise.all([api.listAccounts(), api.session()])
    accounts.value = accountData ?? []
    session.value = sessionData
    syncSelectedAccountExports()
  }

  function setActiveSection(section: SectionKey) {
    activeSection.value = section
    mobileNavOpen.value = false
  }

  function sectionIcon(section: SectionKey) {
    const map = {
      accounts: User,
      notifications: Bell,
      proxies: Connection,
      taskConfig: Setting,
      bws: Document,
      taskStatus: Monitor,
    }
    return map[section]
  }

  function syncSelectedAccountExports() {
    const accountIds = new Set(accounts.value.map((account) => account.id))
    selectedAccountExportIds.value = selectedAccountExportIds.value.filter((id) => accountIds.has(id))
  }

  function selectAllAccountExports() {
    selectedAccountExportIds.value = accounts.value.map((account) => account.id)
  }

  function clearAccountExportSelection() {
    selectedAccountExportIds.value = []
  }

  function openAccountExportDialog() {
    syncSelectedAccountExports()
    accountExportDialogVisible.value = true
  }

  async function editAccount(account: Account) {
    editingAccountId.value = account.id
    accountForm.name = account.name
    accountForm.cookie = ''
    accountForm.note = account.note
    accountEditDialogVisible.value = true
    if (!account.hasCookie) {
      return
    }
    const accountId = account.id
    await run(async () => {
      const result = await api.getAccountCookie(accountId)
      if (editingAccountId.value === accountId) {
        accountForm.cookie = result.cookie
      }
    }, '账号 Cookie 已载入')
  }

  async function startQRLogin() {
    await run(async () => {
      stopQRAutoPoll()
      const result = await api.startUserQRLogin()
      qrLogin.qrcodeKey = result.qrcodeKey
      qrLogin.qrImageDataUrl = result.qrImageDataUrl
      qrLogin.status = 'generated'
      qrLogin.message = '二维码已生成，正在等待扫码'
      qrLogin.lastCheckedAt = ''
      startQRAutoPoll()
      void pollQRLoginOnce(true)
    })
  }

  async function pollQRLogin() {
    await run(() => pollQRLoginOnce(false))
  }

  function startQRAutoPoll() {
    stopQRAutoPoll()
    qrLogin.autoPolling = true
    qrPollTimer = window.setInterval(() => {
      void pollQRLoginOnce(true)
    }, QR_AUTO_POLL_MS)
  }

  function stopQRAutoPoll() {
    if (qrPollTimer !== undefined) {
      window.clearInterval(qrPollTimer)
      qrPollTimer = undefined
    }
    qrLogin.autoPolling = false
  }

  async function pollQRLoginOnce(auto: boolean) {
    if (!qrLogin.qrcodeKey || qrLogin.polling) {
      return
    }

    const pollKey = qrLogin.qrcodeKey
    qrLogin.polling = true
    try {
      const result = await api.pollUserQRLogin({ qrcodeKey: pollKey })
      if (pollKey !== qrLogin.qrcodeKey) {
        return
      }
      qrLogin.status = result.status as QRLoginStatus
      qrLogin.message = result.message
      qrLogin.lastCheckedAt = new Date().toLocaleTimeString()

      if (result.status === 'confirmed') {
        stopQRAutoPoll()
        qrLogin.polling = false
        if (!result.token) {
          throw new Error('登录响应缺少 token')
        }
        const username = result.user?.username ?? result.username ?? ''
        completeUserLogin(result.token, result.expiresAt ?? '', username)
        resetQRLogin()
        await bootConsole()
        notice.value = `欢迎，${username || '用户'}`
        return
      }

      if (result.status === 'waiting_scan' || result.status === 'waiting_confirm') {
        if (!auto) {
          notice.value = result.message
        }
        return
      }

      stopQRAutoPoll()
      if (!auto) {
        throw new Error(result.message)
      }
    } catch (err) {
      if (pollKey !== qrLogin.qrcodeKey) {
        return
      }
      stopQRAutoPoll()
      qrLogin.status = 'failed'
      qrLogin.message = err instanceof Error ? err.message : String(err)
      if (!auto) {
        throw err
      }
      error.value = qrLogin.message
    } finally {
      if (pollKey === qrLogin.qrcodeKey) {
        qrLogin.polling = false
      }
    }
  }

  async function saveAccount() {
    await run(async () => {
      if (editingAccountId.value) {
        await api.updateAccount(editingAccountId.value, accountForm)
      } else {
        await api.createAccount(accountForm)
      }
      resetAccountForm()
      accountEditDialogVisible.value = false
      await refreshAccountsAndSession()
    }, '账号信息已保存')
  }

  async function verifyAccount(id: number) {
    await run(async () => {
      const result = await api.verifyAccount(id)
      await refreshAccountsAndSession()
      if (!result.loggedIn) {
        throw new Error(result.message)
      }
    }, '登录态验证成功')
  }

  async function copyAccountCookie(id: number) {
    await run(async () => {
      const result = await api.getAccountCookie(id)
      await writeClipboardText(result.cookie)
    }, 'Cookie 已复制')
  }

  async function exportSelectedAccounts() {
    await run(async () => {
      const accountIds = [...selectedAccountExportIds.value]
      if (accountIds.length === 0) {
        throw new Error('请先选择要导出的账号')
      }
      const payload = await api.exportAccounts({ accountIds })
      const text = JSON.stringify(payload, null, 2)
      downloadJSONFile(accountsExportFileName(), text)
      accountExportDialogVisible.value = false
      notice.value = `已导出 ${payload.accounts.length} 个账号`
    })
  }

  function accountsExportFileName() {
    const stamp = new Date().toISOString().replace(/[:.]/g, '-')
    return `biligo-accounts-${stamp}.json`
  }

  function downloadJSONFile(filename: string, text: string) {
    const blob = new Blob([text], { type: 'application/json;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }

  async function writeClipboardText(text: string) {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return
    }

    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.left = '-9999px'
    textarea.style.top = '0'
    document.body.appendChild(textarea)
    textarea.select()
    const copied = document.execCommand('copy')
    document.body.removeChild(textarea)
    if (!copied) {
      throw new Error('浏览器拒绝写入剪贴板')
    }
  }

  async function deleteAccount(id: number) {
    await run(async () => {
      await api.deleteAccount(id)
      await refreshAccountsAndSession()
    }, '账号已删除')
  }

  function resetNotificationForm() {
    editingNotificationId.value = null
    Object.assign(notificationForm, {
      name: '',
      provider: 'pushplus',
      config: {
        token: '',
      },
    })
  }

  function editNotification(notification: Notification) {
    editingNotificationId.value = notification.id
    Object.assign(notificationForm, {
      name: notification.name,
      provider: notification.provider,
      config: {
        token: notification.config?.token ?? '',
      },
    })
    activeSection.value = 'notifications'
  }

  async function saveNotification() {
    await run(async () => {
      const payload: NotificationInput = {
        name: notificationForm.name.trim(),
        provider: notificationForm.provider,
        config: {
          token: (notificationForm.config.token ?? '').trim(),
        },
      }
      if (editingNotificationId.value) {
        await api.updateNotification(editingNotificationId.value, payload)
      } else {
        await api.createNotification(payload)
      }
      notifications.value = await api.listNotifications()
      resetNotificationForm()
    }, editingNotificationId.value ? '通知接口已更新' : '通知接口已添加')
  }

  async function confirmDeleteNotification(notification: Notification) {
    try {
      await ElMessageBox.confirm(
        `确认删除通知接口「${notification.name}」？`,
        '删除通知接口',
        {
          confirmButtonText: '删除',
          cancelButtonText: '取消',
          type: 'warning',
          confirmButtonClass: 'el-button--danger',
        },
      )
    } catch {
      return
    }
    await run(async () => {
      await api.deleteNotification(notification.id)
      notifications.value = await api.listNotifications()
      if (editingNotificationId.value === notification.id) {
        resetNotificationForm()
      }
    }, '通知接口已删除')
  }

  async function testNotification(id: number) {
    await run(async () => {
      const notification = await api.testNotification(id)
      notifications.value = await api.listNotifications()
      if (notification.lastTestStatus === 'error') {
        throw new Error(notification.lastTestMessage || '通知接口检测失败')
      }
    }, '测试推送已发送')
  }

  async function setNotificationEnabled(id: number, enabled: boolean) {
    await run(async () => {
      if (enabled) {
        await api.enableNotification(id)
      } else {
        await api.disableNotification(id)
      }
      notifications.value = await api.listNotifications()
    }, enabled ? '通知接口已启用' : '通知接口已停用')
  }

  function resetProxyGroupForm() {
    editingProxyGroupId.value = null
    Object.assign(proxyGroupForm, {
      name: '',
      type: 'static',
      apiProvider: 'kuaidaili_dps',
      apiConfig: {
        secretId: '',
        secretKey: '',
        signType: 'hmacsha1',
        num: '5',
        pullTimes: '1',
        pullBeforeMinutes: '5',
        proxyProtocol: 'http',
      },
    })
  }

  function resetProxyNodeForm() {
    editingProxyNodeId.value = null
    Object.assign(proxyNodeForm, {
      name: '',
      protocol: 'http',
      host: '',
      port: 0,
      username: '',
      password: '',
    })
  }

  async function refreshProxyGroups() {
    proxyGroups.value = await api.listProxyGroups()
  }

  async function selectProxyGroup(group: ProxyGroup) {
    selectedProxyGroupId.value = group.id
    proxyNodes.value = await api.listProxyNodes(group.id)
    resetProxyNodeForm()
  }

  async function handleProxyGroupSelection(id: number) {
    const group = proxyGroups.value.find((item) => item.id === id)
    if (group) {
      await selectProxyGroup(group)
    }
  }

  function editProxyGroup(group: ProxyGroup) {
    editingProxyGroupId.value = group.id
    Object.assign(proxyGroupForm, {
      name: group.name,
      type: group.type || 'static',
      apiProvider: group.apiProvider || 'kuaidaili_dps',
      apiConfig: {
        secretId: group.apiConfig?.secretId ?? '',
        secretKey: group.apiConfig?.secretKey ?? '',
        secretToken: group.apiConfig?.secretToken ?? '',
        signType: group.apiConfig?.signType ?? 'hmacsha1',
        num: group.apiConfig?.num ?? '5',
        pullTimes: group.apiConfig?.pullTimes ?? '1',
        pullBeforeMinutes: group.apiConfig?.pullBeforeMinutes ?? '5',
        proxyProtocol: group.apiConfig?.proxyProtocol ?? 'http',
        area: group.apiConfig?.area ?? '',
      },
    })
    selectedProxyGroupId.value = group.id
    activeSection.value = 'proxies'
    void selectProxyGroup(group)
  }

  async function saveProxyGroup() {
    await run(async () => {
      const payload: ProxyGroupInput = {
        name: proxyGroupForm.name.trim(),
        type: proxyGroupForm.type,
        apiProvider: proxyGroupForm.type === 'api' ? proxyGroupForm.apiProvider : '',
        apiConfig: proxyGroupForm.type === 'api' ? cleanConfig(proxyGroupForm.apiConfig) : {},
      }
      const group = editingProxyGroupId.value
        ? await api.updateProxyGroup(editingProxyGroupId.value, payload)
        : await api.createProxyGroup(payload)
      await refreshProxyGroups()
      selectedProxyGroupId.value = group.id
      proxyNodes.value = await api.listProxyNodes(group.id)
      resetProxyGroupForm()
    }, editingProxyGroupId.value ? '代理组已更新' : '代理组已创建')
  }

  async function confirmDeleteProxyGroup(group: ProxyGroup) {
    try {
      await ElMessageBox.confirm(`确认删除代理组「${group.name}」？`, '删除代理组', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger',
      })
    } catch {
      return
    }
    await run(async () => {
      await api.deleteProxyGroup(group.id)
      if (selectedProxyGroupId.value === group.id) {
        selectedProxyGroupId.value = null
        proxyNodes.value = []
      }
      await refreshProxyGroups()
    }, '代理组已删除')
  }

  function editProxyNode(node: ProxyNode) {
    editingProxyNodeId.value = node.id
    Object.assign(proxyNodeForm, {
      name: node.name,
      protocol: node.protocol || 'http',
      host: node.host,
      port: node.port,
      username: node.username,
      password: node.password,
    })
  }

  async function saveProxyNode() {
    await run(async () => {
      if (selectedProxyGroupId.value === null) {
        throw new Error('请先选择代理组')
      }
      if (proxyNodeFormLocked.value) {
        throw new Error('API 代理组不支持手动添加代理节点，请使用拉取')
      }
      if (editingProxyNodeId.value) {
        await api.updateProxyNode(editingProxyNodeId.value, proxyNodeForm)
      } else {
        await api.createProxyNode(selectedProxyGroupId.value, proxyNodeForm)
      }
      proxyNodes.value = await api.listProxyNodes(selectedProxyGroupId.value)
      await refreshProxyGroups()
      resetProxyNodeForm()
    }, editingProxyNodeId.value ? '代理节点已更新' : '代理节点已添加')
  }

  async function deleteProxyNode(node: ProxyNode) {
    await run(async () => {
      await api.deleteProxyNode(node.id)
      if (selectedProxyGroupId.value !== null) {
        proxyNodes.value = await api.listProxyNodes(selectedProxyGroupId.value)
      }
      await refreshProxyGroups()
    }, '代理节点已删除')
  }

  async function testProxyGroup(group: ProxyGroup) {
    await run(async () => {
      const tested = await api.testProxyGroup(group.id)
      await refreshProxyGroups()
      if (selectedProxyGroupId.value === group.id) {
        proxyNodes.value = await api.listProxyNodes(group.id)
      }
      if (tested.lastTestStatus === 'error') {
        throw new Error(tested.lastTestMessage || '代理检测失败')
      }
    }, '代理检测完成')
  }

  async function pullProxyGroup(group: ProxyGroup) {
    await run(async () => {
      await api.pullProxyGroup(group.id)
      await refreshProxyGroups()
      selectedProxyGroupId.value = group.id
      proxyNodes.value = await api.listProxyNodes(group.id)
    }, '代理已拉取')
  }

  function resetTaskForm() {
    editingTaskId.value = null
    resetTicketProjectSelection()
    Object.assign(taskForm, {
      name: '',
      accountId: accounts.value[0]?.id ?? 0,
      proxyGroupId: 0,
      proxyMode: 'round_robin',
      superMode: false,
      superModeBase: defaultSuperModeBase,
      projectId: 0,
      projectName: '',
      screenId: 0,
      skuId: 0,
      sessionName: '',
      ticketLevel: '',
      ticketDisplay: '',
      ticketPrice: 0,
      saleStart: '',
      saleStatus: '',
      linkId: 0,
      isHotProject: false,
      taskMode: 'rush',
      durationMode: 'limited',
      selectedTickets: [],
      rushDurationSeconds: 600,
      orderType: 1,
      payMoney: 0,
      buyerInfo: [],
      buyer: '',
      tel: '',
      deliverInfo: undefined,
      phone: '',
      timeSyncStrategy: 'bilibili',
      quantity: 1,
      startAt: '',
      endAt: '',
      pollIntervalMillis: 1000,
      rushPollIntervalMillis: 1000,
      restockPollIntervalMillis: 1000,
    })
  }

  function editTask(task: Task) {
    editingTaskId.value = task.id
    Object.assign(taskForm, {
      name: task.name,
      accountId: task.accountId,
      proxyGroupId: task.proxyGroupId || 0,
      proxyMode: task.proxyMode || 'round_robin',
      superMode: Boolean(task.superMode),
      superModeBase: normalizeSuperModeBase(task.superModeBase),
      projectId: task.projectId,
      projectName: task.projectName,
      screenId: task.screenId,
      skuId: task.skuId,
      sessionName: task.sessionName,
      ticketLevel: task.ticketLevel,
      ticketDisplay: task.ticketDisplay,
      ticketPrice: task.ticketPrice,
      saleStart: normalizeSaleStartInput(task.saleStart),
      saleStatus: task.saleStatus,
      linkId: task.linkId,
      isHotProject: task.isHotProject,
      taskMode: task.taskMode || 'rush',
      durationMode: task.durationMode || 'limited',
      selectedTickets: task.selectedTickets ?? [],
      rushDurationSeconds: task.rushDurationSeconds || 600,
      orderType: task.orderType,
      payMoney: task.payMoney,
      buyerInfo: task.buyerInfo ?? [],
      buyer: task.buyer,
      tel: task.tel,
      deliverInfo: task.deliverInfo,
      phone: task.phone,
      timeSyncStrategy: task.timeSyncStrategy || 'bilibili',
      quantity: task.quantity,
      startAt: task.startAt,
      endAt: task.endAt,
      pollIntervalMillis: task.pollIntervalMillis || 1000,
      rushPollIntervalMillis: task.rushPollIntervalMillis || task.pollIntervalMillis || 1000,
      restockPollIntervalMillis: task.restockPollIntervalMillis || task.pollIntervalMillis || 1000,
    })
    restoreTicketSelectionFromTask(task)
    activeSection.value = 'taskConfig'
  }

  async function saveTask() {
    await run(async () => {
      if (hasRestockTaskSection.value) {
        applySelectedRestockTickets()
      }
      if (hasRushTaskSection.value) {
        taskForm.saleStart = normalizeSaleStartInput(taskForm.saleStart)
      }
      if (hasRushTaskSection.value && (!taskForm.ticketDisplay || taskForm.skuId <= 0)) {
        throw new Error('请先获取票务信息并选择票信息')
      }
      if (hasRestockTaskSection.value && taskForm.selectedTickets.length === 0) {
        throw new Error('回流蹲票模式请至少选择一个票种')
      }
      if (hasRushTaskSection.value && !taskForm.saleStart.trim()) {
        throw new Error('抢票模式需要票档起售时间')
      }
      if (hasRushTaskSection.value && !parseTaskTime(taskForm.saleStart)) {
        throw new Error('抢票模式需要设置合法起售时间')
      }
      if (isHybridTaskForm.value && taskForm.rushDurationSeconds <= 0) {
        throw new Error('抢票+回流捡漏模式需要设置大于 0 的抢票持续秒数')
      }
      if (isHybridTaskForm.value && taskForm.rushPollIntervalMillis <= 0) {
        throw new Error('抢票+回流捡漏模式需要设置大于 0 的抢票阶段重试间隔')
      }
      if (isHybridTaskForm.value && taskForm.restockPollIntervalMillis <= 0) {
        throw new Error('抢票+回流捡漏模式需要设置大于 0 的回流阶段重试间隔')
      }
      if (hasRushTaskSection.value && taskForm.proxyMode === 'concurrent' && Number(taskForm.proxyGroupId || 0) <= 0) {
        throw new Error('并发代理需要选择代理组')
      }
      if (hasRestockTaskSection.value && taskForm.durationMode === 'limited' && !taskForm.endAt.trim()) {
        throw new Error('回流捡漏有限模式需要设置截止时间')
      }
      if (taskForm.buyerInfo.length === 0 || !taskForm.deliverInfo?.id) {
        throw new Error('请先选择购票人和收货地址')
      }
      normalizeTaskModeFields()
      taskForm.quantity = taskForm.buyerInfo.length
      taskForm.payMoney = taskForm.ticketPrice * taskForm.quantity
      if (editingTaskId.value) {
        await api.updateTask(editingTaskId.value, taskForm)
      } else {
        await api.createTask(taskForm)
      }
      resetTaskForm()
      tasks.value = await api.listTasks()
      logs.value = await api.listLogs()
    }, '任务配置已保存')
  }

  function resetTicketProjectSelection() {
    ticketProjectInput.value = ''
    fetchedTicketProject.value = null
    ticketOptions.value = []
    ticketBuyers.value = []
    ticketAddresses.value = []
    selectedTicketValue.value = ''
    selectedTicketValues.value = []
    selectedBuyerIndexes.value = []
    selectedAddressId.value = 0
  }

  function clearSelectedTicketFields() {
    Object.assign(taskForm, {
      projectId: 0,
      projectName: '',
      screenId: 0,
      skuId: 0,
      sessionName: '',
      ticketLevel: '',
      ticketDisplay: '',
      ticketPrice: 0,
      saleStart: '',
      saleStatus: '',
      linkId: 0,
      isHotProject: false,
      selectedTickets: [],
      rushDurationSeconds: 600,
      payMoney: 0,
      endAt: '',
    })
  }

  function normalizeTaskModeFields() {
    if (!['rush', 'restock', 'rush_restock'].includes(taskForm.taskMode)) {
      taskForm.taskMode = 'rush'
    }
    taskForm.proxyGroupId = Number(taskForm.proxyGroupId || 0)
    if (!hasRushTaskSection.value) {
      taskForm.proxyGroupId = 0
      taskForm.proxyMode = 'round_robin'
    }
    if (Number(taskForm.proxyGroupId || 0) <= 0) {
      taskForm.proxyMode = 'round_robin'
    }
    if (!['round_robin', 'concurrent'].includes(taskForm.proxyMode)) {
      taskForm.proxyMode = 'round_robin'
    }
    taskForm.superModeBase = normalizeSuperModeBase(taskForm.superModeBase)
    if (taskForm.pollIntervalMillis <= 0) {
      taskForm.pollIntervalMillis = 1000
    }
    if (!isHybridTaskForm.value) {
      taskForm.rushPollIntervalMillis = taskForm.pollIntervalMillis
      taskForm.restockPollIntervalMillis = taskForm.pollIntervalMillis
    } else if (taskForm.rushPollIntervalMillis <= 0) {
      taskForm.rushPollIntervalMillis = taskForm.pollIntervalMillis
    }
    if (isHybridTaskForm.value && taskForm.restockPollIntervalMillis <= 0) {
      taskForm.restockPollIntervalMillis = taskForm.pollIntervalMillis
    }
    if (!hasRestockTaskSection.value) {
      taskForm.durationMode = 'limited'
      taskForm.selectedTickets = selectedTicketOption.value ? [selectedTicketOption.value] : []
      return
    }
    applySelectedRestockTickets()
    if (taskForm.rushDurationSeconds <= 0) {
      taskForm.rushDurationSeconds = 600
    }
    if (taskForm.durationMode !== 'unlimited') {
      taskForm.durationMode = 'limited'
      return
    }
    taskForm.endAt = ''
  }

  function normalizeBWSTaskProxyFields() {
    if (!['scheduled', 'immediate'].includes(bwsTaskForm.mode)) {
      bwsTaskForm.mode = 'scheduled'
    }
    bwsTaskForm.proxyGroupId = Number(bwsTaskForm.proxyGroupId || 0)
    if (Number(bwsTaskForm.proxyGroupId || 0) <= 0) {
      bwsTaskForm.proxyGroupId = 0
      bwsTaskForm.proxyMode = 'round_robin'
    }
    if (!['round_robin', 'concurrent'].includes(bwsTaskForm.proxyMode)) {
      bwsTaskForm.proxyMode = 'round_robin'
    }
  }

  function handleTaskModeChange() {
    normalizeTaskModeFields()
    if (hasRestockTaskSection.value && selectedTicketValue.value && selectedTicketValues.value.length === 0) {
      selectedTicketValues.value = [selectedTicketValue.value]
      applySelectedRestockTickets()
    }
    if (hasRushTaskSection.value && !selectedTicketValue.value && selectedTicketValues.value.length > 0) {
      selectedTicketValue.value = selectedTicketValues.value[0]
      selectTicketOption()
    }
    if (hasRushTaskSection.value && selectedTicketOption.value && !hasRestockTaskSection.value && !taskForm.endAt) {
      taskForm.endAt = defaultEndAtFromSaleStart(selectedTicketOption.value.saleStart)
    }
  }

  function handleDurationModeChange() {
    normalizeTaskModeFields()
  }

  function clearPurchaseFields() {
    selectedBuyerIndexes.value = []
    selectedAddressId.value = 0
    Object.assign(taskForm, {
      buyerInfo: [],
      buyer: '',
      tel: '',
      deliverInfo: undefined,
      phone: '',
      quantity: 1,
      payMoney: 0,
    })
  }

  function handleTaskAccountChange() {
    ticketBuyers.value = []
    ticketAddresses.value = []
    clearPurchaseFields()
  }

  function ticketProjectHistorySuggestions(queryString: string, cb: (items: TicketProjectHistorySuggestion[]) => void) {
    const keyword = queryString.trim().toLowerCase()
    const suggestions = ticketProjectHistories.value
      .filter((history) => {
        if (!keyword) {
          return true
        }
        return (
          String(history.projectId).includes(keyword) ||
          history.projectName.toLowerCase().includes(keyword) ||
          history.venueName.toLowerCase().includes(keyword)
        )
      })
      .map((history) => ({
        ...history,
        value: String(history.projectId),
      }))
    cb(suggestions)
  }

  function selectTicketProjectHistorySuggestion(history: TicketProjectHistorySuggestion) {
    ticketProjectInput.value = String(history.projectId)
    applyTicketProjectHistory()
  }

  function applyTicketProjectHistory() {
    const raw = ticketProjectInput.value.trim()
    const history = ticketProjectHistories.value.find((item) => String(item.projectId) === raw)
    ticketOptions.value = []
    selectedTicketValue.value = ''
    selectedTicketValues.value = []
    clearSelectedTicketFields()
    clearPurchaseFields()
    if (!history) {
      fetchedTicketProject.value = null
      return
    }
    fetchedTicketProject.value = {
      projectId: history.projectId,
      projectName: history.projectName,
      projectUrl: '',
      username: '',
      phone: '',
      venueName: history.venueName,
      venueAddress: history.venueAddress,
      startAt: history.startAt,
      endAt: history.endAt,
      isHotProject: false,
      hasETicket: false,
      salesDates: [],
      ticketOptions: [],
      buyers: [],
      addresses: [],
    }
  }

  async function fetchTicketProject() {
    await run(async () => {
      const projectInput = ticketProjectInput.value.trim()
      if (!projectInput) {
        throw new Error('请输入抢票项目 ID')
      }
      const previousTicketValue = selectedTicketValue.value
      const previousTicketValues = [...selectedTicketValues.value]
      const project = await api.fetchTicketProject({
        projectInput,
        accountId: 0,
      })
      fetchedTicketProject.value = project
      ticketProjectInput.value = String(project.projectId)
      ticketOptions.value = project.ticketOptions ?? []
      ticketBuyers.value = []
      ticketAddresses.value = []
      clearPurchaseFields()
      selectedTicketValues.value = previousTicketValues.filter((value) => ticketOptions.value.some((ticket) => ticket.value === value))
      if (hasRestockTaskSection.value && selectedTicketValues.value.length > 0) {
        applySelectedRestockTickets(true)
      } else if (previousTicketValue && ticketOptions.value.some((ticket) => ticket.value === previousTicketValue)) {
        selectedTicketValue.value = previousTicketValue
        selectTicketOption(true)
      } else {
        selectedTicketValue.value = ''
        selectedTicketValues.value = []
        clearSelectedTicketFields()
        taskForm.projectId = project.projectId
        taskForm.projectName = project.projectName
        taskForm.name = autoTaskName(project.projectName)
      }
      ticketProjectHistories.value = await api.listTicketProjectHistory()
    }, '票务信息已获取')
  }

  async function fetchTicketAccountContext() {
    await run(async () => {
      if (taskForm.accountId <= 0) {
        throw new Error('请先选择账号')
      }
      if (!hasFetchedTicketInfo.value || !fetchedTicketProject.value) {
        throw new Error('请先获取票务信息')
      }

      const context = await api.fetchTicketAccountContext({
        projectInput: String(fetchedTicketProject.value.projectId),
        accountId: taskForm.accountId,
      })
      applyTicketAccountContext(context)
    }, '账号信息已获取')
  }

  function applyTicketAccountContext(context: TicketAccountContext) {
    ticketBuyers.value = context.buyers ?? []
    ticketAddresses.value = context.addresses ?? []
    clearPurchaseFields()
    taskForm.phone = context.phone ?? ''
  }

  function selectTicketOption(forceNameUpdate = false) {
    const ticket = selectedTicketOption.value
    if (!ticket || !fetchedTicketProject.value) {
      clearSelectedTicketFields()
      return
    }
    const defaultEndAt = defaultEndAtFromSaleStart(ticket.saleStart)
    Object.assign(taskForm, {
      projectId: ticket.projectId,
      projectName: fetchedTicketProject.value.projectName,
      screenId: ticket.screenId,
      skuId: ticket.skuId,
      sessionName: ticket.screenName,
      ticketLevel: ticket.ticketLevel,
      ticketDisplay: ticket.display,
      ticketPrice: ticket.price,
      saleStart: normalizeSaleStartInput(ticket.saleStart),
      saleStatus: ticket.saleStatus,
      linkId: ticket.linkId ?? 0,
      isHotProject: ticket.isHotProject,
      payMoney: ticket.price * taskForm.buyerInfo.length,
      endAt: defaultEndAt,
    })
    if (forceNameUpdate || !taskForm.name.trim()) {
      taskForm.name = autoTaskName(fetchedTicketProject.value.projectName, ticket)
    }
  }

  function selectRestockTickets() {
    applySelectedRestockTickets()
  }

  function applySelectedRestockTickets(forceNameUpdate = false) {
    const tickets = selectedTicketOptions.value
    taskForm.selectedTickets = tickets
    if (tickets.length === 0) {
      if (isRestockTaskForm.value) {
        clearSelectedTicketFields()
      }
      return
    }
    if (!fetchedTicketProject.value) {
      return
    }
    const primary = tickets[0]
    if (!isRestockTaskForm.value) {
      taskForm.projectId = primary.projectId
      taskForm.projectName = fetchedTicketProject.value.projectName
      if (forceNameUpdate || !taskForm.name.trim()) {
        taskForm.name = autoTaskName(fetchedTicketProject.value.projectName)
      }
      return
    }
    Object.assign(taskForm, {
      projectId: primary.projectId,
      projectName: fetchedTicketProject.value.projectName,
      screenId: primary.screenId,
      skuId: primary.skuId,
      sessionName: primary.screenName,
      ticketLevel: primary.ticketLevel,
      ticketDisplay: primary.display,
      ticketPrice: primary.price,
      saleStart: normalizeSaleStartInput(primary.saleStart),
      saleStatus: primary.saleStatus,
      linkId: primary.linkId ?? 0,
      isHotProject: primary.isHotProject,
      payMoney: primary.price * taskForm.buyerInfo.length,
    })
    if (forceNameUpdate || !taskForm.name.trim()) {
      taskForm.name = autoTaskName(fetchedTicketProject.value.projectName, primary)
    }
  }

  function autoTaskName(projectName: string, ticket?: TicketOption) {
    if (taskForm.taskMode === 'restock') {
      return [projectName, '回流蹲票'].filter(Boolean).join(' ')
    }
    if (taskForm.taskMode === 'rush_restock') {
      return [projectName, '抢票+回流'].filter(Boolean).join(' ')
    }
    return [projectName, ticket?.screenName, ticket?.ticketLevel].filter(Boolean).join(' ')
  }

  function handleSaleStartChange() {
    taskForm.saleStart = normalizeSaleStartInput(taskForm.saleStart)
  }

  function updateSelectedBuyers() {
    const indexes = selectedBuyerIndexes.value
      .map((item) => Number(item))
      .filter((item) => Number.isInteger(item) && item >= 0 && item < ticketBuyers.value.length)
    selectedBuyerIndexes.value = Array.from(new Set(indexes))
    taskForm.buyerInfo = selectedBuyerIndexes.value.map((index) => ticketBuyers.value[index])
    taskForm.quantity = Math.max(taskForm.buyerInfo.length, 1)
    taskForm.payMoney = taskForm.ticketPrice * taskForm.buyerInfo.length
  }

  function updateSelectedAddress() {
    const address = ticketAddresses.value.find((item) => item.id === selectedAddressId.value)
    taskForm.deliverInfo = address
    if (address) {
      if (!taskForm.buyer) {
        taskForm.buyer = address.name
      }
      if (!taskForm.tel) {
        taskForm.tel = address.phone
      }
    }
  }

  function restoreTicketSelectionFromTask(task: Task) {
    ticketProjectInput.value = task.projectId > 0 ? String(task.projectId) : ''
    fetchedTicketProject.value = task.projectId > 0
      ? {
          projectId: task.projectId,
          projectName: task.projectName,
          projectUrl: '',
          username: '',
          phone: task.phone,
          venueName: '',
          venueAddress: '',
          startAt: '',
          endAt: '',
          isHotProject: task.isHotProject,
          hasETicket: false,
          salesDates: [],
          ticketOptions: [],
          buyers: task.buyerInfo ?? [],
          addresses: task.deliverInfo ? [task.deliverInfo] : [],
        }
      : null

    const restored = task.ticketDisplay
      ? {
          value: `${task.projectId}:${task.screenId}:${task.skuId}:${task.linkId}`,
          display: task.ticketDisplay,
          projectId: task.projectId,
          screenId: task.screenId,
          skuId: task.skuId,
          screenName: task.sessionName,
          ticketLevel: task.ticketLevel,
          price: task.ticketPrice,
          priceText: task.ticketPrice > 0 ? `￥${(task.ticketPrice / 100).toFixed(2).replace(/\.?0+$/, '')}` : '',
          saleStatus: task.saleStatus,
          saleStart: task.saleStart,
          isHotProject: task.isHotProject,
          linkId: task.linkId,
          clickable: false,
        }
      : null
    const restoredSelectedTickets = task.selectedTickets ?? []
    const restoredOptions = [...restoredSelectedTickets]
    if (restored && !restoredOptions.some((ticket) => ticket.value === restored.value)) {
      restoredOptions.unshift(restored)
    }
    ticketOptions.value = restoredOptions
    selectedTicketValue.value = restored?.value ?? restoredOptions[0]?.value ?? ''
    selectedTicketValues.value = (task.taskMode === 'restock' || task.taskMode === 'rush_restock')
      ? restoredSelectedTickets.map((ticket) => ticket.value)
      : selectedTicketValue.value ? [selectedTicketValue.value] : []
    ticketBuyers.value = task.buyerInfo ?? []
    ticketAddresses.value = task.deliverInfo ? [task.deliverInfo] : []
    selectedBuyerIndexes.value = task.buyerInfo?.map((_, index) => index) ?? []
    selectedAddressId.value = task.deliverInfo?.id ?? 0
  }

  function resetBWSTaskForm() {
    editingBWSTaskId.value = null
    selectedBWSItemId.value = null
    Object.assign(bwsTaskForm, {
      name: '',
      accountId: bwsQueryForm.accountId || accounts.value[0]?.id || 0,
      proxyGroupId: 0,
      proxyMode: 'round_robin',
      reserveType: bwsQueryForm.reserveType || 'activity',
      year: bwsQueryForm.year || '202601',
      reserveDate: '',
      ticketNo: '',
      reserveId: 0,
      title: '',
      reserveBeginTime: 0,
      mode: 'scheduled',
      timeSyncStrategy: 'bilibili',
      pollIntervalMillis: 1000,
    })
  }

  async function fetchBWSInfo() {
    await run(async () => {
      if (bwsQueryForm.accountId <= 0) {
        throw new Error('请先选择账号')
      }
      const info = await api.fetchBWSInfo({
        accountId: bwsQueryForm.accountId,
        year: bwsQueryForm.year.trim() || '202601',
        reserveDates: bwsQueryForm.reserveDates.trim() || '20260710,20260711,20260712',
        reserveType: bwsQueryForm.reserveType,
      })
      bwsInfo.value = info
      bwsItems.value = info.items ?? []
      bwsQueryForm.year = info.year || bwsQueryForm.year
      bwsQueryForm.reserveDates = info.reserveDates || bwsQueryForm.reserveDates
      bwsQueryForm.reserveType = info.reserveType || bwsQueryForm.reserveType
      if (!bwsTaskForm.accountId) {
        bwsTaskForm.accountId = bwsQueryForm.accountId
      }
    }, 'BWS 预约列表已拉取')
  }

  async function fetchBWSMyReservations() {
    await run(async () => {
      if (bwsQueryForm.accountId <= 0) {
        throw new Error('请先选择账号')
      }
      const result = await api.fetchBWSMyReservations(bwsQueryForm.accountId, bwsQueryForm.year.trim() || '202601')
      bwsMyReservations.value = result.items ?? []
    }, '我的 BWS 预约已刷新')
  }

  function selectBWSItem(item: BWSItem) {
    selectedBWSItemId.value = item.reserveId
    Object.assign(bwsTaskForm, {
      name: `BWS ${item.title}`.trim(),
      accountId: bwsQueryForm.accountId || bwsTaskForm.accountId || accounts.value[0]?.id || 0,
      reserveType: bwsQueryForm.reserveType || 'activity',
      year: bwsQueryForm.year || '202601',
      reserveDate: item.date,
      ticketNo: item.ticketNo,
      reserveId: item.reserveId,
      title: item.title,
      reserveBeginTime: item.effectiveReserveBeginTime || item.reserveBeginTime,
    })
  }

  function editBWSTask(task: BWSTask) {
    editingBWSTaskId.value = task.id
    selectedBWSItemId.value = task.reserveId
    Object.assign(bwsTaskForm, {
      name: task.name,
      accountId: task.accountId,
      proxyGroupId: task.proxyGroupId || 0,
      proxyMode: task.proxyMode || 'round_robin',
      reserveType: task.reserveType || 'activity',
      year: task.year || '202601',
      reserveDate: task.reserveDate,
      ticketNo: task.ticketNo,
      reserveId: task.reserveId,
      title: task.title,
      reserveBeginTime: task.reserveBeginTime,
      mode: task.mode || 'scheduled',
      timeSyncStrategy: task.timeSyncStrategy || 'bilibili',
      pollIntervalMillis: task.pollIntervalMillis || 1000,
    })
    activeSection.value = 'bws'
  }

  async function saveBWSTask() {
    await run(async () => {
      normalizeBWSTaskProxyFields()
      if (!canSaveBWSTask.value) {
        throw new Error('请先选择 BWS 预约条目并补全任务配置')
      }
      if (bwsTaskForm.proxyMode === 'concurrent' && Number(bwsTaskForm.proxyGroupId || 0) <= 0) {
        throw new Error('BWS 并发代理需要选择代理组')
      }
      if (editingBWSTaskId.value) {
        await api.updateBWSTask(editingBWSTaskId.value, bwsTaskForm)
      } else {
        await api.createBWSTask(bwsTaskForm)
      }
      bwsTasks.value = await api.listBWSTasks()
      resetBWSTaskForm()
    }, editingBWSTaskId.value ? 'BWS 任务已更新' : 'BWS 任务已创建')
  }

  async function dispatchBWSTask(id: number) {
    await run(async () => {
      await api.dispatchBWSTask(id)
      bwsTasks.value = await api.listBWSTasks()
      if (selectedLogKind.value === 'bws' && selectedBWSTaskId.value) {
        bwsLogs.value = await api.listBWSTaskLogs(selectedBWSTaskId.value)
      } else if (selectedLogKind.value === null) {
        bwsLogs.value = await api.listBWSLogs()
      }
    }, 'BWS 任务已启动')
  }

  async function pauseBWSTask(id: number) {
    await run(async () => {
      await api.pauseBWSTask(id)
      bwsTasks.value = await api.listBWSTasks()
      if (selectedLogKind.value === 'bws' && selectedBWSTaskId.value) {
        bwsLogs.value = await api.listBWSTaskLogs(selectedBWSTaskId.value)
      } else if (selectedLogKind.value === null) {
        bwsLogs.value = await api.listBWSLogs()
      }
    }, 'BWS 任务已停止')
  }

  async function confirmDeleteBWSTask(task: BWSTask) {
    try {
      await ElMessageBox.confirm(`确认删除 BWS 预约任务「${task.name || task.title}」？`, '删除 BWS 任务', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger',
      })
    } catch {
      return
    }
    await run(async () => {
      await api.deleteBWSTask(task.id)
      if (selectedBWSTaskId.value === task.id) {
        selectedBWSTaskId.value = null
        if (selectedLogKind.value === 'bws') {
          selectedLogKind.value = null
          mobileLogMode.value = 'empty'
        }
      }
      bwsTasks.value = await api.listBWSTasks()
      bwsLogs.value = selectedLogKind.value === null ? await api.listBWSLogs() : []
    }, 'BWS 任务已删除')
  }

  async function selectBWSTaskLog(task: BWSTask) {
    selectedBWSTaskId.value = task.id
    selectedTaskId.value = null
    selectedLogKind.value = 'bws'
    activeSection.value = 'taskStatus'
    await run(async () => {
      bwsLogs.value = await api.listBWSTaskLogs(task.id)
      logs.value = []
    })
  }

  function upsertBWSTask(task: BWSTask) {
    const index = bwsTasks.value.findIndex((item) => item.id === task.id)
    if (index >= 0) {
      bwsTasks.value.splice(index, 1, task)
    } else {
      bwsTasks.value.unshift(task)
    }
  }

  function addBWSLog(log: BWSTaskLog) {
    if (bwsLogs.value.some((item) => item.id === log.id)) {
      return
    }
    if (selectedLogKind.value === 'ticket') {
      return
    }
    if (selectedLogKind.value === 'bws' && selectedBWSTaskId.value && log.taskId !== selectedBWSTaskId.value) {
      return
    }
    bwsLogs.value = [log, ...bwsLogs.value].slice(0, 200)
  }

  async function dispatchTask(id: number) {
    await run(async () => {
      await api.dispatchTask(id)
      tasks.value = await api.listTasks()
      logs.value = await api.listLogs(selectedLogKind.value === 'ticket' ? selectedTaskId.value ?? undefined : undefined)
    }, '任务已下发')
  }

  async function startTask(id: number) {
    await run(async () => {
      await api.dispatchTask(id)
      tasks.value = await api.listTasks()
      logs.value = await api.listLogs(selectedLogKind.value === 'ticket' ? selectedTaskId.value ?? undefined : undefined)
    }, '任务已启动')
  }

  async function stopTask(id: number) {
    await run(async () => {
      await api.pauseTask(id)
      tasks.value = await api.listTasks()
      logs.value = await api.listLogs(selectedLogKind.value === 'ticket' ? selectedTaskId.value ?? undefined : undefined)
    }, '任务已停止')
  }

  async function confirmDeleteTask(task: Task) {
    const taskName = task.name.trim() || `#${task.id}`
    try {
      await ElMessageBox.confirm(
        `如果任务正在运行，删除会同时停止后台任务。此操作不可撤销。`,
        `确认删除任务「${taskName}」？`,
        {
          confirmButtonText: '删除',
          cancelButtonText: '取消',
          type: 'warning',
          confirmButtonClass: 'el-button--danger',
        },
      )
    } catch {
      return
    }
    await deleteTask(task.id)
  }

  async function deleteTask(id: number) {
    await run(async () => {
      await api.deleteTask(id)
      if (selectedTaskId.value === id) {
        selectedTaskId.value = null
        if (selectedLogKind.value === 'ticket') {
          selectedLogKind.value = null
          mobileLogMode.value = 'empty'
        }
      }
      tasks.value = await api.listTasks()
      logs.value = await api.listLogs()
    }, '任务已删除')
  }

  async function selectTaskLog(task: Task) {
    selectedTaskId.value = task.id
    selectedBWSTaskId.value = null
    selectedLogKind.value = 'ticket'
    await run(async () => {
      logs.value = await api.listLogs(task.id)
      bwsLogs.value = []
    })
  }

  async function showAllLogs() {
    mobileLogMode.value = 'empty'
    selectedTaskId.value = null
    selectedLogKind.value = null
    await run(async () => {
      const [taskLogs, bwsTaskLogs] = await Promise.all([api.listLogs(), api.listBWSLogs()])
      logs.value = taskLogs ?? []
      bwsLogs.value = bwsTaskLogs ?? []
    })
  }

  async function showAllMobileLogs() {
    mobileLogMode.value = 'all'
    selectedTaskId.value = null
    selectedLogKind.value = null
    await run(async () => {
      const [taskLogs, bwsTaskLogs] = await Promise.all([api.listLogs(), api.listBWSLogs()])
      logs.value = taskLogs ?? []
      bwsLogs.value = bwsTaskLogs ?? []
    })
  }

  function upsertTask(task: Task) {
    const index = tasks.value.findIndex((item) => item.id === task.id)
    if (index >= 0) {
      tasks.value.splice(index, 1, task)
    } else {
      tasks.value.unshift(task)
    }
  }

  function addLog(log: TaskLog) {
    if (logs.value.some((item) => item.id === log.id)) {
      return
    }
    if (selectedLogKind.value === 'bws') {
      return
    }
    if (selectedLogKind.value === 'ticket' && selectedTaskId.value && log.taskId !== selectedTaskId.value) {
      return
    }
    logs.value = [log, ...logs.value].slice(0, 200)
  }

  async function selectManagedTaskLog(row: ManagedTaskRow) {
    mobileLogMode.value = 'task'
    if (row.kind === 'ticket') {
      await selectTaskLog(row.task)
      return
    }
    await selectBWSTaskLog(row.bwsTask)
  }

  async function startManagedTask(row: ManagedTaskRow) {
    if (row.kind === 'ticket') {
      await startTask(row.id)
      return
    }
    await dispatchBWSTask(row.id)
  }

  async function stopManagedTask(row: ManagedTaskRow) {
    if (row.kind === 'ticket') {
      await stopTask(row.id)
      return
    }
    await pauseBWSTask(row.id)
  }

  async function confirmDeleteManagedTask(row: ManagedTaskRow) {
    if (row.kind === 'ticket') {
      await confirmDeleteTask(row.task)
      return
    }
    await confirmDeleteBWSTask(row.bwsTask)
  }

  function connectEvents() {
    eventSource?.close()
    const token = getUserAuthToken()
    if (!token) {
      sseStatus.value = '未登录'
      return
    }
    const source = new EventSource(`${API_BASE}/api/events?token=${encodeURIComponent(token)}`)
    eventSource = source
    sseStatus.value = '连接中'

    source.addEventListener('open', () => {
      sseStatus.value = '已连接'
    })
    source.addEventListener('error', () => {
      sseStatus.value = '重连中'
    })
    source.addEventListener('snapshot', (event) => {
      const snapshot = JSON.parse((event as MessageEvent).data) as EventSnapshot
      tasks.value = snapshot.tasks ?? []
      logs.value = snapshot.logs ?? []
      bwsTasks.value = snapshot.bwsTasks ?? bwsTasks.value
      bwsLogs.value = snapshot.bwsLogs ?? bwsLogs.value
    })
    source.addEventListener('task.updated', (event) => {
      upsertTask(JSON.parse((event as MessageEvent).data) as Task)
    })
    source.addEventListener('task.deleted', (event) => {
      const payload = JSON.parse((event as MessageEvent).data) as { id: number }
      tasks.value = tasks.value.filter((task) => task.id !== payload.id)
      if (selectedTaskId.value === payload.id) {
        selectedTaskId.value = null
        if (selectedLogKind.value === 'ticket') {
          selectedLogKind.value = null
          mobileLogMode.value = 'empty'
        }
      }
    })
    source.addEventListener('log.created', (event) => {
      addLog(JSON.parse((event as MessageEvent).data) as TaskLog)
    })
    source.addEventListener('bws.task.updated', (event) => {
      upsertBWSTask(JSON.parse((event as MessageEvent).data) as BWSTask)
    })
    source.addEventListener('bws.task.deleted', (event) => {
      const payload = JSON.parse((event as MessageEvent).data) as { id: number }
      bwsTasks.value = bwsTasks.value.filter((task) => task.id !== payload.id)
      if (selectedBWSTaskId.value === payload.id) {
        selectedBWSTaskId.value = null
        if (selectedLogKind.value === 'bws') {
          selectedLogKind.value = null
          mobileLogMode.value = 'empty'
        }
      }
    })
    source.addEventListener('bws.log.created', (event) => {
      addBWSLog(JSON.parse((event as MessageEvent).data) as BWSTaskLog)
    })
  }

  function closeEvents() {
    eventSource?.close()
    eventSource = null
    sseStatus.value = '未连接'
  }

  function startClock() {
    if (clockTimer !== undefined) {
      return
    }
    clockTimer = window.setInterval(() => {
      nowMs.value = Date.now()
    }, 1000)
  }

  function stopClock() {
    if (clockTimer === undefined) {
      return
    }
    window.clearInterval(clockTimer)
    clockTimer = undefined
  }

  function calibratedNowMs(task: Task) {
    return nowMs.value + (task.timeOffsetMillis || 0)
  }

  function countdownText(task: Task) {
    if (task.taskMode === 'restock') {
      if (task.durationMode === 'unlimited') {
        return '无限检测'
      }
      const endAt = parseTaskTime(task.endAt)
      if (!endAt) {
        return '未设置截止时间'
      }
      const remaining = endAt.getTime() - nowMs.value
      if (remaining <= 0) {
        return '已到截止时间'
      }
      return formatDuration(remaining)
    }
    if (task.taskMode === 'rush_restock') {
      const saleStart = parseTaskTime(task.saleStart)
      if (!saleStart) {
        return '-'
      }
      const saleRemaining = saleStart.getTime() - calibratedNowMs(task)
      if (saleRemaining > 0) {
        return formatDuration(saleRemaining)
      }
      if (task.durationMode === 'unlimited') {
        return '抢票/回流中'
      }
      const endAt = parseTaskTime(task.endAt)
      if (!endAt) {
        return '未设置截止时间'
      }
      const remaining = endAt.getTime() - nowMs.value
      if (remaining <= 0) {
        return '已到截止时间'
      }
      return `回流剩余${formatDuration(remaining)}`
    }
    const target = parseTaskTime(task.saleStart)
    if (!target) {
      return '-'
    }
    const remaining = target.getTime() - calibratedNowMs(task)
    if (remaining <= 0) {
      return '已到起售时间'
    }
    return formatDuration(remaining)
  }

  function bwsCountdownText(task: BWSTask) {
    if (task.mode === 'immediate') {
      return '立刻抢'
    }
    if (!task.reserveBeginTime) {
      return '-'
    }
    const remaining = task.reserveBeginTime * 1000 - (nowMs.value + (task.timeOffsetMillis || 0))
    if (remaining <= 0) {
      return '已到预约时间'
    }
    return formatDuration(remaining)
  }

  function managedTaskTimeText(row: ManagedTaskRow) {
    return row.kind === 'ticket' ? countdownText(row.task) : bwsCountdownText(row.bwsTask)
  }

  async function copyPaymentUrl(task: Task) {
    await run(async () => {
      await writeClipboardText(task.paymentUrl)
    }, '支付链接已复制')
  }

  const qrStatusLabel = computed(() => {
    const map: Record<QRLoginStatus, string> = {
      idle: '未生成',
      generated: '已生成',
      waiting_scan: '等待扫码',
      waiting_confirm: '等待确认',
      confirmed: '已添加',
      expired: '已过期',
      failed: '失败',
    }
    return map[qrLogin.status]
  })

  const qrStatusMessage = computed(() => qrLogin.message || '尚未生成登录二维码。')
  const qrMessageAlertType = computed(() => {
    if (qrLogin.status === 'confirmed') {
      return 'success'
    }
    if (qrLogin.status === 'expired' || qrLogin.status === 'failed') {
      return 'error'
    }
    if (qrLogin.status === 'waiting_scan' || qrLogin.status === 'waiting_confirm') {
      return 'warning'
    }
    return 'info'
  })

  function qrStatusClass() {
    if (qrLogin.status === 'confirmed') {
      return 'ready'
    }
    if (qrLogin.status === 'expired' || qrLogin.status === 'failed') {
      return 'bad'
    }
    if (qrLogin.status === 'waiting_scan' || qrLogin.status === 'waiting_confirm' || qrLogin.status === 'generated') {
      return 'idle'
    }
    return 'idle'
  }

  function qrStatusTagType() {
    if (qrLogin.status === 'confirmed') {
      return 'success'
    }
    if (qrLogin.status === 'expired' || qrLogin.status === 'failed') {
      return 'danger'
    }
    return 'warning'
  }

  onMounted(async () => {
    await initializeUserAuth()
  })

  onUnmounted(() => {
    stopQRAutoPoll()
    closeEvents()
    stopClock()
  })

  return {
    activeSection,
    loading,
    error,
    notice,
    health,
    session,
    mobileNavOpen,
    userAuth,
    accounts,
    notifications,
    proxyGroups,
    proxyNodes,
    ticketProjectHistories,
    tasks,
    logs,
    bwsTasks,
    bwsLogs,
    editingAccountId,
    editingNotificationId,
    editingProxyGroupId,
    editingProxyNodeId,
    editingTaskId,
    editingBWSTaskId,
    accountForm,
    accountExportDialogVisible,
    accountEditDialogVisible,
    selectedAccountExportIds,
    notificationForm,
    proxyGroupForm,
    proxyNodeForm,
    qrLogin,
    taskForm,
    bwsQueryForm,
    bwsTaskForm,
    bwsInfo,
    bwsItems,
    bwsMyReservations,
    selectedBWSItemId,
    selectedBWSTaskId,
    bwsHideEnded,
    bwsHideReserved,
    selectedTaskId,
    selectedProxyGroupId,
    mobileLogMode,
    selectedLogKind,
    ticketProjectInput,
    fetchedTicketProject,
    ticketOptions,
    ticketBuyers,
    ticketAddresses,
    selectedTicketValue,
    selectedTicketValues,
    selectedBuyerIndexes,
    selectedAddressId,
    nowMs,
    sseStatus,
    managedTasks,
    selectedManagedTask,
    selectedLogSubtitle,
    managedLogs,
    selectedAccountExportCount,
    pendingTasks,
    issuedTasks,
    selectedTicketOption,
    selectedTicketOptions,
    hasFetchedTicketInfo,
    isRestockTaskForm,
    isHybridTaskForm,
    hasRushTaskSection,
    hasRestockTaskSection,
    isRestockUnlimitedTaskForm,
    selectableProxyGroups,
    canSaveTask,
    canSaveNotification,
    canSaveProxyGroup,
    selectedProxyGroup,
    selectedProxyGroupIsAPI,
    proxyNodeFormLocked,
    canSaveProxyNode,
    filteredBWSItems,
    canSaveBWSTask,
    qrStatusLabel,
    qrStatusMessage,
    qrMessageAlertType,
    setActiveSection,
    sectionIcon,
    loadAll,
    logoutUser,
    openAccountExportDialog,
    selectAllAccountExports,
    clearAccountExportSelection,
    editAccount,
    saveAccount,
    verifyAccount,
    copyAccountCookie,
    exportSelectedAccounts,
    deleteAccount,
    startQRLogin,
    pollQRLogin,
    qrStatusClass,
    qrStatusTagType,
    resetNotificationForm,
    editNotification,
    saveNotification,
    confirmDeleteNotification,
    testNotification,
    setNotificationEnabled,
    resetProxyGroupForm,
    resetProxyNodeForm,
    selectProxyGroup,
    handleProxyGroupSelection,
    editProxyGroup,
    saveProxyGroup,
    confirmDeleteProxyGroup,
    editProxyNode,
    saveProxyNode,
    deleteProxyNode,
    testProxyGroup,
    pullProxyGroup,
    resetTaskForm,
    editTask,
    saveTask,
    handleTaskModeChange,
    handleDurationModeChange,
    handleTaskAccountChange,
    ticketProjectHistorySuggestions,
    selectTicketProjectHistorySuggestion,
    applyTicketProjectHistory,
    fetchTicketProject,
    fetchTicketAccountContext,
    selectTicketOption,
    selectRestockTickets,
    handleSaleStartChange,
    updateSelectedBuyers,
    updateSelectedAddress,
    normalizeTaskModeFields,
    normalizeBWSTaskProxyFields,
    resetBWSTaskForm,
    fetchBWSInfo,
    fetchBWSMyReservations,
    selectBWSItem,
    editBWSTask,
    saveBWSTask,
    dispatchBWSTask,
    pauseBWSTask,
    confirmDeleteBWSTask,
    selectBWSTaskLog,
    dispatchTask,
    startTask,
    stopTask,
    confirmDeleteTask,
    deleteTask,
    selectTaskLog,
    showAllLogs,
    showAllMobileLogs,
    selectManagedTaskLog,
    startManagedTask,
    stopManagedTask,
    confirmDeleteManagedTask,
    countdownText,
    bwsCountdownText,
    managedTaskTimeText,
    copyPaymentUrl,
  }
}

export type AppStore = ReturnType<typeof createStore>

let storeSingleton: AppStore | null = null

export function useAppStore(): AppStore {
  if (!storeSingleton) {
    storeSingleton = createStore()
  }
  return storeSingleton
}
