import type {
  Category,
  CategoryInput,
  Comment,
  CreateTicketInput,
  Equipment,
  EquipmentInput,
  EquipmentStatus,
  Ticket,
  TicketFilter,
  TicketPage,
  TicketStatus,
  UpdateTicketInput,
} from './types'

const BASE = '/api/v1'

/** Mirrors the backend error format: {"error": {"code", "message", "details"}} */
export class ApiError extends Error {
  status: number
  code: string
  details: Record<string, string>

  constructor(status: number, code: string, message: string, details: Record<string, string> = {}) {
    super(message)
    this.status = status
    this.code = code
    this.details = details
  }
}

/** Validation message for a form field, taken from error.details. */
export function fieldError(error: unknown, field: string): string | undefined {
  return error instanceof ApiError ? error.details[field] : undefined
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response
  try {
    res = await fetch(BASE + path, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'NETWORK_ERROR', 'Сервер недоступен')
  }

  if (res.status === 204) return undefined as T

  const data = await res.json().catch(() => null)
  if (!res.ok) {
    const e = data?.error
    throw new ApiError(res.status, e?.code ?? 'UNKNOWN', e?.message ?? `HTTP ${res.status}`, e?.details)
  }
  return data as T
}

function query(params: Record<string, string | number | boolean | undefined>): string {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') q.set(k, String(v))
  }
  const s = q.toString()
  return s ? `?${s}` : ''
}

type List<T> = { items: T[] }

export const api = {
  categories: {
    list: () => request<List<Category>>('GET', '/categories').then((r) => r.items),
    create: (in_: CategoryInput) => request<Category>('POST', '/categories', in_),
    update: (id: string, in_: CategoryInput) => request<Category>('PUT', `/categories/${id}`, in_),
    remove: (id: string) => request<void>('DELETE', `/categories/${id}`),
  },
  equipment: {
    list: (status?: EquipmentStatus | '') =>
      request<List<Equipment>>('GET', `/equipment${query({ status })}`).then((r) => r.items),
    create: (in_: EquipmentInput) => request<Equipment>('POST', '/equipment', in_),
    update: (id: string, in_: EquipmentInput) => request<Equipment>('PUT', `/equipment/${id}`, in_),
    remove: (id: string) => request<void>('DELETE', `/equipment/${id}`),
  },
  tickets: {
    list: (f: TicketFilter) => request<TicketPage>('GET', `/tickets${query({ ...f })}`),
    get: (id: string) => request<Ticket>('GET', `/tickets/${id}`),
    create: (in_: CreateTicketInput) => request<Ticket>('POST', '/tickets', in_),
    update: (id: string, in_: UpdateTicketInput) => request<Ticket>('PUT', `/tickets/${id}`, in_),
    changeStatus: (id: string, status: TicketStatus) =>
      request<Ticket>('PATCH', `/tickets/${id}/status`, { status }),
    remove: (id: string) => request<void>('DELETE', `/tickets/${id}`),
  },
  comments: {
    list: (ticketId: string) =>
      request<List<Comment>>('GET', `/tickets/${ticketId}/comments`).then((r) => r.items),
    create: (ticketId: string, content: string) =>
      request<Comment>('POST', `/tickets/${ticketId}/comments`, { content }),
    update: (ticketId: string, id: string, content: string) =>
      request<Comment>('PUT', `/tickets/${ticketId}/comments/${id}`, { content }),
    remove: (ticketId: string, id: string) => request<void>('DELETE', `/tickets/${ticketId}/comments/${id}`),
  },
}
