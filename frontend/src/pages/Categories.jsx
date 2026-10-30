import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import { useToast } from '../toast/ToastContext';
import { validateCategoryName } from '../util/validate';

// Categories (FR2): create, rename, retype (while unused), delete and view
// all with a type filter.
export default function Categories() {
  const toast = useToast();
  const [rows, setRows] = useState(null);
  const [filter, setFilter] = useState('');
  const [name, setName] = useState('');
  const [type, setType] = useState('EXPENSE');
  const [nameError, setNameError] = useState(null);
  const [editing, setEditing] = useState(null); // {id, name, type}
  const [editError, setEditError] = useState(null);

  const load = useCallback(
    async (f) => {
      try {
        setRows(await api(`/categories${f ? `?type=${f}` : ''}`));
      } catch (err) {
        toast.error(err.message);
      }
    },
    [toast],
  );

  useEffect(() => {
    load(filter);
  }, [load, filter]);

  async function onCreate(e) {
    e.preventDefault();
    const err = validateCategoryName(name);
    setNameError(err);
    if (err) return;
    try {
      await api('/categories', { method: 'POST', body: { name: name.trim(), type } });
      toast.success('Category created');
      setName('');
      load(filter);
    } catch (err) {
      toast.error(err.message);
    }
  }

  async function onSaveEdit(e) {
    e.preventDefault();
    const err = validateCategoryName(editing.name);
    setEditError(err);
    if (err) return;
    try {
      await api(`/categories/${editing.id}`, {
        method: 'PUT',
        body: { name: editing.name.trim(), type: editing.type },
      });
      toast.success('Category updated');
      setEditing(null);
      load(filter);
    } catch (err) {
      toast.error(err.message);
    }
  }

  async function onDelete(id) {
    if (!window.confirm('Delete this category?')) return;
    try {
      await api(`/categories/${id}`, { method: 'DELETE' });
      toast.success('Category deleted');
      load(filter);
    } catch (err) {
      // 409 CATEGORY_IN_USE (AC12) arrives here as a friendly message.
      toast.error(err.message);
    }
  }

  return (
    <>
      <div className="page-head">
        <h1>Categories</h1>
        <div className="month-pickers">
          <select value={filter} onChange={(e) => setFilter(e.target.value)} aria-label="Filter by type">
            <option value="">All types</option>
            <option value="INCOME">Income</option>
            <option value="EXPENSE">Expense</option>
          </select>
        </div>
      </div>

      <form className="card form-grid" onSubmit={onCreate} noValidate>
        <div className="form-row">
          <label>Name
            <input
              type="text"
              maxLength={100}
              placeholder="e.g. Food, Salary"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            {nameError && <span className="field-error">{nameError}</span>}
          </label>
          <label>Type
            <div className="radio-row">
              <label className="radio">
                <input type="radio" name="newcattype" checked={type === 'EXPENSE'} onChange={() => setType('EXPENSE')} />
                Expense
              </label>
              <label className="radio">
                <input type="radio" name="newcattype" checked={type === 'INCOME'} onChange={() => setType('INCOME')} />
                Income
              </label>
            </div>
          </label>
          <div className="form-actions align-end">
            <button type="submit" className="btn btn-primary">Add category</button>
          </div>
        </div>
      </form>

      <section className="card">
        {rows === null ? (
          <p className="muted">Loading…</p>
        ) : rows.length === 0 ? (
          <p className="muted">No categories yet — add your first above.</p>
        ) : (
          <table>
            <thead>
              <tr><th>Name</th><th>Type</th><th className="num">Actions</th></tr>
            </thead>
            <tbody>
              {rows.map((c) =>
                editing && editing.id === c.id ? (
                  <tr key={c.id}>
                    <td>
                      <input
                        type="text"
                        maxLength={100}
                        value={editing.name}
                        onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                      />
                      {editError && <span className="field-error">{editError}</span>}
                    </td>
                    <td>
                      <select
                        value={editing.type}
                        onChange={(e) => setEditing({ ...editing, type: e.target.value })}
                      >
                        <option value="EXPENSE">Expense</option>
                        <option value="INCOME">Income</option>
                      </select>
                    </td>
                    <td className="num">
                      <button className="btn btn-small btn-primary" onClick={onSaveEdit}>Save</button>{' '}
                      <button className="btn btn-small" onClick={() => setEditing(null)}>Cancel</button>
                    </td>
                  </tr>
                ) : (
                  <tr key={c.id}>
                    <td>{c.name}</td>
                    <td>
                      <span className={`badge badge-${c.type === 'INCOME' ? 'income' : 'expense'}`}>
                        {c.type === 'INCOME' ? 'Income' : 'Expense'}
                      </span>
                    </td>
                    <td className="num">
                      <button
                        className="btn btn-small"
                        onClick={() => { setEditing({ id: c.id, name: c.name, type: c.type }); setEditError(null); }}
                      >
                        Edit
                      </button>{' '}
                      <button className="btn btn-small btn-danger" onClick={() => onDelete(c.id)}>Delete</button>
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
