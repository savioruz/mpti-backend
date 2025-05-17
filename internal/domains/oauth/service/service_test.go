package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/savioruz/goth/internal/domains/user/mock"
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/jwt"
	log "github.com/savioruz/goth/pkg/logger/mock"
	"github.com/savioruz/goth/pkg/oauth"
	mockOAuth "github.com/savioruz/goth/pkg/oauth/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
)

func init() {
	jwt.Initialize("test-app", "test-secret-key", time.Hour, time.Hour*24)
}

func TestOauthService_GetGoogleAuthURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGoogleProvider := mockOAuth.NewMockGoogleProviderIface(ctrl)
	mockQuerier := mock.NewMockQuerier(ctrl)
	mockPgx, _ := pgxmock.NewPool()
	mockLogger := log.NewMockInterface(ctrl)

	service := New(mockPgx, mockQuerier, mockGoogleProvider, mockLogger)

	expectedURL := "https://accounts.google.com/o/oauth2/auth?client_id=test&redirect_uri=test&response_type=code&scope=openid+profile+email"
	mockGoogleProvider.EXPECT().GetAuthURL().Return(expectedURL)

	url := service.GetGoogleAuthURL()
	assert.Equal(t, expectedURL, url)
}

func TestOauthService_HandleGoogleCallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockGoogleProvider := mockOAuth.NewMockGoogleProviderIface(ctrl)
	mockQuerier := mock.NewMockQuerier(ctrl)
	mockPgx, _ := pgxmock.NewPool()
	mockLogger := log.NewMockInterface(ctrl)
	mockError := errors.New("error")

	service := New(mockPgx, mockQuerier, mockGoogleProvider, mockLogger)

	mockCode := "test-auth-code"
	mockToken := &oauth2.Token{AccessToken: "test-access-token"}
	mockUserInfo := &oauth.GoogleUserInfo{
		Email:         "test@example.com",
		Name:          "Test User",
		Picture:       "https://example.com/profile.jpg",
		VerifiedEmail: true,
	}

	mockID := uuid.New()
	mockUser := repository.User{
		ID:           pgtype.UUID{Bytes: mockID, Valid: true},
		Email:        "test@example.com",
		Level:        "1",
		FullName:     pgtype.Text{String: "Test User", Valid: true},
		IsVerified:   pgtype.Bool{Bool: true, Valid: true},
		ProfileImage: pgtype.Text{String: "https://example.com/profile.jpg", Valid: true},
		CreatedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	t.Run("error: exchange code failure", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())
		mockGoogleProvider.EXPECT().Exchange(mockCode).Return(nil, mockError)

		res, err := service.HandleGoogleCallback(ctx, mockCode)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: get user info failure", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())
		mockGoogleProvider.EXPECT().Exchange(mockCode).Return(mockToken, nil)
		mockGoogleProvider.EXPECT().GetUserInfo(mockToken).Return(nil, mockError)

		res, err := service.HandleGoogleCallback(ctx, mockCode)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: transaction begin failure", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())
		mockGoogleProvider.EXPECT().Exchange(mockCode).Return(mockToken, nil)
		mockGoogleProvider.EXPECT().GetUserInfo(mockToken).Return(mockUserInfo, nil)
		mockPgx.ExpectBegin().WillReturnError(mockError)

		res, err := service.HandleGoogleCallback(ctx, mockCode)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("success: existing user", func(t *testing.T) {
		mockGoogleProvider.EXPECT().Exchange(mockCode).Return(mockToken, nil)
		mockGoogleProvider.EXPECT().GetUserInfo(mockToken).Return(mockUserInfo, nil)
		mockPgx.ExpectBegin()
		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), mockUserInfo.Email).
			Return(mockUser, nil)
		mockPgx.ExpectCommit()
		mockPgx.ExpectRollback()

		res, err := service.HandleGoogleCallback(ctx, mockCode)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.AccessToken)
		assert.NotEmpty(t, res.RefreshToken)
	})

	t.Run("success: new user", func(t *testing.T) {
		mockPgx, _ = pgxmock.NewPool()
		service = New(mockPgx, mockQuerier, mockGoogleProvider, mockLogger)

		mockGoogleProvider.EXPECT().Exchange(mockCode).Return(mockToken, nil)
		mockGoogleProvider.EXPECT().GetUserInfo(mockToken).Return(mockUserInfo, nil)
		mockPgx.ExpectBegin()
		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), mockUserInfo.Email).
			Return(repository.User{}, errors.New("user not found"))

		mockQuerier.EXPECT().
			CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(mockUser, nil)

		mockPgx.ExpectCommit()
		mockPgx.ExpectRollback()

		res, err := service.HandleGoogleCallback(ctx, mockCode)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.AccessToken)
		assert.NotEmpty(t, res.RefreshToken)
	})

	t.Run("error: create user failure", func(t *testing.T) {
		mockPgx, _ = pgxmock.NewPool()
		service = New(mockPgx, mockQuerier, mockGoogleProvider, mockLogger)

		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

		mockGoogleProvider.EXPECT().Exchange(mockCode).Return(mockToken, nil)
		mockGoogleProvider.EXPECT().GetUserInfo(mockToken).Return(mockUserInfo, nil)
		mockPgx.ExpectBegin()
		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), mockUserInfo.Email).
			Return(repository.User{}, errors.New("user not found"))
		mockQuerier.EXPECT().
			CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(repository.User{}, mockError)
		mockPgx.ExpectRollback()

		res, err := service.HandleGoogleCallback(ctx, mockCode)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: transaction commit failure", func(t *testing.T) {
		mockPgx, _ = pgxmock.NewPool()
		service = New(mockPgx, mockQuerier, mockGoogleProvider, mockLogger)

		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

		mockGoogleProvider.EXPECT().Exchange(mockCode).Return(mockToken, nil)
		mockGoogleProvider.EXPECT().GetUserInfo(mockToken).Return(mockUserInfo, nil)
		mockPgx.ExpectBegin()
		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), mockUserInfo.Email).
			Return(mockUser, nil)
		mockPgx.ExpectCommit().WillReturnError(mockError)
		mockPgx.ExpectRollback()

		res, err := service.HandleGoogleCallback(ctx, mockCode)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})
}
