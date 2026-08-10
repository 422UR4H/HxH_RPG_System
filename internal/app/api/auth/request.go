package auth

type RegisterRequest struct {
	Body RegisterRequestBody `json:"body"`
}

type RegisterRequestBody struct {
	Nick        string `json:"nick" required:"true"`
	Email       string `json:"email" required:"true"`
	Password    string `json:"password" required:"true"`
	ConfirmPass string `json:"confirmPass" required:"true"`
}

type LoginRequest struct {
	Body LoginRequestBody `json:"body" required:"true"`
}

type LoginRequestBody struct {
	Email    string `json:"email" required:"true"`
	Password string `json:"password" required:"true"`
}
