package access

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSignerProducesExpiringEdDSAToken(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewSigner(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}))
	if err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	raw, err := signer.Sign(42)
	if err != nil {
		t.Fatal(err)
	}
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
			t.Fatalf("algorithm = %s", token.Method.Alg())
		}
		return publicKey, nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("JWT invalid: %v", err)
	}
	if claims.Subject != strconv.FormatInt(42, 10) || claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(before.Add(14*time.Minute)) || claims.ExpiresAt.Time.After(time.Now().Add(16*time.Minute)) {
		t.Fatalf("claims = %+v", claims)
	}
}
