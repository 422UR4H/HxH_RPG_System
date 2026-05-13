package auth

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/google/uuid"
)

type RegisterInput struct {
	Nick        string `json:"nick" required:"true"`
	Email       string `json:"email" required:"true"`
	Password    string `json:"password" required:"true"`
	ConfirmPass string `json:"confirm_pass" required:"true"`
}

type IRegister interface {
	Register(ctx context.Context, user *RegisterInput) error
}

type RegisterUC struct {
	repo IRepository
}

func NewRegisterUC(repo IRepository) *RegisterUC {
	return &RegisterUC{
		repo: repo,
	}
}

func (uc *RegisterUC) Register(ctx context.Context, input *RegisterInput) error {
	if input.Nick == "" {
		return user.ErrMissingNick
	}
	nickLength := len(input.Nick)
	if nickLength < 3 || nickLength > 20 {
		return user.ErrInvalidNickLength
	}
	if input.Email == "" {
		return user.ErrMissingEmail
	}
	emailLength := len(input.Email)
	if emailLength < 12 || emailLength > 64 {
		return user.ErrInvalidEmailLength
	}
	if input.Password == "" {
		return user.ErrMissingPassword
	}
	if input.ConfirmPass == "" {
		return user.ErrMissingConfirmPass
	}
	passwordLength := len(input.Password)
	if passwordLength < 8 {
		return user.ErrPasswordMinLenght
	}
	if passwordLength > 64 {
		return user.ErrPasswordMaxLenght
	}
	if input.Password != input.ConfirmPass {
		return user.ErrMismatchPassword
	}

	// TODO: improve validation unifying these 2 or 3 db calls
	exists, err := uc.repo.ExistsUserWithNick(ctx, input.Nick)
	if err != nil {
		return err
	} else if exists {
		return user.ErrNickAlreadyExists
	}

	exists, err = uc.repo.ExistsUserWithEmail(ctx, input.Email)
	if err != nil {
		return err
	} else if exists {
		return user.ErrEmailAlreadyExists
	}

	err = uc.repo.CreateUser(ctx, &user.User{
		UUID:      uuid.New(),
		Nick:      input.Nick,
		Email:     input.Email,
		Password:  input.Password,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return err
	}
	return nil
}
