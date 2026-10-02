import { BrowserRouter, Navigate, NavLink, Route, Routes } from 'react-router-dom'
import { CategoriesPage } from './pages/CategoriesPage'
import { EquipmentPage } from './pages/EquipmentPage'
import { TicketPage } from './pages/TicketPage'
import { TicketsPage } from './pages/TicketsPage'

export default function App() {
  return (
    <BrowserRouter>
      <header className="topbar">
        <div className="topbar-inner">
          <span className="logo">🛠 Repair Desk</span>
          <nav>
            <NavLink to="/tickets">Заявки</NavLink>
            <NavLink to="/equipment">Оборудование</NavLink>
            <NavLink to="/categories">Категории</NavLink>
          </nav>
        </div>
      </header>
      <main className="container">
        <Routes>
          <Route path="/" element={<Navigate to="/tickets" replace />} />
          <Route path="/tickets" element={<TicketsPage />} />
          <Route path="/tickets/:id" element={<TicketPage />} />
          <Route path="/equipment" element={<EquipmentPage />} />
          <Route path="/categories" element={<CategoriesPage />} />
          <Route path="*" element={<p className="empty">Страница не найдена</p>} />
        </Routes>
      </main>
    </BrowserRouter>
  )
}
