package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/minight/minight-terminal/internal/config"
	"github.com/minight/minight-terminal/internal/mcpserver"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	server := mcpserver.BuildMCPServer(cfg)
	return server.Run(context.Background(), &mcp.StdioTransport{})
}
