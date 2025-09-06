# OrderPulse API (Go)
Minimal backend for the OrderPulse frontend.
Streams order events over **SSE/WS**, accepts **telemetry**, exposes **/healthz**, **/readyz**, and **/metrics**.

## Endpoints
- `GET /api/stream/events` → SSE stream (Bearer required). Supports `Last-Event-ID`, `?since=`, `?types=`, `?statuses=`.
- `GET /api/ws` → WebSocket stream (Bearer required).
- `POST /api/telemetry` → Error/metric ingestion (Bearer optional). 64KB body limit.
- `GET /healthz`, `GET /readyz`
- `GET /metrics` → Prometheus.

## Env
PORT=8080
CORS_ORIGINS=http://localhost:5173,http://localhost:3000
MOCK_ENABLED=true
JWT_HS256_SECRET=
BACKOFF_MAX=30s

## Run
go mod tidy
go run ./cmd/orderpulse-api

## Quick checks
curl -H "Authorization: Bearer demo" -H "Accept: text/event-stream" http://localhost:8080/api/stream/events
curl -H "Authorization: Bearer demo" -H "Accept: text/event-stream" "http://localhost:8080/api/stream/events?since=5m"
curl -X POST http://localhost:8080/api/telemetry -H "Authorization: Bearer demo" -H "Content-Type: application/json" -d '{"type":"error","message":"Bearer ABC","tags":{"url":"https://x?k=apiKey=Z"}}'

## Frontend
VITE_SSE_URL=http://localhost:8080/api/stream/events
VITE_WS_URL=ws://localhost:8080/api/ws

## Docker
docker build -t orderpulse-api:dev .
docker run --rm -p 8080:8080 --env-file .env orderpulse-api:dev

## Security
CORS allowlist, WS origin check, JWT, Request-ID, secure headers, rate limit, telemetry masking, 64KB limit.
EOF