// Package format produit les représentations de sortie partagées par la CLI et
// le serveur MCP : JSON (machine) et Markdown concis (lisible par un LLM).
package format

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/NicolasGuilloux/lafourche-mcp/internal/lafourche"
)

// JSON sérialise n'importe quelle valeur en JSON indenté.
func JSON(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ProductsMarkdown rend une liste de produits dans un format adapté aux LLM.
// L'identifiant de variant est mis en avant car requis par add_to_cart.
func ProductsMarkdown(products []lafourche.Product) string {
	if len(products) == 0 {
		return "Aucun produit trouvé."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d produit(s) trouvé(s) :\n", len(products))
	for _, p := range products {
		for _, v := range p.Variants {
			name := p.Title
			if v.Title != "" && v.Title != "Default Title" {
				name += " — " + v.Title
			}
			fmt.Fprintf(&b, "\n- **%s**", name)
			if p.Vendor != "" {
				fmt.Fprintf(&b, " (%s)", p.Vendor)
			}
			fmt.Fprintf(&b, "\n  - prix : %s %s", price(v.Price), v.Currency)
			if v.CompareAt != "" && v.CompareAt != v.Price {
				fmt.Fprintf(&b, " (au lieu de %s %s)", price(v.CompareAt), v.Currency)
			}
			fmt.Fprintf(&b, "\n  - disponibilité : %s", dispo(v.Available))
			fmt.Fprintf(&b, "\n  - variant_id : `%s`", v.VariantID)
			if p.URL != "" {
				fmt.Fprintf(&b, "\n  - url : %s", p.URL)
			}
		}
	}
	return b.String()
}

// BasketMarkdown rend le panier du compte dans un format adapté aux LLM.
func BasketMarkdown(b *lafourche.Basket) string {
	if b == nil || len(b.Lines) == 0 {
		return "Le panier est vide."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Panier : %d article(s), total %s %s\n", b.TotalItems, price(b.Total), b.Currency)
	for _, l := range b.Lines {
		name := l.Name
		if name == "" {
			name = l.SKU
		}
		fmt.Fprintf(&sb, "\n- %d × %s", l.Quantity, name)
		if l.Subtotal != "" {
			fmt.Fprintf(&sb, " — %s %s", price(l.Subtotal), b.Currency)
		}
		fmt.Fprintf(&sb, " (sku %s)", l.SKU)
	}
	return sb.String()
}

// OrdersMarkdown rend les commandes dans un format adapté aux LLM.
func OrdersMarkdown(orders []lafourche.Order) string {
	if len(orders) == 0 {
		return "Aucune commande trouvée."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d commande(s) :\n", len(orders))
	for _, o := range orders {
		fmt.Fprintf(&b, "\n- **%s** (%s) — %s · total %s %s", o.Name, o.ProcessedAt, status(o.Status), price(o.Total), o.Currency)
		if o.Fulfillment != "" {
			fmt.Fprintf(&b, " · livraison : %s", o.Fulfillment)
		}
		for _, l := range o.Lines {
			fmt.Fprintf(&b, "\n  - %d × %s", l.Quantity, l.Title)
		}
	}
	return b.String()
}

// OrderDetailMarkdown rend le détail complet d'une commande.
func OrderDetailMarkdown(o *lafourche.Order) string {
	if o == nil {
		return "Commande introuvable."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Commande %s\n", o.Name)
	fmt.Fprintf(&b, "- date : %s\n", o.ProcessedAt)
	fmt.Fprintf(&b, "- statut : %s", status(o.Status))
	if o.Fulfillment != "" {
		fmt.Fprintf(&b, " · livraison : %s", o.Fulfillment)
	}
	fmt.Fprintf(&b, "\n- total : %s %s\n", price(o.Total), o.Currency)
	if o.ShipTo != "" {
		fmt.Fprintf(&b, "- adresse : %s\n", o.ShipTo)
	}
	if o.Tracking != "" || o.Carrier != "" {
		fmt.Fprintf(&b, "- suivi : %s %s\n", o.Carrier, o.Tracking)
	}
	if o.URL != "" {
		fmt.Fprintf(&b, "- lien : %s\n", o.URL)
	}
	fmt.Fprintf(&b, "\n## Articles (%d)", len(o.Lines))
	for _, l := range o.Lines {
		fmt.Fprintf(&b, "\n- %d × %s", l.Quantity, l.Title)
		if l.Price != "" {
			fmt.Fprintf(&b, " — %s %s", price(l.Price), o.Currency)
		}
		if l.Vendor != "" {
			fmt.Fprintf(&b, " (%s)", l.Vendor)
		}
	}
	return b.String()
}

// UserInfoMarkdown rend les infos du compte connecté.
func UserInfoMarkdown(u *lafourche.UserInfo) string {
	if u == nil {
		return "Non connecté."
	}
	var b strings.Builder
	b.WriteString("Compte connecté :\n")
	if u.Name != "" {
		fmt.Fprintf(&b, "- nom : %s\n", u.Name)
	}
	fmt.Fprintf(&b, "- email : %s%s\n", u.Email, verified(u.EmailVerified))
	fmt.Fprintf(&b, "- identifiant : %s\n", u.UserID)
	if u.Provider != "" {
		fmt.Fprintf(&b, "- méthode : %s\n", u.Provider)
	}
	fmt.Fprintf(&b, "- jeton valide jusqu'à : %s", u.ExpiresAt.Local().Format("2006-01-02 15:04"))
	return b.String()
}

func verified(ok bool) string {
	if ok {
		return " (vérifié)"
	}
	return " (non vérifié)"
}

func price(s string) string {
	if s == "" {
		return "0.00"
	}
	return s
}

func dispo(available bool) string {
	if available {
		return "en stock"
	}
	return "indisponible"
}

func status(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
