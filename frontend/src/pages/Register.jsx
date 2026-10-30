import { useState } from 'react';
import { Link, Navigate, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { useToast } from '../toast/ToastContext';
import { validateEmail, validatePassword } from '../util/validate';

function PolicyHint({ ok, children }) {
  return <li className={ok ? 'hint-ok' : ''}>{ok ? '✓' : '•'} {children}</li>;
}

export default function Register() {
  const { authed, ready, register } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState({});
  const [busy, setBusy] = useState(false);

  if (ready && authed) return <Navigate to="/" replace />;

  async function onSubmit(e) {
    e.preventDefault();
    const errs = { email: validateEmail(email), password: validatePassword(password) };
    setErrors(errs);
    if (Object.values(errs).some(Boolean)) return;
    setBusy(true);
    try {
      await register(email, password);
      toast.success('Account created — you are signed in');
      navigate('/');
    } catch (err) {
      toast.error(err.message);
    } finally {
      setBusy(false);
    }
  }

  const pw = password;
  return (
    <div className="auth-wrap">
      <div className="auth-card">
        <h1>Create account</h1>
        <form onSubmit={onSubmit} noValidate>
          <label>Email
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              autoFocus
            />
            {errors.email && <span className="field-error">{errors.email}</span>}
          </label>
          <label>Password
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
            />
            <ul className="policy-hints">
              <PolicyHint ok={pw.length >= 8}>At least 8 characters</PolicyHint>
              <PolicyHint ok={/[a-zA-Z]/.test(pw)}>Contains a letter</PolicyHint>
              <PolicyHint ok={/\d/.test(pw)}>Contains a digit</PolicyHint>
            </ul>
            {errors.password && <span className="field-error">{errors.password}</span>}
          </label>
          <button type="submit" className="btn btn-primary btn-block" disabled={busy}>
            {busy ? 'Creating…' : 'Create account'}
          </button>
        </form>
        <p className="muted">Already registered? <Link to="/login">Sign in</Link></p>
      </div>
    </div>
  );
}
