package lafourche

import (
	"os"
	"path/filepath"
)

// Valeurs par défaut découvertes par investigation de shop.lafourche.fr.
// La boutique La Fourche est propulsée par Shopify (la-fourche.myshopify.com).
const (
	DefaultShopDomain = "shop.lafourche.fr"
	DefaultAPIVersion = "2024-10"

	// Token Storefront public extrait du bootstrap de la home Shopify.
	// Il autorise les scopes "unauthenticated_*" (recherche produits, panier).
	DefaultStorefrontToken = "23efc1617fa111fad36f7baab56a7725"

	// Identifiant numérique de la boutique (utilisé par la Customer Account API).
	ShopID = "995655740"

	// Backend membre de La Fourche (≠ Shopify) : app Next.js + API GraphQL
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
	ShopDomain      string
	APIVersion      string
	StorefrontToken string
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
		ShopDomain:        envOr("LAFOURCHE_SHOP_DOMAIN", DefaultShopDomain),
		APIVersion:        envOr("LAFOURCHE_API_VERSION", DefaultAPIVersion),
		StorefrontToken:   envOr("LAFOURCHE_STOREFRONT_TOKEN", DefaultStorefrontToken),
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
