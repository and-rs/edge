# Iridium Edge plan

## Goal

Hackathon-scale event-to-market intelligence for crypto and prediction-market operators.

## Core stack

- Frontend: SolidStart in `frontend/`
- Backend: Go in `backend/`
- RPC: Connect-RPC with split protobuf contracts in `proto/`
- AI: OpenAI SDK first, OpenAI-compatible providers supported
- Database: Postgres later; current test drive is live ingest + cache
- Payments: USDC on Arc/Circle via plain HTTP, swappable later
- Infra: self-hosted, no BaaS

## Architecture

- `frontend/`: home free scan, product Pro Signal Room, hackathon about, diagnostics footer
- `backend/`: diagnostics, live ingest, market matching, AI scoring, streaming signals
- `proto/`: generated TS/Go contracts for diagnostics and signals
- Current state caches home test-drive signals in browser session for CSS/UX iteration
- Postgres later stores raw items, events, market links, scores, theses, entitlements

## Pages

- Home: strong intro + gamified free Signal Hunt scan
- Product: Arc testnet USDC payment dynamic + premium gamified Pro Signal Room cards
- Hackathon about: project story, stack, roadmap with more time/money, builder context, useful implementation notes

## Feature slices

### 1. Signal Hunt free scan

- Home test drive becomes feedback-oriented, not dense analytics
- 2 event sources × 5 events each max
- Low-density cards: headline, source→market, one-line reason, confidence
- Three-step loop: find events → link markets → reveal edge
- Animations reward each stage: scanner meter, card flips, locked high-score pulse, CTA light-up
- Only 1 free thesis revealed; 2–3 strong cards blur/lock behind USDC

### 2. Pro Signal Room

- Arc-unlocked paid contrast: free finds smoke; paid explains tradeable fire
- 3 sources × 10–15 events each
- Planned sources: CoinDesk, CryptoPanic, Cointelegraph RSS or The Defiant RSS
- Venues: Kalshi + Polymarket API/snapshot fallback
- Unlocks deeper model, full thesis, catalyst timing, risk, market rationale, matched keywords, venue liquidity, alert/export
- Premium cards expand from teaser into full rationale after payment

### 3. Ingest and mapping

- CoinDesk RSS events live
- Kalshi market fetch live
- Rule-first market mapping live with typed match states
- Add second event source next, then Polymarket snapshot/API fallback
- Store raw payloads once Postgres lands

### 4. Ranking, thesis, and hook

- Rule score live
- AI thesis/judgment live with disabled/degraded modes
- Stream rule matches first, AI-enriched signals after model returns
- Hook mechanic uses progressive disclosure, completion tension, and instant feedback
- Next: tune scoring, novelty, confidence, recency decay, premium model copy

### 5. Entitlements

- Premium unlocks
- Basic USDC entitlement
- Arc USDC integration is hackathon centerpiece
- Payment must feel central: unlock richer analysis, more sources, more events, saved signal room

## Hackathon scope

### Shipped

- Homepage
- Diagnostics footer
- Split diagnostics/signals proto contracts
- Streamed signal test drive
- CoinDesk → Kalshi rule matching
- AI thesis path
- Frontend 1h session cache for iteration

### Immediate next

- Refactor `SignalTestDrive` into `frontend/src/components/signal-test-drive/`
- Suggested modules: `index.tsx`, `cache.ts`, `stage-card.tsx`, `signal-card.tsx`, `loading-stream.tsx`, `animations.ts`
- Standardize enter/reveal animations with homepage patterns
- Build Signal Hunt animations and locked-card hook
- Reduce home scan size and information density
- Add second event source
- Plan Product page Arc payment flow and Pro Signal Room contrast
- Add Hackathon About page structure

### Next after hook

- Dedicated Product page implementation
- Arc testnet USDC entitlement path
- Polymarket integration or snapshot fallback
- Postgres persistence
- Deploy both

### Skip or fake first

- Perfect X ingest
- Real-time push alerts outside current stream
- Trading/execution automation
- Deep portfolio logic
- Multi-service architecture

## Backend shape

- One Go service
- Connect handlers for frontend
- Signals support unary and server-streaming RPCs
- Keep backend cleanup limited until demo UX is strong
- Provider adapters behind interfaces
- SQL migrations when persistence starts