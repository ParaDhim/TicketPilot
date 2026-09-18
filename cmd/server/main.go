package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/parasdhiman/ticketpilot/internal/agent"
	"github.com/parasdhiman/ticketpilot/internal/db"
	"github.com/parasdhiman/ticketpilot/internal/graph"
	"github.com/parasdhiman/ticketpilot/internal/llm"
)

func main() {
	ctx := context.Background()
	d, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}

	llmClient := llm.NewClient()

	agt := &agent.Agent{
		DB:        d,
		LLMClient: llmClient,
		Interval:  10 * time.Second,
	}

	go agt.Start(ctx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	resolver := &graph.Resolver{DB: d}
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
