package graph

import (
	"context"

	"github.com/parasdhiman/ticketpilot/internal/db"
	"github.com/parasdhiman/ticketpilot/internal/graph/model"
)

type Resolver struct {
	DB *db.DB
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) OverrideDecision(ctx context.Context, ticketID string, newAction string) (*model.Decision, error) {
	return r.DB.OverrideDecision(ctx, ticketID, newAction)
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Tickets(ctx context.Context, status *string, category *string) ([]*model.Ticket, error) {
	return r.DB.GetTickets(ctx, status, category)
}

func (r *queryResolver) Decisions(ctx context.Context, ticketID string) ([]*model.Decision, error) {
	return r.DB.GetDecisions(ctx, ticketID)
}
