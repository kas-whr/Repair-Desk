import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { api, fieldError } from '../api/client'
import type { Equipment, EquipmentStatus } from '../api/types'
import { EQUIPMENT_STATUSES } from '../api/types'
import { Empty, EquipmentStatusBadge, ErrorBanner, Field, Modal } from '../components/ui'
import { equipmentStatusLabel } from '../labels'

export function EquipmentPage() {
  const [items, setItems] = useState<Equipment[] | null>(null)
  const [status, setStatus] = useState<EquipmentStatus | ''>('')
  const [error, setError] = useState<unknown>(null)
  const [editing, setEditing] = useState<Equipment | 'new' | null>(null)

  const load = useCallback(() => {
    api.equipment.list(status).then(setItems).catch(setError)
  }, [status])

  useEffect(load, [load])

  async function remove(e: Equipment) {
    if (!confirm(`Удалить «${e.name}»? Связанные заявки сохранятся, но без привязки к оборудованию.`)) return
    try {
      await api.equipment.remove(e.id)
      load()
    } catch (err) {
      setError(err)
    }
  }

  return (
    <section>
      <div className="page-header">
        <h1>Оборудование</h1>
        <button className="btn primary" onClick={() => setEditing('new')}>
          + Добавить
        </button>
      </div>

      <div className="filters">
        <select value={status} onChange={(e) => setStatus(e.target.value as EquipmentStatus | '')} aria-label="Состояние">
          <option value="">Все состояния</option>
          {EQUIPMENT_STATUSES.map((s) => (
            <option key={s} value={s}>
              {equipmentStatusLabel[s]}
            </option>
          ))}
        </select>
      </div>

      <ErrorBanner error={error} />
      {items && items.length === 0 && <Empty>Оборудование не найдено</Empty>}
      {items && items.length > 0 && (
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Название</th>
                <th>Инв. номер</th>
                <th>Местоположение</th>
                <th>Состояние</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {items.map((e) => (
                <tr key={e.id}>
                  <td>{e.name}</td>
                  <td className="mono">{e.inventory_number}</td>
                  <td>{e.location}</td>
                  <td>
                    <EquipmentStatusBadge status={e.status} />
                  </td>
                  <td className="row-actions">
                    <button className="link-btn" onClick={() => setEditing(e)}>
                      Изменить
                    </button>
                    <button className="link-btn danger" onClick={() => remove(e)}>
                      Удалить
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {editing && (
        <Modal title={editing === 'new' ? 'Новое оборудование' : 'Редактирование'} onClose={() => setEditing(null)}>
          <EquipmentForm
            initial={editing === 'new' ? undefined : editing}
            onCancel={() => setEditing(null)}
            onSaved={() => {
              setEditing(null)
              load()
            }}
          />
        </Modal>
      )}
    </section>
  )
}

function EquipmentForm({
  initial,
  onCancel,
  onSaved,
}: {
  initial?: Equipment
  onCancel: () => void
  onSaved: () => void
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [inventoryNumber, setInventoryNumber] = useState(initial?.inventory_number ?? '')
  const [location, setLocation] = useState(initial?.location ?? '')
  const [status, setStatus] = useState<EquipmentStatus>(initial?.status ?? 'ACTIVE')
  const [error, setError] = useState<unknown>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    const in_ = { name, inventory_number: inventoryNumber, location, status }
    try {
      if (initial) await api.equipment.update(initial.id, in_)
      else await api.equipment.create(in_)
      onSaved()
    } catch (err) {
      setError(err)
    }
  }

  return (
    <form onSubmit={submit} className="form">
      <ErrorBanner error={error} />
      <Field label="Название" error={fieldError(error, 'name')}>
        <input value={name} onChange={(e) => setName(e.target.value)} autoFocus required />
      </Field>
      <div className="form-row">
        <Field label="Инвентарный номер" error={fieldError(error, 'inventory_number')}>
          <input value={inventoryNumber} onChange={(e) => setInventoryNumber(e.target.value)} required />
        </Field>
        <Field label="Состояние" error={fieldError(error, 'status')}>
          <select value={status} onChange={(e) => setStatus(e.target.value as EquipmentStatus)}>
            {EQUIPMENT_STATUSES.map((s) => (
              <option key={s} value={s}>
                {equipmentStatusLabel[s]}
              </option>
            ))}
          </select>
        </Field>
      </div>
      <Field label="Местоположение" error={fieldError(error, 'location')}>
        <input value={location} onChange={(e) => setLocation(e.target.value)} required />
      </Field>
      <div className="form-actions">
        <button type="button" className="btn" onClick={onCancel}>
          Отмена
        </button>
        <button type="submit" className="btn primary">
          Сохранить
        </button>
      </div>
    </form>
  )
}
