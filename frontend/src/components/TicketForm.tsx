import { useState, type FormEvent } from 'react'
import { fieldError } from '../api/client'
import type { Category, CreateTicketInput, Equipment, Priority, Ticket, UpdateTicketInput } from '../api/types'
import { PRIORITIES } from '../api/types'
import { equipmentStatusLabel, priorityLabel } from '../labels'
import { ErrorBanner, Field } from './ui'

type Props = {
  categories: Category[]
  equipment: Equipment[]
  ticket?: Ticket
  onCreate?: (in_: CreateTicketInput) => Promise<void>
  onUpdate?: (in_: UpdateTicketInput) => Promise<void>
  onCancel: () => void
}

/** Create form when ticket is undefined, edit form otherwise (category is not editable). */
export function TicketForm({ categories, equipment, ticket, onCreate, onUpdate, onCancel }: Props) {
  const [title, setTitle] = useState(ticket?.title ?? '')
  const [description, setDescription] = useState(ticket?.description ?? '')
  const [priority, setPriority] = useState<Priority>(ticket?.priority ?? 'MEDIUM')
  const [categoryId, setCategoryId] = useState(ticket?.category_id ?? '')
  const [equipmentId, setEquipmentId] = useState(ticket?.equipment_id ?? '')
  const [error, setError] = useState<unknown>(null)
  const [saving, setSaving] = useState(false)

  // Retired equipment cannot be assigned, but keep the one already linked to the ticket.
  const equipmentOptions = equipment.filter((e) => e.status !== 'RETIRED' || e.id === ticket?.equipment_id)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    const common = { title, description, priority, equipment_id: equipmentId || null }
    try {
      if (ticket) await onUpdate?.(common)
      else await onCreate?.({ ...common, category_id: categoryId })
    } catch (err) {
      setError(err)
      setSaving(false)
    }
  }

  return (
    <form onSubmit={submit} className="form">
      <ErrorBanner error={error} />
      <Field label="Заголовок" error={fieldError(error, 'title')}>
        <input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={255} autoFocus required />
      </Field>
      <Field label="Описание проблемы" error={fieldError(error, 'description')}>
        <textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={4} />
      </Field>
      <div className="form-row">
        <Field label="Категория" error={fieldError(error, 'category_id')}>
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
            disabled={!!ticket}
            required
            title={ticket ? 'Категорию нельзя изменить после создания' : undefined}
          >
            <option value="">— выберите —</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} (SLA {c.sla_hours} ч)
              </option>
            ))}
          </select>
        </Field>
        <Field label="Приоритет" error={fieldError(error, 'priority')}>
          <select value={priority} onChange={(e) => setPriority(e.target.value as Priority)}>
            {PRIORITIES.map((p) => (
              <option key={p} value={p}>
                {priorityLabel[p]}
              </option>
            ))}
          </select>
        </Field>
      </div>
      <Field label="Оборудование (необязательно)" error={fieldError(error, 'equipment_id')}>
        <select value={equipmentId} onChange={(e) => setEquipmentId(e.target.value)}>
          <option value="">— не указано —</option>
          {equipmentOptions.map((e) => (
            <option key={e.id} value={e.id}>
              {e.name} · {e.inventory_number}
              {e.status !== 'ACTIVE' ? ` (${equipmentStatusLabel[e.status]})` : ''}
            </option>
          ))}
        </select>
      </Field>
      <div className="form-actions">
        <button type="button" className="btn" onClick={onCancel}>
          Отмена
        </button>
        <button type="submit" className="btn primary" disabled={saving}>
          {ticket ? 'Сохранить' : 'Создать заявку'}
        </button>
      </div>
    </form>
  )
}
