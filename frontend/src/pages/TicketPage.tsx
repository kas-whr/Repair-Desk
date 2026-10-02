import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Category, Comment, Equipment, Ticket, TicketStatus } from '../api/types'
import { TicketForm } from '../components/TicketForm'
import { Empty, ErrorBanner, Modal, OverdueBadge, PriorityBadge, StatusBadge } from '../components/ui'
import { formatDate, transitionLabel } from '../labels'

export function TicketPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()

  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [comments, setComments] = useState<Comment[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [equipment, setEquipment] = useState<Equipment[]>([])
  const [error, setError] = useState<unknown>(null)
  const [editing, setEditing] = useState(false)
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => {
    Promise.all([api.tickets.get(id), api.comments.list(id), api.categories.list(), api.equipment.list()])
      .then(([t, cm, c, e]) => {
        setTicket(t)
        setComments(cm)
        setCategories(c)
        setEquipment(e)
      })
      .catch(setError)
  }, [id])

  useEffect(load, [load])

  async function run(action: () => Promise<void>) {
    setBusy(true)
    setError(null)
    try {
      await action()
    } catch (err) {
      setError(err)
    } finally {
      setBusy(false)
    }
  }

  const changeStatus = (to: TicketStatus) => run(async () => setTicket(await api.tickets.changeStatus(id, to)))

  const remove = () => {
    if (!confirm('Удалить заявку? Это действие нельзя отменить.')) return
    run(async () => {
      await api.tickets.remove(id)
      navigate('/tickets')
    })
  }

  if (!ticket) {
    return (
      <section>
        <Link to="/tickets" className="back">
          ← К списку заявок
        </Link>
        <ErrorBanner error={error} />
      </section>
    )
  }

  const closed = ticket.status === 'CLOSED'
  const category = categories.find((c) => c.id === ticket.category_id)
  const eq = equipment.find((e) => e.id === ticket.equipment_id)

  return (
    <section>
      <Link to="/tickets" className="back">
        ← К списку заявок
      </Link>

      <div className="page-header">
        <div>
          <h1>{ticket.title}</h1>
          <div className="badges">
            <StatusBadge status={ticket.status} />
            <PriorityBadge priority={ticket.priority} />
            {ticket.overdue && <OverdueBadge />}
          </div>
        </div>
        <div className="actions">
          {ticket.allowed_transitions.map((to) => (
            <button
              key={to}
              className={`btn ${to === 'CLOSED' ? '' : 'primary'}`}
              disabled={busy}
              onClick={() => changeStatus(to)}
            >
              {transitionLabel(ticket.status, to)}
            </button>
          ))}
          {!closed && (
            <button className="btn" disabled={busy} onClick={() => setEditing(true)}>
              Редактировать
            </button>
          )}
          {ticket.status === 'NEW' && (
            <button className="btn danger" disabled={busy} onClick={remove}>
              Удалить
            </button>
          )}
        </div>
      </div>

      <ErrorBanner error={error} />
      {closed && <div className="notice">Заявка закрыта: изменения и новые комментарии недоступны.</div>}

      <div className="detail-grid">
        <div className="card">
          <h3>Описание</h3>
          <p className="description">{ticket.description || <span className="muted">Без описания</span>}</p>
        </div>
        <dl className="card props">
          <dt>Категория</dt>
          <dd>{category ? `${category.name} (SLA ${category.sla_hours} ч)` : '—'}</dd>
          <dt>Оборудование</dt>
          <dd>{eq ? `${eq.name} · ${eq.inventory_number}, ${eq.location}` : '—'}</dd>
          <dt>Срок выполнения</dt>
          <dd className={ticket.overdue ? 'text-danger' : ''}>{formatDate(ticket.due_at)}</dd>
          <dt>Создана</dt>
          <dd>{formatDate(ticket.created_at)}</dd>
          <dt>Изменена</dt>
          <dd>{formatDate(ticket.updated_at)}</dd>
        </dl>
      </div>

      <Comments
        ticketId={id}
        comments={comments}
        readOnly={closed}
        onChange={() => api.comments.list(id).then(setComments).catch(setError)}
      />

      {editing && (
        <Modal title="Редактирование заявки" onClose={() => setEditing(false)}>
          <TicketForm
            categories={categories}
            equipment={equipment}
            ticket={ticket}
            onCancel={() => setEditing(false)}
            onUpdate={async (in_) => {
              setTicket(await api.tickets.update(id, in_))
              setEditing(false)
            }}
          />
        </Modal>
      )}
    </section>
  )
}

function Comments({
  ticketId,
  comments,
  readOnly,
  onChange,
}: {
  ticketId: string
  comments: Comment[]
  readOnly: boolean
  onChange: () => void
}) {
  const [text, setText] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editText, setEditText] = useState('')
  const [error, setError] = useState<unknown>(null)

  async function guarded(action: () => Promise<unknown>) {
    setError(null)
    try {
      await action()
      onChange()
      return true
    } catch (err) {
      setError(err)
      return false
    }
  }

  async function add(e: FormEvent) {
    e.preventDefault()
    if (await guarded(() => api.comments.create(ticketId, text))) setText('')
  }

  async function saveEdit(id: string) {
    if (await guarded(() => api.comments.update(ticketId, id, editText))) setEditingId(null)
  }

  return (
    <div className="card comments">
      <h3>Комментарии ({comments.length})</h3>
      <ErrorBanner error={error} />
      {comments.length === 0 && <Empty>Комментариев пока нет</Empty>}
      <ul>
        {comments.map((c) => (
          <li key={c.id}>
            <div className="muted small">{formatDate(c.created_at)}</div>
            {editingId === c.id ? (
              <div className="comment-edit">
                <textarea value={editText} onChange={(e) => setEditText(e.target.value)} rows={2} />
                <div className="form-actions">
                  <button className="btn" onClick={() => setEditingId(null)}>
                    Отмена
                  </button>
                  <button className="btn primary" onClick={() => saveEdit(c.id)}>
                    Сохранить
                  </button>
                </div>
              </div>
            ) : (
              <p>{c.content}</p>
            )}
            {!readOnly && editingId !== c.id && (
              <div className="comment-actions">
                <button
                  className="link-btn"
                  onClick={() => {
                    setEditingId(c.id)
                    setEditText(c.content)
                  }}
                >
                  Изменить
                </button>
                <button
                  className="link-btn danger"
                  onClick={() => confirm('Удалить комментарий?') && guarded(() => api.comments.remove(ticketId, c.id))}
                >
                  Удалить
                </button>
              </div>
            )}
          </li>
        ))}
      </ul>
      {!readOnly && (
        <form onSubmit={add} className="comment-form">
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Добавить комментарий…"
            rows={2}
            required
          />
          <button type="submit" className="btn primary" disabled={!text.trim()}>
            Отправить
          </button>
        </form>
      )}
    </div>
  )
}
