package service

import (
	"context"
	"errors"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/logger"
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
}

type authService struct {
	db     postgres.PgxIface
	repo   repository.Querier
	logger logger.Interface
}

func New(db postgres.PgxIface, r repository.Querier, l logger.Interface) AuthService {
	return &authService{
		db:     db,
		repo:   r,
		logger: l,
	}
}

func (s *authService) Register(ctx context.Context, req dto.UserRegisterRequest) (res *dto.UserRegisterResponse, err error) {
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
		Email: req.Email,
		Password: pgtype.Text{
			String: string(password),
			Valid:  true,
		},
		Level: "1",
		FullName: pgtype.Text{
			String: req.Name,
			Valid:  true,
		},
		IsVerified: pgtype.Bool{
			Bool:  false,
			Valid: true,
		},
	})
	if err != nil {
		s.logger.Error("register - service - failed to create user: %w", err)

		return nil, failure.InternalError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("register - service - failed to commit transaction: %w", err)

		return nil, failure.InternalError(err)
	}

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

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(req.Password)); err != nil {
		s.logger.Error("login - service - unauthorized")

		return nil, failure.Unauthorized("unauthorized")
	}

	// TODO: check if user is verified

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
