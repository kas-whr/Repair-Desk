import type { EquipmentStatus, Priority, TicketStatus } from './api/types'

export const statusLabel: Record<TicketStatus, string> = {
  NEW: 'Новая',
  IN_PROGRESS: 'В работе',
  RESOLVED: 'Решена',
  CLOSED: 'Закрыта',
}

/** Button text for a transition into the given status. */
export const transitionLabel = (from: TicketStatus, to: TicketStatus): string => {
  if (to === 'IN_PROGRESS') return from === 'RESOLVED' ? 'Переоткрыть' : 'Взять в работу'
  if (to === 'RESOLVED') return 'Отметить решённой'
  if (to === 'CLOSED') return from === 'NEW' ? 'Отменить' : 'Закрыть'
  return statusLabel[to]
}

export const priorityLabel: Record<Priority, string> = {
  LOW: 'Низкий',
  MEDIUM: 'Средний',
  HIGH: 'Высокий',
  CRITICAL: 'Критический',
}

export const equipmentStatusLabel: Record<EquipmentStatus, string> = {
  ACTIVE: 'Исправно',
  BROKEN: 'Неисправно',
  RETIRED: 'Списано',
}

const dateFmt = new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' })
export const formatDate = (iso: string) => dateFmt.format(new Date(iso))
