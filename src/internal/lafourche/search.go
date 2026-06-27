package lafourche

import "context"

// Product est une vue simplifiée d'un produit La Fourche.
type Product struct {
	Title       string    `json:"title"`
	Handle      string    `json:"handle"`
	Vendor      string    `json:"vendor"`
	ProductType string    `json:"product_type"`
	URL         string    `json:"url"`
	Variants    []Variant `json:"variants"`
}

// Variant est une déclinaison achetable d'un produit (le VariantID sert au panier).
type Variant struct {
	VariantID string `json:"variant_id"`
	Title     string `json:"title"`
	SKU       string `json:"sku"`
	Price     string `json:"price"`
	CompareAt string `json:"compare_at_price,omitempty"`
	Currency  string `json:"currency"`
	Available bool   `json:"available"`
}

const searchQuery = `
query Search($q: String!, $first: Int!) {
  products(first: $first, query: $q) {
    edges {
      node {
        title
        handle
        vendor
        productType
        onlineStoreUrl
        variants(first: 10) {
          edges {
            node {
              id
              title
              sku
              availableForSale
              price { amount currencyCode }
              compareAtPrice { amount currencyCode }
            }
          }
        }
      }
    }
  }
}`

// SearchProducts recherche des produits via l'API Storefront.
func (c *Client) SearchProducts(ctx context.Context, query string, limit int) ([]Product, error) {
	if limit <= 0 {
		limit = 10
	}
	var resp struct {
		Products struct {
			Edges []struct {
				Node struct {
					Title          string `json:"title"`
					Handle         string `json:"handle"`
					Vendor         string `json:"vendor"`
					ProductType    string `json:"productType"`
					OnlineStoreURL string `json:"onlineStoreUrl"`
					Variants       struct {
						Edges []struct {
							Node struct {
								ID               string `json:"id"`
								Title            string `json:"title"`
								SKU              string `json:"sku"`
								AvailableForSale bool   `json:"availableForSale"`
								Price            money  `json:"price"`
								CompareAtPrice   *money `json:"compareAtPrice"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"variants"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"products"`
	}

	err := c.storefront(ctx, searchQuery, map[string]any{"q": query, "first": limit}, &resp)
	if err != nil {
		return nil, err
	}

	products := make([]Product, 0, len(resp.Products.Edges))
	for _, e := range resp.Products.Edges {
		n := e.Node
		p := Product{
			Title:       n.Title,
			Handle:      n.Handle,
			Vendor:      n.Vendor,
			ProductType: n.ProductType,
			URL:         n.OnlineStoreURL,
		}
		if p.URL == "" {
			p.URL = "https://" + c.cfg.ShopDomain + "/products/" + n.Handle
		}
		for _, ve := range n.Variants.Edges {
			v := ve.Node
			variant := Variant{
				VariantID: v.ID,
				Title:     v.Title,
				SKU:       v.SKU,
				Price:     v.Price.Amount,
				Currency:  v.Price.CurrencyCode,
				Available: v.AvailableForSale,
			}
			if v.CompareAtPrice != nil {
				variant.CompareAt = v.CompareAtPrice.Amount
			}
			p.Variants = append(p.Variants, variant)
		}
		products = append(products, p)
	}
	return products, nil
}

type money struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}
