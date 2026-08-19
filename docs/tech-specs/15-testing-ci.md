# Testing & CI Hardening — Implementation Spec v1
Status: ready
Depends-on: none (process/infra spec — no DB migration owned; touches `db/migrations` only via an assertion test)
Migration owner: none — this spec does not add tables. It adds a test that asserts migrations apply cleanly against `test/helpers.go`'s in-memory DB.

0. Verified ground truth      file:line facts + grep proofs of current state
1. Overview / goal            1-para + explicit non-goals
2. API contract               CI is the "API"; no HTTP routes here
3. DB contract                none; migration-apply assertion test
4. UI                         none (Playwright e2e is headless browser smoke, described in §7)
5. Business logic             per-feature test strategy + pyramid
6. Config / env               CI secrets, .golangci.yml, sqlc pin
7. Tests                      coverage targets, Codecov yaml, Makefile hardening, 4 new CI jobs + YAML
8. Future / GPS-provider      Go benchmarks, perf CI, load test
9. Edge cases                 enumerated
10. Phased rollout            build order
11. Open items / VERIFY       decisions before coding
12. File list                 create / modify

---

## 0. Verified ground truth

All statements below were verified against the repo at `/home/abhishek/Desktop/temux/basic` on the working tree.

### Verification Log (principal-engineer QA pass — factual reconciliation)

| # | Claim | Verdict | Correction / Evidence | Sev | Effort |
|---|-------|---------|------------------------|-----|--------|
| 1 | ci.yml has only 3 jobs (test/lint/build) | TRUE | jobs at ci.yml:10 / :51 / :68 | — | — |
| 2 | No coverage gate in ci.yml | TRUE | no `-cover`/Codecov step present | — | — |
| 3 | No gosec/govulncheck/staticcheck in ci.yml; lint = `golangci-lint run --timeout=5m` | TRUE | ci.yml:66; `ls .golangci.yml` → not found | — | — |
| 4 | Makefile `check:` = `check-fmt vet build staticcheck test-race check-security` | TRUE | Makefile:61 | — | — |
| 5 | `check-security` silently skips gosec+govulncheck | TRUE | Makefile:72-76 `echo "…skipped (not installed)"` | — | — |
| 6 | `staticcheck` exits 1 if binary missing | TRUE | Makefile:69 | — | — |
| 7 | `NewTestDB` opens named in-memory SQLite + goose migrations | TRUE | test/helpers.go:21 / :29 / :35 | — | — |
| 8 | 275 `func Test` in repo | TRUE | `grep -rn "func Test" --include=*.go .` = 275 | — | — |
| 9 | 13 top-level internal dirs have zero `_test.go` | TRUE (caveat) | exactly 13 confirmed; of these, `static` & `templates` are non-Go asset dirs (no `.go` at all), so 11 are real Go packages | Low | — |
| 10 | Playwright orphaned: `e2e-test.js` outside `testDir` | TRUE | `e2e-test.js` at repo root; `playwright.config.ts` `testDir:'./test'`; not in any CI job | — | — |
| 11 | `go 1.26` in go.mod | TRUE | go.mod:3 | — | — |
| 12 | sqlc unpinned (`@latest`) in ci.yml | TRUE | ci.yml:43 `go install …/sqlc@latest` | — | — |
| 13 | §3 migration test tables `company_config`/`outbox`/`telemetry_devices` + nums `00040`/`00042`/`00055` | **FALSE** | real schema: `company_settings` (00001), `outbox_events` (00020); **no** `company_config`/`telemetry_devices` table; migrations stop at **00039**. Proposed test would FAIL 3/5 assertions. | High | M |
| 14 | §5 migration refs `00050`/`00054`/`00056`/`00057`/`00059` | **FALSE** | none exist (max 00039). Verified: `hsn_sac_master` absent; tenant FK = `00016`; payment idempotency `event_id` = `00035`. | Med | S |
| 15 | `.golangci.yml` enables `govulncheck` **and** §7 Job B also runs it | **INCONSISTENT** | double-run. Drop `govulncheck` from `.golangci.yml` linters (keep `gosec`+`staticcheck`); run govulncheck only in the separate security job, per §6 note. | Low | S |
| 16 | ci.yml line refs for gofmt/vet/sqlc/build (`:25`/`:32`/`:41`/`:74`) | IMPRECISE | those are step-*name* lines; commands sit at `:27`/`:33`/`:43`/`:75`. `:39`/`:66` are correct. | Low | S |

#### Explicit Decisions (coverage gates, security scanners, e2e)

**D1 — Coverage floor.** *Decision:* adopt an overall **60%** project floor + per-group floors (repository 90%, domain/invoice/payment/booking/auth 80%, integration 70%) via Codecov `codecov.yml`. *Tradeoff:* 60% is low enough that the initial green baseline (post Phases 2-3) passes without heroics, but high enough to catch net regression in the riskiest layer (repository). *Cost:* Codecov is a 3rd-party SaaS; needs `CODECOV_TOKEN` secret and network egress from CI; fork PRs need `if: env.CODECOV_TOKEN != ''` or the upload fails. Alternative (self-hosted `go coverage` diff in CI, no external SaaS) raises infra cost but removes the secret/network dependency.

**D2 — Security scanners in CI.** *Decision:* add a dedicated `security` job running `gosec` (SARIF upload, `-no-fail` for now → triage-first) **and** `govulncheck` (fails job on known CVEs). Keep `gosec`+`staticcheck` in `.golangci.yml`; run `govulncheck` only here (not double-enabled). *Tradeoff:* failing on govulncheck findings may block merges on transitive-dep CVEs the team can't fix immediately; triage-first avoids noise but lets real issues linger. *Cost:* `govulncheck` needs network to the Go vuln DB at runtime (or a pre-cached db); `gosec` install adds ~1 min to the job. Long-term: flip gosec to fail once baselined.

**D3 — e2e strategy.** *Decision:* move orphaned `e2e-test.js` into `test/` so `playwright.config.ts` (`testDir:'./test'`) collects it; add a `e2e` job (`needs: build`) running `npx playwright test` against `go run ./cmd/server/` on `PORT=8092`. *Tradeoff:* Playwright pulls a Chromium download + `--with-deps` apt step (slower, ~3-5 min, needs `ubuntu-latest` with sudo/apt); it is a smoke test only, not a substitute for Go unit/integration coverage. *Cost:* a flaky/browser-version-coupled job can block merges; gate it `if: github.event_name == 'pull_request'` with `failure()` tolerance or mark informational until stabilized. Note `package.json` already has `@playwright/test ^1.62.1` and a `test:ui` script, so the dep is present.

### Current CI (`.github/workflows/ci.yml`)
- Three jobs only: `test` (`.github/workflows/ci.yml:10`), `lint` (`.github/workflows/ci.yml:51`), `build` (`.github/workflows/ci.yml:68`).
- `test` job steps: checkout → `setup-go@v5` go `1.26` → `go mod download` → `gofmt -l -s` check (`.github/workflows/ci.yml:27`, step name at :25) → `go vet ./...` (`.github/workflows/ci.yml:33`, step name at :32) → `CGO_ENABLED=1 go test -v -race -count=1 ./...` (`.github/workflows/ci.yml:39`) → `sqlc generate` freshness check (`.github/workflows/ci.yml:43`, step name at :41).
- **There is NO coverage gate** anywhere in `ci.yml`.
- **There is NO `gosec`, `govulncheck`, or `staticcheck` step** in `ci.yml`. `lint` only runs `golangci-lint run --timeout=5m` (`.github/workflows/ci.yml:66`) and there is **no `.golangci.yml`** config file (verified: `ls .golangci.yml` → not found).
- `build` only does `docker build -t mvtms:latest .` (`.github/workflows/ci.yml:75`, step name at :74).

### Makefile (`Makefile`)
- `check:` (`./Makefile:61`) = `check-fmt vet build staticcheck test-race check-security`. So locally `staticcheck` IS enforced (`./Makefile:67` exits 1 if binary missing) but **CI never calls `make check`** — it calls raw `go test`/`go vet`/`golangci-lint`.
- `check-security:` (`./Makefile:72-76`) **silently skips** both scanners:
  - gosec: `if [ -x "$$bin" ]; then ...; else echo "gosec skipped (not installed)"; fi` — never fails when missing.
  - govulncheck: same silent-skip pattern.
- `staticcheck:` (`./Makefile:67`, exits 1 at :69) fails only if binary absent; does not pin a version.

### Test harness (`test/helpers.go`)
- `NewTestDB(t)` (`./test/helpers.go:21`) opens a unique **named** in-memory SQLite (`file:<name>?mode=memory&cache=shared&_pragma=journal_mode(WAL)`, line 29) and applies goose migrations from `../db/migrations` (`./test/helpers.go:35`). This is the canonical "DB is real SQLite, ephemeral" pattern — there is **no external/container DB** in tests.
- `NewTestServices(t, db)` (`./test/helpers.go:47`) wires `sqlite.NewRepository(db)` + `service.NewServices(...)` + a discard logger. Use this for handler-level tests.
- `NewTestRepo(t, db)` (`./test/helpers.go:69`) returns `*sqlite.SQLRepository`.

### Coverage gaps (grep proof)
- Total `func Test` in repo: **275** (`grep -rn "func Test" --include=*.go . | wc -l`).
- **13 internal packages have ZERO test files** (verified per-package `grep -rln "func Test"` → 0):
  `apiversion`, `driver`, `graphqlservice`, `grpcservice`, `integration`, `module`, `mqttservice`, `openapispec`, `pdf`, `repository`, `static`, `templates`, `vehicle`.
- Note: `internal/repository/sqlite` (the SQL impl) is inside the `repository` package label above and is untested directly. The package `internal/service` DOES have `services_test.go` (verified earlier).

### Toolchain / other
- `go.mod` line 3: `go 1.26` (`./go.mod:3`). Module name `transport-app` (`./go.mod:1`).
- SQLite deps: `modernc.org/sqlite v1.56.0` (`./go.mod:15`), `github.com/pressly/goose/v3 v3.27.3` (`./go.mod:11`). **No `sqlc` version is pinned in go.mod** — `ci.yml:43` does `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` (unpinned, non-reproducible).
- Playwright: `playwright.config.ts` exists with `testDir: './test'` (`./playwright.config.ts`), but the only e2e file `e2e-test.js` lives at repo root `./e2e-test.js` — **outside `testDir` → orphaned, never collected**. Playwright is **NOT referenced in any CI job**.
- `playwright.config.ts` `webServer` runs `go run ./cmd/server/` on `PORT=8092` with `DATABASE_URL="file:/tmp/transport-playwright.db?mode=rwc&cache=shared&_foreign_keys=on&_journal_mode=WAL"` (`./playwright.config.ts`).

---

## 1. Overview / goal

Harden the Go TMS test suite and CI so that coverage, security, and static analysis are real gates (not silently skipped), and so the 13 untested internal packages get a baseline of tests. Goal: a beginner can (a) add per-feature Go tests using the existing `test.NewTestDB`/`NewTestServices` pattern, (b) wire 4 new CI jobs (coverage+Codecov, security, staticcheck, Playwright e2e) by copy-pasting the YAML in §7, and (c) run a hardened local `make check` that fails loudly when scanners are missing.

Non-goals:
- No new product features / no new DB tables (this spec owns no migration).
- Not re-architecting the service layer; we test what exists via the public service/handler surface.
- Not adding real external integrations (GSTN/EWB/FASTag) — those stay behind MOCK adapters per the template conventions; we test the mock path + the adapter interface contract.

---

## 2. API contract

N/A — this spec changes the CI pipeline and local tooling, not HTTP routes. The "interface" is: `make check` exits 0 only when fmt+vet+build+staticcheck+race-tests+security all pass; CI jobs produce artifacts (coverage, sarif, e2e report).

---

## 3. DB contract

**No new tables, no migration owned by this spec.**

However, §3 adds one test that *asserts the migration set is healthy*. Because all tests use `test.NewTestDB` (`./test/helpers.go:21`), any broken migration breaks every test at once — so we add an explicit, fast guard:

`internal/repository/sqlite/migration_apply_test.go` (new file):
```go
package sqlite_test

import (
	"testing"

	"transport-app/test"
)

// TestMigrationsApplyCleanly asserts goose Up runs to completion on a fresh
// named in-memory DB and that a known table from an early + late migration
// exists. Add a new assertion here whenever a new migration is added so a
// missing/renamed table fails CI fast instead of surfacing in unrelated tests.
func TestMigrationsApplyCleanly(t *testing.T) {
	db := test.NewTestDB(t)
	defer db.Close()

	tables := []string{"users", "bookings", "company_settings", "outbox_events", "customers"}
	for _, tbl := range tables {
		// sqlite_master holds the catalog; querying it is the portable way to
		// confirm a table exists without depending on row counts.
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %q to exist after migrations, got: %v", tbl, err)
		}
	}
}
```
Why: asserted tables are verified to exist in `db/migrations` (`company_settings` from `00001_initial.sql`, `outbox_events` from `00020_create_outbox_events.sql`, `users`/`bookings`/`customers` from `00001`/`00006`/`00004`). There is **no** `company_config` or `telemetry_devices` table, and the migration set stops at `00039` — the originally-cited numbers (`00040`/`00042`/`00055`) do not exist. Adjust the `tables` slice if a migration is renamed; this is the single place to assert new-table presence.

---

## 4. UI

No new UI. The Playwright e2e (§7) is a headless smoke test of the existing web app (`internal/static`, `internal/templates`). It reuses `playwright.config.ts`'s `webServer` (`./playwright.config.ts`). The orphaned `e2e-test.js` is moved into `test/` so it is collected (see §7, "Playwright job").

---

## 5. Business logic — per-feature test strategy

Testing pyramid for this repo:
1. **Unit (base, ~70%)** — pure functions / domain aggregates, no DB.
2. **Handler/Service with `test.NewTestServices` (middle, ~25%)** — real SQLite, real service layer, `httptest` for HTTP handlers.
3. **Playwright e2e (top, ~5%)** — one or two critical user journeys only.

General pattern for handler tests (copy-paste skeleton):
```go
func TestXxxHandler(t *testing.T) {
	db := test.NewTestDB(t)
	defer db.Close()
	svc := test.NewTestServices(t, db)
	// build handler with svc, craft *http.Request, httptest.NewRecorder(), assert.
}
```

### 5.1 `internal/integration` (stubs: gstn, ewaybill, accounting, fastag)
- **Target**: 70% (integration subset). Currently 0 tests.
- Add `internal/integration/integration_test.go`:
  - Each client (gstn/ewaybill/accounting/fastag) has a MOCK adapter behind an interface (template convention). Test that, with mock enabled, `PushInvoice`/`GenerateEWB` return canned success and that the **real** client errors when creds missing (config-flagged).
  - Test the `integration` HTTP `handler.go` routes reject when the provider is disabled and succeed when mock enabled.
- Assert multi-tenancy: never trust client `tenant_id` — derive from `auth.ContextUser` (template convention). Test that a cross-tenant request is rejected.

### 5.2 GST calculation (in `internal/invoice`, `internal/domain/invoice`, integration gstn)
- **Target**: 80% (invoice domain).
- Add pure-function unit tests for CGST/SGST/IGST split logic:
  - Intrastate → CGST+SGST at half-rate each.
  - Interstate → IGST at full rate.
  - HSN/SAC based rate lookup (verify the lookup table in `db/migrations`; there is **no** `hsn_sac_master` table as of this audit — migrations stop at `00039`).
  - Rounding to 2 decimals, no paisa drift (use `math/big` or integer paise).
- Fixtures: sample invoice line items JSON, expected tax breakdown JSON.

### 5.3 Event flow (`internal/events`, outbox `outbox_events`)
- **Target**: 70% (integration) for the flow; unit-test the bus.
- `internal/events/bus.go`: unit-test publish/subscribe ordering, at-least-once delivery, and that a failing handler does not ack.
- Outbox: test that a domain event is written to `outbox_events` (table from `00020_create_outbox_events.sql`) and marked `status`/`attempts`/`last_error`; test the relay marks rows `processed` and retries on failure.

### 5.4 Payment webhook (`internal/payment/application/razorpay_webhook.go`, `internal/handlers/payments.go`)
- **Target**: 80% (payment domain).
- Unit-test signature verification (HMAC-SHA256 over `order_id|payment_id`) — both valid and tampered cases.
- Test idempotency: replaying the same `event_id` (idempotency column added at `00035_payment_idempotency`, see `db/migrations`) does not double-credit.
- Test state machine: `created → authorized → captured → refunded` transitions; invalid transitions rejected.
- Handler test via `test.NewTestServices`: POST webhook, assert booking/order marked paid and `payment_id`/`signature` persisted.

### 5.5 GraphQL (`internal/graphqlservice/handler.go`)
- **Target**: unit + a few service-level.
- Add `internal/graphqlservice/handler_test.go`:
  - Build the graphql handler with `test.NewTestServices`.
  - POST a minimal query (e.g. `query { health }` or a real read query against seeded data) and assert 200 + expected JSON shape.
  - Negative: unauthorized query returns auth error; cross-tenant query returns empty/denied.
- Keep GraphQL e2e light; cover 1-2 read queries + 1 mutation.

### 5.6 Mobile (`internal/apiversion`, maybe `internal/grpcservice`)
- **Target**: 80% (auth/booking api surfaces).
- `apiversion`: test version negotiation — unknown version rejected, supported version returns compat schema flag.
- `grpcservice`: add a `grpcservice_test.go` that boots the gRPC server against `test.NewTestServices` and exercises one RPC round-trip (e.g. `Ping` or a read RPC) with a `bufconn` listener (no real port). Negative: unauthenticated RPC rejected.

### 5.7 Repository (`internal/repository/sqlite`)
- **Target**: **90%** (highest).
- This is the biggest gap (0 tests) and the riskiest layer. Add `internal/repository/sqlite/*_test.go` covering each CRUD method using `test.NewTestDB`:
  - Insert → Get returns same entity; Update persists; Delete (soft/hard) reflects.
  - Tenant isolation: rows scoped by `tenant_id` (column added at `00016_add_tenant_id_to_bookings`, see `db/migrations`); test that a repo call with tenant A cannot read tenant B's row.
  - FK/constraint behavior on bad input.
- Because `NewTestDB` applies ALL migrations, every repo test runs against the real schema — high confidence.

### 5.8 `vehicle` / `driver` (`internal/vehicle`, `internal/driver`)
- **Target**: 80%.
- `vehicle`: test registration validation, duplicate-IMEI/registration rejection, document-expiry calc (vehicle columns — verify in `00003_vehicles.sql`; there is no `00054` migration, set stops at `00039`).
- `driver`: test driver aggregate invariants (`internal/driver/domain/aggregate/driver_aggregate.go`), `user_id` link (verify in `db/migrations`; `user_id` is present from the early schema, e.g. `00001_initial`/`00012_rbac`), license-expiry warnings.

### 5.9 `pdf` (`internal/pdf`)
- **Target**: 70% (integration subset).
- Unit-test PDF generation does not panic and produces non-empty bytes for each template (invoice, POD, settlement). Golden-file check optional; assert content contains key fields (invoice no, total).

### 5.10 `module` / `static` / `templates` / `openapispec` / `mqttservice`
- `module`: test module registration/enable-disable flag resolution from config.
- `static` / `templates`: template-render tests — render a known template with a fixture struct, assert key substrings present (catches broken `{{ }}` at test time).
- `openapispec`: assert the generated spec is valid JSON/YAML and contains expected paths (parse, don't just exist).
- `mqttservice`: unit-test the MQTT topic parsing/IMEI-auth logic and the `TelematicsProvider` adapter contract (template convention) with a fake broker (no network). Test message decode → position insert path using `test.NewTestServices`.

---

## 6. Config / env

| var / file | default | purpose | which package / job reads |
|---|---|---|---|
| `CODECOV_TOKEN` | — (repo secret) | Upload coverage to Codecov from CI | coverage job (`codecov/codecov-action`) |
| `GOSEC_VERSION` | `2.21.4` | Pin gosec for reproducible security scan | security job |
| `GOVULNCHECK_VERSION` | `latest` or pinned `go install golang.org/x/vuln/cmd/govulncheck@<ver>` | Pin govulncheck | security job |
| `STATICCHECK_VERSION` | `2025.1.1` (honnef.co/go/tools) | Pin staticcheck (replaces silent skip) | staticcheck job + `make staticcheck` |
| `SQLC_VERSION` | `1.27.0` | Pin sqlc (currently `@latest` in `ci.yml:43`) | ci.yml + Makefile `generate` |
| `.golangci.yml` | new file | Enable gosec/govulncheck/staticcheck via golangci-lint OR keep separate | `lint` job |
| `CGO_ENABLED` | `1` for tests (sqlite needs cgo) | Required for `modernc.org/sqlite` + race | all Go test jobs |

### `.golangci.yml` (create at repo root)
The current `lint` job (`./github/workflows/ci.yml:66`) runs `golangci-lint run` with **no config** — so gosec/govulncheck are NOT enabled. Add this file to turn them on and make the lint gate meaningful:
```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable:
    - govet
    - errcheck
    - staticcheck
    - gosec
    - ineffassign
    - unused
    - misspell
    - revive
  disable:
    - sqlclosecheck   # conflicts with modernc sqlite pooling pattern
  exclusions:
    paths:
      - test/
      - internal/static/

linters-settings:
  gosec:
    excludes:
      - G404   # math/rand usage in non-crypto paths (e.g. demo seeding)

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```
Note: `govulncheck` is deliberately **not** enabled here so it is not run twice (lint job + security job). It runs only in the §7 security job (Job B). Keep `gosec`+`staticcheck` in golangci-lint; the security job needs network for the Go vuln DB (or a pre-cached db).

---

## 7. Tests — coverage, Codecov, Makefile hardening, 4 CI jobs

### 7.1 Coverage targets (enforced via Codecov `codecov.yml`)
| Package / group | Floor |
|---|---|
| `internal/repository/...` | **90%** |
| `internal/domain/...` | 80% |
| `internal/invoice/...` | 80% |
| `internal/payment/...` | 80% |
| `internal/booking/...` | 80% |
| `internal/auth/...` | 80% |
| `internal/integration/...` (event flow + adapters) | 70% |
| **Project overall floor** | **60%** |

### 7.2 `codecov.yml` (create at repo root)
```yaml
coverage:
  status:
    project:
      default:
        target: 60%        # overall floor
        threshold: 1%      # allow 1% regression
        informational: false
    patch:
      default:
        target: 60%
  flags:
    unit:
      paths:
        - internal/          # Go unit/service tests
    e2e:
      paths:
        - e2e/              # Playwright (optional, can be informational)
  component:
    - component_name: "repository"
      paths:
        - internal/repository/
      target: 90%
    - component_name: "domain"
      paths:
        - internal/domain/
      target: 80%
    - component_name: "invoice"
      paths:
        - internal/invoice/
      target: 80%
    - component_name: "payment"
      paths:
        - internal/payment/
      target: 80%
    - component_name: "booking"
      paths:
        - internal/booking/
      target: 80%
    - component_name: "auth"
      paths:
        - internal/auth/
      target: 80%
    - component_name: "integration"
      paths:
        - internal/integration/
        - internal/events/
      target: 70%

comment:
  layout: "reach, diff, flags, files"
  require_changes: false

github_checks:
  annotations: true
```

### 7.3 Hardened Makefile
Replace the silent-skipping `check-security` and unpinned `staticcheck` so a missing scanner **fails** locally (matching the new CI gates). Replace `./Makefile:67-76` with:

```makefile
STATICCHECK_VERSION ?= 2025.1.1
GOSEC_VERSION ?= 2.21.4
GOVULNCHECK_VERSION ?= latest
SQLC_VERSION ?= 1.27.0

staticcheck:
	@bin=$$(command -v staticcheck || echo "$$(go env GOPATH)/bin/staticcheck"); \
	if [ ! -x "$$bin" ]; then \
		echo "installing staticcheck $(STATICCHECK_VERSION)"; \
		go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION); \
		bin="$$(go env GOPATH)/bin/staticcheck"; \
	fi; \
	"$$bin" ./...

check-security:
	@bin=$$(command -v gosec || echo "$$(go env GOPATH)/bin/gosec"); \
	if [ ! -x "$$bin" ]; then \
		echo "installing gosec $(GOSEC_VERSION)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION); \
		bin="$$(go env GOPATH)/bin/gosec"; \
	fi; \
	"$$bin" -no-fail -fmt sarif -out gosec.sarif ./... || true
	@bin=$$(command -v govulncheck || echo "$$(go env GOPATH)/bin/govulncheck"); \
	if [ ! -x "$$bin" ]; then \
		echo "installing govulncheck $(GOVULNCHECK_VERSION)"; \
		go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION); \
		bin="$$(go env GOPATH)/bin/govulncheck"; \
	fi; \
	"$$bin" ./...

generate:
	@bin=$$(command -v sqlc || echo "$$(go env GOPATH)/bin/sqlc"); \
	if [ ! -x "$$bin" ]; then \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION); \
		bin="$$(go env GOPATH)/bin/sqlc"; \
	fi; \
	"$$bin" generate
```
Now `make check` (`./Makefile:61`) genuinely runs security + static analysis and will surface failures. Keep `check:` as `check-fmt vet build staticcheck test-race check-security`.

### 7.4 Four new CI jobs (add to `.github/workflows/ci.yml`)

Append these jobs after the existing `build` job. Pin Go 1.26 (`./go.mod:3`) and `CGO_ENABLED=1` for tests (sqlite + race).

#### Job A — Coverage + Codecov
```yaml
  coverage:
    name: Coverage & Codecov
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: go mod download
      - name: Run tests with coverage
        run: CGO_ENABLED=1 go test -race -count=1 -coverprofile=coverage.out -covermode=atomic ./...
      - name: Upload to Codecov
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          flags: unit
          token: ${{ secrets.CODECOV_TOKEN }}
          fail_ci_if_error: true
```

#### Job B — Security (gosec + govulncheck)
```yaml
  security:
    name: Security scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: go mod download
      - name: Install gosec
        run: go install github.com/securego/gosec/v2/cmd/gosec@2.21.4
      - name: gosec
        run: $(go env GOPATH)/bin/gosec -no-fail -fmt sarif -out gosec.sarif ./...
      - name: Upload gosec SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: gosec.sarif
      - name: govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          $(go env GOPATH)/bin/govulncheck ./...
```

#### Job C — Staticcheck
```yaml
  staticcheck:
    name: Staticcheck
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: go mod download
      - name: Install staticcheck
        run: go install honnef.co/go/tools/cmd/staticcheck@2025.1.1
      - name: Run staticcheck
        run: $(go env GOPATH)/bin/staticcheck ./...
```

#### Job D — Playwright e2e
Moves the orphaned `e2e-test.js` into `test/` so `playwright.config.ts`'s `testDir: './test'` collects it. Steps:
```yaml
  e2e:
    name: E2E (Playwright)
    runs-on: ubuntu-latest
    needs: [build]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Install Playwright
        run: npm install && npx playwright install --with-deps chromium
      - name: Run e2e
        run: npx playwright test
      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: playwright-report/
```
Pre-step (one-time, manual): `git mv e2e-test.js test/e2e-test.js` and rename to `test/e2e.spec.js` if you prefer `.spec` convention; ensure `package.json` has `@playwright/test` as a devDependency and a `"test": "playwright test"` script. `playwright.config.ts` already points `webServer` at `go run ./cmd/server/` on `PORT=8092` (`./playwright.config.ts`), so the job needs no extra server wiring.

### 7.5 Pass-before-merge checklist
- [ ] `make check` exits 0 locally (fmt, vet, build, staticcheck, race tests, security).
- [ ] New feature has unit + (if HTTP) handler test via `test.NewTestServices`.
- [ ] Coverage floors met (repo ≥60%, repository ≥90%, domain/invoice/payment/booking/auth ≥80%, integration ≥70%) — enforced by Codecov status checks.
- [ ] `gosec.sarif` uploaded; govulncheck clean (or findings triaged).
- [ ] `sqlc generate` is a no-op diff (pinned version).
- [ ] Playwright e2e green (or intentionally skipped with reason in PR).

### Anti-Regression Enforcement Gates

Two audit-required gates below are grep-proof CI steps (no Go code, no test binary)
so they fail loudly on the PR that reintroduces a known regression. Owning specs:
**09** (eventbus/tenancy) for gate 1, **10** (auth-rbac-rag) for gate 2.

**1. Tenant-hardcode gate — fail on any hardcoded tenant-1 `TenantID`.**

Exact command (a match is a failure):
```bash
grep -rn 'TenantID:\s*"1"' internal/ cmd/
```
Rationale: multi-tenancy safety — a second customer must never read tenant 1's data.
A literal `"1"` bypasses `auth.ContextUser` scoping and leaks every cross-tenant row.
Scope deliberately covers `internal/` and `cmd/` only: `test/` fixtures legitimately
hardcode `"1"` and must not false-fail the gate. Any production hit is a blocking bug.

Wire into `.github/workflows/ci.yml` as a new `anti-regression` job (runs on every PR,
no Go needed, ~5s):
```yaml
  anti-regression:
    name: Anti-regression gates
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Tenant-hardcode gate
        run: |
          if grep -rn 'TenantID:\s*"1"' internal/ cmd/; then
            echo "FOUND hardcoded tenant-1 TenantID — a second customer would read tenant 1's data"
            exit 1
          fi
      - name: RAG route auth gate
        run: |
          grep -A2 'ragHandler.RegisterRoutes' cmd/server/main.go | grep -q RequireAPIAuth \
            || { echo "RAG routes registered without auth — /api/rag/* arbitrary-file-read regression"; exit 1; }
```

**2. RAG route auth gate — every `/api/rag/*` endpoint MUST sit behind auth.**

Exact command (non-zero exit when the auth middleware is absent):
```bash
grep -A2 'ragHandler.RegisterRoutes' cmd/server/main.go | grep -q RequireAPIAuth
```
Rationale: prevents the `/api/rag/*` arbitrary-file-read regression (critical
security). `/api/rag/upload` and `/api/rag/index` read/write the host filesystem and
must never be reachable unauthenticated. Note: **currently RED** — `ragHandler.RegisterRoutes(r)`
(`cmd/server/main.go:492`) is called on the unprotected router, outside the
`RequireAPIAuth` group (`cmd/server/main.go:444-455`), and `rag.Handler.RegisterRoutes`
(`internal/rag/handler.go:29-36`) applies no middleware itself. Fix is owned by Spec 10;
the gate must be green before merge. If the router is refactored so the `-A2` window
stops matching, the gate fails — re-verify the auth wiring, do not widen the window.

## 8. Future / GPS-provider

- **Go benchmarks** for hot paths: telemetry ingestion (`internal/telemetry` + `internal/mqttservice` decode/insert) and event-bus publish throughput. Add `internal/telemetry/bench_test.go` / `internal/events/bench_test.go` using `testing.B`, with `b.ReportAllocs()`.
- **Perf CI with benchstat**: a scheduled (cron) job that runs `go test -bench=. -benchmem ./internal/telemetry/... ./internal/events/...` and compares against a committed `bench-base.txt` using `benchstat`; fail if regression >X%.
  ```yaml
  perf:
    name: Perf regression (benchstat)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: go mod download
      - run: go test -bench=. -benchmem -run=^$ ./internal/telemetry/... ./internal/events/... > bench-new.txt
      - run: go install golang.org/x/perf/cmd/benchstat@latest
      - run: benchstat bench-base.txt bench-new.txt   # fail if p < 0.05 and worse
  ```
- **Load test**: k6 or vegeta script against the booking + telemetry ingest endpoints; run manually / nightly against a staging instance. Define SLOs (p95 latency, error rate) before automating.
- GPS/telematics: keep `TelematicsProvider` interface (template convention); add a fake provider in tests to validate IMEI-auth + position-enrichment without hardware.

---

## 9. Edge cases

1. **SQLite + race detector**: `modernc.org/sqlite` needs `CGO_ENABLED=1`; the race build fails under `CGO_ENABLED=0` (current `build` job at `ci.yml:74` sets `CGO_ENABLED=0` — fine for build, but tests MUST use `CGO_ENABLED=1` as `test` job does at `ci.yml:39`). Keep them separate.
2. **Shared in-memory DB across pooled conns**: handled by the unique named-DB pattern in `test/helpers.go:28-29`; never use plain `:memory:` or tests flake with "no such table".
3. **govulncheck offline**: CI runners may lack network for the vuln DB — pre-cache or allow the security job to be informational in forks.
4. **Codecov on forks / first run**: `CODECOV_TOKEN` absent → `codecov-action` fails; scope the upload step with `if: ${{ env.CODECOV_TOKEN != '' }}` for fork safety, but require it on `main`.
5. **Playwright chromium deps**: `npx playwright install --with-deps` needs `sudo`/apt on the runner; `ubuntu-latest` supports it.
6. **sqlc drift**: unpinned `@latest` (`ci.yml:43`) can change generated output and falsely fail the freshness check; pin via §6 `SQLC_VERSION`.
7. **Staticcheck too strict after enable**: initially it may flag hundreds of issues; roll out by allowing `//nolint` + fixing incrementally, but do NOT disable the linter.
8. **Multi-tenant tests**: every repo/handler test must assert cross-tenant isolation (template convention) — a common silent bug.

---

## 10. Phased rollout (build order)

1. **Phase 1 — Config & gates (no behavior change)**: add `.golangci.yml`, `codecov.yml`, pin versions in Makefile + `ci.yml`, harden `check-security`/`staticcheck`. Land 4 CI jobs. (No product risk.)
2. **Phase 2 — Repository tests (highest risk)**: add `internal/repository/sqlite/*_test.go` + `migration_apply_test.go` (§3). Drives coverage floor for the 90% group.
3. **Phase 3 — Domain/payment/invoice/booking/auth unit tests** to hit 80% groups.
4. **Phase 4 — Integration/adapters/events/webhook/mqtt** tests (70% group) + GraphQL/mobile handler tests via `NewTestServices`.
5. **Phase 5 — Playwright e2e** (move `e2e-test.js` into `test/`, add job D) + `pdf`/`static`/`templates`/`openapispec`/`module` tests.
6. **Phase 6 — Future perf CI/benchmarks** (§8) as a separate scheduled workflow.

---

## 11. Open items / VERIFY

- **RESOLVED**: `internal/repository` contains only `internal/repository/sqlite` (no `.go` at the parent level); both are zero-tested, so the 90%-target tests target `internal/repository/sqlite` directly.
- **RESOLVED**: `govulncheck` is intentionally excluded from `.golangci.yml` (see §6 note) to avoid running it in both the lint and security jobs; it runs only in the standalone security job (§7 Job B). No plugin-availability check needed.
- **VERIFY**: Playwright `package.json` exists at repo root with `@playwright/test` dep — `e2e-test.js` uses `require('@playwright/test')` (`./e2e-test.js:1`) so it must be installed; confirm before enabling Job D.
- **DECIDE**: overall floor 60% vs stricter; adjust `codecov.yml` `target` if the initial baseline after Phase 2-3 is lower.
- **DECIDE**: whether to fail CI on gosec findings (`-no-fail` currently used for SARIF upload) — recommend triage-first, then flip to fail.

---

## 12. File list

Create:
- `docs/tech-specs/15-testing-ci.md` — this spec.
- `.golangci.yml` — lint config enabling staticcheck/gosec (govulncheck runs in the separate security job, §6/§7 Job B).
- `codecov.yml` — coverage gates (§7.2).
- `internal/repository/sqlite/migration_apply_test.go` — migration health assertion (§3).
- `internal/repository/sqlite/*_test.go` — repo CRUD/tenant tests (§5.7).
- `internal/integration/integration_test.go` — adapter/mock tests (§5.1).
- `internal/graphqlservice/handler_test.go` — GraphQL tests (§5.5).
- `internal/grpcservice/grpcservice_test.go` — bufconn RPC test (§5.6).
- `internal/payment/application/razorpay_webhook_test.go` — webhook/signature/idempotency (§5.4).
- `internal/invoice/...` + `internal/domain/invoice/...` — GST calc tests (§5.2).
- `internal/events/bus_test.go` + outbox relay test (§5.3).
- `internal/vehicle/...`, `internal/driver/...` — validation tests (§5.8).
- `internal/pdf/pdf_test.go`, `internal/static/...`, `internal/templates/...`, `internal/openapispec/...`, `internal/module/...`, `internal/mqttservice/...` — (§5.9/§5.10).
- `test/e2e.spec.js` (moved from `e2e-test.js`) — collected by `playwright.config.ts` (§7 Job D).
- `bench-base.txt` — baseline for benchstat (§8).

Modify:
- `.github/workflows/ci.yml` — replace `sqlc@latest` with pinned; add Jobs A–D (§7.4); keep existing `test`/`lint`/`build`.
- `Makefile` — replace `staticcheck`/`check-security`/`generate` blocks (`./Makefile:67-76`) with pinned, failing versions (§7.3); keep `check:` (`./Makefile:61`).
- `e2e-test.js` → `git mv` to `test/` (orphaned fix, §7 Job D).
