# DEWCON - Atmospheric Moisture Assessment System

## What This Is
A Go web app that converts dewpoint temperature into humorous severity indexes. Two sub-systems under one unified "DEWCON" umbrella:

- **JUICECON** (dewpoint >= 60°F): Humidity severity. Levels descend (JC1 = worst, like DEFCON).
- **CCF / Cunningham Crackle Factor** (dewpoint <= 25°F): Dryness severity. Levels ascend (CCF5 = worst).
- **Comfort zone** (26-59°F): Neither system active.

## Architecture
- Pure Go stdlib, no frameworks. Single binary, embedded static files.
- `internal/index/system.go` is the unified evaluator — it picks JuiceCon or CCF based on dewpoint.
- `internal/juicecon/` and `internal/ccf/` are independent calculators.
- `internal/handler/api.go` serves `/api/juicecon` (kept for backward compat).
- `internal/weather/` fetches from the NWS API (3-step: points → stations → observations).
- `internal/geo/` does ZIP code → lat/lon lookup from embedded JSON.
- `static/` contains vanilla HTML/CSS/JS, embedded via `go:embed`.

## Key Conventions
- Level structs use `*int` (pointer) for the level number; `nil` means all clear.
- CSS theming uses `data-system` and `data-level` attributes on `<body>`.
- JuiceCon colors: warm (yellows → reds). CCF colors: cool (blues → purples).
- Keep the tone: serious presentation of absurd content. "War room meets weather app."

## Build & Verify
```
go build ./...
go vet ./...
```

## Deploy
Fly.io app `juicecon`, region `ord`. Deploys via GitHub Actions on push to `main`, or manually with `fly deploy`.

## Port
Default 8080, override with `PORT` env var.
