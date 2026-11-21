package middleware

import (
	"crypto"
	"fmt"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

type Verifier struct {
	publicKey crypto.PublicKey
}

func NewVerifier(publicKeyPEM []byte) (*Verifier, error) {
	publicKey, err := jwt.ParseEdPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse JWT public key: %w", err)
	}
	return &Verifier{publicKey: publicKey}, nil
}

func (v *Verifier) VerifyToken(tokenString string) (int64, error) {

	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return v.publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}), jwt.WithExpirationRequired())

	if err != nil {
		return 0, fmt.Errorf("Parse or verify token error: %w", err)
	}

	if !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user id in token: %w", err)
	}
	if userID <= 0 {
		return 0, fmt.Errorf("invalid user id in token")
	}

	return userID, nil
}
