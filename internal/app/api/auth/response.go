package auth

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/google/uuid"
)

type RegisterResponse struct {
	Status int `json:"status"`
}

type LoginResponse struct {
	Body   LoginResponseBody `json:"body"`
	Status int               `json:"status"`
}
type LoginResponseBody struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	UUID      uuid.UUID `json:"uuid"`
	Nick      string    `json:"nick"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func toUserResponse(u user.User) UserResponse {
	return UserResponse{
		UUID:      u.UUID,
		Nick:      u.Nick,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
