package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// SessionTTL is how long a freshly issued token (and cookie) stays valid.
	SessionTTL = 24 * time.Hour
	// MaxSessionAge caps how long a login can be extended, measured from the original login.
	MaxSessionAge = 7 * 24 * time.Hour
)

// Claims are the JWT claims: the subject is the user ID.
type Claims struct {
	jwt.RegisteredClaims
	AuthTime int64 `json:"auth_time"` // unix time of the original login
	Remember bool  `json:"remember"`
}

// UserID parses the subject.
func (c *Claims) UserID() (uuid.UUID, error) { return uuid.Parse(c.Subject) }

// SessionExpiry returns when a token issued at now expires: SessionTTL from now,
// but never later than MaxSessionAge after the original login.
func SessionExpiry(now, authTime time.Time) time.Time {
	expiry := now.Add(SessionTTL)
	if limit := authTime.Add(MaxSessionAge); limit.Before(expiry) {
		return limit
	}
	return expiry
}

// TokenManager issues and verifies HS256 JWTs.
type TokenManager struct {
	secret []byte
}

// NewTokenManager signs with secret, which must be long enough to resist brute-forcing.
func NewTokenManager(secret string) (*TokenManager, error) {
	if len(secret) < 16 {
		return nil, errors.New("JWT secret must be at least 16 characters")
	}
	return &TokenManager{secret: []byte(secret)}, nil
}

// Issue signs a token for a user. authTime and remember are carried over unchanged when
// the session is extended, so the 7-day cap and cookie kind survive refreshes.
func (m *TokenManager) Issue(id uuid.UUID, authTime time.Time, remember bool, expiry time.Time) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   id.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
		AuthTime: authTime.Unix(),
		Remember: remember,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse verifies a token's signature and expiry and checks the claims the app relies on.
// Pinning HS256 rejects tokens signed with another algorithm (e.g. "none").
func (m *TokenManager) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if _, err := claims.UserID(); err != nil || claims.AuthTime == 0 {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}
