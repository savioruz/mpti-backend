package xendit

import (
	"github.com/savioruz/goth/config"
	x "github.com/xendit/xendit-go/v7"
)

func New(cfg *config.Config) *x.APIClient {
	return x.NewClient(cfg.Xendit.APIKey)
}
