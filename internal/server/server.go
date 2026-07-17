package server

import (
	"github.com/DGreegman/vaultpay/internal/config"
	"github.com/DGreegman/vaultpay/internal/user"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server wraps the Fiber app and its dependencies

type Server struct {
	app *fiber.App
	cfg *config.Config
	pool *pgxpool.Pool
	userService *user.Service
	validate *validator.Validate
}

// New contructs a server with its routes registered.
// Dependecies are injected, never reached globally.

func New(cfg *config.Config, pool *pgxpool.Pool, userService *user.Service) *Server {
	app := fiber.New(fiber.Config{
		AppName: 		"VaultPay",
		DisableStartupMessage: true,
	})

	validate := validator.New(validator.WithRequiredStructEnabled())
	s := &Server{
		app: app,
		cfg: cfg,
		pool: pool,
		userService: userService,
		validate: validate,

	}

	s.registerRoutes()
	return s
}

// registerRoutes wires every HTTP route. Route live here so there
// is exactly one place to answer "what does this service expose"
func(s *Server) registerRoutes() {
	s.app.Get("/healthz", s.handleHealthz)
	s.app.Get("/readyz", s.handleReadyz)

	v1 := s.app.Group("/v1")

	auth := v1.Group("/auth")
	auth.Post("/register", s.handleRegister)
}

// Listen starts the HTP server. It blocks until the server stops
func (s *Server) Listen() error {
	return s.app.Listen(":" + s.cfg.Port)
}

// Shutdwon gracefully stops the server, allowing in-flight requests to complete before returning.
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}