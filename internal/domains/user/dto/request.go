package dto

type UserRegisterRequest struct {
	Email    string `example:"string@gmail.com" json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
}

type UserLoginRequest struct {
	Email    string `example:"string@gmail.com" json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}
