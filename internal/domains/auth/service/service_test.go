package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/mock"
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/jwt"
	log "github.com/savioruz/goth/pkg/logger/mock"
	mail "github.com/savioruz/goth/pkg/mail/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	jwt.Initialize("test-app", "test-secret-key", time.Hour, time.Hour*24)
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	mockError := errors.New("error")

	registerReq := dto.UserRegisterRequest{
		Email:    "test@gmail.com",
		Password: "password123",
		Name:     "Test User",
	}

	mockID := uuid.New()
	mockUser := repository.User{
		ID:         pgtype.UUID{Bytes: mockID, Valid: true},
		Email:      "test@gmail.com",
		Password:   pgtype.Text{String: "hashedpassword", Valid: true},
		Level:      "1",
		FullName:   pgtype.Text{String: "Test User", Valid: true},
		IsVerified: pgtype.Bool{Bool: false, Valid: true},
		CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	t.Run("error: transaction begin failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mock.NewMockQuerier(ctrl)
		mockPgx, _ := pgxmock.NewPool()
		mockLogger := log.NewMockInterface(ctrl)
		mockMail := mail.NewMockService(ctrl)
		service := New(mockPgx, mockQuerier, mockLogger, mockMail)

		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())
		mockPgx.ExpectBegin().WillReturnError(mockError)

		res, err := service.Register(ctx, registerReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: failure getting user by email", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mock.NewMockQuerier(ctrl)
		mockPgx, _ := pgxmock.NewPool()
		mockLogger := log.NewMockInterface(ctrl)
		mockMail := mail.NewMockService(ctrl)
		service := New(mockPgx, mockQuerier, mockLogger, mockMail)

		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(repository.User{}, mockError)

		res, err := service.Register(ctx, registerReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: user already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mock.NewMockQuerier(ctrl)
		mockPgx, _ := pgxmock.NewPool()
		mockLogger := log.NewMockInterface(ctrl)
		mockMail := mail.NewMockService(ctrl)
		service := New(mockPgx, mockQuerier, mockLogger, mockMail)

		mockLogger.EXPECT().Error(gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(mockUser, nil)

		res, err := service.Register(ctx, registerReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusBadRequest, failure.GetCode(err))
	})

	t.Run("error: failure creating user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mock.NewMockQuerier(ctrl)
		mockPgx, _ := pgxmock.NewPool()
		mockLogger := log.NewMockInterface(ctrl)
		mockMail := mail.NewMockService(ctrl)
		service := New(mockPgx, mockQuerier, mockLogger, mockMail)

		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(repository.User{}, nil)

		mockQuerier.EXPECT().
			CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(repository.User{}, mockError)

		res, err := service.Register(ctx, registerReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: failure creating email verification", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mock.NewMockQuerier(ctrl)
		mockPgx, _ := pgxmock.NewPool()
		mockLogger := log.NewMockInterface(ctrl)
		mockMail := mail.NewMockService(ctrl)
		service := New(mockPgx, mockQuerier, mockLogger, mockMail)

		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(repository.User{}, nil)

		mockQuerier.EXPECT().
			CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(mockUser, nil)

		mockQuerier.EXPECT().
			CreateEmailVerification(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(repository.EmailVerification{}, mockError)

		res, err := service.Register(ctx, registerReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: transaction commit failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mock.NewMockQuerier(ctrl)
		mockPgx, _ := pgxmock.NewPool()
		mockLogger := log.NewMockInterface(ctrl)
		mockMail := mail.NewMockService(ctrl)
		service := New(mockPgx, mockQuerier, mockLogger, mockMail)

		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(repository.User{}, nil)

		mockQuerier.EXPECT().
			CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(mockUser, nil)

		mockQuerier.EXPECT().
			CreateEmailVerification(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(repository.EmailVerification{}, nil)

		mockPgx.ExpectCommit().WillReturnError(mockError)

		res, err := service.Register(ctx, registerReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("success: user registered", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mock.NewMockQuerier(ctrl)
		mockPgx, _ := pgxmock.NewPool()
		mockLogger := log.NewMockInterface(ctrl)
		mockMail := mail.NewMockService(ctrl)
		service := New(mockPgx, mockQuerier, mockLogger, mockMail)

		mockPgx.ExpectBegin()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(repository.User{}, nil)

		mockQuerier.EXPECT().
			CreateUser(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(mockUser, nil)

		mockQuerier.EXPECT().
			CreateEmailVerification(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(repository.EmailVerification{}, nil)

		mockPgx.ExpectCommit()
		mockPgx.ExpectRollback() // For the deferred rollback function

		// Mock the email service call (it runs in a goroutine)
		mockMail.EXPECT().
			SendVerificationEmail(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		res, err := service.Register(ctx, registerReq)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, mockID.String(), res.ID)
		assert.Equal(t, "test@gmail.com", res.Email)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockQuerier := mock.NewMockQuerier(ctrl)
	mockPgx, _ := pgxmock.NewPool()
	mockLogger := log.NewMockInterface(ctrl)
	mockMail := mail.NewMockService(ctrl)
	mockError := errors.New("error")

	service := New(mockPgx, mockQuerier, mockLogger, mockMail)

	loginReq := dto.UserLoginRequest{
		Email:    "test@gmail.com",
		Password: "password123",
	}

	mockID := uuid.New()
	mockUser := func(password string) repository.User {
		return repository.User{
			ID:         pgtype.UUID{Bytes: mockID, Valid: true},
			Email:      "test@gmail.com",
			Password:   pgtype.Text{String: password, Valid: true},
			Level:      "1",
			FullName:   pgtype.Text{String: "Test User", Valid: true},
			IsVerified: pgtype.Bool{Bool: false, Valid: true},
			CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			DeletedAt:  pgtype.Timestamp{Valid: false},
		}
	}

	t.Run("error: transaction begin failure", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin().WillReturnError(mockError)

		res, err := service.Login(ctx, loginReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: failure getting user by email", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(repository.User{}, mockError)

		res, err := service.Login(ctx, loginReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("error: user not found", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(repository.User{}, nil)

		res, err := service.Login(ctx, loginReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusNotFound, failure.GetCode(err))
	})

	t.Run("error: user not verified", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		// Create a user that is not verified
		unverifiedUser := mockUser("hashedpassword")
		// IsVerified is already false in mockUser function

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(unverifiedUser, nil)

		res, err := service.Login(ctx, loginReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusBadRequest, failure.GetCode(err))
	})

	t.Run("error: invalid password", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		// Create a verified user with a password that won't match
		invalidPasswordUser := mockUser("hashedpassword")
		invalidPasswordUser.IsVerified = pgtype.Bool{Bool: true, Valid: true} // Make user verified
		_, _ = bcrypt.GenerateFromPassword([]byte("differentpassword"), bcrypt.DefaultCost)

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(invalidPasswordUser, nil)

		res, err := service.Login(ctx, loginReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusUnauthorized, failure.GetCode(err))
	})

	t.Run("error: transaction commit failure", func(t *testing.T) {
		mockLogger.EXPECT().Error(gomock.Any(), gomock.Any())

		mockPgx.ExpectBegin()
		mockPgx.ExpectRollback()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUserWithValidPassword := mockUser(string(hashedPassword))
		mockUserWithValidPassword.IsVerified = pgtype.Bool{Bool: true, Valid: true} // Make user verified

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(mockUserWithValidPassword, nil)

		mockQuerier.EXPECT().
			UpdateLastLogin(gomock.Any(), gomock.Any(), mockUserWithValidPassword.ID).
			Return(pgtype.UUID{Bytes: mockID, Valid: true}, nil)

		mockPgx.ExpectCommit().WillReturnError(mockError)

		res, err := service.Login(ctx, loginReq)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, http.StatusInternalServerError, failure.GetCode(err))
	})

	t.Run("success: login", func(t *testing.T) {
		mockPgx, _ = pgxmock.NewPool()
		service = New(mockPgx, mockQuerier, mockLogger, mockMail)

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

		mockUserWithValidPassword := mockUser(string(hashedPassword))
		mockUserWithValidPassword.IsVerified = pgtype.Bool{Bool: true, Valid: true} // Make user verified

		mockPgx.ExpectBegin()

		mockQuerier.EXPECT().
			GetUserByEmail(gomock.Any(), gomock.Any(), "test@gmail.com").
			Return(mockUserWithValidPassword, nil)

		mockQuerier.EXPECT().
			UpdateLastLogin(gomock.Any(), gomock.Any(), mockUserWithValidPassword.ID).
			Return(pgtype.UUID{Bytes: mockID, Valid: true}, nil)

		mockPgx.ExpectCommit()
		mockPgx.ExpectRollback()

		res, err := service.Login(ctx, loginReq)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.AccessToken)
		assert.NotEmpty(t, res.RefreshToken)
	})
}
