package format

import (
	"strings"
	"testing"

	"github.com/NicolasGuilloux/lafourche-mcp/internal/lafourche"
)

func TestProductsMarkdown(t *testing.T) {
	out := ProductsMarkdown([]lafourche.Product{{
		Title:  "Miel d'Acacia Bio",
		Vendor: "La Fourche",
		URL:    "https://shop.lafourche.fr/products/miel",
		Variants: []lafourche.Variant{{
			VariantID: "gid://shopify/ProductVariant/123",
			Price:     "9.98", CompareAt: "12.62", Currency: "EUR", Available: true,
		}},
	}})
	for _, want := range []string{"Miel d'Acacia Bio", "9.98 EUR", "au lieu de 12.62", "en stock", "gid://shopify/ProductVariant/123"} {
		if !strings.Contains(out, want) {
			t.Errorf("sortie manque %q:\n%s", want, out)
		}
	}
}

func TestProductsMarkdownEmpty(t *testing.T) {
	if got := ProductsMarkdown(nil); got != "Aucun produit trouvé." {
		t.Errorf("inattendu: %q", got)
	}
}

func TestBasketMarkdown(t *testing.T) {
	out := BasketMarkdown(&lafourche.Basket{
		TotalItems: 2, Total: "17.62", Currency: "EUR",
		Lines: []lafourche.BasketLine{{Name: "Miel", Quantity: 2, Subtotal: "17.62", SKU: "1-ABC"}},
	})
	for _, want := range []string{"2 article(s)", "17.62 EUR", "2 × Miel", "sku 1-ABC"} {
		if !strings.Contains(out, want) {
			t.Errorf("sortie manque %q:\n%s", want, out)
		}
	}
	if got := BasketMarkdown(nil); got != "Le panier est vide." {
		t.Errorf("panier nil: %q", got)
	}
}

func TestJSON(t *testing.T) {
	s, err := JSON(map[string]int{"a": 1})
	if err != nil || !strings.Contains(s, "\"a\": 1") {
		t.Fatalf("JSON: %q err=%v", s, err)
	}
}
