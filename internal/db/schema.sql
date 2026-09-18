CREATE TABLE IF NOT EXISTS tickets (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT '',
    urgency TEXT NOT NULL DEFAULT '',
    risk_score INT NOT NULL DEFAULT 0,
    ground_truth_action TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS decisions (
    id SERIAL PRIMARY KEY,
    ticket_id INT REFERENCES tickets(id),
    action TEXT NOT NULL,
    confidence FLOAT NOT NULL,
    reasoning TEXT NOT NULL,
    overridden_by_human BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
