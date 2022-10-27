package token

import (
	"crypto/ecdsa"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrJWTExpiredToken = errors.New("token has expired")
	ErrJWTInvalidToken = errors.New("invalid token")
)

// JWT payload
type Payload struct {
	ID        string   `json:"jti"`
	UserID    string   `json:"sub"`
	Username  string   `json:"preferred_username"`
	ExpiredAt int64    `json:"exp"`
	Roles     []string `json:"roles"`
}

// Checks if the token payload is valid or not
func (payload *Payload) Valid() error {
	if time.Now().After(time.Unix(payload.ExpiredAt, 0)) {
		return ErrJWTExpiredToken
	}
	return nil
}

type Parser struct {
	PublicKey *ecdsa.PublicKey
}

func (p *Parser) ParseUnverified(token string) (*Payload, error) {
	jwt, _, err := jwt.NewParser().ParseUnverified(token, &Payload{})
	if err != nil {
		return nil, ErrJWTInvalidToken
	}
	payload, ok := jwt.Claims.(*Payload)
	if !ok {
		return nil, ErrJWTInvalidToken
	}
	return payload, nil
}

func (p *Parser) ParseAndValidate(token string) (*Payload, error) {
	// Function for validating payload
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodECDSA)
		if !ok {
			return nil, ErrJWTInvalidToken
		}
		return p.PublicKey, nil
	}

	// Parse and validate using the function
	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		verr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(verr.Inner, ErrJWTExpiredToken) {
			return nil, ErrJWTExpiredToken
		}
		return nil, ErrJWTInvalidToken
	}

	if !jwtToken.Valid {
		return nil, ErrJWTInvalidToken
	}

	// Extract payload from token
	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrJWTInvalidToken
	}

	return payload, nil
}
