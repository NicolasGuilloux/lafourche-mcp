package lafourche

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Session est l'état local persisté entre deux invocations: le panier Storefront
// en cours et les jetons de la Customer Account API (auth/commandes).
type Session struct {
	// ShoppingCartID est l'id du panier du compte (doc Firestore carts/<id>),
	// lu depuis customers/<uid>.shoppingCartId et mis en cache.
	ShoppingCartID string `json:"shopping_cart_id,omitempty"`

	// Jetons d'auth du backend membre (Firebase Auth).
	// AccessToken = ID token Firebase (Bearer), rafraîchissable via RefreshToken.
	AccessToken    string    `json:"access_token,omitempty"`
	RefreshToken   string    `json:"refresh_token,omitempty"`
	FirebaseAPIKey string    `json:"firebase_api_key,omitempty"`
	ExpiresAt      time.Time `json:"expires_at,omitempty"`

	path string
	mu   sync.Mutex
}

// LoadSession lit la session depuis le disque. Un fichier absent n'est pas une
// erreur: on renvoie une session vide rattachée au chemin donné.
func LoadSession(path string) (*Session, error) {
	s := &Session{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	s.path = path
	return s, nil
}

// Save écrit la session sur le disque (0600, le dossier est créé au besoin).
func (s *Session) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// Authenticated indique si un jeton d'accès non expiré est disponible.
func (s *Session) Authenticated() bool {
	return s.AccessToken != "" && time.Now().Before(s.ExpiresAt)
}
