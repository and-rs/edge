# Iridium Edge

# https://edge.baut.dev

Event-to-market intelligence for crypto and prediction-market users.

Iridium Edge watches news, posts, and links, groups them into real events, maps
those events to relevant markets and assets, then ranks what matters now.

Output is signal, not generic summaries:

- what happened
- why it matters
- what is moving
- what still may not be priced in

Target users: crypto speculators, anon operators, bot users, and
prediction-market degenerates.

Current scope: one strong wedge, one useful feed, one clean UX, something that
actually works.

## Stack

- `frontend/` — SolidStart frontend
- `backend/` — Go backend
- `proto/` — protobuf contracts
- Connect-RPC between frontend and backend

## Setup

```nu
just setup
```

## Development

Default ports:

- backend: `:8080`
- frontend: `3000`

Run both:

```nu
just dev
```

Run with alternate ports:

```nu
just dev ':18083' '3303'
```

`just dev` wires `VITE_API_BASE_URL` to chosen backend port and cleans up child
processes on exit.

## Codegen

```nu
just regen
```

## Checks

```nu
just check
```

## Build

```nu
just build-backend
just build-frontend
```

