# rt-gateway

A room-based WebSocket gateway designed for high concurrency (C10k/C100k) and predictable behavior under load.
The goal of this project is not “just a chat server”, but a practical baseline for real-time systems:
clear concurrency ownership, bounded queues (backpressure), and measurable latency.

This repository evolves in milestones. Each milestone must be runnable, observable, and backed by a repeatable load test.

## Why this exists
Real-time backends fail for boring reasons:
- unbounded buffers that turn load into OOM
- slow consumers that stall the entire system
- shared mutable state causing race conditions or tail latency spikes
- “works locally” code with no measurement or profiling

`rt-gateway` exists to demonstrate the opposite:
- a clean connection model (single writer per connection)
- room/actor-style ownership (single-writer per room)
- explicit backpressure policy
- load tests + metrics + profiling as first-class citizens

## Non-goals (deliberately excluded)
This is a gateway/runtime core, not a full product:
- no authentication / accounts
- no persistence / message history
- no distributed routing / sharding (yet)
- no Kubernetes / microservices showcase
- no Redis/Kafka “because it looks enterprise”

Those can be added later, but they are not required to prove real-time competence.

## Architecture (high-level)
- **HTTP + WebSocket server** as the entry point.
- **Connection**: one read loop, one write loop (single-writer to the socket).
- **Rooms**: single-writer event loop per room (actor-style), broadcasts to subscribers.
- **Backpressure**: bounded outgoing queues; slow consumers are handled explicitly (drop/disconnect strategy).

## Milestones
- **M1 (Baseline)**: WS accept + echo + Prometheus metrics + graceful shutdown  
  Output: stable local server, observable connection counts.

- **M2 (Rooms)**: join/leave/publish protocol + room manager + room broadcast (single-writer)

- **M3 (Backpressure)**: bounded queues + slow consumer policy + disconnect counters + rate limits (if needed)

- **M4 (Load test & results)**: k6 scripts + results doc (p50/p95/p99) + CPU/heap profiling notes

## Endpoints
- `GET /healthz` -> `ok`
- `GET /metrics` -> Prometheus metrics
- `GET /ws` -> WebSocket endpoint

Default listen address: `:8083` (override with `RTG_ADDR`).

## Requirements
- Go 1.22+ (Linux)
- For high connection counts, raise file descriptor limits:
  - `ulimit -n 200000`

## Run (M1)
```bash
go mod tidy
go run ./cmd/server
```

## Quick test
curl -s http://127.0.0.1:8083/healthz
curl -s http://127.0.0.1:8083/metrics | grep rtgateway_connections_current

# ws echo
websocat -v ws://127.0.0.1:8083/ws
