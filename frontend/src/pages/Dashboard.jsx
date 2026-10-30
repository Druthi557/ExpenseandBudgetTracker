import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import TransactionForm from '../components/TransactionForm';
import { useToast } from '../toast/ToastContext';
import { MONTH_NAMES, formatMoney } from '../util/money';

// Dashboard (FR5): total income, expenses, remaining balance, expense totals
// by category and the five most recent transactions, for a selected month.
// Refetched live after every add/delete so numbers are always current (AC9).
export default function Dashboard() {
  const toast = useToast();
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [data, setData] = useState(null);
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);

  const load = useCallback(
    async (y, m) => {
      setLoading(true);
      try {
        const [dash, cats] = await Promise.all([
          api(`/dashboard?year=${y}&month=${m}`),
          api('/categories'),
        ]);
        setData(dash);
        setCategories(cats);
      } catch (err) {
        toast.error(err.message);
      } finally {
        setLoading(false);
      }
    },
    [toast],
  );

  useEffect(() => {
    load(year, month);
  }, [year, month, load]);

  async function deleteTx(id) {
    if (!window.confirm('Delete this transaction?')) return;
    try {
      await api(`/transactions/${id}`, { method: 'DELETE' });
      toast.success('Transaction deleted');
      load(year, month);
    } catch (err) {
      toast.error(err.message);
    }
  }

  const years = [];
  for (let y = now.getFullYear() - 4; y <= now.getFullYear() + 1; y++) years.push(y);

  return (
    <>
      <div className="page-head">
        <h1>Dashboard</h1>
        <div className="month-pickers">
          <select value={month} onChange={(e) => setMonth(Number(e.target.value))} aria-label="Month">
            {MONTH_NAMES.map((name, i) => (
              <option key={name} value={i + 1}>{name}</option>
            ))}
          </select>
          <select value={year} onChange={(e) => setYear(Number(e.target.value))} aria-label="Year">
            {years.map((y) => (
              <option key={y} value={y}>{y}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="stat-grid">
        <div className="stat stat-income">
          <span className="stat-label">Total income</span>
          <span className="stat-value">{formatMoney(data?.total_income)}</span>
        </div>
        <div className="stat stat-expense">
          <span className="stat-label">Total expenses</span>
          <span className="stat-value">{formatMoney(data?.total_expense)}</span>
        </div>
        <div className={`stat ${data && data.remaining_balance.startsWith('-') ? 'stat-negative' : 'stat-balance'}`}>
          <span className="stat-label">Remaining balance</span>
          <span className="stat-value">{formatMoney(data?.remaining_balance)}</span>
        </div>
      </div>

      <div className="two-col">
        <section className="card">
          <div className="card-head">
            <h2>Expenses by category</h2>
            <span className="muted">{MONTH_NAMES[month - 1]} {year}</span>
          </div>
          {data && data.expenses_by_category.length > 0 ? (
            <table>
              <thead>
                <tr><th>Category</th><th className="num">Total</th></tr>
              </thead>
              <tbody>
                {data.expenses_by_category.map((c) => (
                  <tr key={c.category_id}>
                    <td>{c.category_name}</td>
                    <td className="num">{formatMoney(c.total)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="muted">No expenses recorded in {MONTH_NAMES[month - 1]} {year}.</p>
          )}
        </section>

        <section className="card">
          <div className="card-head">
            <h2>Recent transactions</h2>
            <button className="btn btn-small" onClick={() => setShowForm((v) => !v)}>
              {showForm ? 'Close' : '+ Add'}
            </button>
          </div>
          {showForm && (
            <TransactionForm
              categories={categories}
              onDone={() => {
                setShowForm(false);
                load(year, month);
              }}
              onCancel={() => setShowForm(false)}
            />
          )}
          {data && data.recent_transactions.length > 0 ? (
            <ul className="tx-list">
              {data.recent_transactions.map((t) => (
                <li key={t.id}>
                  <div className="tx-main">
                    <span className="tx-cat">{t.category_name}</span>
                    <span className="muted">{t.date}{t.description ? ` · ${t.description}` : ''}</span>
                  </div>
                  <div className="tx-side">
                    <span className={t.type === 'INCOME' ? 'amount-income' : 'amount-expense'}>
                      {t.type === 'INCOME' ? '+' : '−'}{formatMoney(t.amount)}
                    </span>
                    <button
                      className="icon-btn"
                      onClick={() => deleteTx(t.id)}
                      aria-label={`Delete transaction from ${t.date}`}
                    >
                      🗑
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <p className="muted">No transactions yet.</p>
          )}
        </section>
      </div>
      {loading && <p className="muted">Refreshing…</p>}
    </>
  );
}
