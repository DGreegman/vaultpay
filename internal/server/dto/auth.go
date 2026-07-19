package dto

import "time"

// RegisterRequest is the JSON body for POST /v1/auth/register.
// Validation tags run before any business logic touches the data.

type RegisterRequest struct {
	Email    string  `json:"email"    validate:"required,email"`
	Phone    *string `json:"phone"    validate:"omitempty,e164"`
	Password string  `json:"password" validate:"required,min=8,max=72"`
}

// UserResponse is what we send back. Note what is ABSENT: no
// password_hash, ever. The API contract is chosen field by field.

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Phone     *string   `json:"phone,omitempty"`
	Role      string    `json:"role"`
	KYCTier   int16     `json:"kyc_tier"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}



// Login is the body for POST /v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// TokenResponse carries both tokens back to the client. The access token
// is a short-lived JWT; the refresh token is the raw opaque string whose
// hash we stored.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // access token lifetime, seconds
}

