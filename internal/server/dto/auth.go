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

