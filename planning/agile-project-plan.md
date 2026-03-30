# DEWCON Agile Project Plan

**Last updated:** 2026-03-30
**Maintainer:** Dan Branstrator (@yoladango)
**Cadence:** 2-week sprints, solo developer, part-time (~10-15 hrs/week)

---

## 1. Project Overview

DEWCON is a Go web application that converts dewpoint temperature into humorous severity indexes. It comprises two sub-systems under a unified umbrella:

- **JUICECON** (dewpoint >= 60 F): Humidity severity on a descending scale (JC1 = worst).
- **CCF / Cunningham Crackle Factor** (dewpoint <= 25 F): Dryness severity on a descending scale (CCF1 = worst).
- **Comfort Zone** (26-59 F): Neither system active.

### Current State
- Go 1.23, stdlib only, zero external dependencies
- Frontend: vanilla HTML/CSS/JS embedded via `go:embed`
- NWS API integration (3-step: points -> stations -> observations)
- Deployed on Fly.io (region: `ord`) via GitHub Actions
- **Zero automated tests, no CI gate before deploy**
- ZIP database covers only 736 of ~41,000 US ZIPs (1.8%)
- Recently completed: security hardening (SHA-pinned Actions, Dependabot, branch protection) and CCF scale inversion

---

## 2. Product Vision & Goals

**Vision:** The most absurdly professional dewpoint severity assessment system on the internet -- reliable, fast, and accessible.

**Goals for this planning cycle (12 weeks / 6 sprints):**

| Goal | Measurable Target |
|------|-------------------|
| Test coverage | >= 80% on `internal/` packages |
| CI/CD gate | No deploy without passing build + vet + test |
| ZIP coverage | Full US coverage (~41,000 ZIPs) |
| Response time | < 3s average (via NWS caching) |
| Accessibility | WCAG AA compliance on all text elements |
| Uptime reliability | Health check endpoint, graceful shutdown |

---

## 3. Team & Roles

This is a solo developer project. All roles collapse to one person:

| Role | Person | Responsibilities |
|------|--------|------------------|
| Developer / PM / QA | Dan Branstrator | Everything: planning, coding, testing, deploying, monitoring |

**Process implications:**
- No code review bottleneck, but self-review before merge is expected
- Sprint planning = 30 minutes of backlog grooming at sprint start
- Retrospective = brief journal entry at sprint end
- Branch protection enforces CI checks pass before merge to `main`

---

## 4. Definition of Done

A story is **done** when:

- [ ] Code compiles (`go build ./...`)
- [ ] No vet warnings (`go vet ./...`)
- [ ] All tests pass (`go test -race ./...`)
- [ ] New code has unit tests (table-driven where appropriate)
- [ ] Coverage does not decrease
- [ ] Changes deployed to production via CI/CD pipeline
- [ ] No regressions in existing functionality

For frontend stories, additionally:
- [ ] Tested in Chrome and Firefox
- [ ] No console errors
- [ ] Responsive on mobile viewport

---

## 5. Product Backlog

### Story Point Reference (Fibonacci)

| Points | Solo Dev Effort | Roughly |
|--------|----------------|---------|
| 1 | < 1 hour | Trivial |
| 2 | 1-3 hours | Half a day |
| 3 | 4-6 hours | A day |
| 5 | 7-12 hours | 2-3 days |
| 8 | 13-20 hours | 3-5 days |
| 13 | 20+ hours | Full sprint, consider splitting |

---

### Epic 1: Testing Foundation & CI/CD

> **Goal:** Go from zero tests and no CI gate to a solid testing baseline and a pipeline that prevents broken deploys.

| ID | Story | Points | Priority | Dependencies |
|----|-------|--------|----------|--------------|
| E1-1 | Unit tests for `juicecon.Calculate()` -- all threshold boundaries, nil level, edge cases | 2 | P0 | -- |
| E1-2 | Unit tests for `ccf.Calculate()` -- all threshold boundaries, nil level, edge cases | 2 | P0 | -- |
| E1-3 | Unit tests for `index.Evaluate()` -- system selection, comfort zone, boundary handoff | 2 | P0 | E1-1, E1-2 |
| E1-4 | Unit tests for `celsiusToFahrenheit()` and `LevelDisplay()` | 2 | P0 | -- |
| E1-5 | Add CI workflow step: `go vet`, `go build`, `go test -race -coverprofile` before deploy | 2 | P0 | E1-1 |
| E1-6 | Extract `HTTPDoer` interface from NWS client for testability | 2 | P1 | -- |
| E1-7 | Unit tests for NWS client using `httptest.Server` (success, 500, 429, timeout, nil dewpoint, empty stations) | 5 | P1 | E1-6 |
| E1-8 | HTTP handler tests using `httptest` (valid ZIP, invalid ZIP, missing params, NWS failure) | 5 | P1 | E1-6 |
| E1-9 | Unit tests for `geo.LookupZIP()` (valid, invalid, edge cases) | 2 | P1 | -- |
| E1-10 | Add coverage threshold gate to CI (80% for `internal/`) | 2 | P1 | E1-5 |

**Acceptance Criteria for Epic:**
- All `internal/` packages have test files
- CI pipeline runs on every push/PR and gates deploy
- Coverage report generated and threshold enforced
- `go test -race ./...` passes clean

---

### Epic 2: Core Infrastructure Improvements

> **Goal:** Make the application more robust, observable, and production-ready.

| ID | Story | Points | Priority | Dependencies |
|----|-------|--------|----------|--------------|
| E2-1 | Add `/healthz` endpoint (returns 200 + JSON with app version and uptime) | 2 | P1 | -- |
| E2-2 | Propagate `context.Context` through all NWS API calls | 2 | P1 | -- |
| E2-3 | Graceful shutdown with OS signal handling (`SIGTERM`, `SIGINT`) | 2 | P2 | -- |
| E2-4 | Fix temperature nil handling -- return explicit error instead of silent 0.0 F default | 2 | P2 | E1-7 |
| E2-5 | Sanitize error messages -- don't leak NWS URLs to clients | 2 | P2 | -- |
| E2-6 | Fix unchecked `w.Write()` errors in `main.go` | 1 | P2 | -- |
| E2-7 | Remove `/test` endpoint from production builds | 2 | P2 | -- |
| E2-8 | DRY up duplicate `intPtr` helper into a shared `internal/ptr` package | 1 | P3 | -- |
| E2-9 | Structured logging with `slog` | 5 | P3 | -- |
| E2-10 | Rename API to `/api/dewcon`, keep `/api/juicecon` as backward-compat alias | 5 | P3 | E1-8 |

**Acceptance Criteria for Epic:**
- Health check available for Fly.io monitoring
- NWS calls respect request cancellation via context
- No internal details leaked in error responses
- Application shuts down cleanly on deploy

---

### Epic 3: Data & Performance

> **Goal:** Full US ZIP coverage and dramatically faster response times.

| ID | Story | Points | Priority | Dependencies |
|----|-------|--------|----------|--------------|
| E3-1 | Expand embedded ZIP database to full US coverage (~41,000 ZIPs) | 5 | P1 | -- |
| E3-2 | Add in-memory NWS response cache with TTL (station observations, 10-15 min TTL) | 5 | P1 | E2-2 |
| E3-3 | Cache NWS points-to-station mapping (long TTL, rarely changes) | 2 | P1 | E3-2 |
| E3-4 | Add cache-hit/miss metrics to `/healthz` response | 2 | P2 | E3-2, E2-1 |

**Acceptance Criteria for Epic:**
- Any valid US ZIP code returns results
- Repeated requests for same area served from cache (< 100ms)
- Cache expires and refreshes automatically
- No stale data older than 15 minutes

---

### Epic 4: Security & Hardening

> **Goal:** Defense-in-depth for a publicly deployed application.

| ID | Story | Points | Priority | Dependencies |
|----|-------|--------|----------|--------------|
| E4-1 | Add security headers middleware (CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy) | 2 | P2 | -- |
| E4-2 | Add rate limiting middleware (per-IP, token bucket or fixed window) | 3 | P2 | -- |
| E4-3 | Run Docker container as non-root user | 1 | P2 | -- |
| E4-4 | Validate and sanitize all user inputs (ZIP format, coordinate ranges) | 2 | P2 | E1-8 |

**Acceptance Criteria for Epic:**
- Security headers present on all responses (verifiable via curl)
- Rate limiting returns 429 when threshold exceeded
- Docker container runs as UID 1000+
- Invalid inputs return clear 400 errors, not 500s

---

### Epic 5: User Experience & Accessibility

> **Goal:** Make DEWCON usable by everyone and polished on all devices.

| ID | Story | Points | Priority | Dependencies |
|----|-------|--------|----------|--------------|
| E5-1 | Fix color contrast ratios to meet WCAG AA (4.5:1 minimum) | 2 | P2 | -- |
| E5-2 | Add `role="dialog"` and `aria-modal` to all modals | 1 | P2 | -- |
| E5-3 | Add focus trap to modals (tab cycling, Escape to close) | 2 | P2 | E5-2 |
| E5-4 | Add `aria-live` regions for loading states and result updates | 2 | P2 | -- |
| E5-5 | Auto-refresh interval on frontend (configurable, default 15 min) | 2 | P2 | -- |
| E5-6 | PWA manifest and meta tags | 2 | P3 | -- |
| E5-7 | Frontend keyboard navigation audit and fixes | 5 | P3 | E5-3 |

**Acceptance Criteria for Epic:**
- All text passes WCAG AA contrast ratio check
- Modals are fully keyboard-navigable and screen-reader friendly
- Loading states announced to assistive technology
- Auto-refresh keeps data current without manual interaction

---

### Epic 6: Feature Enhancements

> **Goal:** Expand DEWCON's capabilities once the foundation is solid.

| ID | Story | Points | Priority | Dependencies |
|----|-------|--------|----------|--------------|
| E6-1 | Update `juicecon-spec.md` to match current system (CCF inversion, unified model) | 2 | P2 | -- |
| E6-2 | Social sharing (Open Graph meta tags, share button with pre-filled text) | 5 | P3 | -- |
| E6-3 | City/name-based location search (autocomplete against ZIP database) | 8 | P3 | E3-1 |
| E6-4 | Forecast mode -- NWS forecast API integration for upcoming conditions | 8 | P3 | E2-2, E3-2 |
| E6-5 | Staging environment on Fly.io | 5 | P3 | E1-5 |

**Acceptance Criteria for Epic:**
- Spec document is accurate and current
- Shared links render with proper preview cards
- Users can search by city name or ZIP
- Forecast shows upcoming DEWCON level changes

---

## 6. Sprint Plan

### Sprint 1: Testing Foundation (Weeks 1-2)

**Theme:** Get from zero tests to tested core logic and a CI gate.

| Story | Points | Notes |
|-------|--------|-------|
| E1-1: Unit tests for `juicecon.Calculate()` | 2 | Table-driven, all boundaries |
| E1-2: Unit tests for `ccf.Calculate()` | 2 | Table-driven, all boundaries |
| E1-3: Unit tests for `index.Evaluate()` | 2 | System selection + comfort zone |
| E1-4: Unit tests for `celsiusToFahrenheit()`, `LevelDisplay()` | 2 | Utility functions |
| E1-5: CI workflow: vet + build + test before deploy | 2 | Modify `fly-deploy.yml` |
| E1-9: Unit tests for `geo.LookupZIP()` | 2 | Valid, invalid, edge ZIPs |

**Sprint Goal:** Core business logic is tested and no code reaches production without passing CI.
**Velocity:** 12 points

---

### Sprint 2: Testability & Infrastructure (Weeks 3-4)

**Theme:** Make the rest of the codebase testable and add production infrastructure.

| Story | Points | Notes |
|-------|--------|-------|
| E1-6: Extract `HTTPDoer` interface | 2 | Enables mocking NWS client |
| E1-7: NWS client tests with `httptest` | 5 | Error scenarios critical |
| E1-8: HTTP handler tests | 5 | Depends on E1-6 |
| E2-1: `/healthz` endpoint | 2 | Fly.io health checks |
| E2-2: Context propagation through NWS calls | 2 | Needed for cache + timeouts |

**Sprint Goal:** Full test coverage on HTTP layer, health check deployed.
**Velocity:** 16 points

---

### Sprint 3: Data & Performance (Weeks 5-6)

**Theme:** Full ZIP coverage and NWS response caching.

| Story | Points | Notes |
|-------|--------|-------|
| E3-1: Expand ZIP database to full US coverage | 5 | Source data, regenerate `zips.json` |
| E3-2: NWS response cache with TTL | 5 | In-memory, concurrent-safe |
| E3-3: Cache station mapping (long TTL) | 2 | Low-change data |
| E1-10: Coverage threshold gate (80%) | 2 | Should be achievable after Sprint 2 |
| E2-4: Fix temperature nil handling | 2 | Return error, not silent 0.0 |

**Sprint Goal:** Any US ZIP works, responses are fast on cache hit, coverage gate enforced.
**Velocity:** 16 points

---

### Sprint 4: Security & Hardening (Weeks 7-8)

**Theme:** Lock down the production application.

| Story | Points | Notes |
|-------|--------|-------|
| E4-1: Security headers middleware | 2 | CSP, X-Frame-Options, etc. |
| E4-2: Rate limiting middleware | 3 | Per-IP, reasonable defaults |
| E4-3: Non-root Docker container | 1 | Dockerfile change |
| E4-4: Input validation and sanitization | 2 | ZIP format, coordinate ranges |
| E2-3: Graceful shutdown | 2 | Signal handling |
| E2-5: Sanitize error messages | 2 | No URL leaks |
| E2-6: Fix unchecked `w.Write()` errors | 1 | Quick fix |
| E2-7: Remove `/test` endpoint from production | 2 | Quick fix |

**Sprint Goal:** Application is hardened against common attack vectors and runs securely.
**Velocity:** 15 points

---

### Sprint 5: Accessibility & UX (Weeks 9-10)

**Theme:** Make DEWCON accessible and polished.

| Story | Points | Notes |
|-------|--------|-------|
| E5-1: Fix color contrast ratios (WCAG AA) | 2 | Audit all themes/levels |
| E5-2: Modal ARIA attributes | 1 | `role="dialog"`, `aria-modal` |
| E5-3: Focus trap on modals | 2 | Tab cycling, Escape to close |
| E5-4: `aria-live` regions for loading | 2 | Announce state changes |
| E5-5: Auto-refresh interval | 2 | Frontend timer, configurable |
| E6-1: Update `juicecon-spec.md` | 2 | Bring spec current |
| E2-8: DRY up `intPtr` helper | 1 | Quick refactor |

**Sprint Goal:** DEWCON is fully accessible and keeps data fresh automatically.
**Velocity:** 12 points

---

### Sprint 6: Features & Polish (Weeks 11-12)

**Theme:** Add new capabilities and polish the experience.

| Story | Points | Notes |
|-------|--------|-------|
| E2-9: Structured logging with `slog` | 5 | Replace ad-hoc logging |
| E2-10: Rename API to `/api/dewcon` + alias | 5 | Backward compat required |
| E5-6: PWA manifest and meta tags | 2 | Installable on mobile |
| E6-2: Social sharing (OG tags + share button) | 5 | Preview cards on share |
| E3-4: Cache metrics on `/healthz` | 2 | Observability |

**Sprint Goal:** Improved observability, modern API naming, and shareable results.
**Velocity:** 19 points

---

### Backlog (Unprioritized / Future Sprints)

These stories are scoped but not scheduled. Pull them in if sprint capacity allows or plan them in a future cycle:

- E5-7: Frontend keyboard navigation audit (5 pts)
- E6-3: City/name-based location search (8 pts)
- E6-4: Forecast mode -- NWS forecast API (8 pts)
- E6-5: Staging environment (5 pts)

---

## 7. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| NWS API changes or goes down | Medium | High | Cache layer (E3-2) reduces dependency; degrade gracefully with last-known-good data |
| Full ZIP database inflates binary size significantly | Medium | Low | Measure binary size before/after; consider compressed embedded JSON or alternative lookup |
| Solo developer burnout or unavailability | Medium | High | Keep sprints realistic (10-15 hrs); backlog is prioritized so nothing critical is left half-done |
| Rate limiting misconfigured, blocks legitimate users | Low | Medium | Start with generous limits; add observability before tightening |
| Breaking NWS client refactor (interface extraction) | Low | Medium | Handler tests (E1-8) exist before major refactoring; CI gate catches regressions |
| Coverage threshold blocks deploys on urgent fixes | Low | Medium | Allow manual deploy override; keep threshold at 80% not 100% |
| Fly.io platform issues or pricing changes | Low | Medium | Application is a single binary with no vendor lock-in; can deploy anywhere |

---

## 8. Release Strategy

### Branching Model
- `main` is the production branch
- Feature work on short-lived branches (`feature/E1-1-juicecon-tests`, etc.)
- Branch protection requires CI to pass before merge

### Deployment Pipeline
```
Push to feature branch
    -> CI: go vet -> go build -> go test -race -coverprofile
    -> Coverage threshold check (80% internal/)
    -> PR review (self-review checklist)

Merge to main
    -> CI: same checks
    -> fly deploy (automated, only after CI passes)
```

### Release Cadence
- **Continuous deployment** to production on merge to `main`
- No versioned releases needed for a web app -- every merge is a release
- If a deploy breaks production, revert the merge commit and redeploy

### Rollback Plan
1. `fly releases` to identify the last good release
2. `fly deploy --image <previous-image>` for instant rollback
3. Investigate and fix on a branch, then re-deploy through normal pipeline

---

## Appendix: Story ID Quick Reference

| ID | Summary | Sprint |
|----|---------|--------|
| E1-1 | JuiceCon calculator tests | 1 |
| E1-2 | CCF calculator tests | 1 |
| E1-3 | Index evaluator tests | 1 |
| E1-4 | Utility function tests | 1 |
| E1-5 | CI gate before deploy | 1 |
| E1-6 | HTTPDoer interface extraction | 2 |
| E1-7 | NWS client tests | 2 |
| E1-8 | HTTP handler tests | 2 |
| E1-9 | ZIP lookup tests | 1 |
| E1-10 | Coverage threshold gate | 3 |
| E2-1 | Health check endpoint | 2 |
| E2-2 | Context propagation | 2 |
| E2-3 | Graceful shutdown | 4 |
| E2-4 | Temperature nil fix | 3 |
| E2-5 | Sanitize error messages | 4 |
| E2-6 | Fix unchecked Write errors | 4 |
| E2-7 | Remove /test endpoint | 4 |
| E2-8 | DRY intPtr helper | 5 |
| E2-9 | Structured logging (slog) | 6 |
| E2-10 | Rename API to /api/dewcon | 6 |
| E3-1 | Full US ZIP database | 3 |
| E3-2 | NWS response cache | 3 |
| E3-3 | Station mapping cache | 3 |
| E3-4 | Cache metrics on /healthz | 6 |
| E4-1 | Security headers | 4 |
| E4-2 | Rate limiting | 4 |
| E4-3 | Non-root Docker | 4 |
| E4-4 | Input validation | 4 |
| E5-1 | Color contrast fix | 5 |
| E5-2 | Modal ARIA attributes | 5 |
| E5-3 | Focus trap on modals | 5 |
| E5-4 | aria-live regions | 5 |
| E5-5 | Auto-refresh | 5 |
| E5-6 | PWA manifest | 6 |
| E5-7 | Keyboard nav audit | Backlog |
| E6-1 | Update spec document | 5 |
| E6-2 | Social sharing | 6 |
| E6-3 | City search | Backlog |
| E6-4 | Forecast mode | Backlog |
| E6-5 | Staging environment | Backlog |
