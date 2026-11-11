import type { BWSTask, Task, TicketProjectHistory } from '../api'

export type SectionKey = 'accounts' | 'notifications' | 'proxies' | 'taskConfig' | 'bws' | 'taskStatus'
export type QRLoginStatus = 'idle' | 'generated' | 'waiting_scan' | 'waiting_confirm' | 'confirmed' | 'expired' | 'failed'
export type TicketProjectHistorySuggestion = TicketProjectHistory & { value: string }
export type ManagedTaskKind = 'ticket' | 'bws'
export type ManagedTaskRow =
  | { key: string; kind: 'ticket'; id: number; task: Task; bwsTask?: never }
  | { key: string; kind: 'bws'; id: number; task?: never; bwsTask: BWSTask }
export type ManagedLogRow = {
  key: string
  kind: ManagedTaskKind
  id: number
  taskId: number
  level: string
  message: string
  createdAt: string
}
