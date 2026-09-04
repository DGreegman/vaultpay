package server

import (
	"github.com/DGreegman/vaultpay/internal/config"
	"github.com/DGreegman/vaultpay/internal/session"
	"github.com/DGreegman/vaultpay/internal/token"
	"github.com/DGreegman/vaultpay/internal/user"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	recoverer "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server wraps the Fiber app and its dependencies

type Server struct {
	app *fiber.App
	cfg *config.Config
	pool *pgxpool.Pool
	userService *user.Service
	tokenManager *token.Manager
	sessionService *session.Service
	validate *validator.Validate
}

// New contructs a server with its routes registered.
// Dependecies are injected, never reached globally.

func New(cfg *config.Config, pool *pgxpool.Pool, userService *user.Service, tokenmanager *token.Manager, sessionService *session.Service) *Server {
	app := fiber.New(fiber.Config{
		AppName: 		"VaultPay",
		DisableStartupMessage: true,
		ErrorHandler: errorHandler,
	})

	validate := validator.New(validator.WithRequiredStructEnabled())
	s := &Server{
		app: app,
		cfg: cfg,
		pool: pool,
		userService: userService,
		tokenManager: tokenmanager,
		sessionService: sessionService,
		validate: validate,

	}

	s.registerRoutes()
	return s
}

// registerRoutes wires every HTTP route. Route live here so there
// is exactly one place to answer "what does this service expose"
func(s *Server) registerRoutes() {
	// recover turns a panic into a 500 instead of dead process, It must be first, so it wraps every handler and every middleware after it
	s.app.Use(recoverer.New(recoverer.Config{
		EnableStackTrace: true,
	}))
	// requestID stamps every request with a UUID and echoes it in the X-Request-ID header, so a user's bug report maps to exact log lines
	s.app.Use(requestid.New())
	s.app.Use(logger.New(logger.Config{
		Format: "${time} ${locals:requestid} ${status} ${method} ${path} ${latency}\n",
	}))
	s.app.Get("/healthz", s.handleHealthz)
	s.app.Get("/readyz", s.handleReadyz)

	v1 := s.app.Group("/v1")

	auth := v1.Group("/auth")
	auth.Post("/register", s.handleRegister)
	auth.Post("/login", s.handleLogin)
	auth.Post("/refresh", s.handleRefresh)
	auth.Post("/logout", s.handleLogout)
}

// Listen starts the HTP server. It blocks until the server stops
func (s *Server) Listen() error {
	return s.app.Listen(":" + s.cfg.Port)
}

// Shutdwon gracefully stops the server, allowing in-flight requests to complete before returning.
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}