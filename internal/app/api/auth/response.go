package auth

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"

type RegisterResponse struct {
	Status int `json:"status"`
}

type LoginResponse struct {
	Body   LoginResponseBody `json:"body"`
	Status int               `json:"status"`
}
type LoginResponseBody struct {
	Token string    `json:"token"`
	User  user.User `json:"user"`
}
