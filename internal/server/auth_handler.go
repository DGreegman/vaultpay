package server

import (
	"errors"
	"log"

	"github.com/DGreegman/vaultpay/internal/server/dto"
	"github.com/DGreegman/vaultpay/internal/session"
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

func (s *Server) handleLogin(c *fiber.Ctx) error{

	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid_body", "request body is not a valid JSON")
	}

	if err := s.validate.Struct(req); err != nil {
		return writeValidationError(c, err)
	}

	// 1. Verify Credentials (timing-safe, vague on failure)
	u, err := s.userService.Authenticate(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			return writeError(c, fiber.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		}
		return writeError(c, fiber.StatusInternalServerError, "internal_error", "something went wrong")
	}
	
	// 2. Mint the short-lived access token (stateless JWT)
	accessToken, err := s.tokenManager.GenerateAccessToken(u.ID, string(u.Role))
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "internal_error", "something went wrong")
	}

	// 3. Issue the long lived refresh token (stateful, stored, new family)
	deviceID := optionalHeader(c, "X-Device-Id")

	ip := clientIP(c)

	issueed, err := s.sessionService.Issue(c.Context(), u.ID, deviceID, ip)

	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "internal_error", "something went wrong")
	}

	// 4. Return both tokens.
	return c.JSON(dto.TokenResponse{
		AccessToken: accessToken,
		RefreshToken: issueed.RawToken,
		TokenType: "Bearer",
		ExpiresIn: 900, // 15 mins
	})
}

// optionalHeader returns a pointer to the header value, or nil if absent —
// matching the nullable columns in the sessions table.
func optionalHeader(c *fiber.Ctx, name string) *string{
	v := c.Get(name)
	if v == ""{
		return nil
	}
	return &v
}

// clientIP returns the caller's IP as a pointer for the nullable column.
func clientIP(c *fiber.Ctx) *string {
	ip := c.IP()
	if ip == ""{
		return nil
	}
	return &ip
}

func (s *Server) handleRefresh(c *fiber.Ctx) error {

	var req dto.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invlid_body", "request body is not a valid JSON")
	}

	if err := s.validate.Struct(req); err != nil {
		return writeValidationError(c, err)
	}

	deviceID := optionalHeader(c, "X-Device-Id")
	ip := clientIP(c)

	// Rotate: validate the presented token, mark it used, mint a
	// replacement in the same family. Detects reuse (theft) internally.

	issued, err := s.sessionService.Rotate(c.Context(), req.RefreshToken, deviceID, ip)

	if err != nil {
		switch  {
		case errors.Is(err, session.ErrTokenReused):
			// The whole family was just revoked inside Rotate. Return the
			// same 401 as any token - we do not reveal to the caller that was detected.
			return writeError(c, fiber.StatusUnauthorized, "invalid_token", "refresh token is invalid")
		case errors.Is(err, session.ErrInvalidRefreshToken):
			return writeError(c, fiber.StatusUnauthorized, "invalid_token", "refresh token is invalid")
		default:
			log.Printf("refresh failed: %v", err)
			return writeError(c, fiber.StatusInternalServerError, "internal_error", "something went wrong")
		}
	}
	// The rotated token belongs to the same user; we need their role for
	// the new access token. Fetch the user fresh.
	u, err := s.userService.GetByID(c.Context(), issued.Session.UserID)
	if err != nil {
		log.Printf("refresh: get user: %v", err)
		return writeError(c, fiber.StatusInternalServerError, "internal_server", "something went wrong")
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(u.ID, string(u.Role))

	return c.JSON(dto.TokenResponse{
		AccessToken: accessToken,
		RefreshToken: issued.RawToken,
		TokenType: "Bearer",
		ExpiresIn: 900,
	})
}

