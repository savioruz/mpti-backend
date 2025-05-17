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
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/mock"
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/failure"
	log "github.com/savioruz/goth/pkg/logger/mock"
	redis "github.com/savioruz/goth/pkg/redis/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUserService_Profile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cfg := &config.Config{
		Cache: config.Cache{
			Duration: 300,
		},
	}
	mockQuerier := mock.NewMockQuerier(ctrl)
	mockPgx, _ := pgxmock.NewPool()
	mockRedis := redis.NewMockIRedisCache(ctrl)
	mockLogger := log.NewMockInterface(ctrl)
	mockError := errors.New("error")

	service := New(mockPgx, mockQuerier, mockRedis, cfg, mockLogger)

	mockID := uuid.New()
	profileMock := repository.User{
		ID:           pgtype.UUID{Bytes: mockID, Valid: true},
		Email:        "string@gmail.com",
		Password:     pgtype.Text{String: "strongpassword", Valid: true},
		Level:        "user",
		GoogleID:     pgtype.Text{String: "google123", Valid: true},
		FullName:     pgtype.Text{String: "Test User", Valid: true},
		ProfileImage: pgtype.Text{String: "https://example.com/profile.jpg", Valid: true},
		IsVerified:   pgtype.Bool{Bool: true, Valid: true},
		LastLogin:    pgtype.Timestamp{Time: time.Now(), Valid: true},
		CreatedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
		DeletedAt:    pgtype.Timestamp{Valid: false},
	}

	t.Run("error: failure getting user by email", func(t *testing.T) {
		mockRedis.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockError)
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "error@gmail.com").
			Return(repository.User{}, mockError).
			Times(1)

		res, err := service.Profile(ctx, "error@gmail.com")

		assert.Error(t, err)
		assert.Equal(t, dto.UserProfileResponse{}, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: user not found", func(t *testing.T) {
		mockRedis.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockError)
		mockLogger.EXPECT().Error(gomock.Any())

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "notfound@gmail.com").
			Return(repository.User{}, nil).
			Times(1)

		res, err := service.Profile(ctx, "notfound@gmail.com")

		assert.Error(t, err)
		assert.Equal(t, dto.UserProfileResponse{}, res)
		assert.Equal(t, http.StatusNotFound, failure.GetCode(err))
	})

	t.Run("success: from database", func(t *testing.T) {
		mockRedis.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockError)
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
		mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

		mockRedis.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "string@gmail.com").
			Return(profileMock, nil).
			Times(1)

		res, err := service.Profile(ctx, "string@gmail.com")

		assert.NoError(t, err)
		assert.Equal(t, "string@gmail.com", res.Email)
		assert.Equal(t, "Test User", res.Name)
		assert.Equal(t, "https://example.com/profile.jpg", res.ProfileImage)
	})

	t.Run("success: from cache", func(t *testing.T) {
		var cachedResponse dto.UserProfileResponse
		cachedResponse = cachedResponse.ToProfileResponse(profileMock)

		mockRedis.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			SetArg(2, cachedResponse).Return(nil)

		res, err := service.Profile(ctx, "string@gmail.com")

		assert.NoError(t, err)
		assert.Equal(t, "string@gmail.com", res.Email)
		assert.Equal(t, "Test User", res.Name)
		assert.Equal(t, "https://example.com/profile.jpg", res.ProfileImage)
	})
}
