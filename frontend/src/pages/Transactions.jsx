import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import TransactionForm from '../components/TransactionForm';
import { useToast } from '../toast/ToastContext';
import { formatMoney } from '../util/money';

// Transactions (FR3): filterable newest-first list with full CRUD.
export default function Transactions() {
  const toast = useToast();
  const [categories, setCategories] = useState([]);
  const [rows, setRows] = useState(null);
  const [type, setType] = useState('');
  const [categoryId, setCategoryId] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [form, setForm] = useState(null); // null | {mode:'create'} | {mode:'edit', tx}

  const load = useCallback(
    async (f) => {
      const params = new URLSearchParams();
      if (f.type) params.set('type', f.type);
      if (f.categoryId) params.set('category_id', f.categoryId);
      if (f.from) params.set('from', f.from);
      if (f.to) params.set('to', f.to);
      try {
        const [list, cats] = await Promise.all([
          api(`/transactions?${params.toString()}`),
          api('/categories'),
        ]);
        setRows(list);
        setCategories(cats);
      } catch (err) {
        toast.error(err.message);
      }
    },
    [toast],
  );

  useEffect(() => {
    load({ type, categoryId, from, to });
  }, [load, type, categoryId, from, to]);

  async function deleteTx(id) {
    if (!window.confirm('Delete this transaction?')) return;
    try {
      await api(`/transactions/${id}`, { method: 'DELETE' });
      toast.success('Transaction deleted');
      load({ type, categoryId, from, to });
    } catch (err) {
      toast.error(err.message);
    }
  }

  function done() {
    setForm(null);
    load({ type, categoryId, from, to });
  }

  return (
    <>
      <div className="page-head">
        <h1>Transactions</h1>
        <button className="btn btn-primary" onClick={() => setForm({ mode: 'create' })}>
          + Add transaction
        </button>
      </div>

      {form && (
        <TransactionForm
          categories={categories}
          initial={form.mode === 'edit' ? form.tx : null}
          onDone={done}
          onCancel={() => setForm(null)}
        />
      )}

      <div className="filter-bar card">
        <label>Type
          <select value={type} onChange={(e) => setType(e.target.value)}>
            <option value="">All types</option>
            <option value="INCOME">Income</option>
            <option value="EXPENSE">Expense</option>
          </select>
        </label>
        <label>Category
          <select value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
            <option value="">All categories</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>{c.name} ({c.type === 'INCOME' ? 'Income' : 'Expense'})</option>
            ))}
          </select>
        </label>
        <label>From
          <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
        </label>
        <label>To
          <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
        </label>
        {(type || categoryId || from || to) && (
          <button
            className="btn btn-ghost"
            onClick={() => { setType(''); setCategoryId(''); setFrom(''); setTo(''); }}
          >
            Clear filters
          </button>
        )}
      </div>

      <section className="card">
        {rows === null ? (
          <p className="muted">Loading…</p>
        ) : rows.length === 0 ? (
          <p className="muted">No transactions match. Add one or relax the filters.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Date</th><th>Type</th><th>Category</th><th>Description</th>
                <th className="num">Amount</th><th />
              </tr>
            </thead>
            <tbody>
              {rows.map((t) => (
                <tr key={t.id}>
                  <td>{t.date}</td>
                  <td><span className={`badge badge-${t.type === 'INCOME' ? 'income' : 'expense'}`}>
                    {t.type === 'INCOME' ? 'Income' : 'Expense'}
                  </span></td>
                  <td>{t.category_name}</td>
                  <td className="desc">{t.description || <span className="muted">—</span>}</td>
                  <td className={`num ${t.type === 'INCOME' ? 'amount-income' : 'amount-expense'}`}>
                    {t.type === 'INCOME' ? '+' : '−'}{formatMoney(t.amount)}
                  </td>
                  <td className="num">
                    <button className="btn btn-small" onClick={() => setForm({ mode: 'edit', tx: t })}>Edit</button>{' '}
                    <button className="btn btn-small btn-danger" onClick={() => deleteTx(t.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </>
  );
}
