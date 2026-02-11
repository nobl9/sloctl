package internal

import (
	"context"
	"log/slog"
	"os"

	"github.com/nobl9/nobl9-go/sdk"
)

func newMCPServer(client *sdk.Client) mcpServer {
	return mcpServer{
		client: client,
	}
}

type mcpServer struct {
	client *sdk.Client
}

func (s mcpServer) Start() error {
	slog.Info("Starting Nobl9 MCP proxy", "version", getBuildVersion())
	// Simply pass stdin/stdout to the SDK - it handles all the complexity
	return s.client.MCP().V1().ProxyStream(
		context.Background(),
		os.Stdin,
		os.Stdout,
	)
}
