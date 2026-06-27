package lafourche

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// UserInfo décrit l'utilisateur connecté, extrait des claims du jeton Firebase.
type UserInfo struct {
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Name          string    `json:"name,omitempty"`
	UserID        string    `json:"user_id"`
	Provider      string    `json:"sign_in_provider,omitempty"`
	ExpiresAt     time.Time `json:"token_expires_at"`
}

// firebaseClaims mappe les claims utiles d'un ID token Firebase.
type firebaseClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	UserID        string `json:"user_id"`
	Sub           string `json:"sub"`
	Exp           int64  `json:"exp"`
	Firebase      struct {
		SignInProvider string `json:"sign_in_provider"`
	} `json:"firebase"`
}

// UserInfo renvoie les informations du compte connecté. Rafraîchit le jeton si
// nécessaire pour disposer de claims à jour.
func (c *Client) UserInfo(ctx context.Context) (*UserInfo, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	claims, err := parseFirebaseClaims(c.session.AccessToken)
	if err != nil {
		return nil, err
	}
	uid := claims.UserID
	if uid == "" {
		uid = claims.Sub
	}
	info := &UserInfo{
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		UserID:        uid,
		Provider:      claims.Firebase.SignInProvider,
		ExpiresAt:     time.Unix(claims.Exp, 0),
	}
	return info, nil
}

func parseFirebaseClaims(token string) (*firebaseClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("jeton invalide")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("décodage du jeton : %w", err)
	}
	var claims firebaseClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("claims illisibles : %w", err)
	}
	return &claims, nil
}
