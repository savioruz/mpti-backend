package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/jwt"
	"github.com/savioruz/goth/pkg/oauth"
)

type OAuthService interface {
	GetGoogleAuthURL() string
	HandleGoogleCallback(ctx context.Context, code string) (res *dto.UserLoginResponse, err error)
}

type oauthService struct {
	db             postgres.PgxIface
	repo           repository.Querier
	googleProvider oauth.GoogleProviderIface
	logger         logger.Interface
}

func New(db postgres.PgxIface, repo repository.Querier, googleProvider oauth.GoogleProviderIface, l logger.Interface) OAuthService {
	return &oauthService{
		db:             db,
		repo:           repo,
		googleProvider: googleProvider,
		logger:         l,
	}
}

func (s *oauthService) GetGoogleAuthURL() string {
	return s.googleProvider.GetAuthURL()
}

func (s *oauthService) HandleGoogleCallback(ctx context.Context, code string) (res *dto.UserLoginResponse, err error) {
	token, err := s.googleProvider.Exchange(code)
	if err != nil {
		s.logger.Error("google callback - service - failed to exchange code: %w", err)

		return nil, failure.InternalError(err)
	}

	userInfo, err := s.googleProvider.GetUserInfo(token)
	if err != nil {
		s.logger.Error("google callback - service - failed to get user info: %w", err)

		return nil, failure.InternalError(err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("google callback - service - failed to begin transaction: %w", err)

		return nil, failure.InternalError(err)
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error("google callback - service - failed to rollback transaction: %w", err)
		}
	}(tx, ctx)

	// Check if a user exists
	user, err := s.repo.GetUserByEmail(ctx, tx, userInfo.Email)
	if err != nil {
		params := repository.CreateUserParams{
			Email:        userInfo.Email,
			FullName:     pgtype.Text{String: userInfo.Name, Valid: true},
			IsVerified:   pgtype.Bool{Bool: userInfo.VerifiedEmail, Valid: true},
			Level:        "1",
			ProfileImage: pgtype.Text{String: userInfo.Picture, Valid: true},
		}

		user, err = s.repo.CreateUser(ctx, tx, params)
		if err != nil {
			s.logger.Error("google callback - service - failed to create user: %w", err)

			return nil, failure.InternalError(err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("google callback - service - failed to commit transaction: %w", err)

		return nil, failure.InternalError(err)
	}

	accessToken, err := jwt.GenerateAccessToken(user.ID.String(), user.Email, user.Level)
	if err != nil {
		s.logger.Error("google callback - service - failed to generate access token: %w", err)

		return nil, failure.InternalError(err)
	}

	refreshToken, err := jwt.GenerateRefreshToken(user.ID.String(), user.Email, user.Level)
	if err != nil {
		s.logger.Error("google callback - service - failed to generate refresh token: %w", err)

		return nil, failure.InternalError(err)
	}

	res = new(dto.UserLoginResponse).ToLoginResponse(accessToken, refreshToken)

	return res, nil
}
