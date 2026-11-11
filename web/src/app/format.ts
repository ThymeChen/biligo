import type {
  Account,
  BWSItem,
  BWSTask,
  ProxyGroup,
  Task,
  TicketAddress,
  TicketBuyer,
} from '../api'
import { defaultSuperModeBase, superModeBaseOptions, chinaTimeZoneOffset, chinaTimeZoneOffsetMillis, explicitTimeZonePattern } from './constants'
import type { ManagedTaskRow } from './types'

export function cleanConfig(config: Record<string, string>) {
  const output: Record<string, string> = {}
  Object.entries(config).forEach(([key, value]) => {
    const normalized = (value ?? '').trim()
    if (normalized) {
      output[key] = normalized
    }
  })
  return output
}

export function normalizeSuperModeBase(base?: string) {
  const normalized = (base ?? '').trim()
  return superModeBaseOptions.some((option) => option.value === normalized) ? normalized : defaultSuperModeBase
}

export function defaultEndAtFromSaleStart(saleStart: string) {
  const parsed = parseTaskTime(saleStart)
  if (!parsed) {
    return ''
  }
  return formatDateTimeInput(new Date(parsed.getTime() + 10 * 60 * 1000))
}

export function normalizeSaleStartInput(saleStart: string) {
  const parsed = parseTaskTime(saleStart)
  if (!parsed) {
    return saleStart.trim()
  }
  return formatDateTimeSecondInput(parsed)
}

export function formatDateTimeInput(date: Date) {
  const pad = (value: number) => String(value).padStart(2, '0')
  const chinaDate = new Date(date.getTime() + chinaTimeZoneOffsetMillis)
  return [
    chinaDate.getUTCFullYear(),
    pad(chinaDate.getUTCMonth() + 1),
    pad(chinaDate.getUTCDate()),
  ].join('-') + `T${pad(chinaDate.getUTCHours())}:${pad(chinaDate.getUTCMinutes())}`
}

export function formatDateTimeSecondInput(date: Date) {
  const pad = (value: number) => String(value).padStart(2, '0')
  const chinaDate = new Date(date.getTime() + chinaTimeZoneOffsetMillis)
  return [
    chinaDate.getUTCFullYear(),
    pad(chinaDate.getUTCMonth() + 1),
    pad(chinaDate.getUTCDate()),
  ].join('-') + ` ${pad(chinaDate.getUTCHours())}:${pad(chinaDate.getUTCMinutes())}:${pad(chinaDate.getUTCSeconds())}`
}

export function buyerLabel(buyer: TicketBuyer) {
  return `${buyer.name || '未命名购票人'}${buyer.personalId ? ` - ${buyer.personalId}` : ''}`
}

export function taskBuyerSummary(task: Task) {
  const buyers = task.buyerInfo ?? []
  if (buyers.length === 0) {
    return '实名购票人：-'
  }
  return `实名购票人：${buyers.map((buyer) => buyerLabel(buyer)).join('、')}`
}

export function taskTicketSummary(task: Task) {
  const selectedTickets = task.selectedTickets ?? []
  if ((task.taskMode === 'restock' || task.taskMode === 'rush_restock') && selectedTickets.length > 0) {
    const names = selectedTickets.map((ticket) => ticket.display || `${ticket.screenName || '-'} / ${ticket.ticketLevel || '-'}`)
    const preview = names.slice(0, 2).join('、')
    const suffix = names.length > 2 ? ` 等 ${names.length} 个` : `${names.length} 个`
    if (task.taskMode === 'rush_restock') {
      const rush = task.ticketDisplay || `${task.sessionName || '-'} / ${task.ticketLevel || '-'}`
      return `抢票：${rush}；${task.rushDurationSeconds || 600} 秒后回流：${preview}（${suffix}）`
    }
    const current = task.ticketDisplay ? `；当前：${task.ticketDisplay}` : ''
    return `回流票种：${preview}（${suffix}）${current}`
  }
  const display = task.ticketDisplay || `${task.sessionName || '-'} / ${task.ticketLevel || '-'}`
  return `${display} / ${task.quantity} 张`
}

export function addressLabel(address: TicketAddress) {
  return `${address.name || '未命名地址'} - ${address.phone || '-'} - ${address.fullAddress || address.addr || '-'}`
}

export function formatMoney(value: number) {
  if (!value) {
    return '￥0'
  }
  return `￥${(value / 100).toFixed(2).replace(/\.?0+$/, '')}`
}

export function formatDuration(durationMs: number) {
  if (durationMs < 0) {
    durationMs = 0
  }
  const totalSeconds = Math.ceil(durationMs / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return `${hours}小时${minutes}分${seconds}秒`
}

export function parseTaskTime(value: string) {
  const normalized = normalizeTaskTime(value)
  if (!normalized) {
    return null
  }
  const parsed = new Date(normalized)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }
  return parsed
}

export function normalizeTaskTime(value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return ''
  }
  let normalized = trimmed.includes('T') ? trimmed : trimmed.replace(' ', 'T')
  if (/^\d{4}-\d{2}-\d{2}$/.test(normalized)) {
    normalized = `${normalized}T00:00:00`
  }
  if (!explicitTimeZonePattern.test(normalized)) {
    normalized = `${normalized}${chinaTimeZoneOffset}`
  }
  return normalized
}

export function timeSyncStrategyLabel(strategy: string) {
  return strategy === 'local' ? '本地时间' : '哔哩哔哩时间'
}

export function timeSyncSummary(task: Task) {
  if (task.taskMode === 'restock') {
    return task.durationMode === 'unlimited' ? '回流捡漏 · 无限模式' : `回流捡漏 · 截止 ${task.endAt || '-'}`
  }
  if (task.taskMode === 'rush_restock') {
    const restock = task.durationMode === 'unlimited' ? '无限回流' : `回流截止 ${task.endAt || '-'}`
    const sync = task.timeSyncedAt
      ? `${timeSyncStrategyLabel(task.timeSyncStrategy)} · offset ${(task.timeOffsetMillis || 0) >= 0 ? '+' : ''}${task.timeOffsetMillis || 0}ms`
      : `${timeSyncStrategyLabel(task.timeSyncStrategy)} · 未同步`
    return `抢票+回流 · ${task.rushDurationSeconds || 600}秒后切换 · ${restock} · ${sync}`
  }
  if (!task.timeSyncedAt) {
    return `${timeSyncStrategyLabel(task.timeSyncStrategy)} · 未同步`
  }
  const offset = task.timeOffsetMillis || 0
  return `${timeSyncStrategyLabel(task.timeSyncStrategy)} · offset ${offset >= 0 ? '+' : ''}${offset}ms`
}

export function lastCheckedSummary(task: Pick<Task, 'lastCheckedAt'>) {
  return `最近检测：${lastCheckedText(task.lastCheckedAt)}`
}

export function lastCheckedText(value: string) {
  if (!value) {
    return '-'
  }
  const checkedAt = parseTaskTime(value)
  if (!checkedAt) {
    return value
  }
  const milliseconds = String(checkedAt.getMilliseconds()).padStart(3, '0')
  return `${checkedAt.toLocaleString('zh-CN', { hour12: false })}.${milliseconds}`
}

export function taskModeLabel(task: Pick<Task, 'taskMode'>) {
  if (task.taskMode === 'restock') {
    return '回流捡漏'
  }
  if (task.taskMode === 'rush_restock') {
    return '抢票+回流'
  }
  return '抢票'
}

export function taskModeTagType(task: Pick<Task, 'taskMode'>) {
  if (task.taskMode === 'restock') {
    return 'warning'
  }
  if (task.taskMode === 'rush_restock') {
    return 'success'
  }
  return 'info'
}

export function statusLabel(status: string) {
  const map: Record<string, string> = {
    draft: '草稿',
    waiting_start: '等待起售',
    waiting_payment: '待支付',
    succeeded: '已成功',
    success: '已成功',
    duplicate_order: '重复订单',
    paused: '已停止',
    running: '运行中',
    failed: '失败',
    waiting_user: '等待用户',
  }
  return map[status] ?? status
}

export function bwsReserveTypeLabel(value: string) {
  return value === 'goods' ? '商品购买资格' : '活动预约'
}

export function bwsTaskModeLabel(value: string) {
  return value === 'immediate' ? '立刻抢' : '等待开票'
}

export function bwsReserveTimeLabel(item: Pick<BWSItem, 'effectiveReserveBeginTime' | 'reserveBeginTime'> | Pick<BWSTask, 'reserveBeginTime'>) {
  const timestamp = 'effectiveReserveBeginTime' in item
    ? item.effectiveReserveBeginTime || item.reserveBeginTime
    : item.reserveBeginTime
  return formatUnixSeconds(timestamp)
}

export function managedTaskKey(kind: 'ticket' | 'bws', id: number) {
  return `${kind}-${id}`
}

export function managedTaskUpdatedAt(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? row.task.updatedAt : row.bwsTask.updatedAt
}

export function compareDateText(left: string, right: string) {
  const leftTime = Date.parse(left)
  const rightTime = Date.parse(right)
  if (!Number.isNaN(leftTime) && !Number.isNaN(rightTime)) {
    return leftTime - rightTime
  }
  return left.localeCompare(right)
}

export function managedTaskName(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? row.task.name : row.bwsTask.name
}

export function managedTaskAccountName(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? row.task.accountName : row.bwsTask.accountName
}

export function managedTaskStatus(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? row.task.status : row.bwsTask.status
}

export function managedTaskLastMessage(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? row.task.lastMessage : row.bwsTask.lastMessage
}

export function managedTaskLastCheckedAt(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? row.task.lastCheckedAt : row.bwsTask.lastCheckedAt
}

export function managedTaskSourceLabel(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? '会员购' : 'BWS'
}

export function managedTaskModeText(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? taskModeLabel(row.task) : `BWS · ${bwsTaskModeLabel(row.bwsTask.mode)}`
}

export function managedTaskModeTagType(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? taskModeTagType(row.task) : 'success'
}

export function managedTaskDetail(row: ManagedTaskRow) {
  if (row.kind === 'ticket') {
    return taskBuyerSummary(row.task)
  }
  return `${bwsReserveTypeLabel(row.bwsTask.reserveType)} · ${row.bwsTask.reserveDate || '-'} · 票号 ${row.bwsTask.ticketNo || '-'}`
}

export function managedTaskTimeSummary(row: ManagedTaskRow) {
  return row.kind === 'ticket' ? timeSyncSummary(row.task) : bwsTimeSyncSummary(row.bwsTask)
}

export function bwsTimeSyncSummary(task: BWSTask) {
  const sync = task.timeSyncStrategy === 'local'
    ? '本地时间'
    : task.timeSyncedAt
      ? `${timeSyncStrategyLabel(task.timeSyncStrategy)} · offset ${(task.timeOffsetMillis || 0) >= 0 ? '+' : ''}${task.timeOffsetMillis || 0}ms`
      : `${timeSyncStrategyLabel(task.timeSyncStrategy)} · 未同步`
  return `${bwsReserveTypeLabel(task.reserveType)} · ${bwsTaskModeLabel(task.mode)} · ${sync}`
}

export function bwsManagedSummary(task: BWSTask) {
  return `${task.title || task.name} · ${bwsReserveTypeLabel(task.reserveType)} · ${task.reserveDate || '-'}`
}

export function formatUnixSeconds(value: number) {
  if (!value) {
    return '-'
  }
  return new Date(value * 1000).toLocaleString('zh-CN', { hour12: false })
}

export function bwsItemTagText(item: BWSItem) {
  const tags = []
  if (item.isVipTicket) {
    tags.push(item.userIsVip ? 'VIP优先' : 'VIP场次')
  }
  if (item.isSecondPay) {
    tags.push('二次付费')
  }
  if (item.reserved) {
    tags.push('已预约')
  }
  if (item.state === 3) {
    tags.push('已结束')
  }
  return tags.join(' · ')
}

export function accountStatusLabel(account: Account) {
  const map: Record<string, string> = {
    logged_in: '已登录',
    configured: '待验证',
    login_invalid: '登录失效',
    missing_cookie: '缺少 Cookie',
  }
  if (!account.hasCookie) {
    return '缺少 Cookie'
  }
  return map[account.status] ?? account.status
}

export function accountStatusClass(account: Account) {
  if (!account.hasCookie) {
    return 'idle'
  }
  if (account.status === 'logged_in') {
    return 'ready'
  }
  if (account.status === 'login_invalid') {
    return 'bad'
  }
  return 'idle'
}

export function accountStatusTagType(account: Account) {
  if (!account.hasCookie) {
    return 'warning'
  }
  if (account.status === 'logged_in') {
    return 'success'
  }
  if (account.status === 'login_invalid') {
    return 'danger'
  }
  return 'warning'
}

export function notificationProviderLabel(provider: string) {
  if (provider === 'pushplus') {
    return 'PushPlus'
  }
  if (provider === 'bark') {
    return 'Bark'
  }
  return provider || '-'
}

export function notificationTokenLabel(provider: string) {
  return provider === 'bark' ? 'Bark Token 或完整推送地址' : 'PushPlus Token'
}

export function notificationTokenPlaceholder(provider: string) {
  return provider === 'bark'
    ? '例如：jmGYKxxxxx 或 https://bark.example.app/jmGYKxxxxx'
    : '填写 PushPlus Token'
}

export function notificationTestTagType(status: string) {
  if (status === 'success') {
    return 'success'
  }
  if (status === 'error') {
    return 'danger'
  }
  return 'info'
}

export function proxyGroupTypeLabel(type: string) {
  return type === 'api' ? 'API 代理组' : '普通代理组'
}

export function proxyProtocolLabel(protocol: string) {
  if (protocol === 'socks5') {
    return 'SOCKS5'
  }
  return (protocol || 'http').toUpperCase()
}

export function proxyTestTagType(status: string) {
  if (status === 'success') {
    return 'success'
  }
  if (status === 'error') {
    return 'danger'
  }
  return 'info'
}

export function proxyGroupOptionLabel(group: ProxyGroup) {
  const available = group.availableNodeCount > 0 ? `${group.availableNodeCount}/${group.nodeCount} 可用` : `${group.nodeCount} 节点`
  return `${group.name} · ${proxyGroupTypeLabel(group.type)} · ${available}`
}

export function proxyModeLabel(mode: string) {
  return mode === 'concurrent' ? '并发代理' : '循环代理'
}

export function taskProxySummary(task: { proxyGroupName?: string; proxyMode?: string }) {
  if (!task.proxyGroupName) {
    return ''
  }
  return `代理组：${task.proxyGroupName} · ${proxyModeLabel(task.proxyMode || 'round_robin')}`
}

export function taskStatusClass(status: string) {
  if (status === 'waiting_payment' || status === 'succeeded') {
    return 'ready'
  }
  if (status === 'failed' || status === 'waiting_user') {
    return 'bad'
  }
  return 'idle'
}

export function taskStatusTagType(status: string) {
  if (status === 'waiting_payment' || status === 'succeeded') {
    return 'success'
  }
  if (status === 'failed' || status === 'waiting_user') {
    return 'danger'
  }
  if (status === 'running' || status === 'waiting_start') {
    return 'warning'
  }
  return 'info'
}

export function logLevelTagType(level: string) {
  if (level === 'warn') {
    return 'warning'
  }
  if (level === 'error') {
    return 'danger'
  }
  return 'info'
}
