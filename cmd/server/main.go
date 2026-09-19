package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/parasdhiman/ticketpilot/internal/agent"
	"github.com/parasdhiman/ticketpilot/internal/db"
	"github.com/parasdhiman/ticketpilot/internal/graph"
	"github.com/parasdhiman/ticketpilot/internal/llm"
	"github.com/parasdhiman/ticketpilot/internal/mcpserver"
)

func main() {
	ctx := context.Background()
	d, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}

	// 1. Initialize MCP Server
	mcpSrv := mcpserver.NewServer(d)

	// 2. Initialize in-process MCP Client for the Agent
	mcpClient, err := client.NewInProcessClient(mcpSrv.MCPServer)
	if err != nil {
		log.Fatalf("Failed to create in-process MCP client: %v", err)
	}

	if err := mcpClient.Start(ctx); err != nil {
		log.Fatalf("Failed to start in-process MCP client: %v", err)
	}

	// Initialize the MCP client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "TicketPilot Agent",
		Version: "1.0.0",
	}

	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize MCP client: %v", err)
	}

	llmClient := llm.NewClient()

	agt := &agent.Agent{
		DB:        d,
		LLMClient: llmClient,
		MCPClient: mcpClient,
		Interval:  10 * time.Second,
	}

	go agt.Start(ctx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 3. Expose MCP Server over HTTP SSE
	mcpHTTPServer := server.NewStreamableHTTPServer(mcpSrv.MCPServer)
	http.Handle("/mcp/", mcpHTTPServer)

	resolver := &graph.Resolver{DB: d}
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Printf("connect to http://localhost:%s/mcp/sse for MCP Server", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
