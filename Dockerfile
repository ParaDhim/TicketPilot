FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /ticketpilot cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /seed cmd/seed/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /metrics cmd/metrics/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /ticketpilot ./ 
COPY --from=builder /seed ./ 
COPY --from=builder /metrics ./

EXPOSE 8080
CMD ["./ticketpilot"]
