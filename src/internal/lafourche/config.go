package lafourche

import (
	"os"
	"path/filepath"
)

// Valeurs par défaut découvertes par investigation du front lafourche.fr.
const (
	// Recherche produits : index Algolia du site (mêmes résultats et prix membres).
	// Clé « search-only » publique extraite du bootstrap du front.
	DefaultAlgoliaAppID  = "SPM5J6SZTM"
	DefaultAlgoliaAPIKey = "ca66381c136c56785ec5fb8e95a70ad7"
	DefaultAlgoliaIndex  = "production_products"

	// Backend membre de La Fourche : app Next.js + API GraphQL
	// « lego », auth Firebase. Sert au login et aux commandes.
	DefaultMemberAPIURL = "https://api.lafourche.fr/graphql"
	DefaultLFChannel    = "default:fr_FR"
	MemberSiteURL       = "https://lafourche.fr"

	// Clé API Web Firebase du projet La Fourche (découverte dans le front).
	// Permet le login email/mot de passe et le refresh, sans navigateur.
	DefaultFirebaseAPIKey = "AIzaSyDt_BPgkSXBnG_4VBmSXL04jMW03kj7whg"

	// Projet Firebase/Firestore de La Fourche. Le panier du compte (synchronisé
	// mobile/web) est stocké dans Firestore : customers/<uid>.shoppingCartId puis
	// carts/<shoppingCartId> (map SKU -> quantité).
	DefaultFirebaseProjectID = "production-la-fourche"
)

// Config regroupe les paramètres d'un Client. Tous surchargeables par env.
type Config struct {
	// Algolia (recherche produits).
	AlgoliaAppID  string
	AlgoliaAPIKey string
	AlgoliaIndex  string
	// SessionPath: fichier JSON où l'on persiste le panier et les jetons d'auth.
	SessionPath string
	// MemberAPIURL: endpoint GraphQL du backend membre (commandes, compte).
	MemberAPIURL string
	// LFChannel: valeur de l'en-tête lf-channel requis par l'API membre.
	LFChannel string
	// FirebaseAPIKey: clé Web Firebase (login email/mdp + refresh).
	FirebaseAPIKey string
	// FirebaseProjectID: projet Firestore (panier du compte).
	FirebaseProjectID string
}

// ConfigFromEnv construit une Config à partir des variables d'environnement,
// en retombant sur les valeurs par défaut.
func ConfigFromEnv() Config {
	return Config{
		AlgoliaAppID:      envOr("LAFOURCHE_ALGOLIA_APP_ID", DefaultAlgoliaAppID),
		AlgoliaAPIKey:     envOr("LAFOURCHE_ALGOLIA_API_KEY", DefaultAlgoliaAPIKey),
		AlgoliaIndex:      envOr("LAFOURCHE_ALGOLIA_INDEX", DefaultAlgoliaIndex),
		SessionPath:       envOr("LAFOURCHE_SESSION_PATH", defaultSessionPath()),
		MemberAPIURL:      envOr("LAFOURCHE_MEMBER_API_URL", DefaultMemberAPIURL),
		LFChannel:         envOr("LAFOURCHE_LF_CHANNEL", DefaultLFChannel),
		FirebaseAPIKey:    envOr("LAFOURCHE_FIREBASE_API_KEY", DefaultFirebaseAPIKey),
		FirebaseProjectID: envOr("LAFOURCHE_FIREBASE_PROJECT_ID", DefaultFirebaseProjectID),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultSessionPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "lafourche", "session.json")
	}
	return "lafourche-session.json"
}
