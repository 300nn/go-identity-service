package outbox

type UserRegisteredPayload struct {
	UserID int64  `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}
