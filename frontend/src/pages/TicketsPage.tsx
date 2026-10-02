import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Category, Equipment, Priority, TicketPage, TicketStatus } from '../api/types'
import { PRIORITIES, TICKET_STATUSES } from '../api/types'
import { TicketForm } from '../components/TicketForm'
import { Empty, ErrorBanner, Modal, OverdueBadge, PriorityBadge, StatusBadge } from '../components/ui'
import { formatDate, priorityLabel, statusLabel } from '../labels'

const PAGE_SIZE = 20

export function TicketsPage() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const status = (params.get('status') ?? '') as TicketStatus | ''
  const priority = (params.get('priority') ?? '') as Priority | ''
  const categoryId = params.get('category_id') ?? ''
  const overdue = params.get('overdue') === 'true'
  const offset = Number(params.get('offset') ?? 0)

  const [page, setPage] = useState<TicketPage | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [equipment, setEquipment] = useState<Equipment[]>([])
  const [error, setError] = useState<unknown>(null)
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    Promise.all([api.categories.list(), api.equipment.list()])
      .then(([c, e]) => {
        setCategories(c)
        setEquipment(e)
      })
      .catch(setError)
  }, [])

  const load = useCallback(() => {
    api.tickets
      .list({ status, priority, category_id: categoryId, overdue: overdue || undefined, limit: PAGE_SIZE, offset })
      .then((p) => {
        setPage(p)
        setError(null)
      })
      .catch(setError)
  }, [status, priority, categoryId, overdue, offset])

  useEffect(load, [load])

  function setFilter(key: string, value: string) {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    next.delete('offset')
    setParams(next)
  }

  function goToOffset(o: number) {
    const next = new URLSearchParams(params)
    if (o > 0) next.set('offset', String(o))
    else next.delete('offset')
    setParams(next)
  }

  const categoryName = (id: string) => categories.find((c) => c.id === id)?.name ?? '—'
  const equipmentName = (id: string | null) => (id ? (equipment.find((e) => e.id === id)?.name ?? '—') : '—')

  return (
    <section>
      <div className="page-header">
        <h1>Заявки</h1>
        <button className="btn primary" onClick={() => setCreating(true)}>
          + Новая заявка
        </button>
      </div>

      <div className="filters">
        <select value={status} onChange={(e) => setFilter('status', e.target.value)} aria-label="Статус">
          <option value="">Все статусы</option>
          {TICKET_STATUSES.map((s) => (
            <option key={s} value={s}>
              {statusLabel[s]}
            </option>
          ))}
        </select>
        <select value={priority} onChange={(e) => setFilter('priority', e.target.value)} aria-label="Приоритет">
          <option value="">Все приоритеты</option>
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>
              {priorityLabel[p]}
            </option>
          ))}
        </select>
        <select value={categoryId} onChange={(e) => setFilter('category_id', e.target.value)} aria-label="Категория">
          <option value="">Все категории</option>
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
        <label className="checkbox">
          <input
            type="checkbox"
            checked={overdue}
            onChange={(e) => setFilter('overdue', e.target.checked ? 'true' : '')}
          />
          Только просроченные
        </label>
      </div>

      <ErrorBanner error={error} />

      {page && page.items.length === 0 && <Empty>Заявок не найдено</Empty>}

      {page && page.items.length > 0 && (
        <div className="table-wrap">
          <table className="table clickable">
            <thead>
              <tr>
                <th>Заявка</th>
                <th>Статус</th>
                <th>Приоритет</th>
                <th>Категория</th>
                <th>Оборудование</th>
                <th>Срок</th>
              </tr>
            </thead>
            <tbody>
              {page.items.map((t) => (
                <tr key={t.id} onClick={() => navigate(`/tickets/${t.id}`)}>
                  <td>
                    <Link to={`/tickets/${t.id}`} onClick={(e) => e.stopPropagation()}>
                      {t.title}
                    </Link>
                    <div className="muted small">создана {formatDate(t.created_at)}</div>
                  </td>
                  <td>
                    <StatusBadge status={t.status} />
                  </td>
                  <td>
                    <PriorityBadge priority={t.priority} />
                  </td>
                  <td>{categoryName(t.category_id)}</td>
                  <td>{equipmentName(t.equipment_id)}</td>
                  <td>
                    {formatDate(t.due_at)} {t.overdue && <OverdueBadge />}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {page && page.total > PAGE_SIZE && (
        <div className="pagination">
          <button className="btn" disabled={offset === 0} onClick={() => goToOffset(Math.max(0, offset - PAGE_SIZE))}>
            ← Назад
          </button>
          <span className="muted">
            {offset + 1}–{Math.min(offset + PAGE_SIZE, page.total)} из {page.total}
          </span>
          <button
            className="btn"
            disabled={offset + PAGE_SIZE >= page.total}
            onClick={() => goToOffset(offset + PAGE_SIZE)}
          >
            Вперёд →
          </button>
        </div>
      )}

      {creating && (
        <Modal title="Новая заявка" onClose={() => setCreating(false)}>
          <TicketForm
            categories={categories}
            equipment={equipment}
            onCancel={() => setCreating(false)}
            onCreate={async (in_) => {
              const t = await api.tickets.create(in_)
              navigate(`/tickets/${t.id}`)
            }}
          />
        </Modal>
      )}
    </section>
  )
}
