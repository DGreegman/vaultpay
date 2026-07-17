package server

import (
	"errors"
	"log"

	"github.com/DGreegman/vaultpay/internal/server/dto"
	"github.com/DGreegman/vaultpay/internal/user"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) handleRegister(c *fiber.Ctx) error {

	var req dto.RegisterRequest

	// 1. Parse JSON. A malformed body is the client's fault -> 400.
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_body", "request body is not a valid JSON")
	}

	// 2. Validate shape before any logic runs.
	if err := s.validate.Struct(req); err != nil {
		return writeValidationError(c, err)
	}

	// 3. Hand off to the service (business logic lives there).
	u, err := s.userService.Register(c.Context(), user.RegisterInput{
		Email: req.Email,
		Phone: req.Phone,
		Password: req.Password,
	})

	if err != nil {
		return mapUserError(c, err)
	}
	// 4. Map domain -> DTO. Choose exactly what crosses the wire.
	return c.Status(fiber.StatusCreated).JSON(dto.UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Phone:     u.Phone,
		Role:      string(u.Role),
		KYCTier:   u.KYCTier,
		Status:    string(u.Status),
		CreatedAt: u.CreatedAt,

	})
}

// mapUserError translates domain errors into HTTP responses.
func mapUserError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, user.ErrEmailTaken):
		return writeError(c, fiber.StatusConflict, "email_taken", "email is already registered")
	case errors.Is(err, user.ErrPhoneTaken):
		return writeError(c, fiber.StatusConflict, "phone_taken", "phone is already registered")
	default:
		log.Printf("unhandled register error: %v", err)
		return writeError(c, fiber.StatusInternalServerError, "internal_error", "something went wrong")
	}
}