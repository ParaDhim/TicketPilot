package agent

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/parasdhiman/ticketpilot/internal/db"
	"github.com/parasdhiman/ticketpilot/internal/llm"
)

type Agent struct {
	DB        *db.DB
	LLMClient *llm.Client
	MCPClient *client.Client // newly added MCP client integration
	Interval  time.Duration
}

const ThresholdRiskScore = 75

func (a *Agent) Start(ctx context.Context) {
	log.Printf("Starting Agent Loop every %v", a.Interval)
	ticker := time.NewTicker(a.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Agent stopping...")
			return
		case <-ticker.C:
			a.processTickets(ctx)
		}
	}
}

func (a *Agent) processTickets(ctx context.Context) {
	tickets, err := a.DB.GetOpenTickets(ctx)
	if err != nil {
		log.Printf("Error fetching open tickets: %v", err)
		return
	}

	for _, t := range tickets {
		log.Printf("Processing ticket %s: %s", t.ID, t.Subject)

		res, err := a.LLMClient.AnalyzeTicket(ctx, t.Subject, t.Body)
		if err != nil {
			log.Printf("Error analyzing ticket %s: %v", t.ID, err)
			continue
		}

		tid, _ := strconv.Atoi(t.ID)
		err = a.DB.SaveLLMOutput(ctx, tid, res.Category, res.Urgency, res.RiskScore)
		if err != nil {
			log.Printf("Error saving LLM output for ticket %s: %v", t.ID, err)
		}

		var action string
		if res.RiskScore > ThresholdRiskScore {
			action = "escalate"
		} else if res.Category == "duplicate" || res.Category == "spam" {
			action = "close"
		} else {
			action = "respond" // Draft reply
		}

		// Connect via MCP to perform the action!
		err = a.callMCPAction(ctx, t.ID, action, res.Reasoning)
		if err != nil {
			log.Printf("Error calling MCP tool for ticket %s: %v", t.ID, err)
		} else {
			log.Printf("Ticket %s decided as %s via MCP", t.ID, action)
		}
	}
}

func (a *Agent) callMCPAction(ctx context.Context, ticketID, action, reasoning string) error {
	req := mcp.CallToolRequest{}

	args := make(map[string]interface{})
	args["ticket_id"] = ticketID

	switch action {
	case "escalate":
		req.Params.Name = "escalate_ticket"
		args["reason"] = reasoning
	case "close":
		req.Params.Name = "close_ticket"
		args["note"] = reasoning
	case "respond":
		req.Params.Name = "respond_to_ticket"
		args["reply"] = reasoning
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	req.Params.Arguments = args

	result, err := a.MCPClient.CallTool(ctx, req)
	if err != nil {
		return err
	}
	if result.IsError {
		return fmt.Errorf("MCP Tool Error: %v", result.Content)
	}
	return nil
}
