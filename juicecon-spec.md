# JUICECON App Technical Specification

## Overview

JUICECON is a **dewpoint severity index** web application that translates raw meteorological data (dewpoint temperature) into a human-relatable discomfort scale. The inverse numbering (5 = least severe, 1 = most severe) mirrors DEFCON's "escalation" pattern.

This specification defines the MVP for a hosted web app built with Go and deployed on Fly.io.

---

## Background: What is JUICECON?

From Joe Tritschler, inventor of the JUICECON system:

> "This is my proprietary 'JUICECON' system of assigning qualitative descriptors to levels of dewpoint in degrees Fahrenheit; similar in concept to the 'DEFCON' system the military uses for defense readiness."

### The Scale

| Level | Dewpoint (°F) | Descriptor | Notes |
|-------|---------------|------------|-------|
| JC5 | 60-64 | Noticeable | Threshold of running A/C at night |
| JC4 | 65-69 | Miserable | — |
| JC3 | 70-72 | Unbearable | — |
| JC2 | 73-74 | Come the fuck on | — |
| JC1 | 75+ | The Ultimate | Very rare event in Ohio |
| — | Below 60 | All Clear | JUICECON not active |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      FRONTEND                           │
│         Static HTML/CSS/JS served by Go                 │
│   - Gets browser location (or manual entry)             │
│   - Calls our API                                       │
│   - Renders the JUICECON display                        │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│                    GO BACKEND                           │
│   - /api/juicecon?lat=X&lon=Y                          │
│   - /api/juicecon?zip=45678                            │
│   - Fetches dewpoint from weather API                   │
│   - Calculates JUICECON level                           │
│   - Returns JSON with level, dewpoint, descriptor       │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│               WEATHER DATA SOURCE                       │
│   National Weather Service API (api.weather.gov)        │
│   - Free, no API key required                           │
│   - US-focused (perfect for Ohio origins)               │
│   - Returns dewpoint in current conditions              │
└─────────────────────────────────────────────────────────┘
```

### Stack Decisions

| Choice | Reasoning |
|--------|-----------|
| **Go backend** | Single binary deployment, handles API abstraction |
| **NWS API** | Free, no key management, authoritative US weather data, has dewpoint |
| **Go serves frontend** | Single deployment artifact, no CORS headaches, simpler infrastructure |
| **Static frontend** | No framework overhead for a single-screen app—just clean HTML/CSS/JS |
| **Fly.io** | Free tier, easy deployment, great for small Go apps |

---

## API Contract

### Endpoint: `GET /api/juicecon`

**Query Parameters:**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `lat` | float | No* | Latitude |
| `lon` | float | No* | Longitude |
| `zip` | string | No* | US ZIP code (5-digit) |

*Must provide either `lat`+`lon` OR `zip`

**Success Response (200):**

```json
{
  "level": 3,
  "levelDisplay": "JUICECON 3",
  "dewpoint": 71.2,
  "descriptor": "Unbearable",
  "description": "The air has weight. You are breathing soup.",
  "location": {
    "city": "Fort Wayne",
    "state": "IN",
    "station": "KFWA"
  },
  "timestamp": "2025-01-07T14:32:00Z",
  "allClear": false
}
```

**All Clear Response (200):**

```json
{
  "level": null,
  "levelDisplay": "ALL CLEAR",
  "dewpoint": 52.4,
  "descriptor": "Comfortable",
  "description": "JUICECON protocols not currently active.",
  "location": {
    "city": "Fort Wayne",
    "state": "IN",
    "station": "KFWA"
  },
  "timestamp": "2025-01-07T14:32:00Z",
  "allClear": true
}
```

**Error Response (400/500):**

```json
{
  "error": "Unable to determine location from ZIP code",
  "code": "INVALID_ZIP"
}
```

---

## JUICECON Calculation Logic

```go
type JuiceconLevel struct {
    Level       *int   // nil when all clear
    Descriptor  string
    Description string
    AllClear    bool
}

func CalculateJuicecon(dewpointF float64) JuiceconLevel {
    switch {
    case dewpointF >= 75:
        return JuiceconLevel{
            Level:       intPtr(1),
            Descriptor:  "The Ultimate",
            Description: "A very rare event. This is not a drill.",
            AllClear:    false,
        }
    case dewpointF >= 73:
        return JuiceconLevel{
            Level:       intPtr(2),
            Descriptor:  "Come The Fuck On",
            Description: "Unacceptable. File complaints with the atmosphere.",
            AllClear:    false,
        }
    case dewpointF >= 70:
        return JuiceconLevel{
            Level:       intPtr(3),
            Descriptor:  "Unbearable",
            Description: "The air has weight. You are breathing soup.",
            AllClear:    false,
        }
    case dewpointF >= 65:
        return JuiceconLevel{
            Level:       intPtr(4),
            Descriptor:  "Miserable",
            Description: "Existence is damp. Consider relocation.",
            AllClear:    false,
        }
    case dewpointF >= 60:
        return JuiceconLevel{
            Level:       intPtr(5),
            Descriptor:  "Noticeable",
            Description: "A/C at night is now justified.",
            AllClear:    false,
        }
    default:
        return JuiceconLevel{
            Level:       nil,
            Descriptor:  "Comfortable",
            Description: "JUICECON protocols not currently active.",
            AllClear:    true,
        }
    }
}

func intPtr(i int) *int {
    return &i
}
```

---

## Frontend Design

### Aesthetic: "Serious Absurdity"

Think DEFCON war room meets weather.gov meets brutalist design. The joke is that we're treating dewpoint with military-grade seriousness.

### Wireframe

```
┌─────────────────────────────────────────────────────────────┐
│  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
│                                                             │
│                      JUICECON  3                            │
│                    ━━━━━━━━━━━━━━━                          │
│                      UNBEARABLE                             │
│                                                             │
│            "The air has weight. You are                     │
│                  breathing soup."                           │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  DEWPOINT         71.2°F                            │   │
│  │  LOCATION         Fort Wayne, IN                    │   │
│  │  STATION          KFWA                              │   │
│  │  UPDATED          2:32 PM EST                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│              [ Change Location ]                            │
│                                                             │
│  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
└─────────────────────────────────────────────────────────────┘
```

### Visual Elements

| Element | Treatment |
|---------|-----------|
| **Level number** | Massive, monospace font, color-coded by severity |
| **Descriptor** | All caps, letter-spaced, below the number |
| **Description** | Smaller, italicized, the humor payload |
| **Data panel** | Stark, tabular, left-aligned labels |
| **Color scheme** | Dark background, level colors from green (clear) → yellow (JC5) → orange → red → deep red (JC1) |

### Color Scale

```css
:root {
  --jc-clear: #22c55e;  /* green */
  --jc-5: #eab308;      /* yellow */
  --jc-4: #f97316;      /* orange */
  --jc-3: #ef4444;      /* red */
  --jc-2: #dc2626;      /* deeper red */
  --jc-1: #991b1b;      /* dark red, pulsing animation */
  
  --bg-dark: #0a0a0a;
  --text-primary: #fafafa;
  --text-secondary: #a1a1aa;
}
```

---

## NWS API Integration

The National Weather Service API is free but has a two-step lookup process.

### Step 1: Points Lookup

```
GET https://api.weather.gov/points/{lat},{lon}
```

Returns grid coordinates and the nearest observation station URL.

**Required Header:**
```
User-Agent: (juicecon.app, contact@example.com)
```

### Step 2: Latest Observation

```
GET https://api.weather.gov/stations/{stationId}/observations/latest
```

Returns current conditions including dewpoint (in Celsius, needs conversion).

### NWS Response Shape (relevant parts)

```json
{
  "properties": {
    "dewpoint": {
      "value": 21.7,
      "unitCode": "wmoUnit:degC"
    },
    "timestamp": "2025-01-07T14:32:00+00:00"
  }
}
```

**Conversion:** `°F = (°C × 9/5) + 32`

### ZIP Code → Coordinates

Embed a ZIP lookup table (~40KB JSON of US ZIP centroids) to keep the app dependency-free and fast.

---

## Project Structure

```
juicecon/
├── main.go                 # Entry point, server setup
├── go.mod
├── go.sum
├── fly.toml               # Fly.io config
├── Dockerfile
│
├── internal/
│   ├── juicecon/
│   │   └── calculator.go  # JUICECON level logic
│   ├── weather/
│   │   ├── nws.go         # NWS API client
│   │   └── types.go       # Response structs
│   ├── geo/
│   │   ├── zip.go         # ZIP code lookup
│   │   └── zips.json      # Embedded ZIP data
│   └── handler/
│       └── api.go         # HTTP handlers
│
├── static/
│   ├── index.html
│   ├── style.css
│   └── app.js             # Geolocation, API calls, DOM updates
│
└── README.md
```

---

## MVP Feature Checklist

| Feature | Priority | Status | Notes |
|---------|----------|--------|-------|
| Browser geolocation | P0 | ⬜ | Core flow |
| ZIP/city manual entry | P0 | ⬜ | Fallback when geo denied |
| JUICECON calculation | P0 | ⬜ | The whole point |
| NWS integration | P0 | ⬜ | Data source |
| Level-appropriate styling | P1 | ⬜ | Color coding by severity |
| JC1 pulsing animation | P2 | ⬜ | Drama |
| Error handling UI | P1 | ⬜ | "Unable to reach NWS" state |
| Loading state | P1 | ⬜ | While fetching |
| Refresh button | P2 | ⬜ | Manual update |
| Auto-refresh interval | P3 | ⬜ | v1.1 maybe |

---

## Deployment

### Fly.io Setup

```bash
# Install flyctl if needed
curl -L https://fly.io/install.sh | sh

# From project root
fly launch

# Deploy
fly deploy
```

### Environment Variables

None required for MVP (NWS API needs no key).

### Domain

TBD - options: juicecon.app, juicecon.io, thejuicecon.com

---

## Future Enhancements (Post-MVP)

- **Forecast mode** — Next 24-48 hours of JUICECON levels
- **Historical comparison** — "JC1 events per decade" trend data
- **Notifications** — Alert when crossing thresholds
- **Joe quotes** — Rotating commentary based on level
- **Social sharing** — "Fort Wayne is currently at JUICECON 2"
- **Seasonal adjustments** — Joe's proposed addendum to the scale

---

## Credits

JUICECON system invented by Joe Tritschler.

> "Shit is real."
