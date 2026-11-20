package access

import (
	"crypto"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenLifetime = 15 * time.Minute

type Signer struct {
	privateKey crypto.PrivateKey
}

func NewSigner(privateKeyPEM []byte) (*Signer, error) {
	privateKey, err := jwt.ParseEdPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse JWT private key: %w", err)
	}
	return &Signer{privateKey: privateKey}, nil
}

func (s *Signer) Sign(userID int64) (string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(TokenLifetime)
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", err
	}
	return signed, nil
}
