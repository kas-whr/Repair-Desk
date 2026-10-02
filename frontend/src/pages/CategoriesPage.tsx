import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { api, fieldError } from '../api/client'
import type { Category } from '../api/types'
import { Empty, ErrorBanner, Field, Modal } from '../components/ui'
import { formatDate } from '../labels'

export function CategoriesPage() {
  const [items, setItems] = useState<Category[] | null>(null)
  const [error, setError] = useState<unknown>(null)
  const [editing, setEditing] = useState<Category | 'new' | null>(null)

  const load = useCallback(() => {
    api.categories.list().then(setItems).catch(setError)
  }, [])

  useEffect(load, [load])

  async function remove(c: Category) {
    if (!confirm(`Удалить категорию «${c.name}»? ВСЕ заявки этой категории тоже будут удалены.`)) return
    try {
      await api.categories.remove(c.id)
      load()
    } catch (err) {
      setError(err)
    }
  }

  return (
    <section>
      <div className="page-header">
        <h1>Категории</h1>
        <button className="btn primary" onClick={() => setEditing('new')}>
          + Добавить
        </button>
      </div>
      <p className="muted">
        Срок выполнения заявки = время создания + SLA категории. Изменение SLA не влияет на уже созданные заявки.
      </p>

      <ErrorBanner error={error} />
      {items && items.length === 0 && <Empty>Категорий нет</Empty>}
      {items && items.length > 0 && (
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Название</th>
                <th>SLA, часов</th>
                <th>Создана</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {items.map((c) => (
                <tr key={c.id}>
                  <td>{c.name}</td>
                  <td>{c.sla_hours}</td>
                  <td>{formatDate(c.created_at)}</td>
                  <td className="row-actions">
                    <button className="link-btn" onClick={() => setEditing(c)}>
                      Изменить
                    </button>
                    <button className="link-btn danger" onClick={() => remove(c)}>
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
        <Modal title={editing === 'new' ? 'Новая категория' : 'Редактирование'} onClose={() => setEditing(null)}>
          <CategoryForm
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

function CategoryForm({
  initial,
  onCancel,
  onSaved,
}: {
  initial?: Category
  onCancel: () => void
  onSaved: () => void
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [slaHours, setSlaHours] = useState(String(initial?.sla_hours ?? 24))
  const [error, setError] = useState<unknown>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    const in_ = { name, sla_hours: Number(slaHours) }
    try {
      if (initial) await api.categories.update(initial.id, in_)
      else await api.categories.create(in_)
      onSaved()
    } catch (err) {
      setError(err)
    }
  }

  return (
    <form onSubmit={submit} className="form">
      <ErrorBanner error={error} />
      <Field label="Название" error={fieldError(error, 'name')}>
        <input value={name} onChange={(e) => setName(e.target.value)} maxLength={100} autoFocus required />
      </Field>
      <Field label="Нормативное время выполнения (SLA), часов" error={fieldError(error, 'sla_hours')}>
        <input type="number" min={1} max={8760} value={slaHours} onChange={(e) => setSlaHours(e.target.value)} required />
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
