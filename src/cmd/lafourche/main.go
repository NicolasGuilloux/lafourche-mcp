// Commande lafourche: CLI et serveur MCP pour piloter La Fourche.
package main

import (
	"fmt"
	"os"

	"github.com/NicolasGuilloux/lafourche-mcp/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "erreur:", err)
		os.Exit(1)
	}
}
