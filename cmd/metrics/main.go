package main

import (
	"context"
	"fmt"
	"log"

	"github.com/parasdhiman/ticketpilot/internal/db"
)

func main() {
	ctx := context.Background()
	d, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}

	rows, err := d.Pool.Query(ctx, `
		SELECT t.id, t.ground_truth_action, d.action
		FROM tickets t
		JOIN decisions d ON t.id = d.ticket_id
		WHERE t.ground_truth_action != ''
	`)
	if err != nil {
		log.Fatalf("Failed to query metrics: %v", err)
	}
	defer rows.Close()

	total := 0
	correct := 0

	actionStats := make(map[string]struct {
		TP int
		FP int
		FN int
	})

	for rows.Next() {
		var id int
		var truth, predicted string
		if err := rows.Scan(&id, &truth, &predicted); err != nil {
			log.Fatalf("Scan error: %v", err)
		}

		total++
		if truth == predicted {
			correct++
		}

		// Initialize stats if not present
		for _, action := range []string{"respond", "escalate", "close"} {
			if _, ok := actionStats[action]; !ok {
				actionStats[action] = struct{ TP, FP, FN int }{}
			}
		}

		if truth == predicted {
			st := actionStats[truth]
			st.TP++
			actionStats[truth] = st
		} else {
			stTrue := actionStats[truth]
			stTrue.FN++
			actionStats[truth] = stTrue

			stPred := actionStats[predicted]
			stPred.FP++
			actionStats[predicted] = stPred
		}
	}

	if total == 0 {
		fmt.Println("No labeled tickets with decisions found.")
		return
	}

	accuracy := float64(correct) / float64(total)
	fmt.Printf("Evaluated %d labeled tickets.\n", total)
	fmt.Printf("Overall Accuracy: %.2f%%\n\n", accuracy*100)

	for _, action := range []string{"respond", "escalate", "close"} {
		st := actionStats[action]
		var precision, recall float64
		if st.TP+st.FP > 0 {
			precision = float64(st.TP) / float64(st.TP+st.FP)
		}
		if st.TP+st.FN > 0 {
			recall = float64(st.TP) / float64(st.TP+st.FN)
		}
		fmt.Printf("Action '%s':\n", action)
		fmt.Printf("  Precision: %.2f%%\n", precision*100)
		fmt.Printf("  Recall:    %.2f%%\n\n", recall*100)
	}
}
