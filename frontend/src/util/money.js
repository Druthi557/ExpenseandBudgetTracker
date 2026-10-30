// Money display and client-side validation (AC3/AC4/AC13): amounts are
// strings end-to-end and never parsed as floats.
const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];

export { MONTH_NAMES };

// formatMoney renders a 2-dp money string with the ₹ symbol (D11):
// "-1234.5" → "-₹1,234.50".
export function formatMoney(s) {
  if (typeof s !== 'string') return '₹—';
  const neg = s.startsWith('-');
  const [intRaw, decRaw] = (neg ? s.slice(1) : s).split('.');
  const int = intRaw || '0';
  const dec = ((decRaw || '') + '00').slice(0, 2);
  const grouped = int.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return `${neg ? '-' : ''}₹${grouped}.${dec}`;
}

// validateAmount checks a raw amount string without floats: digits only, at
// most 2 decimal places, greater than zero, at most 9,999,999,999.99.
// Returns an error message or null when valid.
export function validateAmount(raw) {
  const s = String(raw).trim();
  if (!s) return 'Amount is required';
  if (!/^\d+(\.\d+)?$/.test(s)) return 'Amount must be a positive number (digits only)';
  const m = s.match(/^(\d+)(?:\.(\d+))?$/);
  if (m[2] && m[2].length > 2) return 'Amount can have at most 2 decimal places';
  if (m[1].length > 10) return 'Amount is too large (maximum 9999999999.99)';
  if (!/[1-9]/.test(s.replace('.', ''))) return 'Amount must be greater than zero';
  return null;
}

export function todayISO() {
  const d = new Date();
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${mm}-${dd}`;
}
