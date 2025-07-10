package service

import (
	"context"
	"errors"

	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/helper"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/mail"
	"github.com/savioruz/goth/pkg/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req dto.UserRegisterRequest) (res *dto.UserRegisterResponse, err error)
	Login(ctx context.Context, req dto.UserLoginRequest) (*dto.UserLoginResponse, error)
	VerifyEmail(ctx context.Context, req dto.EmailVerificationRequest) (*dto.EmailVerificationResponse, error)
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error)
	ValidateResetToken(ctx context.Context, req dto.ValidateResetTokenRequest) (*dto.ValidateResetTokenResponse, error)
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error)
}

type authService struct {
	db          postgres.PgxIface
	repo        repository.Querier
	logger      logger.Interface
	mailService mail.Service
}

func New(db postgres.PgxIface, r repository.Querier, l logger.Interface, m mail.Service) AuthService {
	return &authService{
		db:          db,
		repo:        r,
		logger:      l,
		mailService: m,
	}
}

const (
	tokenLength = 32
)

func (s *authService) Register(ctx context.Context, req dto.UserRegisterRequest) (res *dto.UserRegisterResponse, err error) {
	// Validate email domain
	if !helper.IsAllowedEmailDomain(req.Email) {
		s.logger.Error("register - service - email domain not allowed: %s", req.Email)

		return nil, failure.BadRequestFromString("email domain not allowed. Please use a valid email provider like Gmail, Outlook, Yahoo, or iCloud")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("register - service - failed to begin transaction: %w", err)

		return nil, failure.InternalError(err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error("register - service - failed to rollback transaction: %w", err)
		}
	}(tx, ctx)

	exist, err := s.repo.GetUserByEmail(ctx, tx, req.Email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("register - service - failed to get user by email: %w", err)

		return nil, failure.InternalError(err)
	}

	if exist.Email != "" {
		s.logger.Error("register - service - user with email already exists")

		return nil, failure.BadRequestFromString("user already exists")
	}

	password, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("register - service - failed to generate password: %w", err)

		return nil, failure.InternalError(err)
	}

	newUser, err := s.repo.CreateUser(ctx, tx, repository.CreateUserParams{
		Email:      req.Email,
		Password:   helper.PgString(string(password)),
		Level:      constant.UserRoleUser,
		FullName:   helper.PgString(req.Name),
		IsVerified: helper.PgBool(false),
	})
	if err != nil {
		s.logger.Error("register - service - failed to create user: %w", err)

		return nil, failure.InternalError(err)
	}

	// Generate verification token
	token, err := helper.GenerateRandomToken(tokenLength)
	if err != nil {
		s.logger.Error("register - service - failed to generate verification token: %w", err)

		return nil, failure.InternalError(err)
	}

	// Create email verification record
	_, err = s.repo.CreateEmailVerification(ctx, tx, repository.CreateEmailVerificationParams{
		UserID: newUser.ID,
		Token:  token,
	})
	if err != nil {
		s.logger.Error("register - service - failed to create email verification: %w", err)

		return nil, failure.InternalError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("register - service - failed to commit transaction: %w", err)

		return nil, failure.InternalError(err)
	}

	// Send verification email
	go func() {
		if err := s.mailService.SendVerificationEmail(newUser.Email, newUser.FullName.String, token); err != nil {
			s.logger.Error("register - service - failed to send verification email: %w", err)
		}
	}()

	res = new(dto.UserRegisterResponse).ToRegisterResponse(newUser)

	return res, nil
}

func (s *authService) Login(ctx context.Context, req dto.UserLoginRequest) (*dto.UserLoginResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("login - service - failed to begin transaction: %w", err)

		return nil, failure.InternalError(err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error("login - service - failed to rollback transaction: %w", err)
		}
	}(tx, ctx)

	user, err := s.repo.GetUserByEmail(ctx, tx, req.Email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("login - service - failed to get user by email: %w", err)

		return nil, failure.InternalError(err)
	}

	if user.Email == "" {
		s.logger.Error("login - service - user not found")

		return nil, failure.NotFound("user not found")
	}

	// Check if user is verified (skip verification for Google OAuth users)
	if !helper.BoolFromPg(user.IsVerified) && !user.GoogleID.Valid {
		s.logger.Error("login - service - user is not verified")

		return nil, failure.BadRequestFromString("user is not verified")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(req.Password)); err != nil {
		s.logger.Error("login - service - unauthorized")

		return nil, failure.Unauthorized("unauthorized")
	}

	_, err = s.repo.UpdateLastLogin(ctx, tx, user.ID)
	if err != nil {
		s.logger.Error("login - service - failed to update last login: %w", err)

		return nil, failure.InternalError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("login - service - failed to commit transaction: %w", err)

		return nil, failure.InternalError(err)
	}

	accessToken, err := jwt.GenerateAccessToken(user.ID.String(), user.Email, user.Level)
	if err != nil {
		s.logger.Error("login - service - failed to generate access token: %w", err)

		return nil, failure.InternalError(err)
	}

	refreshToken, err := jwt.GenerateRefreshToken(user.ID.String(), user.Email, user.Level)
	if err != nil {
		s.logger.Error("login - service - failed to generate refresh token: %w", err)

		return nil, failure.InternalError(err)
	}

	return &dto.UserLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) VerifyEmail(ctx context.Context, req dto.EmailVerificationRequest) (*dto.EmailVerificationResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("verify-email - service - failed to begin transaction: %w", err)

		return nil, failure.InternalError(err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error("verify-email - service - failed to rollback transaction: %w", err)
		}
	}(tx, ctx)

	verification, err := s.repo.GetEmailVerificationByToken(ctx, tx, req.Token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error("verify-email - service - invalid or expired token")

			return nil, failure.BadRequestFromString("invalid or expired verification token")
		}

		s.logger.Error("verify-email - service - failed to get verification token: %w", err)

		return nil, failure.InternalError(err)
	}

	_, err = s.repo.VerifyEmail(ctx, tx, verification.UserID)
	if err != nil {
		s.logger.Error("verify-email - service - failed to verify email: %w", err)

		return nil, failure.InternalError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("verify-email - service - failed to commit transaction: %w", err)

		return nil, failure.InternalError(err)
	}

	return &dto.EmailVerificationResponse{
		Message: "Email verified successfully",
	}, nil
}

func (s *authService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("forgot-password - service - failed to begin transaction: %w", err)

		return nil, failure.InternalError(err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error("forgot-password - service - failed to rollback transaction: %w", err)
		}
	}(tx, ctx)

	user, err := s.repo.GetUserByEmail(ctx, tx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return success even if user doesn't exist for security
			return &dto.ForgotPasswordResponse{
				Message: "If the email exists, a password reset link has been sent",
			}, nil
		}

		s.logger.Error("forgot-password - service - failed to get user by email: %w", err)

		return nil, failure.InternalError(err)
	}

	// Check if user is a Google OAuth user
	if user.GoogleID.Valid && user.GoogleID.String != "" {
		s.logger.Error("forgot-password - service - cannot reset password for Google OAuth user")

		return nil, failure.BadRequestFromString("cannot reset password for Google OAuth account")
	}

	// Generate reset token
	token, err := helper.GenerateRandomToken(tokenLength)
	if err != nil {
		s.logger.Error("forgot-password - service - failed to generate reset token: %w", err)

		return nil, failure.InternalError(err)
	}

	// Create password reset record
	_, err = s.repo.CreatePasswordReset(ctx, tx, repository.CreatePasswordResetParams{
		UserID: user.ID,
		Token:  token,
	})
	if err != nil {
		s.logger.Error("forgot-password - service - failed to create password reset: %w", err)

		return nil, failure.InternalError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("forgot-password - service - failed to commit transaction: %w", err)

		return nil, failure.InternalError(err)
	}

	// Send reset email
	go func() {
		if err := s.mailService.SendPasswordResetEmail(user.Email, user.FullName.String, token); err != nil {
			s.logger.Error("forgot-password - service - failed to send reset email: %w", err)
		}
	}()

	return &dto.ForgotPasswordResponse{
		Message: "If the email exists, a password reset link has been sent",
	}, nil
}

func (s *authService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("reset-password - service - failed to begin transaction: %w", err)

		return nil, failure.InternalError(err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error("reset-password - service - failed to rollback transaction: %w", err)
		}
	}(tx, ctx)

	resetRecord, err := s.repo.GetPasswordResetByToken(ctx, tx, req.Token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error("reset-password - service - invalid or expired token")

			return nil, failure.BadRequestFromString("invalid or expired reset token")
		}

		s.logger.Error("reset-password - service - failed to get reset token: %w", err)

		return nil, failure.InternalError(err)
	}

	// Check if the user is a Google OAuth user
	user, err := s.repo.GetUserByID(ctx, tx, resetRecord.UserID)
	if err != nil {
		s.logger.Error("reset-password - service - failed to get user: %w", err)

		return nil, failure.InternalError(err)
	}

	if user.GoogleID.Valid && user.GoogleID.String != "" {
		s.logger.Error("reset-password - service - cannot reset password for Google OAuth user")

		return nil, failure.BadRequestFromString("cannot reset password for Google OAuth account")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("reset-password - service - failed to hash password: %w", err)

		return nil, failure.InternalError(err)
	}

	_, err = s.repo.ResetPassword(ctx, tx, repository.ResetPasswordParams{
		Password: pgtype.Text{
			String: string(hashedPassword),
			Valid:  true,
		},
		ID: resetRecord.UserID,
	})
	if err != nil {
		s.logger.Error("reset-password - service - failed to reset password: %w", err)

		return nil, failure.InternalError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("reset-password - service - failed to commit transaction: %w", err)

		return nil, failure.InternalError(err)
	}

	return &dto.ResetPasswordResponse{
		Message: "Password reset successfully",
	}, nil
}

func (s *authService) ValidateResetToken(ctx context.Context, req dto.ValidateResetTokenRequest) (*dto.ValidateResetTokenResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("validate-reset-token - service - failed to begin transaction: %w", err)

		return nil, failure.InternalError(err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error("validate-reset-token - service - failed to rollback transaction: %w", err)
		}
	}(tx, ctx)

	resetRecord, err := s.repo.GetPasswordResetByToken(ctx, tx, req.Token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &dto.ValidateResetTokenResponse{
				Valid:   false,
				Message: "Invalid or expired reset token",
			}, nil
		}

		s.logger.Error("validate-reset-token - service - failed to get reset token: %w", err)

		return nil, failure.InternalError(err)
	}

	// Check if the user is a Google OAuth user
	user, err := s.repo.GetUserByID(ctx, tx, resetRecord.UserID)
	if err != nil {
		s.logger.Error("validate-reset-token - service - failed to get user: %w", err)

		return nil, failure.InternalError(err)
	}

	if user.GoogleID.Valid && user.GoogleID.String != "" {
		return &dto.ValidateResetTokenResponse{
			Valid:   false,
			Message: "Cannot reset password for Google OAuth account",
		}, nil
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("validate-reset-token - service - failed to commit transaction: %w", err)

		return nil, failure.InternalError(err)
	}

	return &dto.ValidateResetTokenResponse{
		Valid:   true,
		Message: "Reset token is valid",
	}, nil
}
