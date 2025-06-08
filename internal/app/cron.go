package app

import (
	"context"
	"github.com/robfig/cron/v3"
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/domains/bookings/service"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"
)

func Cron(db postgres.PgxIface, cfg *config.Config, l logger.Interface) {
	schedulerService := service.NewSchedulerService(db, cfg)

	c := cron.New(cron.WithSeconds())

	_, err := c.AddFunc(cfg.Schedule.BookingsExpiration, func() {
		ctx := context.WithoutCancel(context.Background())

		if err := schedulerService.ExpireOldBookings(ctx); err != nil {
			l.Error("Cron job - ExpireOldBookings failed: %v", err)
		}
	})

	if err != nil {
		l.Error("Cron job - AddFunc failed: %v", err)

		return
	}

	c.Start()
}
