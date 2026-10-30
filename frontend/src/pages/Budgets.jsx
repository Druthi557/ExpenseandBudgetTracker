import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import { useToast } from '../toast/ToastContext';
import { MONTH_NAMES, formatMoney, validateAmount } from '../util/money';

// Budgets (FR4): monthly budgets on expense categories with actual spend,
// remaining balance and clear overspend highlighting (AC7/AC10/AC11).
export default function Budgets() {
  const toast = useToast();
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [rows, setRows] = useState(null);
  const [categories, setCategories] = useState([]);
  const [categoryId, setCategoryId] = useState('');
  const [amount, setAmount] = useState('');
  const [budgetYear, setBudgetYear] = useState(now.getFullYear());
  const [budgetMonth, setBudgetMonth] = useState(now.getMonth() + 1);
  const [errors, setErrors] = useState({});
  const [editing, setEditing] = useState(null); // {id, amount}
  const [editError, setEditError] = useState(null);

  const load = useCallback(
    async (y, m) => {
      try {
        const [list, cats] = await Promise.all([
          api(`/budgets?year=${y}&month=${m}`),
          api('/categories?type=EXPENSE'),
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
    load(year, month);
  }, [load, year, month]);

  async function onCreate(e) {
    e.preventDefault();
    const errs = {
      categoryId: categoryId ? null : 'Choose a category',
      amount: validateAmount(amount),
    };
    setErrors(errs);
    if (Object.values(errs).some(Boolean)) return;
    try {
      await api('/budgets', {
        method: 'POST',
        body: { category_id: Number(categoryId), year: budgetYear, month: budgetMonth, amount },
      });
      toast.success('Budget created');
      setAmount('');
      setCategoryId('');
      setYear(budgetYear);
      setMonth(budgetMonth);
      // Explicit refetch (AC9): when the created budget targets the month
      // already on screen, no state changes, so the effect alone won't reload.
      load(budgetYear, budgetMonth);
    } catch (err) {
      toast.error(err.message);
    }
  }

  async function onSaveAmount(id) {
    const err = validateAmount(editing.amount);
    setEditError(err);
    if (err) return;
    try {
      await api(`/budgets/${id}`, { method: 'PUT', body: { amount: editing.amount } });
      toast.success('Budget updated');
      setEditing(null);
      load(year, month);
    } catch (err) {
      toast.error(err.message);
    }
  }

  async function onDelete(id) {
    if (!window.confirm('Delete this budget?')) return;
    try {
      await api(`/budgets/${id}`, { method: 'DELETE' });
      toast.success('Budget deleted');
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
        <h1>Budgets</h1>
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

      <form className="card form-grid" onSubmit={onCreate} noValidate>
        <div className="form-row">
          <label>Category (expense only)
            <select value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
              <option value="">Select an expense category…</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
            {errors.categoryId && <span className="field-error">{errors.categoryId}</span>}
          </label>
          <label>Month
            <select value={budgetMonth} onChange={(e) => setBudgetMonth(Number(e.target.value))}>
              {MONTH_NAMES.map((name, i) => (
                <option key={name} value={i + 1}>{name}</option>
              ))}
            </select>
          </label>
          <label>Year
            <select value={budgetYear} onChange={(e) => setBudgetYear(Number(e.target.value))}>
              {years.map((y) => (
                <option key={y} value={y}>{y}</option>
              ))}
            </select>
          </label>
        </div>
        <div className="form-row">
          <label>Budget amount (₹)
            <input
              type="text"
              inputMode="decimal"
              placeholder="e.g. 5000.00"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
            {errors.amount && <span className="field-error">{errors.amount}</span>}
          </label>
          <div className="form-actions align-end">
            <button type="submit" className="btn btn-primary">Set budget</button>
          </div>
        </div>
      </form>

      <section className="card">
        <div className="card-head">
          <h2>{MONTH_NAMES[month - 1]} {year}</h2>
        </div>
        {rows === null ? (
          <p className="muted">Loading…</p>
        ) : rows.length === 0 ? (
          <p className="muted">No budgets for {MONTH_NAMES[month - 1]} {year} yet.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Category</th><th className="num">Budget</th><th className="num">Actual</th>
                <th className="num">Remaining</th><th>Status</th><th className="num">Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((b) =>
                editing && editing.id === b.id ? (
                  <tr key={b.id} className={b.over_budget ? 'row-over' : ''}>
                    <td>{b.category_name}</td>
                    <td colSpan={3}>
                      <input
                        type="text"
                        inputMode="decimal"
                        value={editing.amount}
                        onChange={(e) => setEditing({ ...editing, amount: e.target.value })}
                      />
                      {editError && <span className="field-error">{editError}</span>}
                    </td>
                    <td />
                    <td className="num">
                      <button className="btn btn-small btn-primary" onClick={() => onSaveAmount(b.id)}>Save</button>{' '}
                      <button className="btn btn-small" onClick={() => setEditing(null)}>Cancel</button>
                    </td>
                  </tr>
                ) : (
                  <tr key={b.id} className={b.over_budget ? 'row-over' : ''}>
                    <td>{b.category_name}</td>
                    <td className="num">{formatMoney(b.amount)}</td>
                    <td className="num">{formatMoney(b.actual_amount)}</td>
                    <td className={`num ${b.remaining.startsWith('-') ? 'amount-expense' : ''}`}>
                      {formatMoney(b.remaining)}
                    </td>
                    <td>
                      {b.over_budget
                        ? <span className="badge badge-over">Over budget</span>
                        : <span className="badge badge-ok">On track</span>}
                    </td>
                    <td className="num">
                      <button
                        className="btn btn-small"
                        onClick={() => { setEditing({ id: b.id, amount: b.amount }); setEditError(null); }}
                      >
                        Edit
                      </button>{' '}
                      <button className="btn btn-small btn-danger" onClick={() => onDelete(b.id)}>Delete</button>
                    </td>
                  </tr>
                ),
              )}
            </tbody>
          </table>
        )}
      </section>
    </>
  );
}
