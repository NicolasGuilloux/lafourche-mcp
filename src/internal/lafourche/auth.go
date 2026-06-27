package lafourche

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// La connexion se fait directement contre Firebase Auth (le backend membre de
// La Fourche en dépend), via l'API REST Identity Toolkit — sans navigateur.
//
//	POST https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=<apiKey>
//	     { email, password, returnSecureToken: true }
//	  -> { idToken, refreshToken, expiresIn, ... }
//
// L'idToken obtenu est directement accepté comme Bearer par api.lafourche.fr.

// Login authentifie l'utilisateur par email/mot de passe et persiste la session.
func (c *Client) Login(ctx context.Context, email, password string) error {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return fmt.Errorf("email et mot de passe requis")
	}

	endpoint := "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=" +
		url.QueryEscape(c.cfg.FirebaseAPIKey)
	body, _ := json.Marshal(map[string]any{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		Email        string `json:"email"`
		Error        struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if out.IDToken == "" {
		return fmt.Errorf("échec de connexion : %s", firebaseAuthError(out.Error.Message))
	}

	c.session.AccessToken = out.IDToken
	c.session.RefreshToken = out.RefreshToken
	c.session.FirebaseAPIKey = c.cfg.FirebaseAPIKey
	c.session.ExpiresAt = jwtExpiry(out.IDToken)
	return c.session.Save()
}

// firebaseAuthError traduit les codes d'erreur Firebase courants.
func firebaseAuthError(code string) string {
	switch {
	case code == "":
		return "réponse Firebase invalide"
	case strings.HasPrefix(code, "EMAIL_NOT_FOUND"):
		return "email inconnu"
	case strings.HasPrefix(code, "INVALID_PASSWORD"),
		strings.HasPrefix(code, "INVALID_LOGIN_CREDENTIALS"):
		return "email ou mot de passe incorrect"
	case strings.HasPrefix(code, "USER_DISABLED"):
		return "compte désactivé"
	case strings.HasPrefix(code, "TOO_MANY_ATTEMPTS"):
		return "trop de tentatives, réessayez plus tard"
	default:
		return code
	}
}
