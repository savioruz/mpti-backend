package dto

import (
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/helper"
)

type UserRegisterResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type UserLoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserProfileResponse struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	ProfileImage string `json:"profile_image"`
}

type OauthGetURLResponse struct {
	URL   string `json:"url"`
	State string `json:"state"`
}

type UserAdminResponse struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	Level        string `json:"level"`
	ProfileImage string `json:"profile_image,omitempty"`
	IsVerified   bool   `json:"is_verified"`
	LastLogin    string `json:"last_login,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type EmailVerificationResponse struct {
	Message string `json:"message"`
}

type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

type ResetPasswordResponse struct {
	Message string `json:"message"`
}

type ValidateResetTokenResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

func (u *UserRegisterResponse) ToRegisterResponse(user repository.User) *UserRegisterResponse {
	return &UserRegisterResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	}
}

func (u *UserLoginResponse) ToLoginResponse(accessToken, refreshToken string) *UserLoginResponse {
	return &UserLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

func (u UserProfileResponse) ToProfileResponse(user repository.User) UserProfileResponse {
	var name, profileImage string
	if user.FullName.Valid {
		name = user.FullName.String
	}

	if user.ProfileImage.Valid {
		profileImage = user.ProfileImage.String
	}

	return UserProfileResponse{
		Email:        user.Email,
		Name:         name,
		ProfileImage: profileImage,
	}
}

func (u UserAdminResponse) FromModel(model repository.User) UserAdminResponse {
	var fullName, profileImage, lastLogin string

	if model.FullName.Valid {
		fullName = model.FullName.String
	}

	if model.ProfileImage.Valid {
		profileImage = model.ProfileImage.String
	}

	if model.LastLogin.Valid {
		lastLogin = model.LastLogin.Time.Format(constant.FullDateFormat)
	}

	return UserAdminResponse{
		ID:           model.ID.String(),
		Email:        model.Email,
		FullName:     fullName,
		Level:        model.Level,
		ProfileImage: profileImage,
		IsVerified:   model.IsVerified.Bool,
		LastLogin:    lastLogin,
		CreatedAt:    model.CreatedAt.Time.Format(constant.FullDateFormat),
		UpdatedAt:    model.UpdatedAt.Time.Format(constant.FullDateFormat),
	}
}

type PaginatedUserResponse struct {
	Users      []UserAdminResponse `json:"users"`
	TotalItems int                 `json:"total_items"`
	TotalPages int                 `json:"total_pages"`
}

func (p *PaginatedUserResponse) FromModel(users []repository.User, totalItems, limit int) {
	p.TotalItems = totalItems
	p.TotalPages = helper.CalculateTotalPages(totalItems, limit)

	if len(users) == 0 {
		p.Users = []UserAdminResponse{}

		return
	}

	p.Users = make([]UserAdminResponse, len(users))
	for i, user := range users {
		p.Users[i] = UserAdminResponse{}.FromModel(user)
	}
}
