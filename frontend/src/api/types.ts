export type TicketStatus = 'NEW' | 'IN_PROGRESS' | 'RESOLVED' | 'CLOSED'
export type Priority = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
export type EquipmentStatus = 'ACTIVE' | 'BROKEN' | 'RETIRED'

export const TICKET_STATUSES: TicketStatus[] = ['NEW', 'IN_PROGRESS', 'RESOLVED', 'CLOSED']
export const PRIORITIES: Priority[] = ['LOW', 'MEDIUM', 'HIGH', 'CRITICAL']
export const EQUIPMENT_STATUSES: EquipmentStatus[] = ['ACTIVE', 'BROKEN', 'RETIRED']

export interface Category {
  id: string
  name: string
  sla_hours: number
  created_at: string
}

export interface CategoryInput {
  name: string
  sla_hours: number
}

export interface Equipment {
  id: string
  name: string
  inventory_number: string
  location: string
  status: EquipmentStatus
}

export type EquipmentInput = Omit<Equipment, 'id'>

export interface Ticket {
  id: string
  title: string
  description: string
  status: TicketStatus
  priority: Priority
  category_id: string
  equipment_id: string | null
  due_at: string
  created_at: string
  updated_at: string
  overdue: boolean
  allowed_transitions: TicketStatus[]
}

export interface CreateTicketInput {
  title: string
  description: string
  priority: Priority
  category_id: string
  equipment_id: string | null
}

export interface UpdateTicketInput {
  title: string
  description: string
  priority: Priority
  equipment_id: string | null
}

export interface TicketFilter {
  status?: TicketStatus | ''
  priority?: Priority | ''
  category_id?: string
  equipment_id?: string
  overdue?: boolean
  limit?: number
  offset?: number
}

export interface TicketPage {
  items: Ticket[]
  total: number
  limit: number
  offset: number
}

export interface Comment {
  id: string
  ticket_id: string
  content: string
  created_at: string
}
