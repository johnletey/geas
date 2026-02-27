// geas-lsp is an LSP server for .eas (EVM assembly) files.
// It communicates via JSON-RPC 2.0 over stdio.
package main

import (
	"fmt"
	"os"

	"github.com/fjl/geas/internal/lsp"
)

func main() {
	server := lsp.NewServer(os.Stdin, os.Stdout)
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "geas-lsp: %v\n", err)
		os.Exit(1)
	}
}
