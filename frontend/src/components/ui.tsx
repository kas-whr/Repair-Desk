import { useEffect, type ReactNode } from 'react'
import { ApiError } from '../api/client'
import type { EquipmentStatus, Priority, TicketStatus } from '../api/types'
import { equipmentStatusLabel, priorityLabel, statusLabel } from '../labels'

export function StatusBadge({ status }: { status: TicketStatus }) {
  return <span className={`badge status-${status}`}>{statusLabel[status]}</span>
}

export function PriorityBadge({ priority }: { priority: Priority }) {
  return <span className={`badge priority-${priority}`}>{priorityLabel[priority]}</span>
}

export function EquipmentStatusBadge({ status }: { status: EquipmentStatus }) {
  return <span className={`badge eq-${status}`}>{equipmentStatusLabel[status]}</span>
}

export function OverdueBadge() {
  return <span className="badge overdue">Просрочена</span>
}

/** Shows the message of an API error; field details are shown next to form fields. */
export function ErrorBanner({ error }: { error: unknown }) {
  if (!error) return null
  const message = error instanceof Error ? error.message : String(error)
  const code = error instanceof ApiError ? error.code : null
  return (
    <div className="error-banner" role="alert">
      {message}
      {code && code !== 'VALIDATION_ERROR' && <span className="error-code">{code}</span>}
    </div>
  )
}

export function Field({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      {children}
      {error && <span className="field-error">{error}</span>}
    </label>
  )
}

export function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div className="modal-backdrop" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal" role="dialog" aria-modal="true" aria-label={title}>
        <div className="modal-header">
          <h2>{title}</h2>
          <button className="icon-btn" onClick={onClose} aria-label="Закрыть">
            ×
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>
}
