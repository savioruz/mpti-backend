package service

import (
	"context"
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/domains/bookings/repository"
	"github.com/savioruz/goth/pkg/postgres"
)

type SchedulerService struct {
	db   postgres.PgxIface
	repo *repository.Queries
	cfg  *config.Config
}

func NewSchedulerService(db postgres.PgxIface, cfg *config.Config) *SchedulerService {
	return &SchedulerService{
		db:   db,
		repo: repository.New(),
		cfg:  cfg,
	}
}

func (s *SchedulerService) ExpireOldBookings(ctx context.Context) (err error) {
	return s.repo.ExpireOldBookings(ctx, s.db)
}
