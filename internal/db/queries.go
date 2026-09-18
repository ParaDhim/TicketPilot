package db

import (
	"context"
	"fmt"
	"time"

	"github.com/parasdhiman/ticketpilot/internal/graph/model"
)

func (db *DB) GetOpenTickets(ctx context.Context) ([]*model.Ticket, error) {
	rows, err := db.Pool.Query(ctx, "SELECT id, created_at, subject, body, status, category, urgency, risk_score, updated_at FROM tickets WHERE status = 'open'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*model.Ticket
	for rows.Next() {
		var t model.Ticket
		var id int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &createdAt, &t.Subject, &t.Body, &t.Status, &t.Category, &t.Urgency, &t.RiskScore, &updatedAt); err != nil {
			return nil, err
		}
		t.ID = fmt.Sprintf("%d", id)
		t.CreatedAt = createdAt.Format(time.RFC3339)
		t.UpdatedAt = updatedAt.Format(time.RFC3339)
		tickets = append(tickets, &t)
	}
	return tickets, nil
}

func (db *DB) UpdateTicketAndAddDecision(ctx context.Context, ticketID int, action, reasoning string, confidence float64, newStatus string) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO decisions (ticket_id, action, confidence, reasoning) VALUES ($1, $2, $3, $4)", ticketID, action, confidence, reasoning)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE tickets SET status = $1, updated_at = NOW() WHERE id = $2", newStatus, ticketID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (db *DB) SaveLLMOutput(ctx context.Context, ticketID int, category, urgency string, riskScore int) error {
	_, err := db.Pool.Exec(ctx, "UPDATE tickets SET category = $1, urgency = $2, risk_score = $3, updated_at = NOW() WHERE id = $4", category, urgency, riskScore, ticketID)
	return err
}

func (db *DB) GetTickets(ctx context.Context, status *string, category *string) ([]*model.Ticket, error) {
	query := "SELECT id, created_at, subject, body, status, category, urgency, risk_score, updated_at FROM tickets WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}
	if category != nil {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, *category)
		argIdx++
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*model.Ticket
	for rows.Next() {
		var t model.Ticket
		var id int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &createdAt, &t.Subject, &t.Body, &t.Status, &t.Category, &t.Urgency, &t.RiskScore, &updatedAt); err != nil {
			return nil, err
		}
		t.ID = fmt.Sprintf("%d", id)
		t.CreatedAt = createdAt.Format(time.RFC3339)
		t.UpdatedAt = updatedAt.Format(time.RFC3339)
		tickets = append(tickets, &t)
	}
	return tickets, nil
}

func (db *DB) GetDecisions(ctx context.Context, ticketID string) ([]*model.Decision, error) {
	query := "SELECT id, ticket_id, action, confidence, reasoning, overridden_by_human, created_at FROM decisions WHERE ticket_id = $1"
	rows, err := db.Pool.Query(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []*model.Decision
	for rows.Next() {
		var d model.Decision
		var id, tID int
		var createdAt time.Time
		if err := rows.Scan(&id, &tID, &d.Action, &d.Confidence, &d.Reasoning, &d.OverriddenByHuman, &createdAt); err != nil {
			return nil, err
		}
		d.ID = fmt.Sprintf("%d", id)
		d.TicketID = fmt.Sprintf("%d", tID)
		d.CreatedAt = createdAt.Format(time.RFC3339)
		decisions = append(decisions, &d)
	}
	return decisions, nil
}

func (db *DB) OverrideDecision(ctx context.Context, ticketID string, newAction string) (*model.Decision, error) {
	// Let's implement this logic: insert new decision or update the latest one?
	// Specs say: "(marks overridden_by_human = true and updates ticket status)".
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Update the latest decision
	var d model.Decision
	var id, tID int
	var createdAt time.Time

	err = tx.QueryRow(ctx, "UPDATE decisions SET action = $1, overridden_by_human = TRUE WHERE ticket_id = $2 ORDER BY created_at DESC LIMIT 1 RETURNING id, ticket_id, action, confidence, reasoning, overridden_by_human, created_at", newAction, ticketID).
		Scan(&id, &tID, &d.Action, &d.Confidence, &d.Reasoning, &d.OverriddenByHuman, &createdAt)
	if err != nil {
		return nil, err
	}
	d.ID = fmt.Sprintf("%d", id)
	d.TicketID = fmt.Sprintf("%d", tID)
	d.CreatedAt = createdAt.Format(time.RFC3339)

	// Update ticket status
	newStatus := "responded"
	if newAction == "escalate" {
		newStatus = "escalated"
	} else if newAction == "close" {
		newStatus = "closed"
	}

	_, err = tx.Exec(ctx, "UPDATE tickets SET status = $1, updated_at = NOW() WHERE id = $2", newStatus, ticketID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &d, nil
}
