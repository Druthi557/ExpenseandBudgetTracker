import { NavLink, Navigate, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

// Route guard + app chrome: unauthenticated visitors are redirected to the
// login page (FR1.5); authenticated users get the navigation shell.
export default function AuthedLayout() {
  const { authed, ready, logout } = useAuth();
  const navigate = useNavigate();

  if (!ready) {
    return <div className="page page-narrow"><p className="muted">Loading…</p></div>;
  }
  if (!authed) {
    return <Navigate to="/login" replace />;
  }

  async function onLogout() {
    await logout();
    navigate('/login');
  }

  return (
    <div className="shell">
      <header className="topbar">
        <span className="brand">Expense Tracker</span>
        <nav className="mainnav">
          <NavLink to="/" end>Dashboard</NavLink>
          <NavLink to="/transactions">Transactions</NavLink>
          <NavLink to="/categories">Categories</NavLink>
          <NavLink to="/budgets">Budgets</NavLink>
        </nav>
        <button className="btn btn-ghost" onClick={onLogout}>Log out</button>
      </header>
      <main className="page">
        <Outlet />
      </main>
    </div>
  );
}
