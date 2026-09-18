package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	url := os.Getenv("OLLAMA_URL")
	if url == "" {
		url = "http://localhost:11434"
	}
	return &Client{
		BaseURL:    url,
		HTTPClient: &http.Client{},
	}
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
}

type GenerateResponse struct {
	Response string `json:"response"`
}

type LLMResult struct {
	Category  string `json:"category"`
	Urgency   string `json:"urgency"`
	RiskScore int    `json:"risk_score"`
	Reasoning string `json:"reasoning"`
}

func (c *Client) AnalyzeTicket(ctx context.Context, subject, body string) (*LLMResult, error) {
	prompt := fmt.Sprintf(`You are an autonomous support-ticket triage agent.
Read the following ticket and provide:
1. category (e.g. "billing", "technical", "spam", "duplicate", "other")
2. urgency (e.g. "low", "medium", "high")
3. risk_score: integer from 0 to 100 where higher means a higher risk of customer churn, fraud, or SLA breach.
4. reasoning: a one-line string explaining your choices.

Ticket Subject: %s
Ticket Body: %s

Respond ONLY with valid JSON matching exactly this schema:
{
  "category": "string",
  "urgency": "string",
  "risk_score": 0,
  "reasoning": "string"
}`, subject, body)

	reqBody := GenerateRequest{
		Model:  "llama3:latest",
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName != "" {
		reqBody.Model = modelName
	}

	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/generate", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var gResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gResp); err != nil {
		return nil, err
	}

	var result LLMResult
	if err := json.Unmarshal([]byte(gResp.Response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from LLM: %v (response text: %s)", err, gResp.Response)
	}

	return &result, nil
}
