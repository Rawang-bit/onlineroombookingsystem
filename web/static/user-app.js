
let localUsers = [];
let rooms = [];

let state = {
  view: 'weekly', room: '', roomId: null, location: '', bookings: [], pending: null,
  activeUser: null, currentBookingId: null, cancelVerified: false, baseDate: new Date(), currentWeekDates: []
};

const hours = [
  '09:00 AM','09:30 AM','10:00 AM','10:30 AM','11:00 AM','11:30 AM','12:00 PM','12:30 PM',
  '01:00 PM','01:30 PM','02:00 PM','02:30 PM','03:00 PM','03:30 PM','04:00 PM','04:30 PM',
  '05:00 PM','05:30 PM','06:00 PM','06:30 PM','07:00 PM'
];

async function api(path, options = {}) {
  const res = await fetch(path, { headers: { 'Content-Type': 'application/json', ...(options.headers || {}) }, ...options });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || 'Request failed');
  return data;
}

function updateHeaderClock() {
  const now = new Date();
  document.getElementById('clock').innerText = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  document.getElementById('date-label').innerText = now.toDateString().toUpperCase();
}

async function loadRooms() {
  const data = await api('/api/rooms');
  rooms = data.filter(r => r.status === 'Active').map(r => ({ id: r.id, name: r.name, location: r.location, capacity: r.capacity }));
  if (!rooms.length) {
    showMessageModal('No Rooms Available', 'No active meeting rooms are available. Please ask admin to add rooms.', 'circle-alert');
    return;
  }
  if (!state.roomId) {
    state.roomId = rooms[0].id;
    state.room = rooms[0].name;
    state.location = rooms[0].location;
  }
}

async function loadUsers() { localUsers = []; }

function normalizeTimeDisplay(value) {
  if (!value) return '';
  if (value.includes('AM') || value.includes('PM')) return value;
  const [h, m] = value.split(':').map(Number);
  const d = new Date();
  d.setHours(h, m, 0, 0);
  return d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true });
}

async function loadBookings() {
  const data = await api('/api/bookings');
  state.bookings = data.filter(b => b.status !== 'Cancelled').map(b => ({
    id: b.id,
    roomId: b.roomId,
    room: b.roomName || b.room,
    location: b.location || '',
    date: new Date(b.date + 'T00:00:00').toDateString(),
    startTime: normalizeTimeDisplay(b.startTime || b.start),
    endTime: normalizeTimeDisplay(b.endTime || b.end),
    user: b.user,
    email: b.email,
    purpose: b.purpose,
    status: b.status
  }));
}

function toISODate(dateStr) {
  const d = new Date(dateStr);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function populateRoomSwitcher() {
  const roomSwitcher = document.getElementById('roomSwitcher');
  roomSwitcher.innerHTML = rooms.map(room => `<option value="${room.id}" ${Number(room.id) === Number(state.roomId) ? 'selected' : ''}>${room.name} - ${room.location}</option>`).join('');
}

function getMonday(date) {
  const current = new Date(date);
  const dayOfWeek = current.getDay();
  const diff = current.getDate() - dayOfWeek + (dayOfWeek === 0 ? -6 : 1);
  return new Date(new Date(current).setDate(diff));
}

function calculateDates() {
  const current = new Date(state.baseDate);
  const monday = getMonday(current);
  state.currentWeekDates = [];
  for (let i = 0; i < 5; i++) {
    const d = new Date(monday);
    d.setDate(monday.getDate() + i);
    state.currentWeekDates.push({
      short: d.toLocaleDateString('en-US', { weekday: 'short' }),
      day: d.getDate(),
      full: d.toDateString(),
      isToday: d.toDateString() === new Date().toDateString()
    });
  }
  document.getElementById('calendarHeaderDate').innerText = current.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
}

function navigate(direction) {
  if (direction === 0) state.baseDate = new Date();
  else {
    state.baseDate = new Date(state.baseDate);
    state.baseDate.setDate(state.baseDate.getDate() + direction * (state.view === 'weekly' ? 7 : 1));
  }
  calculateDates();
  renderCalendar();
}

function getTimeIndex(timeStr) { return hours.indexOf(timeStr); }
function getMinutesFromTime(timeStr) {
  const [time, modifier] = timeStr.split(' ');
  let [hour, minute] = time.split(':').map(Number);
  if (hour === 12) hour = 0;
  if (modifier === 'PM') hour += 12;
  return hour * 60 + minute;
}
function formatDuration(startTime, endTime) {
  const minutes = getMinutesFromTime(endTime) - getMinutesFromTime(startTime);
  const hrs = Math.floor(minutes / 60);
  const mins = minutes % 60;
  if (hrs > 0 && mins > 0) return `${hrs}h ${mins}m`;
  if (hrs > 0) return `${hrs}h`;
  return `${mins}m`;
}
function getRemainingMinutes(dateStr, endTimeStr) {
  const now = new Date();
  if (dateStr !== now.toDateString()) return null;
  const end = new Date();
  const mins = getMinutesFromTime(endTimeStr);
  end.setHours(Math.floor(mins / 60), mins % 60, 0, 0);
  const diff = Math.floor((end - now) / 60000);
  return diff > 0 && diff < 480 ? diff : null;
}
function isPastSlot(dateStr, timeStr) {
  const now = new Date();
  const slotDate = new Date(dateStr);
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const slotDay = new Date(slotDate.getFullYear(), slotDate.getMonth(), slotDate.getDate());
  if (slotDay < today) return true;
  if (slotDay > today) return false;
  return getMinutesFromTime(timeStr) <= now.getHours() * 60 + now.getMinutes();
}
function getRoomBookings() {
  return state.bookings.filter(b => Number(b.roomId) === Number(state.roomId));
}
function hasBookingConflict(roomId, date, startTime, endTime, excludeId = null) {
  const newStart = getMinutesFromTime(startTime);
  const newEnd = getMinutesFromTime(endTime);
  return state.bookings.some(booking => {
    if (Number(booking.roomId) !== Number(roomId)) return false;
    if (booking.date !== date) return false;
    if (excludeId && booking.id === excludeId) return false;
    return newStart < getMinutesFromTime(booking.endTime) && newEnd > getMinutesFromTime(booking.startTime);
  });
}
function getAvailableEndTimes(roomId, date, startTime) {
  const endTimes = [];
  const startIdx = getTimeIndex(startTime);
  for (let i = startIdx + 1; i < hours.length; i++) {
    const candidateEndTime = hours[i];
    if (hasBookingConflict(roomId, date, startTime, candidateEndTime)) break;
    endTimes.push(candidateEndTime);
  }
  return endTimes;
}

function clearBookingValidation() {
  const el = document.getElementById('bookingValidationMessage');
  el.innerText = '';
  el.classList.add('hidden');
}
function showBookingValidation(message) {
  const el = document.getElementById('bookingValidationMessage');
  el.innerText = message;
  el.classList.remove('hidden');
}
function showMessageModal(title, text, icon = 'info') {
  document.getElementById('messageTitle').innerText = title;
  document.getElementById('messageText').innerText = text;
  const wrap = document.getElementById('messageIconWrap');
  const iconEl = document.getElementById('messageIcon');
  wrap.className = 'w-16 h-16 rounded-2xl flex items-center justify-center mx-auto mb-4';
  if (icon === 'circle-alert') wrap.classList.add('bg-amber-50', 'text-amber-600');
  else if (icon === 'shield-alert') wrap.classList.add('bg-red-50', 'text-red-600');
  else wrap.classList.add('bg-indigo-50', 'text-indigo-600');
  iconEl.setAttribute('data-lucide', icon);
  showStep('message');
}
function resetBookingFormFields() {
  document.getElementById('verifyEmail').value = '';
  document.getElementById('meetingPurpose').value = '';
  document.getElementById('endTimeSelect').innerHTML = '';
  clearBookingValidation();
}
function resetTransientState() {
  state.pending = null;
  state.activeUser = null;
  state.currentBookingId = null;
  state.cancelVerified = false;
}
function updateRoomHeader() {
  document.getElementById('calendarSubtext').innerText = state.room;
  document.getElementById('activeLocationDisplay').innerText = state.location;
  document.getElementById('roomSwitcher').value = state.roomId;
}
async function switchRoom(roomId) {
  const selectedRoom = rooms.find(r => Number(r.id) === Number(roomId));
  if (!selectedRoom) return;
  state.roomId = selectedRoom.id;
  state.room = selectedRoom.name;
  state.location = selectedRoom.location;
  updateRoomHeader();
  await loadBookings();
  renderCalendar();
}
function renderEmptyStateHint(container) {
  if (getRoomBookings().length > 0) return;
  container.insertAdjacentHTML('beforeend', `<div class="absolute bottom-6 right-6 bg-white/90 backdrop-blur border border-slate-200 rounded-2xl px-4 py-3 shadow-lg z-20"><p class="text-[10px] font-black uppercase tracking-widest text-slate-400">Tip</p><p class="text-sm font-bold text-slate-600">Hover a free slot to create a booking in ${state.room}</p></div>`);
}

function renderCalendar() {
  const container = document.getElementById('calendarContainer');
  const roomBookings = getRoomBookings();
  const displayDays = state.view === 'weekly' ? state.currentWeekDates : [{ short: state.baseDate.toLocaleDateString('en-US', { weekday: 'short' }), day: state.baseDate.getDate(), full: state.baseDate.toDateString(), isToday: state.baseDate.toDateString() === new Date().toDateString() }];
  let html = `<div class="relative"><table class="w-full border-collapse min-w-[1000px] relative"><thead><tr><th class="w-24 p-6 sticky-col sticky-header"></th>`;
  displayDays.forEach(dateObj => {
    html += `<th class="p-6 border-r border-slate-100 text-center sticky-header ${dateObj.isToday ? 'is-today-header' : 'bg-white'}"><span class="text-[10px] font-black ${dateObj.isToday ? 'text-indigo-600' : 'text-slate-400'} uppercase block mb-1 tracking-widest">${dateObj.short}</span><span class="${dateObj.isToday ? 'today-number font-black' : 'text-2xl font-black text-slate-700'}">${dateObj.day}</span></th>`;
  });
  html += `</tr></thead><tbody class="divide-y divide-slate-100 relative"><tr class="h-4"><td class="sticky-col"></td>${displayDays.map(() => `<td class="border-r border-slate-100"></td>`).join('')}</tr>`;
  hours.forEach((time, index) => {
    html += `<tr class="h-28"><td class="sticky-col relative"><span class="absolute top-0 right-4 translate-y-[-50%] text-[10px] font-bold text-slate-400 uppercase bg-[#f8fafc] px-1 z-10 tabular-nums">${time}</span></td>`;
    displayDays.forEach(dateObj => {
      const isBooked = roomBookings.some(b => b.date === dateObj.full && getTimeIndex(b.startTime) <= index && getTimeIndex(b.endTime) > index);
      const isPast = isPastSlot(dateObj.full, time);
      html += `<td class="p-2 border-r border-slate-100 relative group">${index < hours.length - 1 && !isBooked && !isPast ? `<button onclick="openBookingFlow('${dateObj.full}', '${time}')" class="opacity-0 group-hover:opacity-100 w-full h-full rounded-2xl border-2 border-dashed border-emerald-200 flex items-center justify-center text-emerald-600 transition-all"><i data-lucide="plus"></i></button>` : isPast && dateObj.full === new Date().toDateString() ? `<div class="w-full h-full rounded-2xl bg-slate-50/70"></div>` : ''}</td>`;
    });
    html += `</tr>`;
  });
  html += `</tbody></table>`;
  const tableWidth = container.clientWidth || 1000;
  const colWidth = (tableWidth - 96) / displayDays.length;
  roomBookings.forEach(booking => {
    const dayIndex = displayDays.findIndex(d => d.full === booking.date);
    if (dayIndex === -1) return;
    const startIdx = getTimeIndex(booking.startTime);
    const endIdx = getTimeIndex(booking.endTime);
    if (startIdx === -1 || endIdx === -1) return;
    const remaining = getRemainingMinutes(booking.date, booking.endTime);
    html += `<div onclick="showDetails(${booking.id})" class="event-card bg-indigo-50 border-indigo-600 animate-pop" style="top: ${120 + (startIdx * 112)}px; left: ${96 + (dayIndex * colWidth) + 8}px; width: ${colWidth - 16}px; height: ${((endIdx - startIdx) * 112) - 16}px;"><div class="flex justify-between items-start mb-1 gap-2"><p class="text-[10px] font-black text-indigo-400 uppercase">${booking.startTime}</p>${remaining ? `<span class="bg-indigo-600 text-white text-[8px] font-black px-1.5 py-0.5 rounded-md uppercase tracking-tighter whitespace-nowrap">${remaining}m left</span>` : ''}</div><p class="font-extrabold text-indigo-900 text-base leading-tight truncate">${booking.purpose}</p><p class="text-[11px] text-indigo-600 font-bold mt-2">${booking.user}</p></div>`;
  });
  container.innerHTML = html + `</div>`;
  renderEmptyStateHint(container);
  lucide.createIcons();
}

function switchView(v) {
  state.view = v;
  ['daily', 'weekly'].forEach(id => {
    const btn = document.getElementById('view-' + id);
    if (btn) btn.className = `px-5 py-2 rounded-xl text-sm font-bold transition-all ${v === id ? 'active-view' : 'text-slate-500'}`;
  });
  calculateDates();
  renderCalendar();
}
function openBookingFlow(date, time) {
  if (isPastSlot(date, time)) { showMessageModal('Invalid Time Slot', 'Past time slots cannot be booked.', 'circle-alert'); return; }
  state.pending = { date, time, room: state.room, roomId: state.roomId };
  clearBookingValidation();
  showStep('verify');
  document.getElementById('verifySubmitBtn').onclick = handleBookingVerify;
}
async function handleBookingVerify() {
  const email = document.getElementById('verifyEmail').value.toLowerCase().trim();
  if (!/.+@.+\..+/.test(email)) { showMessageModal('Invalid Email', 'Please enter a valid email address.', 'circle-alert'); return; }
  let user;
  try {
    user = await api('/api/public/users/validate?email=' + encodeURIComponent(email) + '&t=' + Date.now());
  } catch (err) {
    try {
      const registeredUsers = await api('/api/public/users?t=' + Date.now());
      user = registeredUsers.find(u => String(u.email || '').trim().toLowerCase() === email);
    } catch (_) {
      user = null;
    }
    if (!user) {
      showMessageModal('User Not Registered', 'The email ' + email + ' is not in the registered user list. Please check the exact email spelling in Admin > Users.', 'circle-alert');
      return;
    }
  }
  state.activeUser = user;
  document.getElementById('userNameDisplay').innerText = user.name;
  document.getElementById('userEmailDisplay').innerText = user.email;
  document.getElementById('userInitial').innerText = user.name.charAt(0).toUpperCase();
  document.getElementById('startTimeStatic').value = state.pending.time;
  document.getElementById('selectedRoomDisplay').value = state.pending.room;
  const endSelect = document.getElementById('endTimeSelect');
  endSelect.innerHTML = '';
  const availableEndTimes = getAvailableEndTimes(state.pending.roomId, state.pending.date, state.pending.time);
  availableEndTimes.forEach(time => endSelect.add(new Option(time, time)));
  if (!availableEndTimes.length) { showMessageModal('No End Time Available', 'No valid end times are available for this selected slot.', 'circle-alert'); return; }
  showStep('book');
}
async function submitBooking() {
  clearBookingValidation();
  const meetingPurpose = document.getElementById('meetingPurpose').value.trim();
  const endTime = document.getElementById('endTimeSelect').value;
  if (!state.pending || !state.activeUser) { showBookingValidation('Booking session expired. Please try again.'); return; }
  if (!meetingPurpose || meetingPurpose.length < 3) { showBookingValidation('Meeting title must be at least 3 characters.'); return; }
  if (!endTime) { showBookingValidation('Please select an end time.'); return; }
  if (isPastSlot(state.pending.date, state.pending.time)) { showBookingValidation('Past time slots cannot be booked.'); return; }
  if (hasBookingConflict(state.pending.roomId, state.pending.date, state.pending.time, endTime)) { showBookingValidation('This slot overlaps with an existing booking.'); return; }
  try {
    await api('/api/bookings', { method: 'POST', body: JSON.stringify({ roomId: state.pending.roomId, room: state.pending.room, date: toISODate(state.pending.date), start: state.pending.time, end: endTime, startTime: state.pending.time, endTime, user: state.activeUser.name, email: state.activeUser.email, purpose: meetingPurpose, status: 'Booked' }) });
    await loadBookings();
    document.getElementById('successTitle').innerText = 'Success!';
    document.getElementById('successMessage').innerText = 'Your booking has been created successfully.';
    resetBookingFormFields(); resetTransientState(); showStep('success'); renderCalendar();
  } catch (err) { showBookingValidation(err.message); }
}
function showDetails(id) {
  const b = state.bookings.find(x => x.id === id);
  if (!b) return;
  state.currentBookingId = id; state.cancelVerified = false;
  document.getElementById('viewMeetingTitle').innerText = b.purpose;
  document.getElementById('viewMeetingTime').innerText = `${b.date} | ${b.startTime} - ${b.endTime}`;
  document.getElementById('viewMeetingUser').innerText = b.user;
  document.getElementById('viewMeetingLocation').innerText = b.location || '';
  document.getElementById('viewMeetingEmail').innerText = b.email;
  document.getElementById('viewMeetingDuration').innerText = formatDuration(b.startTime, b.endTime);
  document.getElementById('viewMeetingRoom').innerText = b.room;
  showStep('details');
}
function requestCancelVerify() { showStep('verify'); document.getElementById('verifySubmitBtn').onclick = handleCancelVerify; }
function handleCancelVerify() {
  const email = document.getElementById('verifyEmail').value.toLowerCase().trim();
  const booking = state.bookings.find(b => b.id === state.currentBookingId);
  if (!booking || booking.email.toLowerCase() !== email) { showMessageModal('Access Denied', 'Only the original booking owner can cancel this meeting.', 'shield-alert'); return; }
  state.cancelVerified = true;
  document.getElementById('cancelConfirmText').innerText = `Are you sure you want to cancel "${booking.purpose}" in ${booking.room}?`;
  showStep('cancelConfirm');
}
async function confirmCancelBooking() {
  if (!state.cancelVerified || !state.currentBookingId) return;
  const booking = state.bookings.find(b => b.id === state.currentBookingId);
  try {
    await api(`/api/bookings/${state.currentBookingId}/cancel`, { method: 'POST', body: JSON.stringify({ email: booking.email }) });
    await loadBookings();
    document.getElementById('successTitle').innerText = 'Cancelled';
    document.getElementById('successMessage').innerText = 'The meeting booking has been cancelled successfully.';
    resetBookingFormFields(); resetTransientState(); showStep('success'); renderCalendar();
  } catch (err) { showMessageModal('Cancel Failed', err.message, 'circle-alert'); }
}
function showStep(step) {
  document.getElementById('modalOverlay').classList.remove('hidden');
  ['verify','book','details','cancelConfirm','message','success'].forEach(s => {
    const el = document.getElementById(s + 'Step');
    if (el) el.classList.add('hidden');
  });
  const target = document.getElementById(step + 'Step');
  if (target) target.classList.remove('hidden');
  lucide.createIcons();
}
function closeModal() {
  document.getElementById('modalOverlay').classList.add('hidden');
  resetBookingFormFields(); resetTransientState();
}

window.onload = async () => {
  try {
    await loadRooms(); await loadUsers(); await loadBookings();
    populateRoomSwitcher(); updateRoomHeader(); calculateDates(); renderCalendar(); updateHeaderClock();
  } catch (err) { showMessageModal('Loading Failed', err.message, 'circle-alert'); }
  setInterval(async () => { updateHeaderClock(); try { await loadBookings(); renderCalendar(); } catch {} }, 60000);
  setInterval(() => { document.getElementById('clock').innerText = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }); }, 1000);
};
