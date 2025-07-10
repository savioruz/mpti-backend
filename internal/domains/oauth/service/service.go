package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/helper"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/jwt"
	"github.com/savioruz/goth/pkg/oauth"
)

type OAuthService interface {
	GetGoogleAuthURL() (dto.OauthGetURLResponse, error)
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

func (s *oauthService) GetGoogleAuthURL() (res dto.OauthGetURLResponse, err error) {
	state := helper.GenerateStateToken()

	url := s.googleProvider.GetAuthURL(state)
	if url == "" {
		s.logger.Error("oauth - service - failed to get Google auth URL")

		return res, failure.InternalError(errors.New("failed to get Google auth URL")) //nolint:err113
	}

	res = dto.OauthGetURLResponse{
		URL:   url,
		State: state,
	}

	return res, nil
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
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error("google callback - service - failed to get user by email: %w", err)

			return nil, failure.InternalError(err)
		}

		// User doesn't exist, create a new one
		user, err = s.createGoogleUser(ctx, tx, userInfo)
		if err != nil {
			return nil, err
		}
	} else {
		// User exists, update their Google ID if not already set
		user, err = s.updateExistingUserWithGoogleID(ctx, tx, user, userInfo)
		if err != nil {
			return nil, err
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

func (s *oauthService) createGoogleUser(ctx context.Context, tx pgx.Tx, userInfo *oauth.GoogleUserInfo) (repository.User, error) {
	params := repository.CreateUserParams{
		Email:        userInfo.Email,
		Password:     pgtype.Text{Valid: false}, // No password for OAuth users
		Level:        "1",
		GoogleID:     pgtype.Text{String: userInfo.ID, Valid: true},
		FullName:     pgtype.Text{String: userInfo.Name, Valid: true},
		ProfileImage: pgtype.Text{String: userInfo.Picture, Valid: true},
		IsVerified:   pgtype.Bool{Bool: userInfo.VerifiedEmail, Valid: true},
	}

	user, err := s.repo.CreateUser(ctx, tx, params)
	if err != nil {
		s.logger.Error("google callback - service - failed to create user: %w", err)

		return repository.User{}, failure.InternalError(err)
	}

	return user, nil
}

func (s *oauthService) updateExistingUserWithGoogleID(ctx context.Context, tx pgx.Tx, user repository.User, userInfo *oauth.GoogleUserInfo) (repository.User, error) {
	// Only update if Google ID is not already set
	if user.GoogleID.Valid && user.GoogleID.String != "" {
		return user, nil
	}

	updateParams := repository.UpdateUserParams{
		Email:        user.Email,
		Password:     user.Password,
		GoogleID:     pgtype.Text{String: userInfo.ID, Valid: true},
		FullName:     user.FullName,
		ProfileImage: pgtype.Text{String: userInfo.Picture, Valid: true},
		IsVerified:   pgtype.Bool{Bool: true, Valid: true},
		ID:           user.ID,
	}

	updatedUser, err := s.repo.UpdateUser(ctx, tx, updateParams)
	if err != nil {
		s.logger.Error("google callback - service - failed to update user: %w", err)

		return repository.User{}, failure.InternalError(err)
	}

	return updatedUser, nil
}
