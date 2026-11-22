package auth_test

import (
	"context"
	"testing"

	apiauth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	authUC "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
)

func TestLoginHandler_WrongCredentialsReadInPortuguese(t *testing.T) {
	mock := &mockLogin{fn: func(context.Context, *authUC.LoginInput) (authUC.LoginOutput, error) {
		return authUC.LoginOutput{}, authUC.ErrUnauthorized
	}}

	_, err := apiauth.NewAuthHandler(nil, mock).Login(context.Background(), &apiauth.LoginRequest{})
	if err == nil || err.Error() != "E-mail ou senha incorretos." {
		t.Errorf("err = %v", err)
	}
}
