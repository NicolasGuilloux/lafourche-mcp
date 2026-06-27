package lafourche

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// La recherche produits utilise l'index Algolia du site (`production_products`) :
// mêmes résultats que lafourche.fr, prix membres, et surtout le SKU directement
// utilisable par le panier.

// DefaultSearchPageSize est le nombre de produits par page de recherche.
const DefaultSearchPageSize = 50

// SearchResult est une page de résultats de recherche, avec sa pagination.
type SearchResult struct {
	Products    []Product `json:"products"`
	Page        int       `json:"page"`  // page courante, 1-indexée
	Pages       int       `json:"pages"` // nombre total de pages
	Total       int       `json:"total"` // nombre total de produits
	HitsPerPage int       `json:"hits_per_page"`
}

// Product est une vue simplifiée d'un produit La Fourche.
type Product struct {
	Title       string    `json:"title"`
	Handle      string    `json:"handle"`
	Vendor      string    `json:"vendor"`
	ProductType string    `json:"product_type"`
	URL         string    `json:"url"`
	Image       string    `json:"image,omitempty"`
	Variants    []Variant `json:"variants"`
}

// Variant est une déclinaison achetable d'un produit (le SKU sert au panier).
type Variant struct {
	SKU       string `json:"sku"`
	Title     string `json:"title"`
	Barcode   string `json:"barcode,omitempty"`
	Price     string `json:"price"`
	CompareAt string `json:"compare_at_price,omitempty"`
	Currency  string `json:"currency"`
	Available bool   `json:"available"`
}

// algoliaHit mappe les champs utiles d'un produit dans l'index Algolia.
type algoliaHit struct {
	SKU                string   `json:"sku"`
	Barcode            string   `json:"barcode"`
	Handle             string   `json:"handle"`
	Title              string   `json:"title"`
	Vendor             string   `json:"vendor"`
	ProductType        string   `json:"product_type"`
	Price              float64  `json:"price"`
	CompareAtPrice     *float64 `json:"compare_at_price"`
	InventoryAvailable bool     `json:"inventory_available"`
	Image              string   `json:"image"`
}

// SearchProducts recherche des produits via l'index Algolia de La Fourche.
// page est 1-indexée (défaut 1) ; size est le nombre de produits par page
// (défaut DefaultSearchPageSize = 50).
func (c *Client) SearchProducts(ctx context.Context, query string, page, size int) (*SearchResult, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = DefaultSearchPageSize
	}
	endpoint := fmt.Sprintf("https://%s-dsn.algolia.net/1/indexes/%s/query",
		c.cfg.AlgoliaAppID, url.PathEscape(c.cfg.AlgoliaIndex))

	params := url.Values{}
	params.Set("query", query)
	params.Set("hitsPerPage", strconv.Itoa(size))
	params.Set("page", strconv.Itoa(page-1)) // Algolia : 0-indexé
	body, _ := json.Marshal(map[string]string{"params": params.Encode()})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Algolia-Application-Id", c.cfg.AlgoliaAppID)
	req.Header.Set("X-Algolia-API-Key", c.cfg.AlgoliaAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("recherche Algolia: http %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		Hits        []algoliaHit `json:"hits"`
		NbHits      int          `json:"nbHits"`
		Page        int          `json:"page"`
		NbPages     int          `json:"nbPages"`
		HitsPerPage int          `json:"hitsPerPage"`
		Message     string       `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("réponse Algolia invalide: %w", err)
	}
	if out.Message != "" {
		return nil, fmt.Errorf("recherche Algolia: %s", out.Message)
	}

	products := make([]Product, 0, len(out.Hits))
	for _, h := range out.Hits {
		v := Variant{
			SKU:       h.SKU,
			Barcode:   h.Barcode,
			Price:     formatEuros(h.Price),
			Currency:  "EUR",
			Available: h.InventoryAvailable,
		}
		if h.CompareAtPrice != nil && *h.CompareAtPrice > h.Price {
			v.CompareAt = formatEuros(*h.CompareAtPrice)
		}
		products = append(products, Product{
			Title:       h.Title,
			Handle:      h.Handle,
			Vendor:      h.Vendor,
			ProductType: h.ProductType,
			URL:         MemberSiteURL + "/products/" + h.Handle,
			Image:       h.Image,
			Variants:    []Variant{v},
		})
	}
	return &SearchResult{
		Products:    products,
		Page:        out.Page + 1, // re-1-indexé
		Pages:       out.NbPages,
		Total:       out.NbHits,
		HitsPerPage: out.HitsPerPage,
	}, nil
}

func formatEuros(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
