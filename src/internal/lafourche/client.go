package lafourche

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client est le point d'entrée vers les APIs de La Fourche (Shopify).
type Client struct {
	cfg     Config
	http    *http.Client
	session *Session
}

// New construit un Client à partir d'une Config et charge la session locale.
func New(cfg Config) (*Client, error) {
	sess, err := LoadSession(cfg.SessionPath)
	if err != nil {
		return nil, fmt.Errorf("chargement session: %w", err)
	}
	return &Client{
		cfg:     cfg,
		http:    &http.Client{Timeout: 30 * time.Second},
		session: sess,
	}, nil
}

// Session expose l'état local (panier, jetons).
func (c *Client) Session() *Session { return c.session }

// storefrontEndpoint renvoie l'URL GraphQL Storefront.
func (c *Client) storefrontEndpoint() string {
	return fmt.Sprintf("https://%s/api/%s/graphql.json", c.cfg.ShopDomain, c.cfg.APIVersion)
}

// graphQLError porte les erreurs renvoyées par l'API GraphQL Shopify.
type graphQLError struct {
	Message    string `json:"message"`
	Extensions struct {
		Code           string `json:"code"`
		RequiredAccess string `json:"requiredAccess"`
	} `json:"extensions"`
}

func (e graphQLError) Error() string {
	if e.Extensions.Code != "" {
		return fmt.Sprintf("%s (%s)", e.Message, e.Extensions.Code)
	}
	return e.Message
}

// storefront exécute une requête GraphQL sur l'API Storefront et désérialise
// le champ "data" dans out.
func (c *Client) storefront(ctx context.Context, query string, vars map[string]any, out any) error {
	return c.graphql(ctx, c.storefrontEndpoint(), map[string]string{
		"X-Shopify-Storefront-Access-Token": c.cfg.StorefrontToken,
	}, query, vars, out)
}

// graphql est le helper bas niveau partagé (envoie {query, variables}).
func (c *Client) graphql(ctx context.Context, endpoint string, headers map[string]string, query string, vars map[string]any, out any) error {
	return c.graphqlBody(ctx, endpoint, headers, map[string]any{"query": query, "variables": vars}, out)
}

// graphqlBody envoie un corps GraphQL arbitraire (ex: avec operationName).
func (c *Client) graphqlBody(ctx context.Context, endpoint string, headers map[string]string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(raw))
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphQLError  `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("réponse GraphQL invalide: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return envelope.Errors[0]
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("décodage data: %w", err)
		}
	}
	return nil
}
