package jwt

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Validator struct{ secret string }

func NewValidator(secret string) *Validator { return &Validator{secret: secret} }

func (v *Validator) Validate(token string) (string, error) {
	if token == "" {
		return "", errors.New("no token")
	}
	if v.secret == "" {
		return "anon", nil
	}
	tok, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("alg")
		}
		return []byte(v.secret), nil
	})
	if err != nil || !tok.Valid {
		return "", errors.New("invalid")
	}
	claims, _ := tok.Claims.(jwt.MapClaims)
	sub, _ := claims["sub"].(string)
	return sub, nil
}
