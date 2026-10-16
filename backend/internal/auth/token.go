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

// Roles carried in the token. Users and admins are separate accounts, so the subject
// is a user ID for RoleUser and an admin ID for RoleAdmin.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// Claims are the JWT claims: the subject is the account ID, Role says which table it is in.
type Claims struct {
	jwt.RegisteredClaims
	Role     string `json:"role"`
	AuthTime int64  `json:"auth_time"` // unix time of the original login
	Remember bool   `json:"remember"`
}

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

func NewTokenManager(secret string) (*TokenManager, error) {
	if len(secret) < 16 {
		return nil, errors.New("JWT secret must be at least 16 characters")
	}
	return &TokenManager{secret: []byte(secret)}, nil
}

func (m *TokenManager) Issue(role string, id uuid.UUID, authTime time.Time, remember bool, expiry time.Time) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   id.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
		Role:     role,
		AuthTime: authTime.Unix(),
		Remember: remember,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *TokenManager) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if _, err := claims.UserID(); err != nil || claims.AuthTime == 0 || (claims.Role != RoleUser && claims.Role != RoleAdmin) {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}
