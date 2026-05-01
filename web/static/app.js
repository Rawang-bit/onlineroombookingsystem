const API_BASE = '/api';

async function api(path, options = {}) {
  const token = localStorage.getItem('adminToken');
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  if (token) headers.Authorization = 'Bearer ' + token;
  const res = await fetch(API_BASE + path, { headers, ...options });
  let data = null;
  try { data = await res.json(); } catch (_) {}
  if (!res.ok) throw new Error((data && data.error) || 'Request failed');
  return data;
}

function adminLoggedIn() { return localStorage.getItem('adminLoggedIn') === 'true' && !!localStorage.getItem('adminToken'); }
function requireAuth() {
  const page = window.location.pathname.split('/').pop() || 'index.html';
  const adminPages = ['dashboard.html', 'rooms.html', 'users.html', 'bookings.html', 'book-room.html'];
  if (adminPages.includes(page) && !adminLoggedIn()) window.location.href = 'login.html';
}
function logout() { localStorage.removeItem('adminLoggedIn'); localStorage.removeItem('adminName'); localStorage.removeItem('adminToken'); window.location.href = 'login.html'; }

async function loginAdmin(username, password) {
  const result = await api('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) });
  localStorage.setItem('adminLoggedIn', 'true');
  localStorage.setItem('adminName', result.admin.name);
  localStorage.setItem('adminToken', result.token);
  return result;
}

async function getRooms() { return api('/rooms'); }
async function createRoom(room) { return api('/rooms', { method: 'POST', body: JSON.stringify(room) }); }
async function updateRoom(id, room) { return api('/rooms/' + id, { method: 'PUT', body: JSON.stringify(room) }); }
async function deleteRoomApi(id) { return api('/rooms/' + id, { method: 'DELETE' }); }

async function getBookings() { return api('/bookings'); }
async function createBooking(booking) { return api('/bookings', { method: 'POST', body: JSON.stringify(booking) }); }
async function updateBooking(id, booking) { return api('/bookings/' + id, { method: 'PUT', body: JSON.stringify(booking) }); }
async function cancelBookingApi(id) { return api('/bookings/' + id, { method: 'DELETE' }); }
async function deleteBookingApi(id) { return api('/bookings/' + id + '?hard=1', { method: 'DELETE' }); }

function formatStatus(status) {
  const cls = status === 'Booked' || status === 'Active' ? 'badge-success' : status === 'Cancelled' || status === 'Inactive' ? 'badge-danger' : 'badge-neutral';
  return `<span class="badge ${cls}">${escapeHtml(status)}</span>`;
}
function showMessage(id, text, type='success') {
  const el = document.getElementById(id);
  if (!el) return;
  el.className = `notice ${type === 'success' ? 'notice-success' : 'notice-error'}`;
  el.textContent = text;
  el.style.display = 'block';
  setTimeout(() => { el.style.display = 'none'; }, 3000);
}
function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}
function todayISO() { return new Date().toISOString().slice(0, 10); }
function hasConflictInList(bookings, roomId, date, start, end, excludeId = null) {
  return bookings.some(b =>
    Number(b.roomId) === Number(roomId) &&
    b.date === date &&
    b.status !== 'Cancelled' &&
    Number(b.id) !== Number(excludeId) &&
    start < b.end && end > b.start
  );
}

async function getUsers() { return api('/users'); }
async function createUser(user) { return api('/users', { method: 'POST', body: JSON.stringify(user) }); }
async function updateUser(id, user) { return api('/users/' + id, { method: 'PUT', body: JSON.stringify(user) }); }
async function deleteUserApi(id) { return api('/users/' + id, { method: 'DELETE' }); }
