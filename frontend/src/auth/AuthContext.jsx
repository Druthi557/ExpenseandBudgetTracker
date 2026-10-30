import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { api } from '../api/client';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [authed, setAuthed] = useState(false);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    // D32: no /auth/me endpoint — login state is probed with a cheap
    // authenticated call.
    api('/categories?type=INCOME')
      .then(() => setAuthed(true))
      .catch(() => setAuthed(false))
      .finally(() => setReady(true));
  }, []);

  const login = useCallback(async (email, password) => {
    await api('/auth/login', { method: 'POST', body: { email, password } });
    setAuthed(true);
  }, []);

  const register = useCallback(async (email, password) => {
    await api('/auth/register', { method: 'POST', body: { email, password } });
    setAuthed(true);
  }, []);

  const logout = useCallback(async () => {
    try {
      await api('/auth/logout', { method: 'POST' });
    } finally {
      setAuthed(false);
    }
  }, []);

  return (
    <AuthContext.Provider value={{ authed, ready, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
