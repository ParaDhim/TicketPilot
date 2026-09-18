package main

import (
	"context"
	"fmt"
	"log"

	"github.com/parasdhiman/ticketpilot/internal/db"
)

type SeedTicket struct {
	Subject           string
	Body              string
	GroundTruthAction string // respond, escalate, close
}

func main() {
	ctx := context.Background()
	d, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}

	tickets := []SeedTicket{
		// 20 labeled for metrics
		{"Account locked and need urgent access", "I have a presentation in 20 minutes but my account is completely locked out. Unacceptable! Fix now!", "escalate"},
		{"How do I change my password?", "Hi, I forgot where the password reset page is. Can you point me to it?", "respond"},
		{"Invoice error", "My invoice #8922 is double charging me. Please resolve this immediately as it's a large amount of money.", "escalate"},
		{"Spam: You won a cruise", "Click here to claim your luxury vacation now!!!", "close"},
		{"Duplicate: Password reset", "Like I said in my other ticket, I need a password reset link.", "close"},
		{"Fraud alert on my account", "I am seeing weird transactions from a country I've never visited. Is my account compromised?", "escalate"},
		{"Feature request: dark mode", "Can you please add dark mode? My eyes hurt.", "respond"},
		{"Sales inquiry", "I want to upgrade to Enterprise, who can I talk to?", "respond"},
		{"Spam: SEO Services", "We can rank your website on page 1 of Google! Guaranteed results.", "close"},
		{"System down", "None of my colleagues can log in and the API returns 500 errors. We are losing money by the minute.", "escalate"},
		{"What is your refund policy?", "I couldn't find the refund policy on the website.", "respond"},
		{"Repeated billing issue", "This is the third time I'm complaining about this unauthorized charge. I will sue you if this is not fixed.", "escalate"},
		{"Need receipt for last month", "Could you send me the receipt for August? Thanks.", "respond"},
		{"Data breach question", "I saw on the news your competitor was hacked. Is my data safe with you?", "respond"},
		{"Spam: Buy cheap followers", "10k followers for $5!", "close"},
		{"Duplicate: feature request", "Just another reminder to please add dark mode.", "close"},
		{"App crashes on login", "Every time I enter my credentials the app just closes. I literally can't use the product.", "escalate"},
		{"Can I change my email?", "How do I update my profile email address?", "respond"},
		{"Urgent API rate limit", "Our production app is hitting a rate limit abruptly and we are down. Please lift the limit ASAP.", "escalate"},
		{"Love the app", "Just wanted to say thanks for the great tool!", "close"}, // Or respond, let's say "respond"
	}

	// generate 55 unlabeled
	for i := 0; i < 55; i++ {
		tickets = append(tickets, SeedTicket{
			Subject:           fmt.Sprintf("General inquiry #%d", i),
			Body:              fmt.Sprintf("I have a standard question about my account #%d.", i),
			GroundTruthAction: "",
		})
	}

	for _, t := range tickets {
		_, err := d.Pool.Exec(ctx, "INSERT INTO tickets (subject, body, status, ground_truth_action) VALUES ($1, $2, 'open', $3)", t.Subject, t.Body, t.GroundTruthAction)
		if err != nil {
			log.Printf("Failed to insert: %v", err)
		}
	}

	log.Println("Database seeded with ~75 tickets!")
}
