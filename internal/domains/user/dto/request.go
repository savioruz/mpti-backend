package dto

import "github.com/savioruz/goth/pkg/gdto"

type UserRegisterRequest struct {
	Email    string `example:"string@gmail.com" json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
}

type UserLoginRequest struct {
	Email    string `example:"string@gmail.com" json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type GetUsersRequest struct {
	gdto.PaginationRequest
	Email    string `query:"email" json:"email"`
	FullName string `query:"full_name" json:"full_name"`
	Level    string `query:"level" json:"level"`
}

type UpdateUserRoleRequest struct {
	Level string `json:"level" validate:"required,oneof=1 2 9"`
}

type EmailVerificationRequest struct {
	Token string `query:"token" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type ValidateResetTokenRequest struct {
	Token string `query:"token" validate:"required"`
}
