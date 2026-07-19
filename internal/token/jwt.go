package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// accessTokenTTL is how long an access token is valid. Deliberately
// short: a stolen access token cannot be revoked (it is stateless), so
// we bound the damage by expiry. The refresh token carries revocability.


const accessTokenTTL = 15 * time.Minute

var ErrInvalidToken = errors.New("token: invalid or expired")


// Claims is what we encode inside the access token. It embeds the
// library's RegisteredClaims (exp, iat, sub, etc.) and adds our own.

type Claims struct {
	UserID 		uuid.UUID 	`json:"uid"`
	Role		string		`json:"role"`
	jwt.RegisteredClaims
}

// Manager signs and verifies access tokens. It holds the secret so no
// other code needs to touch it.
type Manager struct {
	secret  []byte 
}

func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// GenerateAccessToken mints a signed, short-lived access token for a user.
func (m *Manager) GenerateAccessToken(userId uuid.UUID, role string) (string, error)  {

	now := time.Now()
	
	claims := Claims{
		UserID: userId,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: userId.String(),
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			Issuer: "vaultpay",
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := tok.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("token: sign %w", err)
	} 
	return signed, nil
	
}

func (m *Manager) Verify(tokenString string) (*Claims, error){
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {

		// Reject any token not signed with the method we expect.
		// Without this check, an attacker can set alg=none and bypass
		// signature verification entirely (algorithm-confusion attack).

		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["algo"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	return claims, nil
	
}

