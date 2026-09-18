package agent

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/parasdhiman/ticketpilot/internal/db"
	"github.com/parasdhiman/ticketpilot/internal/llm"
)

type Agent struct {
	DB        *db.DB
	LLMClient *llm.Client
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

		newStatus := "responded"
		if action == "escalate" {
			newStatus = "escalated"
		} else if action == "close" {
			newStatus = "closed"
		}

		confidence := 0.95 // Arbitrary high default confidence for dummy usage
		err = a.DB.UpdateTicketAndAddDecision(ctx, tid, action, res.Reasoning, confidence, newStatus)
		if err != nil {
			log.Printf("Error updating ticket decision %s: %v", t.ID, err)
		} else {
			log.Printf("Ticket %s decided as %s", t.ID, action)
		}
	}
}
