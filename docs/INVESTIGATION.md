# Investigation — APIs de La Fourche

> Notes de rétro-ingénierie des endpoints utilisés par `lafourche-mcp`.
> La boutique https://shop.lafourche.fr est **propulsée par Shopify**
> (`la-fourche.myshopify.com`, shop id `995655740`).

## 1. Recherche produits & panier — Storefront API (✅ fonctionnel)

API GraphQL Storefront, **non authentifiée** :

```
POST https://shop.lafourche.fr/api/2024-10/graphql.json
Header: X-Shopify-Storefront-Access-Token: 23efc1617fa111fad36f7baab56a7725
```

- Le token public est extrait du bootstrap de la home (`accessToken":"…"`).
  Surcharge possible via `LAFOURCHE_STOREFRONT_TOKEN`.
- Scopes accordés : `unauthenticated_read_product_listings`,
  `unauthenticated_read_checkouts`, `unauthenticated_write_checkouts`…
- ⚠️ Scopes **refusés** : `unauthenticated_read_product_inventory`
  (→ pas de `quantityAvailable`) et `unauthenticated_write_customers`
  (→ pas de `customerAccessTokenCreate`, cf. §3).

Opérations implémentées :
- `products(query:…)` → recherche
- `cartCreate` / `cartLinesAdd` / `cart(id:)` → panier (token de panier persisté
  localement dans la session).

## 2. Recherche rapide — Ajax predictive search (alternative)

```
GET https://shop.lafourche.fr/search/suggest.json?q=miel&resources[type]=product
```

Renvoie un JSON de suggestions. Pratique mais moins riche que la Storefront API ;
non utilisé pour l'instant.

## 3. Connexion & commandes — backend membre lafourche.fr (✅ fonctionnel)

⚠️ Le **compte membre** n'est PAS sur le Shopify `shop.lafourche.fr`, mais sur le
site `lafourche.fr` (app **Next.js**) adossé à un backend GraphQL maison
« lego ». La piste « New Customer Account API Shopify » explorée au départ était
une fausse piste (ce login Shopify n'est pas celui des membres).

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

## 4. Panier — Firestore (panier du compte, synchronisé mobile/web)

Le panier de La Fourche **n'est pas** le panier Shopify ni un panier anonyme : il
est stocké dans **Firestore** et lié au compte (même panier sur le site et l'appli).

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
| `LAFOURCHE_SHOP_DOMAIN` | `shop.lafourche.fr` | Domaine boutique |
| `LAFOURCHE_API_VERSION` | `2024-10` | Version API Shopify |
| `LAFOURCHE_STOREFRONT_TOKEN` | (token public) | Token Storefront |
| `LAFOURCHE_SESSION_PATH` | `$XDG_CONFIG_HOME/lafourche/session.json` | État local (panier + jetons) |
| `LAFOURCHE_MEMBER_API_URL` | `https://api.lafourche.fr/graphql` | API membre (commandes) |
| `LAFOURCHE_LF_CHANNEL` | `default:fr_FR` | En-tête `lf-channel` |
| `LAFOURCHE_EMAIL` / `LAFOURCHE_PASSWORD` | — | Identifiants login (non-interactif) |
| `LAFOURCHE_FIREBASE_API_KEY` | (clé du front) | Clé Web Firebase (login/refresh) |
| `LAFOURCHE_FIREBASE_PROJECT_ID` | `production-la-fourche` | Projet Firestore (panier) |
| `LAFOURCHE_CDP_URL` | `http://localhost:9222` | Endpoint Chrome DevTools (login `--cdp`) |
| `LAFOURCHE_MCP_TRANSPORT` | `stdio` | Transport MCP (`stdio`/`http`) |
| `LAFOURCHE_MCP_ADDR` | `:8080` | Écoute HTTP |
