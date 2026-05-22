# Stint stack plan

## Goal

Hackathon-scale event-to-market intelligence for crypto and prediction-market operators.


## Core stack

- Frontend: SolidStart in `frontend/`
- Backend: Go in `backend/`
- RPC: Connect-RPC with protobuf contracts in `proto/`
- Database: Postgres
- AI: OpenAI first, provider-swappable later
- Payments: USDC on Arc/Circle via plain HTTP, swappable later
- Infra: self-hosted, no BaaS

## Architecture

- `frontend/`: homepage, product page, signal feed, premium gates, auth UI
- `backend/`: ingest, market mapping, scoring, thesis generation, entitlements
- `proto/`: shared API contracts for TS and Go
- Postgres stores raw items, canonical events, asset links, scores, theses, entitlements

## Feature slices

### 1. Intro and product surface

- Sharper homepage for intro, positioning, and demo entry
- Dedicated product page for signal feed, source coverage, market linkage, and ARC USDC path
- Deploy both early with stable demo path

### 2. Ingest and mapping

- RSS/news links first
- Add X/manual links next
- Store raw payload, source, timestamps, hash
- Rule-first market mapping with ticker, venue, keyword dictionaries

### 3. Ranking and thesis

- Score by impact, novelty, confidence, recency decay
- Use deterministic weighted formula first
- AI summary per event: what happened, why it matters now, what may be mispriced

### 4. Entitlements

- Premium unlocks
- Basic USDC entitlement
- ARC USDC integration is hackathon centerpiece

## Hackathon scope

### Ship first

- Homepage + dedicated product page
- Deploy both
- Manual links + RSS ingest
- Event feed
- Rule-based market mapping
- Ranking
- AI thesis
- ARC USDC entitlement path

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