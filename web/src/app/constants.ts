import type { SectionKey } from './types'

export const defaultSuperModeBase = 'show'

export const superModeBaseOptions = [
  { value: 'show', label: 'show.bilibili.com' },
  { value: 'bilibili_cn', label: 'www.bilibili.cn' },
  { value: 'biligo', label: 'www.biligo.com' },
  { value: 'me_bilibili', label: 'me.bilibili.cc' },
]

export const sections: Array<{ key: SectionKey; label: string }> = [
  { key: 'accounts', label: '哔哩哔哩账号管理' },
  { key: 'notifications', label: '通知管理' },
  { key: 'proxies', label: '代理设置' },
  { key: 'taskConfig', label: '抢票' },
  { key: 'bws', label: 'BWS预约' },
  { key: 'taskStatus', label: '任务管理' },
]

export const QR_AUTO_POLL_MS = 2500

export const chinaTimeZoneOffset = '+08:00'
export const chinaTimeZoneOffsetMillis = 8 * 60 * 60 * 1000
export const explicitTimeZonePattern = /(?:Z|[+-]\d{2}:?\d{2})$/i
