import { useState } from 'react';
import { api } from '../api/client';
import { useToast } from '../toast/ToastContext';
import { todayISO, validateAmount } from '../util/money';
import { validateDescription } from '../util/validate';

// Shared transaction create/edit form (FR3.1–FR3.3). The category dropdown is
// pre-filtered by transaction type (AC6), amounts are validated as strings
// without floats (AC4/AC13), and dates come from a date input (AC5).
export default function TransactionForm({ categories, initial = null, onDone, onCancel }) {
  const toast = useToast();
  const [type, setType] = useState(initial?.type || 'EXPENSE');
  const [categoryId, setCategoryId] = useState(initial ? String(initial.category_id) : '');
  const [amount, setAmount] = useState(initial ? initial.amount : '');
  const [date, setDate] = useState(initial ? initial.date : todayISO());
  const [description, setDescription] = useState(initial?.description || '');
  const [errors, setErrors] = useState({});
  const [busy, setBusy] = useState(false);

  const filtered = categories.filter((c) => c.type === type);

  async function onSubmit(e) {
    e.preventDefault();
    const errs = {
      categoryId: filtered.some((c) => String(c.id) === categoryId)
        ? null
        : 'Choose a category',
      amount: validateAmount(amount),
      date: date ? null : 'Date is required',
      description: validateDescription(description),
    };
    setErrors(errs);
    if (Object.values(errs).some(Boolean)) return;

    setBusy(true);
    try {
      const body = {
        type,
        category_id: Number(categoryId),
        amount,
        date,
        description: description.trim(),
      };
      const saved = initial
        ? await api(`/transactions/${initial.id}`, { method: 'PUT', body })
        : await api('/transactions', { method: 'POST', body });
      toast.success(initial ? 'Transaction updated' : 'Transaction added');
      onDone(saved);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="card form-grid" onSubmit={onSubmit} noValidate>
      <div className="form-row">
        <label>Type
          <div className="radio-row">
            <label className="radio">
              <input type="radio" name="txtype" checked={type === 'EXPENSE'} onChange={() => setType('EXPENSE')} />
              Expense
            </label>
            <label className="radio">
              <input type="radio" name="txtype" checked={type === 'INCOME'} onChange={() => setType('INCOME')} />
              Income
            </label>
          </div>
        </label>
        <label>Category
          <select value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
            <option value="">Select a category…</option>
            {filtered.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
          {errors.categoryId && <span className="field-error">{errors.categoryId}</span>}
        </label>
      </div>
      <div className="form-row">
        <label>Amount (₹)
          <input
            type="text"
            inputMode="decimal"
            placeholder="e.g. 250.00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          {errors.amount && <span className="field-error">{errors.amount}</span>}
        </label>
        <label>Date
          <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          {errors.date && <span className="field-error">{errors.date}</span>}
        </label>
      </div>
      <label>Description (optional)
        <input
          type="text"
          maxLength={500}
          placeholder="e.g. Groceries at the market"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        {errors.description && <span className="field-error">{errors.description}</span>}
      </label>
      <div className="form-actions">
        <button type="submit" className="btn btn-primary" disabled={busy}>
          {busy ? 'Saving…' : initial ? 'Save changes' : 'Add transaction'}
        </button>
        <button type="button" className="btn btn-ghost" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}
