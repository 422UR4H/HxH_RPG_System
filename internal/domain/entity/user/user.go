package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        int       `json:"-"`
	UUID      uuid.UUID `json:"uuid"`
	Nick      string    `json:"nick"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
