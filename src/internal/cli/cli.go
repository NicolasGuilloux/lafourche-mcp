// Package cli implémente la commande `lafourche` (CLI + serveur MCP).
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/NicolasGuilloux/lafourche-mcp/internal/format"
	"github.com/NicolasGuilloux/lafourche-mcp/internal/lafourche"
	"github.com/NicolasGuilloux/lafourche-mcp/internal/mcpserver"
)

type clientFactory func() (*lafourche.Client, error)

// Execute est le point d'entrée de la CLI.
func Execute() error {
	root := &cobra.Command{
		Use:           "lafourche",
		Short:         "CLI et serveur MCP pour piloter La Fourche (épicerie bio)",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       mcpserver.Version,
	}

	var outFormat string
	root.PersistentFlags().StringVar(&outFormat, "format", "text", "format de sortie : text|json")

	newClient := func() (*lafourche.Client, error) {
		return lafourche.New(lafourche.ConfigFromEnv())
	}

	root.AddCommand(
		loginCmd(newClient),
		logoutCmd(newClient),
		infoCmd(newClient, &outFormat),
		ordersCmd(newClient, &outFormat),
		searchCmd(newClient, &outFormat),
		basketCmd(newClient, &outFormat),
		mcpCmd(newClient),
	)
	return root.Execute()
}

// --- login / logout ---

func loginCmd(newClient clientFactory) *cobra.Command {
	var email, password string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Se connecter à La Fourche (email/mot de passe)",
		Long: "Se connecter à La Fourche par email/mot de passe (auth Firebase, sans navigateur).\n\n" +
			"Identifiants via --email/--password ou les variables LAFOURCHE_EMAIL/\n" +
			"LAFOURCHE_PASSWORD ; sinon ils sont demandés interactivement.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			if email == "" {
				email = envOr("LAFOURCHE_EMAIL", "")
			}
			if password == "" {
				password = envOr("LAFOURCHE_PASSWORD", "")
			}
			if email == "" {
				if email, err = promptLine("Email : "); err != nil {
					return err
				}
			}
			if password == "" {
				if password, err = promptPassword("Mot de passe : "); err != nil {
					return err
				}
			}
			if err := client.Login(cmd.Context(), email, password); err != nil {
				return err
			}
			fmt.Println("Connecté ✅")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "email du compte (ou LAFOURCHE_EMAIL)")
	cmd.Flags().StringVar(&password, "password", "", "mot de passe (ou LAFOURCHE_PASSWORD)")
	return cmd
}

func logoutCmd(newClient clientFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Se déconnecter (efface les jetons locaux)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			if err := client.Logout(); err != nil {
				return err
			}
			fmt.Println("Déconnecté.")
			return nil
		},
	}
}

// --- info ---

func infoCmd(newClient clientFactory, outFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Afficher les infos de l'utilisateur connecté",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			info, err := client.UserInfo(cmd.Context())
			if err != nil {
				return err
			}
			if *outFormat == "json" {
				return printJSON(info)
			}
			fmt.Println(format.UserInfoMarkdown(info))
			return nil
		},
	}
}

// --- orders ---

func ordersCmd(newClient clientFactory, outFormat *string) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "orders [number]",
		Short: "Lister les commandes, ou afficher le détail d'une commande",
		Long: "Sans argument : liste les dernières commandes.\n" +
			"Avec un numéro : affiche le détail (articles) de la commande.",
		Args:    cobra.MaximumNArgs(1),
		Example: "  lafourche orders\n  lafourche orders 002955843",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			// Détail d'une commande
			if len(args) == 1 {
				order, err := client.GetOrder(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if *outFormat == "json" {
					return printJSON(order)
				}
				fmt.Println(format.OrderDetailMarkdown(order))
				return nil
			}
			// Liste
			orders, err := client.LastOrders(cmd.Context(), limit)
			if err != nil {
				return err
			}
			if *outFormat == "json" {
				return printJSON(orders)
			}
			if len(orders) == 0 {
				fmt.Println("Aucune commande.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "COMMANDE\tDATE\tSTATUT\tTOTAL")
			for _, o := range orders {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s %s\n", o.Name, o.ProcessedAt, o.Status, o.Total, o.Currency)
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "nombre max de commandes (liste)")
	return cmd
}

// --- search ---

func searchCmd(newClient clientFactory, outFormat *string) *cobra.Command {
	var page, size int
	cmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Rechercher des produits (50 par page)",
		Args:    cobra.MinimumNArgs(1),
		Example: "  lafourche search miel acacia\n  lafourche search chocolat --page 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			res, err := client.SearchProducts(cmd.Context(), strings.Join(args, " "), page, size)
			if err != nil {
				return err
			}
			if *outFormat == "json" {
				return printJSON(res)
			}
			if len(res.Products) == 0 {
				fmt.Println("Aucun produit trouvé.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "PRIX\tDISPO\tPRODUIT\tSKU")
			for _, p := range res.Products {
				for _, v := range p.Variants {
					dispo := "✗"
					if v.Available {
						dispo = "✓"
					}
					fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\n", v.Price, v.Currency, dispo, p.Title, v.SKU)
				}
			}
			w.Flush()
			fmt.Printf("\nPage %d/%d — %d produit(s) au total", res.Page, res.Pages, res.Total)
			if res.Page < res.Pages {
				fmt.Printf(" (page suivante : --page %d)", res.Page+1)
			}
			fmt.Println()
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "numéro de page (1-indexée)")
	cmd.Flags().IntVar(&size, "size", lafourche.DefaultSearchPageSize, "produits par page")
	return cmd
}

// --- basket ---

func basketCmd(newClient clientFactory, outFormat *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "basket",
		Short: "Gérer le panier",
	}
	cmd.AddCommand(
		basketGetCmd(newClient, outFormat),
		basketAddCmd(newClient, outFormat),
		basketRemoveCmd(newClient, outFormat),
	)
	return cmd
}

func basketGetCmd(newClient clientFactory, outFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Afficher le contenu du panier",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			basket, err := client.GetBasket(cmd.Context())
			if err != nil {
				return err
			}
			if *outFormat == "json" {
				return printJSON(basket)
			}
			printBasket(basket)
			return nil
		},
	}
}

func basketAddCmd(newClient clientFactory, outFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:     "add <sku> [qty]",
		Short:   "Ajouter un produit au panier (qté par défaut : 1)",
		Args:    cobra.RangeArgs(1, 2),
		Example: "  lafourche basket add 1-NTV-207 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			qty := parseQty(args, 1)
			basket, err := client.BasketAdd(cmd.Context(), args[0], qty)
			if err != nil {
				return err
			}
			if *outFormat == "json" {
				return printJSON(basket)
			}
			fmt.Printf("Ajouté. Panier : %d article(s), total %s %s\n", basket.TotalItems, basket.Total, basket.Currency)
			return nil
		},
	}
}

func basketRemoveCmd(newClient clientFactory, outFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:     "remove <sku> [qty]",
		Short:   "Retirer un produit (par défaut : retire tout)",
		Args:    cobra.RangeArgs(1, 2),
		Example: "  lafourche basket remove 1-NTV-207\n  lafourche basket remove 1-NTV-207 1",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			qty := parseQty(args, 0) // 0 => tout retirer
			basket, err := client.BasketRemove(cmd.Context(), args[0], qty)
			if err != nil {
				return err
			}
			if *outFormat == "json" {
				return printJSON(basket)
			}
			fmt.Printf("Retiré. Panier : %d article(s), total %s %s\n", basket.TotalItems, basket.Total, basket.Currency)
			return nil
		},
	}
}

// --- mcp ---

func mcpCmd(newClient clientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Démarrer le serveur MCP (stdio)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			return mcpserver.ServeStdio(mcpserver.New(client))
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:     "http [addr]",
		Short:   "Démarrer le serveur MCP (Streamable HTTP, défaut :8080)",
		Args:    cobra.MaximumNArgs(1),
		Example: "  lafourche mcp http\n  lafourche mcp http :9000",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			addr := envOr("MCP_ADDR", ":8080")
			if len(args) == 1 {
				addr = args[0]
			}
			fmt.Fprintf(os.Stderr, "Serveur MCP HTTP sur %s\n", addr)
			return mcpserver.ServeHTTP(mcpserver.New(client), addr)
		},
	})
	return cmd
}

// --- helpers ---

func printBasket(basket *lafourche.Basket) {
	if basket == nil || len(basket.Lines) == 0 {
		fmt.Println("Panier vide.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "QTÉ\tPRODUIT\tSOUS-TOTAL\tSKU")
	for _, l := range basket.Lines {
		name := l.Name
		if name == "" {
			name = l.SKU
		}
		fmt.Fprintf(w, "%d\t%s\t%s %s\t%s\n", l.Quantity, name, l.Subtotal, basket.Currency, l.SKU)
	}
	w.Flush()
	fmt.Printf("\nTotal : %s %s\n", basket.Total, basket.Currency)
}

func parseQty(args []string, def int) int {
	if len(args) < 2 {
		return def
	}
	var q int
	if _, err := fmt.Sscanf(args[1], "%d", &q); err != nil {
		return def
	}
	return q
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func promptLine(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	s, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(s), err
}

func promptPassword(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return strings.TrimSpace(string(b)), err
}
