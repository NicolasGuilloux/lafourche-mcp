// Package mcpserver expose les fonctionnalités La Fourche en tant que serveur MCP.
package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/NicolasGuilloux/lafourche-mcp/internal/format"
	"github.com/NicolasGuilloux/lafourche-mcp/internal/lafourche"
)

// Version du serveur MCP.
const Version = "0.1.0"

// New construit le serveur MCP et enregistre les outils La Fourche.
func New(client *lafourche.Client) *server.MCPServer {
	s := server.NewMCPServer(
		"lafourche-mcp",
		Version,
		server.WithToolCapabilities(true),
		server.WithInstructions("Outils pour piloter La Fourche (épicerie bio): recherche produits, panier du compte, commandes."),
	)
	registerTools(s, client)
	return s
}

// ServeStdio lance le serveur sur le transport stdio.
func ServeStdio(s *server.MCPServer) error {
	return server.ServeStdio(s)
}

// ServeHTTP lance le serveur sur le transport HTTP streamable.
func ServeHTTP(s *server.MCPServer, addr string) error {
	httpSrv := server.NewStreamableHTTPServer(s)
	return httpSrv.Start(addr)
}

func registerTools(s *server.MCPServer, client *lafourche.Client) {
	s.AddTool(
		mcp.NewTool("search_products",
			mcp.WithDescription("Recherche des produits sur La Fourche par mots-clés (50 par page)."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Termes de recherche, ex: 'miel acacia'.")),
			mcp.WithNumber("page", mcp.Description("Numéro de page, 1-indexée (défaut 1).")),
			mcp.WithNumber("size", mcp.Description("Produits par page (défaut 50).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query := req.GetString("query", "")
			if query == "" {
				return mcp.NewToolResultError("le paramètre 'query' est requis"), nil
			}
			res, err := client.SearchProducts(ctx, query, req.GetInt("page", 1), req.GetInt("size", 0))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.ProductsMarkdown(res)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("basket_add",
			mcp.WithDescription("Ajoute un produit au panier du compte (nécessite d'être connecté)."),
			mcp.WithString("sku", mcp.Required(), mcp.Description("SKU du produit (issu de search_products), ex: '1-NTV-207'.")),
			mcp.WithNumber("quantity", mcp.Description("Quantité à ajouter (défaut 1).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sku := req.GetString("sku", "")
			if sku == "" {
				return mcp.NewToolResultError("le paramètre 'sku' est requis"), nil
			}
			basket, err := client.BasketAdd(ctx, sku, req.GetInt("quantity", 1))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.BasketMarkdown(basket)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("basket_remove",
			mcp.WithDescription("Retire un produit du panier (par défaut, retire tout)."),
			mcp.WithString("sku", mcp.Required(), mcp.Description("SKU du produit à retirer.")),
			mcp.WithNumber("quantity", mcp.Description("Quantité à retirer (défaut : tout).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sku := req.GetString("sku", "")
			if sku == "" {
				return mcp.NewToolResultError("le paramètre 'sku' est requis"), nil
			}
			basket, err := client.BasketRemove(ctx, sku, req.GetInt("quantity", 0))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.BasketMarkdown(basket)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("basket_get",
			mcp.WithDescription("Affiche le panier du compte (nécessite d'être connecté)."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			basket, err := client.GetBasket(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.BasketMarkdown(basket)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("user_info",
			mcp.WithDescription("Affiche les informations du compte connecté (nécessite d'être connecté)."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			info, err := client.UserInfo(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.UserInfoMarkdown(info)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("orders",
			mcp.WithDescription("Liste les dernières commandes (nécessite d'être connecté)."),
			mcp.WithNumber("limit", mcp.Description("Nombre max de commandes (défaut 10).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orders, err := client.LastOrders(ctx, req.GetInt("limit", 10))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.OrdersMarkdown(orders)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("order_detail",
			mcp.WithDescription("Affiche le détail (articles) d'une commande par son numéro."),
			mcp.WithString("number", mcp.Required(), mcp.Description("Numéro de commande, ex: '002955843'.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			number := req.GetString("number", "")
			if number == "" {
				return mcp.NewToolResultError("le paramètre 'number' est requis"), nil
			}
			order, err := client.GetOrder(ctx, number)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(format.OrderDetailMarkdown(order)), nil
		},
	)
}
