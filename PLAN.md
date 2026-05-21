# Stint stack plan

## Goal

Personal hackathon-scale event-to-market intelligence for crypto and
prediction-market users.

## Core stack

- Frontend: SolidStart in `frontend/`
- Backend: Go in `backend/`
- RPC: Connect-RPC with protobuf contracts in `proto/`
- Database: Postgres
- AI: OpenAI first, provider-swappable later
- Payments: USDC on Arc/Circle via plain HTTP, swappable later
- Infra: self-hosted, no BaaS

## Architecture

- `frontend/`: signal feed, watchlists, premium gates, auth UI
- `backend/`: ingest, clustering, market mapping, scoring, thesis generation,
  entitlements
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
- Start rule-first: ticker cashtags, venue dictionaries, keyword mapping
- Add AI assist later if needed

### 4. Ranking

- Score by impact, novelty, confidence, recency decay
- Use deterministic weighted formula first
- Store score breakdown for debugging

### 5. Thesis generation

- AI summary per canonical event
- Output:
  - what happened
  - why it matters now
  - what is moving vs not priced in

### 6. Product surface

- Signal feed
- Saved watchlists
- Premium unlocks
- Basic USDC entitlement

## Hackathon scope

### Ship first

- Manual links + RSS ingest
- Event feed
- Dedupe
- Basic clustering
- Rule-based market mapping
- Ranking
- AI thesis
- Premium flag + simple entitlement record

### Skip or fake first

- Perfect X ingest
- Real-time push alerts
- Trading/execution automation
- Deep portfolio logic
- Multi-service architecture

## Backend shape

- One Go service
- Connect handlers for frontend
- Background pollers/workers in same binary
- SQL migrations
- Provider adapters behind interfaces
