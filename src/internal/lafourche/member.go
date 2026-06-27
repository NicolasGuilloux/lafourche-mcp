package lafourche

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrNotAuthenticated est renvoyée quand une opération nécessite une connexion.
var ErrNotAuthenticated = errors.New("non connecté : lancez `lafourche login`")

// Order est une vue simplifiée d'une commande (backend membre La Fourche).
type Order struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	ProcessedAt string      `json:"processed_at"`
	Status      string      `json:"financial_status"`
	Fulfillment string      `json:"fulfillment_status"`
	Total       string      `json:"total"`
	Currency    string      `json:"currency"`
	URL         string      `json:"tracking_url,omitempty"`
	Tracking    string      `json:"tracking_number,omitempty"`
	Carrier     string      `json:"carrier,omitempty"`
	ShipTo      string      `json:"ship_to,omitempty"`
	Lines       []OrderLine `json:"lines,omitempty"`
}

// OrderLine est une ligne de commande.
type OrderLine struct {
	Title    string `json:"title"`
	Quantity int    `json:"quantity"`
	Price    string `json:"price"`
	SKU      string `json:"sku,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
}

const getCustomerOrderQuery = `query GetCustomerOrder($cursor: String, $pageSize: Int) {
  getCustomerOrder(cursor: $cursor, pageSize: $pageSize) {
    cursor
    items {
      id name financialStatus fulfillmentStatus createdAt totalPrice parcelTrackingUrl
      shippingAddress { name address1 address2 zip city country }
      fulfillments { trackingCompany trackingUrl trackingNumber }
      lineItems { title quantity price sku vendor }
    }
  }
}`

// rawOrder mappe la forme brute renvoyée par l'API.
type rawOrder struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	FinancialStatus   string `json:"financialStatus"`
	FulfillmentStatus string `json:"fulfillmentStatus"`
	CreatedAt         string `json:"createdAt"`
	TotalPrice        string `json:"totalPrice"`
	ParcelTrackingURL string `json:"parcelTrackingUrl"`
	ShippingAddress   struct {
		Name     string `json:"name"`
		Address1 string `json:"address1"`
		Address2 string `json:"address2"`
		Zip      string `json:"zip"`
		City     string `json:"city"`
		Country  string `json:"country"`
	} `json:"shippingAddress"`
	Fulfillments []struct {
		TrackingCompany string `json:"trackingCompany"`
		TrackingURL     string `json:"trackingUrl"`
		TrackingNumber  string `json:"trackingNumber"`
	} `json:"fulfillments"`
	LineItems []struct {
		Title    string `json:"title"`
		Quantity int    `json:"quantity"`
		Price    string `json:"price"`
		SKU      string `json:"sku"`
		Vendor   string `json:"vendor"`
	} `json:"lineItems"`
}

func (r rawOrder) toOrder() Order {
	o := Order{
		ID:          r.ID,
		Name:        r.Name,
		ProcessedAt: r.CreatedAt,
		Status:      r.FinancialStatus,
		Fulfillment: r.FulfillmentStatus,
		Total:       r.TotalPrice,
		Currency:    "EUR",
		URL:         r.ParcelTrackingURL,
	}
	if a := r.ShippingAddress; a.City != "" || a.Address1 != "" {
		parts := []string{a.Name, strings.TrimSpace(a.Address1 + " " + a.Address2), strings.TrimSpace(a.Zip + " " + a.City), a.Country}
		var nz []string
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				nz = append(nz, p)
			}
		}
		o.ShipTo = strings.Join(nz, ", ")
	}
	if len(r.Fulfillments) > 0 {
		f := r.Fulfillments[0]
		o.Carrier = f.TrackingCompany
		o.Tracking = f.TrackingNumber
		if f.TrackingURL != "" {
			o.URL = f.TrackingURL
		}
	}
	for _, li := range r.LineItems {
		o.Lines = append(o.Lines, OrderLine{Title: li.Title, Quantity: li.Quantity, Price: li.Price, SKU: li.SKU, Vendor: li.Vendor})
	}
	return o
}

// fetchOrdersPage récupère une page de commandes (cursor pour la suivante).
func (c *Client) fetchOrdersPage(ctx context.Context, pageSize int, cursor string) ([]Order, string, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, "", err
	}
	vars := map[string]any{"pageSize": pageSize}
	if cursor != "" {
		vars["cursor"] = cursor
	}
	var resp struct {
		GetCustomerOrder struct {
			Cursor string     `json:"cursor"`
			Items  []rawOrder `json:"items"`
		} `json:"getCustomerOrder"`
	}
	if err := c.memberGraphQL(ctx, "GetCustomerOrder", getCustomerOrderQuery, vars, &resp); err != nil {
		return nil, "", err
	}
	orders := make([]Order, 0, len(resp.GetCustomerOrder.Items))
	for _, it := range resp.GetCustomerOrder.Items {
		orders = append(orders, it.toOrder())
	}
	return orders, resp.GetCustomerOrder.Cursor, nil
}

// LastOrders renvoie les dernières commandes du membre connecté.
func (c *Client) LastOrders(ctx context.Context, limit int) ([]Order, error) {
	if limit <= 0 {
		limit = 10
	}
	orders, _, err := c.fetchOrdersPage(ctx, limit, "")
	return orders, err
}

// GetOrder renvoie le détail d'une commande par son numéro (ex: "002955843"
// ou "2955843"), en paginant si nécessaire.
func (c *Client) GetOrder(ctx context.Context, number string) (*Order, error) {
	cursor := ""
	for page := 0; page < 10; page++ {
		orders, next, err := c.fetchOrdersPage(ctx, 50, cursor)
		if err != nil {
			return nil, err
		}
		for i := range orders {
			if orderNumberMatch(orders[i].Name, number) {
				return &orders[i], nil
			}
		}
		if next == "" || len(orders) == 0 {
			break
		}
		cursor = next
	}
	return nil, fmt.Errorf("commande %q introuvable", number)
}

func orderNumberMatch(name, query string) bool {
	n := strings.TrimLeft(strings.TrimSpace(name), "0")
	q := strings.TrimLeft(strings.TrimSpace(query), "0")
	return name == query || n == q
}

// memberGraphQL exécute une opération GraphQL authentifiée sur l'API membre.
func (c *Client) memberGraphQL(ctx context.Context, opName, query string, vars map[string]any, out any) error {
	headers := map[string]string{
		"Authorization": "Bearer " + c.session.AccessToken,
		"lf-channel":    c.cfg.LFChannel,
		"Origin":        MemberSiteURL,
	}
	body := map[string]any{"operationName": opName, "query": query, "variables": vars}
	return c.graphqlBody(ctx, c.cfg.MemberAPIURL, headers, body, out)
}

// ensureToken garantit un ID token Firebase valide, en le rafraîchissant via
// l'API securetoken de Google si nécessaire.
func (c *Client) ensureToken(ctx context.Context) error {
	s := c.session
	if s.AccessToken != "" && time.Now().Add(60*time.Second).Before(s.ExpiresAt) {
		return nil
	}
	if s.RefreshToken == "" || s.FirebaseAPIKey == "" {
		return ErrNotAuthenticated
	}

	endpoint := "https://securetoken.googleapis.com/v1/token?key=" + url.QueryEscape(s.FirebaseAPIKey)
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {s.RefreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tok struct {
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
		Error        struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return err
	}
	if tok.IDToken == "" {
		if tok.Error.Message != "" {
			return fmt.Errorf("rafraîchissement Firebase : %s (relancez `lafourche login`)", tok.Error.Message)
		}
		return ErrNotAuthenticated
	}
	s.AccessToken = tok.IDToken
	if tok.RefreshToken != "" {
		s.RefreshToken = tok.RefreshToken
	}
	s.ExpiresAt = jwtExpiry(tok.IDToken)
	return s.Save()
}

// Logout efface les jetons d'auth locaux (conserve le panier).
func (c *Client) Logout() error {
	c.session.AccessToken = ""
	c.session.RefreshToken = ""
	c.session.FirebaseAPIKey = ""
	c.session.ExpiresAt = time.Time{}
	return c.session.Save()
}

// jwtExpiry lit le champ exp d'un JWT ; défaut now+50min si illisible.
func jwtExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		if payload, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var claims struct {
				Exp int64 `json:"exp"`
			}
			if json.Unmarshal(payload, &claims) == nil && claims.Exp > 0 {
				return time.Unix(claims.Exp, 0)
			}
		}
	}
	return time.Now().Add(50 * time.Minute)
}
