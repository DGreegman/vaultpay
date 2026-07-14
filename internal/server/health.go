package server

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)


// handleHealthz reports that the process is alive.
// It must never depend on downstream services — an orchestrator
// uses this to decide whether to RESTART the container.
func (s *Server) handleHealthz(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

// handleReadyz reports that the process is ready to serve traffic.
// Unlike healthz, this DOES check dependencies (database, etc.) —
// an orchestrator uses it to decide whether to ROUTE traffic here.
func (s *Server) handleReadyz(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	if err := s.pool.Ping(ctx); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "not ready",
			"error":  "database unreachable",
		})
	}

	return c.JSON(fiber.Map{
		"status": "ready",
	})
}