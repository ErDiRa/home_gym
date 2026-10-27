// Package jwt signs and checks the session tokens as HS256 JWTs.
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/modules/identity/app"
)

// claims are the JWT claims: the subject is the user id.
type claims struct {
	jwt.RegisteredClaims
	AuthTime int64 `json:"auth_time"` // unix time of the original login
	Remember bool  `json:"remember"`
}

// Tokens implements app.Tokens.
type Tokens struct {
	secret []byte
}

// NewTokens signs with secret, which must be long enough to resist brute-forcing.
func NewTokens(secret string) (*Tokens, error) {
	if len(secret) < 16 {
		return nil, errors.New("JWT secret must be at least 16 characters")
	}
	return &Tokens{secret: []byte(secret)}, nil
}

// Issue signs a token with the claims, valid until expiry.
func (t *Tokens) Issue(c app.Claims, expiry time.Time) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   c.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
		AuthTime: c.AuthTime.Unix(),
		Remember: c.Remember,
	}).SignedString(t.secret)
}

// Parse verifies a token's signature and expiry and checks the claims the app relies on.
// Pinning HS256 rejects tokens signed with another algorithm (e.g. "none").
func (t *Tokens) Parse(token string) (app.Claims, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return app.Claims{}, err
	}
	userID, err := uuid.Parse(c.Subject)
	if err != nil || c.AuthTime == 0 {
		return app.Claims{}, errors.New("invalid claims")
	}
	return app.Claims{UserID: userID, AuthTime: time.Unix(c.AuthTime, 0), Remember: c.Remember}, nil
}
