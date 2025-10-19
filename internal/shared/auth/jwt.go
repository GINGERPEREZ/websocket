package auth

import (
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
	secret []byte
	now    func() time.Time
}

func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{secret: []byte(strings.TrimSpace(secret)), now: time.Now}
}

func (v *JWTValidator) Validate(token string) (*Claims, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrMissingToken
	}

	if len(v.secret) == 0 {
		return nil, fmt.Errorf("%w: jwt secret not configured", ErrInvalidToken)
	}

	claims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
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
