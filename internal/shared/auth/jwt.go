package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingToken = errors.New("missing token")
	ErrInvalidToken = errors.New("invalid token")
)

type Claims struct {
	SessionID string   `json:"sid"`
	Roles     []string `json:"roles"`
	jwt.RegisteredClaims
}

type TokenValidator interface {
	Validate(token string) (*Claims, error)
}

type JWTValidator struct {
	secret    []byte
	publicKey *rsa.PublicKey
	now       func() time.Time
}

// NewJWTValidator creates a validator that uses HMAC (HS256) with the provided secret.
// Deprecated: Use NewJWTValidatorWithPublicKey for RS256 support.
func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{secret: []byte(strings.TrimSpace(secret)), now: time.Now}
}

// NewJWTValidatorWithPublicKey creates a validator that supports RS256 with RSA public key.
// If publicKeyPEM is provided, RS256 is used. Otherwise, falls back to HMAC with secret.
func NewJWTValidatorWithPublicKey(secret, publicKeyPEM string) *JWTValidator {
	v := &JWTValidator{
		secret: []byte(strings.TrimSpace(secret)),
		now:    time.Now,
	}

	if publicKeyPEM != "" {
		if key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM)); err == nil {
			v.publicKey = key
		}
	}

	return v
}

func (v *JWTValidator) Validate(token string) (*Claims, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrMissingToken
	}

	// Determine which key to use based on configuration
	if v.publicKey == nil && len(v.secret) == 0 {
		return nil, fmt.Errorf("%w: jwt key not configured (neither public key nor secret)", ErrInvalidToken)
	}

	claims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		// If we have a public key configured, expect RS256
		if v.publicKey != nil {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v, expected RS256", t.Header["alg"])
			}
			return v.publicKey, nil
		}

		// Fall back to HMAC if no public key
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.secret, nil
	}, jwt.WithLeeway(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	if claims.RegisteredClaims.Subject == "" {
		return nil, fmt.Errorf("%w: missing subject", ErrInvalidToken)
	}

	if claims.SessionID == "" {
		claims.SessionID = claims.RegisteredClaims.ID
	}
	if claims.SessionID == "" && claims.RegisteredClaims.Subject != "" {
		if claims.RegisteredClaims.ExpiresAt != nil {
			claims.SessionID = fmt.Sprintf("%s:%d", claims.RegisteredClaims.Subject, claims.RegisteredClaims.ExpiresAt.Unix())
		} else {
			claims.SessionID = claims.RegisteredClaims.Subject
		}
	}
	if claims.SessionID == "" {
		return nil, fmt.Errorf("%w: missing session id", ErrInvalidToken)
	}

	if exp := claims.RegisteredClaims.ExpiresAt; exp != nil && !exp.Time.After(v.now()) {
		return nil, fmt.Errorf("%w: token expired", ErrInvalidToken)
	}

	return claims, nil
}
