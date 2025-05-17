package httpserver

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	_defaultAddr            = ":3000"
	_defaultReadTimeout     = 5 * time.Second
	_defaultWriteTimeout    = 5 * time.Second
	_defaultShutdownTimeout = 3 * time.Second
)

type Server struct {
	App    *fiber.App
	notify chan error

	address         string
	readTimeout     time.Duration
	writeTimeout    time.Duration
	shutdownTimeout time.Duration
}

func New(opts ...Option) *Server {
	s := &Server{
		App:             nil,
		notify:          make(chan error, 1),
		address:         _defaultAddr,
		readTimeout:     _defaultReadTimeout,
		writeTimeout:    _defaultWriteTimeout,
		shutdownTimeout: _defaultShutdownTimeout,
	}

	// Custom options
	for _, opt := range opts {
		opt(s)
	}

	if s.App == nil {
		s.App = fiber.New(fiber.Config{
			Prefork:      false,
			ReadTimeout:  s.readTimeout,
			WriteTimeout: s.writeTimeout,
			IdleTimeout:  s.shutdownTimeout,
			JSONEncoder:  json.Marshal,
			JSONDecoder:  json.Unmarshal,
		})
	}

	return s
}

func (s *Server) Start() {
	go func() {
		s.notify <- s.App.Listen(s.address)
		close(s.notify)
	}()
}

func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) Shutdown() error {
	if err := s.App.ShutdownWithTimeout(s.shutdownTimeout); err != nil {
		return fmt.Errorf("httpserver: shutdown error: %w", err)
	}

	return nil
}
