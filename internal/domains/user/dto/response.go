package dto

import "github.com/savioruz/goth/internal/domains/user/repository"

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
