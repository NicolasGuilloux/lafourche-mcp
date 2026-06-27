package lafourche

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
)

// Le panier de La Fourche est le panier du compte, synchronisé mobile/web :
//   - source de vérité : Firestore carts/<shoppingCartId> = { SKU: quantité }
//   - enrichissement (noms, prix membres, total) : mutation createCart de l'API
//     api.lafourche.fr (les montants y sont en centimes).

// Basket est une vue simplifiée du panier du compte.
type Basket struct {
	TotalItems int          `json:"total_items"`
	Total      string       `json:"total"`
	Currency   string       `json:"currency"`
	Lines      []BasketLine `json:"lines,omitempty"`
}

// BasketLine est une ligne du panier.
type BasketLine struct {
	SKU       string `json:"sku"`
	Name      string `json:"name,omitempty"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price,omitempty"`
	Subtotal  string `json:"subtotal,omitempty"`
	Vendor    string `json:"vendor,omitempty"`
}

// shoppingCartID renvoie l'id du panier du compte (depuis Firestore), en cache.
func (c *Client) shoppingCartID(ctx context.Context) (string, error) {
	if c.session.ShoppingCartID != "" {
		return c.session.ShoppingCartID, nil
	}
	uid, err := c.userID(ctx)
	if err != nil {
		return "", err
	}
	fields, _, err := c.firestoreGet(ctx, "customers/"+uid)
	if err != nil {
		return "", err
	}
	v, ok := fields["shoppingCartId"]
	if !ok || v.StringValue == nil || *v.StringValue == "" {
		return "", fmt.Errorf("aucun panier associé au compte")
	}
	c.session.ShoppingCartID = *v.StringValue
	_ = c.session.Save()
	return c.session.ShoppingCartID, nil
}

// userID renvoie l'identifiant Firebase de l'utilisateur connecté.
func (c *Client) userID(ctx context.Context) (string, error) {
	if err := c.ensureToken(ctx); err != nil {
		return "", err
	}
	claims, err := parseFirebaseClaims(c.session.AccessToken)
	if err != nil {
		return "", err
	}
	if claims.UserID != "" {
		return claims.UserID, nil
	}
	if claims.Sub != "" {
		return claims.Sub, nil
	}
	return "", ErrNotAuthenticated
}

// fetchCartItems lit la map SKU -> quantité du panier Firestore.
func (c *Client) fetchCartItems(ctx context.Context, scid string) (map[string]int, error) {
	fields, status, err := c.firestoreGet(ctx, "carts/"+scid)
	if err != nil {
		return nil, err
	}
	items := make(map[string]int)
	if status == http.StatusNotFound {
		return items, nil
	}
	for sku, v := range fields {
		if v.IntegerValue == nil {
			continue
		}
		if q, err := strconv.Atoi(*v.IntegerValue); err == nil && q > 0 {
			items[sku] = q
		}
	}
	return items, nil
}

// GetBasket renvoie le panier du compte (enrichi noms/prix membres).
func (c *Client) GetBasket(ctx context.Context) (*Basket, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	scid, err := c.shoppingCartID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := c.fetchCartItems(ctx, scid)
	if err != nil {
		return nil, err
	}
	return c.enrichBasket(ctx, items)
}

// BasketAdd ajoute une quantité d'un SKU au panier du compte.
func (c *Client) BasketAdd(ctx context.Context, sku string, quantity int) (*Basket, error) {
	if quantity <= 0 {
		quantity = 1
	}
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	scid, err := c.shoppingCartID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := c.fetchCartItems(ctx, scid)
	if err != nil {
		return nil, err
	}
	if err := c.writeQuantity(ctx, scid, sku, items[sku]+quantity); err != nil {
		return nil, err
	}
	return c.GetBasket(ctx)
}

// BasketRemove retire une quantité d'un SKU (par défaut : retire tout).
func (c *Client) BasketRemove(ctx context.Context, sku string, quantity int) (*Basket, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	scid, err := c.shoppingCartID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := c.fetchCartItems(ctx, scid)
	if err != nil {
		return nil, err
	}
	cur, ok := items[sku]
	if !ok {
		return nil, fmt.Errorf("produit %s absent du panier", sku)
	}
	newQty := cur - quantity
	if quantity <= 0 {
		newQty = 0
	}
	if err := c.writeQuantity(ctx, scid, sku, newQty); err != nil {
		return nil, err
	}
	return c.GetBasket(ctx)
}

// writeQuantity fixe la quantité d'un SKU (0 => suppression du champ).
func (c *Client) writeQuantity(ctx context.Context, scid, sku string, qty int) error {
	if qty <= 0 {
		// Champ dans le mask mais absent du corps => supprimé.
		return c.firestorePatch(ctx, "carts/"+scid, []string{sku}, map[string]fsValue{})
	}
	s := strconv.Itoa(qty)
	return c.firestorePatch(ctx, "carts/"+scid, []string{sku},
		map[string]fsValue{sku: {IntegerValue: &s}})
}

// enrichBasket résout les noms et prix membres via createCart (montants en
// centimes). En cas d'échec, retombe sur un panier minimal (SKU + quantité).
func (c *Client) enrichBasket(ctx context.Context, items map[string]int) (*Basket, error) {
	if len(items) == 0 {
		return &Basket{Currency: "EUR"}, nil
	}
	list := make([]map[string]any, 0, len(items))
	for sku, q := range items {
		list = append(list, map[string]any{"sku": sku, "quantity": q})
	}
	query := `mutation CreateCart($cart: ClientCart!) {
  createCart(cart: $cart) {
    cart { total totalItems items { sku name quantity unitPrice subtotal productVendor } }
    errors { sku error message }
  }
}`
	var resp struct {
		CreateCart struct {
			Cart *struct {
				Total      int64 `json:"total"`
				TotalItems int   `json:"totalItems"`
				Items      []struct {
					SKU           string `json:"sku"`
					Name          string `json:"name"`
					Quantity      int    `json:"quantity"`
					UnitPrice     int64  `json:"unitPrice"`
					Subtotal      int64  `json:"subtotal"`
					ProductVendor string `json:"productVendor"`
				} `json:"items"`
			} `json:"cart"`
		} `json:"createCart"`
	}
	err := c.memberGraphQL(ctx, "CreateCart", query,
		map[string]any{"cart": map[string]any{"items": list}}, &resp)
	if err != nil || resp.CreateCart.Cart == nil {
		return basketFromItems(items), nil // dégradé : SKU + quantité
	}
	cart := resp.CreateCart.Cart
	b := &Basket{
		TotalItems: cart.TotalItems,
		Total:      centsToEuros(cart.Total),
		Currency:   "EUR",
	}
	for _, it := range cart.Items {
		b.Lines = append(b.Lines, BasketLine{
			SKU:       it.SKU,
			Name:      it.Name,
			Quantity:  it.Quantity,
			UnitPrice: centsToEuros(it.UnitPrice),
			Subtotal:  centsToEuros(it.Subtotal),
			Vendor:    it.ProductVendor,
		})
	}
	return b, nil
}

// basketFromItems construit un panier minimal (sans enrichissement).
func basketFromItems(items map[string]int) *Basket {
	b := &Basket{Currency: "EUR"}
	skus := make([]string, 0, len(items))
	for sku := range items {
		skus = append(skus, sku)
	}
	sort.Strings(skus)
	for _, sku := range skus {
		b.TotalItems += items[sku]
		b.Lines = append(b.Lines, BasketLine{SKU: sku, Quantity: items[sku]})
	}
	return b
}

func centsToEuros(c int64) string {
	return strconv.FormatFloat(float64(c)/100, 'f', 2, 64)
}
