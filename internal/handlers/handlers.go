package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"smartbook-go/internal/models"
	"smartbook-go/internal/store"
)

type Handler struct{ Store *store.Store }

func New(s *store.Store) *Handler { return &Handler{Store: s} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("GET /api/users/validate", h.validateUser)
	mux.HandleFunc("GET /api/public/users/validate", h.validateUser)
	mux.HandleFunc("GET /api/public/users", h.publicUsers)
	mux.HandleFunc("GET /api/users", h.requireAdmin(h.users))
	mux.HandleFunc("POST /api/users", h.requireAdmin(h.createUser))
	mux.HandleFunc("PUT /api/users/", h.requireAdmin(h.updateUser))
	mux.HandleFunc("DELETE /api/users/", h.requireAdmin(h.deleteUser))
	mux.HandleFunc("GET /api/rooms", h.rooms)
	mux.HandleFunc("POST /api/rooms", h.requireAdmin(h.createRoom))
	mux.HandleFunc("PUT /api/rooms/", h.requireAdmin(h.updateRoom))
	mux.HandleFunc("DELETE /api/rooms/", h.requireAdmin(h.deleteRoom))
	mux.HandleFunc("GET /api/bookings", h.listBookings)
	mux.HandleFunc("POST /api/bookings", h.createBooking)
	mux.HandleFunc("PUT /api/bookings/", h.requireAdmin(h.updateBooking))
	mux.HandleFunc("DELETE /api/bookings/", h.requireAdmin(h.deleteBooking))
	mux.HandleFunc("POST /api/bookings/", h.publicBookingAction)
}
func (h *Handler) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token != "local-admin-session" {
			writeError(w, http.StatusUnauthorized, "admin login required")
			return
		}
		next(w, r)
	}
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if decode(w, r, &req) {
		if a, ok := h.Store.Login(req.Username, req.Password); ok {
			writeJSON(w, 200, models.LoginResponse{Admin: a, Token: "local-admin-session"})
			return
		}
		writeError(w, 401, "invalid username or password")
	}
}
func (h *Handler) users(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, h.Store.ListUsers())
}
func (h *Handler) publicUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, h.Store.ListUsers())
}
func (h *Handler) validateUser(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	u, ok := h.Store.GetUserByEmail(email)
	if !ok {
		writeError(w, http.StatusNotFound, "user is not pre-registered")
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"ok":    true,
		"id":    u.ID,
		"email": u.Email,
		"name":  u.Name,
	})
}
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req models.UserRequest
	if !decode(w, r, &req) {
		return
	}
	u, err := h.Store.CreateUser(req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, u)
}
func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/api/users/")
	if !ok {
		return
	}
	var req models.UserRequest
	if !decode(w, r, &req) {
		return
	}
	u, err := h.Store.UpdateUser(id, req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, u)
}
func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/api/users/")
	if !ok {
		return
	}
	if err := h.Store.DeleteUser(id); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}
func (h *Handler) rooms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, h.Store.ListRooms())
}
func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	var req models.RoomRequest
	if !decode(w, r, &req) {
		return
	}
	room, err := h.Store.CreateRoom(req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, room)
}
func (h *Handler) updateRoom(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/api/rooms/")
	if !ok {
		return
	}
	var req models.RoomRequest
	if !decode(w, r, &req) {
		return
	}
	room, err := h.Store.UpdateRoom(id, req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, room)
}
func (h *Handler) deleteRoom(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/api/rooms/")
	if !ok {
		return
	}
	if err := h.Store.DeleteRoom(id); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}
func (h *Handler) listBookings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, h.Store.ListBookings(r.URL.Query().Get("room")))
}
func (h *Handler) createBooking(w http.ResponseWriter, r *http.Request) {
	var req models.BookingRequest
	if !decode(w, r, &req) {
		return
	}
	b, err := h.Store.CreateBooking(req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, b)
}
func (h *Handler) updateBooking(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/api/bookings/")
	if !ok {
		return
	}
	var req models.BookingRequest
	if !decode(w, r, &req) {
		return
	}
	b, err := h.Store.UpdateBooking(id, req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, b)
}
func (h *Handler) deleteBooking(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/api/bookings/")
	if !ok {
		return
	}
	hard := r.URL.Query().Get("hard") == "1"
	var err error
	if hard {
		err = h.Store.DeleteBooking(id)
	} else {
		err = h.Store.CancelBooking(id)
	}
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (h *Handler) publicBookingAction(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if !strings.HasSuffix(path, "/cancel") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	idPart := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/bookings/"), "/cancel")
	idPart = strings.Trim(idPart, "/")
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := h.Store.CancelBookingByEmail(id, req.Email); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "cancelled"})
}

func pathID(w http.ResponseWriter, r *http.Request, prefix string) (int64, bool) {
	part := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	id, err := strconv.ParseInt(part, 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return 0, false
	}
	return id, true
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, 400, "invalid JSON request")
		return false
	}
	return true
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Error: message})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
