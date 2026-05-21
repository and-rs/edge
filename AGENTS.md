# Stint agent instructions

## Stack

- Frontend in `frontend/`
- Backend in `backend/`
- Shared contracts in `proto/`
- Postgres for storage
- Connect-RPC for API
- Go for backend implementation

## Go rules

- `:=` is banned.
- Always use explicit `var` declarations or explicit typed declarations where
  possible.
- Prefer clarity over shorthand.
- Keep functions small.
- No unused code.
- No placeholder TODO comments.

## API rules

- Prefer protobuf contract changes in `proto/` first.
- Regenerate clients/servers after proto changes.
- Keep frontend and backend aligned through generated code.

## Architecture

- One backend service first.
- No microservice sprawl.
- Background jobs can live in same Go binary.
- Keep provider integrations swappable.
