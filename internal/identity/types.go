package identity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	FullName     string    `json:"fullname"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	Password     string    `json:"-"`
	TokenVersion int       `json:"-"`
	CreateAt     time.Time `json:"create_at"`
	UpdateAt     time.Time `json:"update_at"`
	DeleteAt     time.Time `json:"delete_at"`
}
