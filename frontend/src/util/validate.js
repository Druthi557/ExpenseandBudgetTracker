// Client-side form validation (AC3) — mirrors the backend rules (D8 etc.).
export function validateEmail(raw) {
  const email = String(raw).trim().toLowerCase();
  if (!email) return 'Email is required';
  if (email.length > 255) return 'Email must be at most 255 characters';
  if (!/^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$/.test(email)) {
    return 'Enter a valid email address';
  }
  return null;
}

// Full registration password policy (D8).
export function validatePassword(raw) {
  const pw = String(raw);
  if (pw.length < 8) return 'Password must be at least 8 characters';
  if (pw.length > 72) return 'Password must be at most 72 characters';
  if (!/[a-zA-Z]/.test(pw) || !/\d/.test(pw)) {
    return 'Password must contain at least one letter and one digit';
  }
  return null;
}

export function validateCategoryName(raw) {
  const name = String(raw).trim();
  if (!name) return 'Category name is required';
  if (name.length > 100) return 'Category name must be at most 100 characters';
  return null;
}

export function validateDescription(raw) {
  if (String(raw).length > 500) return 'Description must be at most 500 characters';
  return null;
}

// Login only requires presence/format, not the registration policy.
export function validateLoginPassword(raw) {
  if (!String(raw)) return 'Password is required';
  return null;
}
