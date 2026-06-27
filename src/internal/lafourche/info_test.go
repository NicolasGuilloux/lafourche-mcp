package lafourche

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// makeJWT forge un JWT non signé (payload base64url) pour les tests.
func makeJWT(claims map[string]any) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]any{"alg": "RS256", "typ": "JWT"}) + "." + enc(claims) + ".sig"
}

func TestParseFirebaseClaims(t *testing.T) {
	token := makeJWT(map[string]any{
		"name":           "Jean Bio",
		"email":          "jean@example.com",
		"email_verified": true,
		"user_id":        "abc123",
		"sub":            "abc123",
		"exp":            1782561302,
		"firebase":       map[string]any{"sign_in_provider": "password"},
	})
	c, err := parseFirebaseClaims(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.Email != "jean@example.com" || c.Name != "Jean Bio" || c.UserID != "abc123" {
		t.Errorf("claims inattendues: %+v", c)
	}
	if !c.EmailVerified || c.Firebase.SignInProvider != "password" {
		t.Errorf("flags inattendus: %+v", c)
	}
	if c.Exp != 1782561302 {
		t.Errorf("exp inattendu: %d", c.Exp)
	}
}

func TestParseFirebaseClaimsInvalid(t *testing.T) {
	if _, err := parseFirebaseClaims("not-a-jwt"); err == nil {
		t.Error("attendu une erreur pour un jeton invalide")
	}
}
