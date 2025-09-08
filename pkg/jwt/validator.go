package jwt

import (
	"errors"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

type Validator struct {
	keys map[string]string
	skew time.Duration
}

func New(keys map[string]string, skew time.Duration) *Validator {
	return &Validator{keys: keys, skew: skew}
}

func (v *Validator) Validate(token string) (string, error) {
	if token == "" {
		return "", errors.New("no token")
	}
	if len(v.keys) == 0 {
		return "anon", nil
	}

	parser := jwtv5.NewParser(jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}))
	tok, err := parser.Parse(token, func(t *jwtv5.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" && len(v.keys) == 1 {
			for _, s := range v.keys {
				return []byte(s), nil
			}
		}
		sec, ok := v.keys[kid]
		if !ok {
			return nil, errors.New("unknown kid")
		}
		return []byte(sec), nil
	})
	if err != nil || !tok.Valid {
		return "", errors.New("invalid")
	}

	claims, ok := tok.Claims.(jwtv5.MapClaims)
	if !ok {
		return "", errors.New("claims")
	}

	now := time.Now()
	validateOpts := []jwtv5.ParserOption{
		jwtv5.WithLeeway(v.skew),
	}
	_ = validateOpts

	if expRaw, ok := claims["exp"]; ok {
		if exp, err := toTime(expRaw); err == nil && now.After(exp.Add(v.skew)) {
			return "", errors.New("expired")
		}
	}
	if nbfRaw, ok := claims["nbf"]; ok {
		if nbf, err := toTime(nbfRaw); err == nil && now.Add(v.skew).Before(nbf) {
			return "", errors.New("not yet valid")
		}
	}
	if iatRaw, ok := claims["iat"]; ok {
		if iat, err := toTime(iatRaw); err == nil && now.Before(iat.Add(-v.skew)) {
			return "", errors.New("issued in the future")
		}
	}

	sub, _ := claims["sub"].(string)
	return sub, nil
}

func toTime(x any) (time.Time, error) {
	switch v := x.(type) {
	case float64:
		return time.Unix(int64(v), 0), nil
	case int64:
		return time.Unix(v, 0), nil
	case jsonNumber:
		i, err := v.Int64()
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(i, 0), nil
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("bad time")
}

type jsonNumber interface{ Int64() (int64, error) }
