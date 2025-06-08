package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/session"
	pgUser "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/user"
	"github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"golang.org/x/crypto/bcrypt"
)

type LoginInput struct {
	Email    string `json:"email" required:"true"`
	Password string `json:"password" required:"true"`
}

type LoginOutput struct {
	Token string     `json:"token"`
	User  *user.User `json:"user"`
}

type ILogin interface {
	Login(ctx context.Context, user *LoginInput) (LoginOutput, error)
}

type LoginUC struct {
	sessions    *sync.Map
	repo        IRepository
	sessionRepo session.IRepository
}

func NewLoginUC(
	sessions *sync.Map, repo IRepository, sessionRepo session.IRepository,
) *LoginUC {
	return &LoginUC{
		sessions:    sessions,
		repo:        repo,
		sessionRepo: sessionRepo,
	}
}

func (uc *LoginUC) Login(
	ctx context.Context, input *LoginInput) (LoginOutput, error) {

	if input.Email == "" {
		return LoginOutput{}, user.ErrMissingEmail
	}
	emailLength := len(input.Email)
	if emailLength < 12 || emailLength > 64 {
		return LoginOutput{}, user.ErrInvalidEmailLength
	}
	if input.Password == "" {
		return LoginOutput{}, user.ErrMissingPassword
	}
	passwordLength := len(input.Password)
	if passwordLength < 8 {
		return LoginOutput{}, user.ErrPasswordMinLenght
	}
	if passwordLength > 64 {
		return LoginOutput{}, user.ErrPasswordMaxLenght
	}

	userEntity, err := uc.repo.GetUserByEmail(ctx, input.Email)
	if err == pgUser.ErrEmailNotFound {
		return LoginOutput{}, ErrUnauthorized
	}
	if err != nil {
		return LoginOutput{}, err
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(userEntity.Password), []byte(input.Password),
	)
	if err != nil {
		return LoginOutput{}, ErrUnauthorized
	}

	token, err := auth.GenerateToken(userEntity.UUID)
	if err != nil {
		return LoginOutput{}, err
	}
	uc.sessions.Store(userEntity.UUID, token)

	err = uc.sessionRepo.CreateSession(ctx, userEntity.UUID, token)
	if err != nil {
		fmt.Println("failed to persist session:", err)
	}
	return LoginOutput{Token: token, User: userEntity}, nil
}
