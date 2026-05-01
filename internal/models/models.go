package models

type User struct {
	ID    int64  `json:"id,omitempty"`
	Email string `json:"email"`
	Name  string `json:"name"`
}
type UserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Admin struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type Room struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Location string `json:"location"`
	Status   string `json:"status"`
}

type Booking struct {
	ID       int64  `json:"id"`
	User     string `json:"user"`
	Email    string `json:"email"`
	RoomID   int64  `json:"roomId"`
	RoomName string `json:"roomName"`
	Date     string `json:"date"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Purpose  string `json:"purpose"`
	Status   string `json:"status"`

	// Compatibility fields for the original calendar frontend.
	Room      string `json:"room,omitempty"`
	Location  string `json:"location,omitempty"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Admin Admin  `json:"admin"`
	Token string `json:"token"`
}

type RoomRequest struct {
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Location string `json:"location"`
	Status   string `json:"status"`
}

type BookingRequest struct {
	User    string `json:"user"`
	Email   string `json:"email"`
	RoomID  int64  `json:"roomId"`
	Room    string `json:"room"`
	Date    string `json:"date"`
	Start   string `json:"start"`
	End     string `json:"end"`
	Purpose string `json:"purpose"`
	Status  string `json:"status"`

	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
