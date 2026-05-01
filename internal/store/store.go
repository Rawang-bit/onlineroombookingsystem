package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"smartbook-go/internal/models"
)

type Store struct{ db *sql.DB }

func New(databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres connection failed: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.seed(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admins (id BIGSERIAL PRIMARY KEY, username TEXT UNIQUE NOT NULL, password TEXT NOT NULL, name TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());`,
		`CREATE TABLE IF NOT EXISTS users (id BIGSERIAL PRIMARY KEY, email TEXT UNIQUE NOT NULL, name TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());`,
		`CREATE TABLE IF NOT EXISTS rooms (id BIGSERIAL PRIMARY KEY, name TEXT UNIQUE NOT NULL, capacity INTEGER NOT NULL DEFAULT 1, location TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'Active', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());`,
		`CREATE TABLE IF NOT EXISTS bookings (id BIGSERIAL PRIMARY KEY, user_name TEXT NOT NULL, email TEXT NOT NULL, room_id BIGINT NOT NULL REFERENCES rooms(id) ON DELETE RESTRICT, booking_date DATE NOT NULL, start_time TEXT NOT NULL, end_time TEXT NOT NULL, purpose TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'Booked', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW());`,
		`CREATE INDEX IF NOT EXISTS idx_bookings_room_date ON bookings(room_id, booking_date);`,
	}
	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

func (s *Store) seed(ctx context.Context) error {
	admins := []struct{ u, p, n string }{{"admin", "admin123", "System Admin"}, {"manager", "manager123", "Booking Manager"}}
	for _, a := range admins {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO admins(username,password,name) VALUES($1,$2,$3) ON CONFLICT(username) DO NOTHING`, a.u, a.p, a.n); err != nil {
			return err
		}
	}
	rooms := []models.Room{{Name: "Conference Room A", Capacity: 10, Location: "Floor 1", Status: "Active"}, {Name: "Meeting Room B", Capacity: 8, Location: "Floor 2", Status: "Active"}, {Name: "Board Room", Capacity: 15, Location: "Floor 3", Status: "Active"}, {Name: "Executive Boardroom", Capacity: 18, Location: "Level 4, North Wing", Status: "Active"}, {Name: "Innovation Hub", Capacity: 12, Location: "Level 3, East Wing", Status: "Active"}}
	for _, r := range rooms {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO rooms(name,capacity,location,status) VALUES($1,$2,$3,$4) ON CONFLICT(name) DO NOTHING`, r.Name, r.Capacity, r.Location, r.Status); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Login(username, password string) (models.Admin, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var a models.Admin
	err := s.db.QueryRowContext(ctx, `SELECT id, username, name FROM admins WHERE username=$1 AND password=$2`, strings.TrimSpace(username), strings.TrimSpace(password)).Scan(&a.ID, &a.Username, &a.Name)
	return a, err == nil
}

func (s *Store) ListUsers() []models.User {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT id,email,name FROM users ORDER BY name`)
	if err != nil {
		return []models.User{}
	}
	defer rows.Close()
	out := []models.User{}
	for rows.Next() {
		var u models.User
		if rows.Scan(&u.ID, &u.Email, &u.Name) == nil {
			out = append(out, u)
		}
	}
	return out
}

func (s *Store) ListRooms() []models.Room {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,capacity,location,status FROM rooms ORDER BY id`)
	if err != nil {
		return []models.Room{}
	}
	defer rows.Close()
	out := []models.Room{}
	for rows.Next() {
		var r models.Room
		if rows.Scan(&r.ID, &r.Name, &r.Capacity, &r.Location, &r.Status) == nil {
			out = append(out, r)
		}
	}
	return out
}
func (s *Store) CreateRoom(req models.RoomRequest) (models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	normalizeRoom(&req)
	if req.Name == "" || req.Location == "" || req.Capacity < 1 {
		return models.Room{}, errors.New("room name, location, and capacity are required")
	}
	var r models.Room
	err := s.db.QueryRowContext(ctx, `INSERT INTO rooms(name,capacity,location,status) VALUES($1,$2,$3,$4) RETURNING id,name,capacity,location,status`, req.Name, req.Capacity, req.Location, req.Status).Scan(&r.ID, &r.Name, &r.Capacity, &r.Location, &r.Status)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return r, errors.New("room name already exists")
		}
		return r, err
	}
	return r, nil
}
func (s *Store) UpdateRoom(id int64, req models.RoomRequest) (models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	normalizeRoom(&req)
	if req.Name == "" || req.Location == "" || req.Capacity < 1 {
		return models.Room{}, errors.New("room name, location, and capacity are required")
	}
	var r models.Room
	err := s.db.QueryRowContext(ctx, `UPDATE rooms SET name=$1,capacity=$2,location=$3,status=$4 WHERE id=$5 RETURNING id,name,capacity,location,status`, req.Name, req.Capacity, req.Location, req.Status, id).Scan(&r.ID, &r.Name, &r.Capacity, &r.Location, &r.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return r, errors.New("room not found")
	}
	return r, err
}
func (s *Store) DeleteRoom(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := s.db.ExecContext(ctx, `DELETE FROM rooms WHERE id=$1`, id)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") {
			return errors.New("cannot delete room with existing bookings")
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("room not found")
	}
	return nil
}
func normalizeRoom(r *models.RoomRequest) {
	r.Name = strings.TrimSpace(r.Name)
	r.Location = strings.TrimSpace(r.Location)
	if r.Status == "" {
		r.Status = "Active"
	}
}

func (s *Store) ListBookings(roomFilter string) []models.Booking {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT b.id,b.user_name,b.email,b.room_id,r.name,b.booking_date::text,b.start_time,b.end_time,b.purpose,b.status,r.location FROM bookings b JOIN rooms r ON r.id=b.room_id`
	args := []any{}
	if strings.TrimSpace(roomFilter) != "" {
		query += ` WHERE r.name=$1 OR b.room_id::text=$1`
		args = append(args, roomFilter)
	}
	query += ` ORDER BY b.booking_date DESC, b.start_time DESC, b.id DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return []models.Booking{}
	}
	defer rows.Close()
	out := []models.Booking{}
	for rows.Next() {
		var b models.Booking
		if rows.Scan(&b.ID, &b.User, &b.Email, &b.RoomID, &b.RoomName, &b.Date, &b.Start, &b.End, &b.Purpose, &b.Status, &b.Location) == nil {
			fillCompat(&b)
			out = append(out, b)
		}
	}
	return out
}
func (s *Store) CreateBooking(req models.BookingRequest) (models.Booking, error) {
	return s.saveBooking(0, req)
}
func (s *Store) UpdateBooking(id int64, req models.BookingRequest) (models.Booking, error) {
	return s.saveBooking(id, req)
}
func (s *Store) saveBooking(id int64, req models.BookingRequest) (models.Booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	normalizeBooking(&req)
	if req.RoomID == 0 && req.Room != "" {
		_ = s.db.QueryRowContext(ctx, `SELECT id FROM rooms WHERE name=$1`, req.Room).Scan(&req.RoomID)
	}
	if req.Email == "" || req.RoomID == 0 || req.Date == "" || req.Start == "" || req.End == "" || len(req.Purpose) < 3 {
		return models.Booking{}, errors.New("all booking fields are required and purpose must be at least 3 characters")
	}
	bookingDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return models.Booking{}, errors.New("invalid booking date")
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if bookingDate.Before(today) {
		return models.Booking{}, errors.New("previous dates cannot be booked")
	}
	registeredUser, ok := s.GetUserByEmail(req.Email)
	if !ok {
		return models.Booking{}, errors.New("this email is not registered. Please contact admin to pre-register the user")
	}
	req.User = registeredUser.Name
	start, err := minutes(req.Start)
	if err != nil {
		return models.Booking{}, err
	}
	if bookingDate.Equal(today) && start <= now.Hour()*60+now.Minute() {
		return models.Booking{}, errors.New("past time slots cannot be booked")
	}
	end, err := minutes(req.End)
	if err != nil {
		return models.Booking{}, err
	}
	if end <= start {
		return models.Booking{}, errors.New("end time must be later than start time")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return models.Booking{}, err
	}
	defer tx.Rollback()
	var roomName, loc, roomStatus string
	if err := tx.QueryRowContext(ctx, `SELECT name,location,status FROM rooms WHERE id=$1`, req.RoomID).Scan(&roomName, &loc, &roomStatus); err != nil {
		return models.Booking{}, errors.New("room not found")
	}
	if roomStatus != "Active" {
		return models.Booking{}, errors.New("inactive rooms cannot be booked")
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,start_time,end_time FROM bookings WHERE room_id=$1 AND booking_date=$2 AND status <> 'Cancelled' FOR UPDATE`, req.RoomID, req.Date)
	if err != nil {
		return models.Booking{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var bid int64
		var s1, e1 string
		if rows.Scan(&bid, &s1, &e1) != nil {
			continue
		}
		if id != 0 && bid == id {
			continue
		}
		a, _ := minutes(s1)
		b, _ := minutes(e1)
		if start < b && end > a {
			return models.Booking{}, errors.New("this room is already booked for the selected time slot")
		}
	}
	status := req.Status
	if status == "" {
		status = "Booked"
	}
	var b models.Booking
	if id == 0 {
		err = tx.QueryRowContext(ctx, `INSERT INTO bookings(user_name,email,room_id,booking_date,start_time,end_time,purpose,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,user_name,email,room_id,booking_date::text,start_time,end_time,purpose,status`, req.User, req.Email, req.RoomID, req.Date, req.Start, req.End, req.Purpose, status).Scan(&b.ID, &b.User, &b.Email, &b.RoomID, &b.Date, &b.Start, &b.End, &b.Purpose, &b.Status)
	} else {
		err = tx.QueryRowContext(ctx, `UPDATE bookings SET user_name=$1,email=$2,room_id=$3,booking_date=$4,start_time=$5,end_time=$6,purpose=$7,status=$8,updated_at=NOW() WHERE id=$9 RETURNING id,user_name,email,room_id,booking_date::text,start_time,end_time,purpose,status`, req.User, req.Email, req.RoomID, req.Date, req.Start, req.End, req.Purpose, status, id).Scan(&b.ID, &b.User, &b.Email, &b.RoomID, &b.Date, &b.Start, &b.End, &b.Purpose, &b.Status)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return b, errors.New("booking not found")
	}
	if err != nil {
		return b, err
	}
	if err := tx.Commit(); err != nil {
		return b, err
	}
	b.RoomName = roomName
	b.Location = loc
	fillCompat(&b)
	return b, nil
}
func (s *Store) CancelBooking(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := s.db.ExecContext(ctx, `UPDATE bookings SET status='Cancelled',updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("booking not found")
	}
	return nil
}

func (s *Store) CancelBookingByEmail(id int64, email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := s.db.ExecContext(ctx, `UPDATE bookings SET status='Cancelled',updated_at=NOW() WHERE id=$1 AND lower(email)=lower($2)`, id, strings.TrimSpace(email))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("only the original booking owner can cancel this meeting")
	}
	return nil
}

func (s *Store) DeleteBooking(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := s.db.ExecContext(ctx, `DELETE FROM bookings WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("booking not found")
	}
	return nil
}

func normalizeBooking(b *models.BookingRequest) {
	b.User = strings.TrimSpace(b.User)
	b.Email = strings.ToLower(strings.TrimSpace(b.Email))
	b.Purpose = strings.TrimSpace(b.Purpose)
	if b.Start == "" {
		b.Start = b.StartTime
	}
	if b.End == "" {
		b.End = b.EndTime
	}
}
func fillCompat(b *models.Booking) { b.Room = b.RoomName; b.StartTime = b.Start; b.EndTime = b.End }
func minutes(t string) (int, error) {
	t = strings.TrimSpace(t)
	if p, err := time.Parse("15:04", t); err == nil {
		return p.Hour()*60 + p.Minute(), nil
	}
	if p, err := time.Parse("03:04 PM", t); err == nil {
		return p.Hour()*60 + p.Minute(), nil
	}
	return 0, fmt.Errorf("invalid time format: %s", t)
}

func normalizeUser(u *models.UserRequest) {
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	u.Name = strings.TrimSpace(u.Name)
}

func (s *Store) GetUserByEmail(email string) (models.User, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var u models.User
	err := s.db.QueryRowContext(ctx, `SELECT id,email,name FROM users WHERE lower(email)=lower($1)`, strings.TrimSpace(email)).Scan(&u.ID, &u.Email, &u.Name)
	return u, err == nil
}

func (s *Store) CreateUser(req models.UserRequest) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	normalizeUser(&req)
	if req.Email == "" || req.Name == "" || !strings.Contains(req.Email, "@") {
		return models.User{}, errors.New("valid user name and email are required")
	}
	var u models.User
	err := s.db.QueryRowContext(ctx, `INSERT INTO users(email,name) VALUES($1,$2) RETURNING id,email,name`, req.Email, req.Name).Scan(&u.ID, &u.Email, &u.Name)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return u, errors.New("user email already exists")
		}
		return u, err
	}
	return u, nil
}

func (s *Store) UpdateUser(id int64, req models.UserRequest) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	normalizeUser(&req)
	if req.Email == "" || req.Name == "" || !strings.Contains(req.Email, "@") {
		return models.User{}, errors.New("valid user name and email are required")
	}
	var u models.User
	err := s.db.QueryRowContext(ctx, `UPDATE users SET email=$1,name=$2 WHERE id=$3 RETURNING id,email,name`, req.Email, req.Name, id).Scan(&u.ID, &u.Email, &u.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return u, errors.New("user not found")
	}
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return u, errors.New("user email already exists")
		}
		return u, err
	}
	return u, nil
}

func (s *Store) DeleteUser(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("user not found")
	}
	return nil
}
