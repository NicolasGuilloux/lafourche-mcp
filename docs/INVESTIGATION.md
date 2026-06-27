# Investigation — APIs de La Fourche

> Notes de rétro-ingénierie des endpoints utilisés par `lafourche-mcp`.
> Le front lafourche.fr (Next.js) s'appuie sur un back-end membre
> (`api.lafourche.fr` + Firebase/Firestore) et sur Algolia pour la recherche.

## 1. Recherche produits — Algolia (✅ fonctionnel)

Le site utilise **Algolia** (config extraite du `__NEXT_DATA__.runtimeConfig`,
clé « search-only » publique) :

```
POST https://SPM5J6SZTM-dsn.algolia.net/1/indexes/production_products/query
Headers: X-Algolia-Application-Id: SPM5J6SZTM
         X-Algolia-API-Key: ca66381c136c56785ec5fb8e95a70ad7
Body:    {"params":"query=p%C3%A2te&hitsPerPage=20"}
```

- App `SPM5J6SZTM`, index produits `production_products` (mêmes résultats que
  lafourche.fr, **prix membres**). « pâte » → ~840 hits.
- Chaque hit porte le **`sku`** (ex. `6-LAZ-102`) — directement utilisable par le
  panier — ainsi que `title`, `vendor`, `price`, `compare_at_price`, `handle`,
  `image`, `inventory_available`, `barcode`, `product_type`, catégories…
- Surcharge via `LAFOURCHE_ALGOLIA_APP_ID` / `_API_KEY` / `_INDEX`.

## 2. Connexion & commandes — backend membre lafourche.fr (✅ fonctionnel)

Le **compte membre** est sur `lafourche.fr` (app **Next.js**) adossé à un backend
GraphQL (`api.lafourche.fr`) et à **Firebase/Firestore**.

```
Login (UI)   : https://lafourche.fr/account/login   (email/mdp, Google, FB, Apple)
API GraphQL  : POST https://api.lafourche.fr/graphql  (Apollo, introspection OFF)
En-têtes     : Authorization: Bearer <idToken>
               lf-channel: default:fr_FR
```

### Auth = Firebase Auth

Le Bearer est un **ID token Firebase** (`iss: securetoken.google.com/production-la-fourche`,
exp ~1h), gardé en mémoire JS. Le **refresh token** Firebase est stocké dans
`IndexedDB` (`firebaseLocalStorageDb` → `firebaseLocalStorage`), avec l'`apiKey`
Firebase (`AIzaSyDt_BPgkSXBnG_4VBmSXL04jMW03kj7whg`).

Renouvellement (sans navigateur) :
```
POST https://securetoken.googleapis.com/v1/token?key=<apiKey>
  grant_type=refresh_token&refresh_token=<refreshToken>
  -> { id_token, refresh_token, expires_in }
```

### Stratégie de connexion implémentée

Email / mot de passe, sans navigateur (`src/internal/lafourche/auth.go`) :
```
POST https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=<apiKey>
     { email, password, returnSecureToken: true }  -> { idToken, refreshToken }
```
La clé Web Firebase (`AIzaSyDt_BPgkSXBnG_4VBmSXL04jMW03kj7whg`) est extraite du
front. L'`idToken` est directement accepté comme Bearer par `api.lafourche.fr` et
par Firestore. Le refresh token permet ensuite aux commandes authentifiées de
fonctionner pendant des semaines sans re-login (renouvellement automatique via
`securetoken.googleapis.com`).

> Le CDP (Chrome DevTools) n'a servi qu'à l'**exploration** (découverte des
> endpoints) ; il ne fait pas partie de l'outil livré. Les comptes 100 % SSO
> (Google/Facebook/Apple) ne sont pas couverts par le login email/mot de passe.

### Requête commandes

Opération `GetCustomerOrder` (route `/account/orders`) :
```graphql
query GetCustomerOrder($cursor: String, $pageSize: Int) {
  getCustomerOrder(cursor: $cursor, pageSize: $pageSize) {
    cursor
    items { id name financialStatus fulfillmentStatus createdAt totalPrice
            parcelTrackingUrl lineItems { title quantity price } }
  }
}
```

> 💡 Autres opérations repérées sur l'espace membre : `GetUpcomingInvoice`,
> `GetCurrentPaymentMethods` (abonnement / facturation), à exploiter plus tard.

### Config Next.js utile (`__NEXT_DATA__.runtimeConfig`)

```
legoUrl    = https://api.lafourche.fr   (backend membre)
appBaseUrl = https://lafourche.fr
authProviders = [email, google, facebook, apple]
```

## 3. Panier — Firestore (panier du compte, synchronisé mobile/web)

Le panier de La Fourche est stocké dans **Firestore** et lié au compte : c'est le
même panier sur le site et l'appli mobile.

```
Projet Firestore : production-la-fourche
customers/<uid>            -> champ shoppingCartId (id du panier)
carts/<shoppingCartId>     -> document = map { "SKU": <quantité entière>, ... }
```

Accès via l'API REST Firestore avec le Bearer Firebase :
```
Lecture : GET  https://firestore.googleapis.com/v1/projects/production-la-fourche/databases/(default)/documents/carts/<id>
Écriture: PATCH …/carts/<id>?updateMask.fieldPaths=`<SKU>`  body {"fields":{"<SKU>":{"integerValue":"N"}}}
           (champ dans le mask mais absent du corps => suppression de la ligne)
```

Enrichissement (noms, **prix membres**, total) : mutation `createCart` de
`api.lafourche.fr` — on lui envoie la liste `{sku, quantity}` du panier Firestore
et elle renvoie les libellés + montants **en centimes**. `createCart` ne sert qu'à
l'affichage (la vérité reste le doc Firestore).

> Le doc client (`customers/<uid>`) contient aussi : firstName, lastName, member,
> savings, orderCount, currentSubscription, address… (exploitable pour `info`).

## Récapitulatif des variables d'environnement

| Variable | Défaut | Rôle |
|---|---|---|
| `LAFOURCHE_ALGOLIA_APP_ID` | `SPM5J6SZTM` | App Algolia (recherche) |
| `LAFOURCHE_ALGOLIA_API_KEY` | (clé search-only) | Clé Algolia |
| `LAFOURCHE_ALGOLIA_INDEX` | `production_products` | Index produits |
| `LAFOURCHE_SESSION_PATH` | `$XDG_CONFIG_HOME/lafourche/session.json` | État local (panier + jetons) |
| `LAFOURCHE_MEMBER_API_URL` | `https://api.lafourche.fr/graphql` | API membre (commandes) |
| `LAFOURCHE_LF_CHANNEL` | `default:fr_FR` | En-tête `lf-channel` |
| `LAFOURCHE_EMAIL` / `LAFOURCHE_PASSWORD` | — | Identifiants login (non-interactif) |
| `LAFOURCHE_FIREBASE_API_KEY` | (clé du front) | Clé Web Firebase (login/refresh) |
| `LAFOURCHE_FIREBASE_PROJECT_ID` | `production-la-fourche` | Projet Firestore (panier) |
| `LAFOURCHE_CDP_URL` | `http://localhost:9222` | Endpoint Chrome DevTools (login `--cdp`) |
| `LAFOURCHE_MCP_TRANSPORT` | `stdio` | Transport MCP (`stdio`/`http`) |
| `LAFOURCHE_MCP_ADDR` | `:8080` | Écoute HTTP |
