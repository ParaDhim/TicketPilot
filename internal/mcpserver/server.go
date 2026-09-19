package mcpserver

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/parasdhiman/ticketpilot/internal/db"
)

type Server struct {
	MCPServer *server.MCPServer
	DB        *db.DB
}

func NewServer(d *db.DB) *Server {
	s := server.NewMCPServer("TicketPilot MCP", "1.0.0")

	srv := &Server{
		MCPServer: s,
		DB:        d,
	}

	// Register tools
	s.AddTool(mcp.NewTool("get_ticket_status",
		mcp.WithDescription("Fetch current status and metadata for a ticket by ID"),
		mcp.WithString("ticket_id",
			mcp.Required(),
			mcp.Description("The ID of the ticket"),
		),
	), srv.handleGetTicketStatus)

	s.AddTool(mcp.NewTool("escalate_ticket",
		mcp.WithDescription("Escalate a ticket with a reason"),
		mcp.WithString("ticket_id",
			mcp.Required(),
			mcp.Description("The ID of the ticket"),
		),
		mcp.WithString("reason",
			mcp.Required(),
			mcp.Description("The reasoning for escalating"),
		),
	), srv.handleEscalateTicket)

	s.AddTool(mcp.NewTool("close_ticket",
		mcp.WithDescription("Close a ticket with a resolution note"),
		mcp.WithString("ticket_id",
			mcp.Required(),
			mcp.Description("The ID of the ticket"),
		),
		mcp.WithString("note",
			mcp.Required(),
			mcp.Description("The resolution note or reason for closing"),
		),
	), srv.handleCloseTicket)

	s.AddTool(mcp.NewTool("respond_to_ticket",
		mcp.WithDescription("Draft a response to a ticket"),
		mcp.WithString("ticket_id",
			mcp.Required(),
			mcp.Description("The ID of the ticket"),
		),
		mcp.WithString("reply",
			mcp.Required(),
			mcp.Description("The drafted reply or reasoning"),
		),
	), srv.handleRespondTicket)

	s.AddTool(mcp.NewTool("override_classification",
		mcp.WithDescription("Manually override the agent's classification for a ticket"),
		mcp.WithString("ticket_id",
			mcp.Required(),
			mcp.Description("The ID of the ticket"),
		),
		mcp.WithString("new_action",
			mcp.Required(),
			mcp.Description("The new action to override with (e.g. escalate, close, respond)"),
		),
	), srv.handleOverrideClassification)

	return srv
}

func (s *Server) handleGetTicketStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	ticketIDStr, ok := args["ticket_id"].(string)
	if !ok {
		return mcp.NewToolResultError("ticket_id must be a string"), nil
	}

	tickets, dberr := s.DB.GetTickets(ctx, nil, nil)
	if dberr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DB error: %v", dberr)), nil
	}

	for _, t := range tickets {
		if t.ID == ticketIDStr {
			return mcp.NewToolResultText(fmt.Sprintf("Ticket %s: Subject: %s, Status: %s, Category: %s, Risk: %d", t.ID, t.Subject, t.Status, t.Category, t.RiskScore)), nil
		}
	}

	return mcp.NewToolResultError("Ticket not found"), nil
}

func (s *Server) handleEscalateTicket(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	ticketIDStr, _ := args["ticket_id"].(string)
	reason, _ := args["reason"].(string)

	tid, parseErr := strconv.Atoi(ticketIDStr)
	if parseErr != nil {
		return mcp.NewToolResultError("invalid ticket_id"), nil
	}

	err := s.DB.UpdateTicketAndAddDecision(ctx, tid, "escalate", reason, 0.95, "escalated")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DB failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ticket %s escalated successfully", ticketIDStr)), nil
}

func (s *Server) handleCloseTicket(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	ticketIDStr, _ := args["ticket_id"].(string)
	note, _ := args["note"].(string)

	tid, parseErr := strconv.Atoi(ticketIDStr)
	if parseErr != nil {
		return mcp.NewToolResultError("invalid ticket_id"), nil
	}

	err := s.DB.UpdateTicketAndAddDecision(ctx, tid, "close", note, 0.95, "closed")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DB failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ticket %s closed successfully", ticketIDStr)), nil
}

func (s *Server) handleRespondTicket(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	ticketIDStr, _ := args["ticket_id"].(string)
	reply, _ := args["reply"].(string)

	tid, parseErr := strconv.Atoi(ticketIDStr)
	if parseErr != nil {
		return mcp.NewToolResultError("invalid ticket_id"), nil
	}

	err := s.DB.UpdateTicketAndAddDecision(ctx, tid, "respond", reply, 0.95, "responded")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DB failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ticket %s responded successfully", ticketIDStr)), nil
}

func (s *Server) handleOverrideClassification(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}
	ticketIDStr, _ := args["ticket_id"].(string)
	newAction, _ := args["new_action"].(string)

	d, err := s.DB.OverrideDecision(ctx, ticketIDStr, newAction)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Override failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Ticket %s overridden to %s (Decision ID: %s)", ticketIDStr, d.Action, d.ID)), nil
}
