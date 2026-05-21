# Stint stack plan

## Goal

Personal hackathon-scale tooling for fast event/news intelligence.

## Core stack

- Frontend: SolidStart in `frontend/`
- Backend: Go in `backend/`
- RPC: Connect-RPC with protobuf contracts in `proto/`
- Database: Postgres
- AI: OpenAI first, provider-swappable later
- Payments: USDC via HTTP-accessible services/libs
- Infra: self-hosted, no BaaS

## Architecture

- `frontend/`: feed, watchlists, premium gates, auth UI
- `backend/`: ingest, clustering, scoring, thesis generation, entitlements
- `proto/`: shared API contracts for TS and Go
- Postgres stores raw items, canonical events, asset links, scores, theses,
  watchlists, entitlements

## Feature slices

### 1. Ingest

- RSS/news links
- X/manual links
- Normalize into common source item shape
- Store raw payload, source, timestamps, hash

### 2. Dedupe and clustering

- Exact dedupe by normalized URL/content hash
- Soft dedupe by title similarity + source proximity + time window
- Cluster related source items into one canonical event

### 3. Event mapping

- Map event to assets, markets, sectors, watchlists
- Start rule-first: ticker cashtags, symbol dictionary, keyword mapping
- Add AI assist later if needed

### 4. Ranking

- Score by impact, novelty, confidence, recency decay
- Use deterministic weighted formula first
- Store score breakdown for debugging

### 5. Thesis generation

- AI summary per canonical event
- Output:
  - what happened
  - why it matters
  - what likely moves
- Keep short and structured

### 6. Product surface

- Alert feed
- Saved watchlists
- Premium unlock
- Basic USDC entitlement

## Hackathon scope

### Ship first

- Manual links + RSS ingest
- Event feed
- Dedupe
- Basic clustering
- Rule-based asset mapping
- Ranking
- AI thesis
- Premium flag + simple entitlement record

### Skip or fake first

- Perfect X ingest
- Real-time push alerts
- Complex onchain automation
- Deep portfolio logic
- Multi-service architecture

## Backend shape

- One Go service
- Connect handlers for frontend
- Background pollers/workers in same binary
- SQL migrations
- Provider adapters behind interfaces
