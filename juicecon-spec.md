# DEWCON Technical Specification

## Overview

**DEWCON** (Atmospheric Moisture Assessment System) is a web application that translates raw dewpoint temperature into humorous severity indexes. It operates two sub-systems under one unified umbrella:

- **JUICECON** (dewpoint >= 60°F): Humidity severity. Levels descend (JC1 = worst, like DEFCON).
- **CCF / Cunningham Crackle Factor** (dewpoint <= 25°F): Dryness severity. Levels descend (CCF1 = worst, same convention).
- **Comfort zone** (26-59°F): Neither system active. All Clear.

The inverse numbering (1 = most severe, 5 = least severe) mirrors DEFCON's "escalation" pattern across both sub-systems.

Built with Go and deployed on Fly.io.

---

## Background

### JUICECON

From Joe Tritschler, inventor of the JUICECON system:

> "This is my proprietary 'JUICECON' system of assigning qualitative descriptors to levels of dewpoint in degrees Fahrenheit; similar in concept to the 'DEFCON' system the military uses for defense readiness."

### CCF (Cunningham Crackle Factor)

From Sheila Cunningham, inventor of the CCF system:

The CCF tracks the opposite end of the moisture spectrum — the desiccation zone. When dewpoint drops low enough, static shock risk escalates, skin cracks, and the atmosphere itself seems to crackle. The CCF quantifies this misery.

---

## The Scales

### JUICECON (High Dewpoint — Humidity)

| Level | Dewpoint (°F) | Descriptor | Description |
|-------|---------------|------------|-------------|
| JC1 | 75+ | The Ultimate | A very rare event. This is not a drill. |
| JC2 | 73-74 | Come The Fuck On | Unacceptable. File complaints with the atmosphere. |
| JC3 | 70-72 | Unbearable | The air has weight. You are breathing soup. |
| JC4 | 65-69 | Miserable | Existence is damp. Consider relocation. |
| JC5 | 60-64 | Noticeable | A/C at night is now justified. |
| — | Below 60 | All Clear | JUICECON protocols not currently active. |

### CCF (Low Dewpoint — Dryness)

| Level | Dewpoint (°F) | Descriptor | Description |
|-------|---------------|------------|-------------|
| CCF1 | 2 or below | Walking EMP | I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes! |
| CCF2 | 3-8 | Husk Hands | Hands are husks of their former selves. |
| CCF3 | 9-14 | Lotion Failure | This lotion is NOT cutting it. |
| CCF4 | 15-19 | Humidifier Check | Do we even have a working humidifier? |
| CCF5 | 20-25 | Cotton Mouth | Cotton mouth but no alcohol? Hmmmm. Saline nasal rinse under consideration. |
| — | Above 25 | All Clear | CCF protocols not currently active. |

### Comfort Zone (26-59°F)

Neither sub-system is active. The DEWCON system reports "All Clear" with descriptor "Comfortable" and the message: "All atmospheric moisture protocols inactive. Conditions nominal."

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      FRONTEND                           │
│         Static HTML/CSS/JS served by Go                 │
│   - Gets browser location (or manual ZIP entry)         │
│   - Calls /api/juicecon                                 │
│   - Renders the DEWCON display (JuiceCon or CCF)        │
│   - CSS theming via data-system / data-level on <body>  │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│                    GO BACKEND                           │
│   - /api/juicecon?lat=X&lon=Y                          │
│   - /api/juicecon?zip=45678                             │
│   - /api/juicecon?dewpoint=XX  (test mode)              │
│   - /healthz  (health check)                            │
│   - Fetches dewpoint from NWS API                       │
│   - Unified evaluator picks JuiceCon or CCF             │
│   - Returns JSON with level, dewpoint, descriptor       │
│   - In-memory cache (10-min TTL per location)           │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│               WEATHER DATA SOURCE                       │
│   National Weather Service API (api.weather.gov)        │
│   - Free, no API key required                           │
│   - US-focused (perfect for Ohio origins)               │
│   - 3-step lookup: points → stations → observation      │
│   - Station fallback: tries up to 5 stations            │
│   - Returns dewpoint and temperature in Celsius          │
└─────────────────────────────────────────────────────────┘
```

### Stack Decisions

| Choice | Reasoning |
|--------|-----------|
| **Go backend** | Single binary deployment, handles API abstraction |
| **NWS API** | Free, no key management, authoritative US weather data, has dewpoint |
| **Go serves frontend** | Single deployment artifact, no CORS headaches, simpler infrastructure |
| **Static frontend** | No framework overhead for a single-screen app — just clean HTML/CSS/JS |
| **Fly.io** | Easy deployment, great for small Go apps |
| **Embedded static files** | `go:embed` bakes HTML/CSS/JS into the binary |

---

## API Contract

### Endpoint: `GET /api/juicecon`

**Query Parameters:**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `lat` | float | No* | Latitude |
| `lon` | float | No* | Longitude |
| `zip` | string | No* | US ZIP code (5-digit) |
| `dewpoint` | float | No** | Test mode: override dewpoint value in °F |

*Must provide either `lat`+`lon` OR `zip` (unless using test mode).

**When `dewpoint` is provided, the NWS API is bypassed entirely. Location is reported as "Test Mode, SIM" with station "KTEST". Temperature is simulated as dewpoint + 10°F.

**Success Response (200):**

```json
{
  "activeSystem": "juicecon",
  "systemName": "JUICECON",
  "level": 3,
  "levelDisplay": "JUICECON 3",
  "dewpoint": 71.2,
  "temperature": 81.2,
  "descriptor": "Unbearable",
  "description": "The air has weight. You are breathing soup.",
  "location": {
    "city": "Fort Wayne",
    "state": "IN",
    "station": "KFWA"
  },
  "timestamp": "2025-07-15T14:32:00Z",
  "allClear": false
}
```

**CCF Active Response (200):**

```json
{
  "activeSystem": "ccf",
  "systemName": "CCF",
  "level": 3,
  "levelDisplay": "CCF 3",
  "dewpoint": 12.0,
  "temperature": 22.0,
  "descriptor": "Lotion Failure",
  "description": "This lotion is NOT cutting it.",
  "location": {
    "city": "Fort Wayne",
    "state": "IN",
    "station": "KFWA"
  },
  "timestamp": "2026-01-20T09:15:00Z",
  "allClear": false
}
```

**All Clear Response (200):**

```json
{
  "activeSystem": "none",
  "systemName": "",
  "level": null,
  "levelDisplay": "ALL CLEAR",
  "dewpoint": 52.4,
  "temperature": 62.4,
  "descriptor": "Comfortable",
  "description": "All atmospheric moisture protocols inactive. Conditions nominal.",
  "location": {
    "city": "Fort Wayne",
    "state": "IN",
    "station": "KFWA"
  },
  "timestamp": "2025-10-07T14:32:00Z",
  "allClear": true
}
```

**Test Mode Response (200):**

```json
{
  "activeSystem": "ccf",
  "systemName": "CCF",
  "level": 1,
  "levelDisplay": "CCF 1",
  "dewpoint": -5.0,
  "temperature": 5.0,
  "descriptor": "Walking EMP",
  "description": "I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes!",
  "location": {
    "city": "Test Mode",
    "state": "SIM",
    "station": "KTEST"
  },
  "timestamp": "2026-01-20T09:15:00Z",
  "allClear": false
}
```

**Response Field Reference:**

| Field | Type | Description |
|-------|------|-------------|
| `activeSystem` | string | `"juicecon"`, `"ccf"`, or `"none"` |
| `systemName` | string | `"JUICECON"`, `"CCF"`, or `""` (empty when comfort zone) |
| `level` | *int | 1-5 when a system is active, `null` when all clear |
| `levelDisplay` | string | e.g. `"JUICECON 3"`, `"CCF 1"`, or `"ALL CLEAR"` |
| `dewpoint` | float64 | Dewpoint in °F |
| `temperature` | *float64 | Air temperature in °F; `null` if NWS did not report it |
| `descriptor` | string | Human-readable severity label |
| `description` | string | The humor payload |
| `location` | object | `{ city, state, station }` |
| `timestamp` | string | ISO 8601 UTC timestamp of the observation |
| `allClear` | bool | `true` when dewpoint is in the comfort zone (26-59°F) |

**Error Response (400/500/502):**

```json
{
  "error": "Unable to retrieve weather data. Please try again.",
  "code": "WEATHER_API_ERROR"
}
```

Error codes: `INVALID_PARAMS`, `METHOD_NOT_ALLOWED`, `WEATHER_API_ERROR`.

---

### Endpoint: `GET /healthz`

Health check endpoint for uptime monitoring and Fly.io health probes.

**Response (200):**

```json
{
  "status": "ok",
  "uptime": "2h35m10s"
}
```

---

## Unified Evaluation Logic

The unified evaluator (`internal/index/system.go`) routes to the correct sub-system:

```
Dewpoint >= 60°F  →  JuiceCon calculator
Dewpoint 26-59°F  →  Comfort zone (all clear, no sub-system)
Dewpoint <= 25°F  →  CCF calculator
```

Both sub-systems return:
- `Level` (`*int`): pointer to the severity level (1-5), or `nil` for all clear
- `Descriptor`: human-readable label
- `Description`: the humor payload
- `AllClear`: boolean

The evaluator wraps these into a unified `Result` that adds `ActiveSystem`, `SystemName`, and `LevelDisplay`.

---

## NWS API Integration

The National Weather Service API is free, requires no API key, and provides authoritative US weather data.

### Three-Step Lookup

**Step 1: Points Lookup**

```
GET https://api.weather.gov/points/{lat},{lon}
```

Returns grid coordinates, the nearest city/state, and the observation stations URL.

**Step 2: Stations List**

```
GET {observationStationsURL}
```

Returns an ordered list of nearby observation stations.

**Step 3: Latest Observation (with fallback)**

```
GET https://api.weather.gov/stations/{stationId}/observations/latest
```

Returns current conditions including dewpoint and temperature in Celsius.

The client tries up to 5 stations in order. If a station returns no dewpoint data (some stations have stale or missing readings), it moves on to the next. This fallback behavior significantly improves reliability.

**Required Request Headers:**
```
User-Agent: (juicecon.app, contact@juicecon.app)
Accept: application/geo+json
```

**Conversion:** `°F = (°C × 9/5) + 32`

### Caching

Observations are cached in memory with a **10-minute TTL**. The cache key is the lat/lon rounded to 2 decimal places, so nearby requests share the same cached observation and do not result in redundant NWS API calls.

### ZIP Code Lookup

An embedded JSON file (`internal/geo/zips.json`) maps 33,121 US ZIP codes to lat/lon centroids. This keeps the app dependency-free and fast — no external geocoding service needed.

---

## Project Structure

```
juicecon/
├── main.go                    # Entry point, HTTP server, routes, embedded static files
├── go.mod
├── go.sum
├── fly.toml                   # Fly.io config
├── Dockerfile                 # Multi-stage alpine build
│
├── internal/
│   ├── index/
│   │   ├── system.go          # Unified DEWCON evaluator (picks JuiceCon or CCF)
│   │   └── system_test.go
│   ├── juicecon/
│   │   ├── calculator.go      # JUICECON level logic (humidity)
│   │   └── calculator_test.go
│   ├── ccf/
│   │   ├── calculator.go      # CCF level logic (dryness)
│   │   └── calculator_test.go
│   ├── weather/
│   │   ├── nws.go             # NWS API client (3-step lookup, station fallback)
│   │   ├── nws_test.go
│   │   ├── cache.go           # In-memory observation cache (10-min TTL)
│   │   ├── cache_test.go
│   │   └── types.go           # Observation struct, NWS response types
│   ├── geo/
│   │   ├── zip.go             # ZIP code → lat/lon lookup
│   │   ├── zip_test.go
│   │   └── zips.json          # Embedded ZIP data (33,121 entries)
│   ├── handler/
│   │   ├── api.go             # /api/juicecon HTTP handler
│   │   └── api_test.go
│   └── middleware/
│       ├── middleware.go       # Security headers, per-IP rate limiting
│       └── middleware_test.go
│
├── static/
│   ├── index.html             # Main page
│   ├── style.css              # Level-based color theming, animations
│   └── app.js                 # Geolocation, API calls, DOM updates
│
└── juicecon-spec.md           # This file
```

---

## Frontend Design

### Aesthetic: "Serious Absurdity"

Think DEFCON war room meets weather.gov meets brutalist design. The joke is that we're treating dewpoint with military-grade seriousness.

### CSS Theming

The frontend sets `data-system` and `data-level` attributes on `<body>` to drive color theming:

- `data-system`: `"juicecon"`, `"ccf"`, or `"none"`
- `data-level`: `"1"` through `"5"`, or `"clear"`

### Color Scales

**JuiceCon (warm — yellows to reds):**

```css
--jc-clear: #22c55e;  /* green */
--jc-5: #eab308;      /* yellow */
--jc-4: #f97316;      /* orange */
--jc-3: #ef4444;      /* red */
--jc-2: #dc2626;      /* deeper red */
--jc-1: #991b1b;      /* dark red — pulsing animation */
```

**CCF (cool — blues to purples):**

```css
--ccf-clear: #22c55e;  /* green */
--ccf-5: #7dd3fc;      /* light blue */
--ccf-4: #38bdf8;      /* blue */
--ccf-3: #6366f1;      /* indigo */
--ccf-2: #8b5cf6;      /* violet */
--ccf-1: #c084fc;      /* purple — crackle animation */
```

### Animations

- **JC1**: Pulsing opacity animation (slow, ominous breathing)
- **CCF1**: Crackle animation (rapid text-shadow jitter simulating static discharge)

### Visual Elements

| Element | Treatment |
|---------|-----------|
| **Level number** | Massive (140px), monospace font, color-coded by severity |
| **System prefix** | All caps, letter-spaced — shows "JUICECON", "CCF", or "DEWCON" |
| **Descriptor** | All caps, letter-spaced, below the number |
| **Description** | Smaller, italicized, the humor payload |
| **Data panel** | Stark, tabular — protocol, temperature, dewpoint, location, station, updated |
| **Background** | Dark (#0a0a0a), monospace everything |

---

## Middleware

### Security Headers

Applied to all routes:
- `Content-Security-Policy`: `default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'`
- `X-Content-Type-Options`: `nosniff`
- `X-Frame-Options`: `DENY`
- `Referrer-Policy`: `strict-origin-when-cross-origin`

### Rate Limiting

Applied to `/api/` routes only. Per-IP rate limiting: 60 requests per minute. Stale visitor entries are cleaned up every 5 minutes.

---

## Test Mode

Append `?dewpoint=XX` to the API endpoint (or `?_dewpoint=XX` in the browser URL) to bypass the NWS API entirely and simulate any dewpoint value.

This is useful for:
- Testing specific JUICECON or CCF levels
- Verifying threshold boundaries
- Demonstrating the system without waiting for real weather conditions
- Frontend development

Examples:
- `/api/juicecon?dewpoint=76` — JC1 (The Ultimate)
- `/api/juicecon?dewpoint=45` — All Clear (comfort zone)
- `/api/juicecon?dewpoint=0` — CCF1 (Walking EMP)
- `/?_dewpoint=73` — Browser test mode, shows JC2

---

## Deployment

### Fly.io

App name: `juicecon`, region: `ord` (Chicago).

```bash
fly deploy
```

Also deploys via GitHub Actions on push to `main`.

### Dockerfile

Multi-stage alpine build. Produces a single static binary.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server listen port |

No other environment variables required (NWS API needs no key).

---

## Future Enhancements

- **Forecast mode** — Next 24-48 hours of DEWCON levels
- **Historical comparison** — "JC1 events per decade" trend data
- **Notifications** — Alert when crossing thresholds
- **Joe quotes** — Rotating commentary based on level
- **Social sharing** — "Fort Wayne is currently at JUICECON 2"
- **Seasonal adjustments** — Joe's proposed addendum to the scale

---

## Credits

JUICECON system created by Joe Tritschler.
Cunningham Crackle Factor created by Sheila Cunningham.
App designed and developed by Dan Branstrator.

> "Shit is real."
