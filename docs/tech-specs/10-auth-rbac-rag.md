# Auth / RBAC Hardening + RAG Security & Multi-Tenant — Implementation Spec v1

Status: ready
Depends-on: 00012_rbac.sql (roles/permissions), 00027_add_driver_role.sql (driver role id=5),
           00001_initial.sql (sessions, users), 00002_drivers.sql (drivers),
           08 (auth base), internal/agent/* (knowledge_search tool)
Migration owner: db/migrations/00056_*.sql   (reserved exactly here in 00-migration-ownership-index.md)

> RAG DB migration: reserve the next free number **00062** in 00-migration-ownership-index.md
> (append a row `00062 | RAG multi-tenant vectors + provider registry | 10`).
> All RAG DDL below lives in `db/migrations/00062_rag_vectors.sql`.

This spec has two halves:
- **Part A — Auth / RBAC hardening** (migration 00056): encrypted signed sessions,
  revocable API tokens, tenant-scoped Casbin, finish the DRIVER role, kill the
  non-revocable session fallback, tenant derived from identity not hardcoded.
- **Part B — RAG** (migration 00062): provider pluggability, ANN / hybrid search,
  dimension validation + reindex guard, multi-tenant isolation, document lifecycle,
  **AUTH on all RAG routes**, hybrid search + reranking.

---

## 0. Verified ground truth (file:line proofs of current state)

Run these greps yourself before coding; every claim below is reproducible.

### 0.1 CRITICAL — RAG routes are completely unauthenticated
`internal/rag/handler.go:29-36` registers 6 endpoints with **no** auth middleware:

```
r.Post("/api/rag/search", h.handleSearch)
r.Post("/api/rag/index", h.handleIndex)
r.Get ("/api/rag/stats", h.handleStats)
r.Post("/api/rag/reindex", h.handleReindex)
r.Post("/api/rag/teach", h.handleTeach)
r.Post("/api/rag/upload", h.handleUpload)
```

`handler.go:29 func (h *Handler) RegisterRoutes(r chi.Router)` — the `r` is a plain
`chi.Router`, not wrapped by `middleware.RequireAPIAuth`. `handleIndex`/`handleReindex`
(`handler.go:106,161`) accept an arbitrary `directory` from the body and read files off
disk — an anonymous attacker can point `directory` at `/etc` or the repo root and exfiltrate
source via `/api/rag/search`. This is a **CRITICAL** security gap and the #1 thing to fix.

The agent already calls this unauthenticated service:
`internal/agent/subagents.go:108` `Tools: []*RegisteredTool{ragSearchTool(opts.RagService)}`
and `ragSearchTool` (`subagents.go:126`) calls `svc.Query` directly.

### 0.2 Sessions are sign-only, NOT encrypted
`internal/auth/session.go:54`:

```
signer: securecookie.New([]byte(cookieSecret), nil),  // blockKey = nil => SIGNED ONLY
```

`securecookie.New(hashKey, blockKey)` with `blockKey == nil` produces a cookie that is
**HMAC-authenticated but plaintext-readable**. Anyone who reads the cookie sees the JSON
(`SessionData{UserID, Role, Name, Expires, Token}` — `session.go:70-76`) in cleartext. Only
tampering is prevented, not disclosure. `mustEncode` (`session.go:112-118`) just base64-encodes
the signed blob.

### 0.3 Non-revocable session fallback exists
`internal/handlers/auth.go:100-104` (`Register`):

```
if sessResult, err := h.Services.Auth.CreateSessionForUser(...); err == nil && sessResult != nil {
    h.AuthStore.CreateSessionWithToken(w, ... sessResult.SessionToken)
} else {
    h.AuthStore.CreateSession(w, user.ID.String(), ...)  // <- NON-REVocABLE: no DB row, no token
}
```

`CreateSession` (`session.go:87-89`) sets `Token:""`, so `ValidateSession`
(`session.go:136`) skips server-side validation entirely -> the session can never be revoked,
and `Logout` (`handlers/auth.go:236` -> `RevokeSession` `session.go:163-173`) cannot revoke it.
This fallback must be removed; registration must hard-fail if server-side session creation fails.

### 0.4 Casbin is read-from-DB but write-only-to-DB, with no tenant scoping
`internal/auth/casbin.go`:
- `LoadPolicy` (`casbin.go:40-87`) reads `role_permissions` + `user_roles` — but **no tenant**.
- `SavePolicy` (`casbin.go:90-92`) is a no-op (`return nil`).
- `AddPolicy`/`RemovePolicy`/`RemoveFilteredPolicy` (`casbin.go:94-104`) are all no-ops.
- `AddRoleForUser`/`DeleteRolesForUser` (`casbin.go:151-160`) call
  `enforcer.AddRoleForUser` which only mutates **in-memory** state (because the persist
  adapter's `AddPolicy` is a no-op). Result: `handlers/users.go:105,154-155`
  (`_ = h.AuthSrv.AddRoleForUser(...)`) updates Casbin memory but **not the DB**, and any
  `Reload()` (`casbin.go:146`) would **discard** those in-memory edits by re-reading old DB rows
  -> silent permission drift between restarts.

Also `CasbinModel` (`casbin.go:12-27`) has no domain/tenant dimension (`g = _, _` only).

### 0.5 API token claims are base64 plaintext (no encryption)
`internal/auth/apitoken.go:43`:

```
b64payload := base64.RawURLEncoding.EncodeToString(payload)  // payload = json(claims)
sig := computeHMAC(secret, b64payload)
return b64payload + "." + sig, nil
```

`APITokenClaims` (`apitoken.go:14-20`) carries `uid, role, tid, iat, exp`. The payload is
**base64 of JSON** — trivially decoded by anyone to read `uid`/`role`/`tid` (only the HMAC
prevents forgery, not disclosure). There is **no `api_tokens` table** anywhere in
`db/migrations/`; revocation is impossible except by deleting the user. `ParseAPIToken`
(`apitoken.go:49`) never consults a DB — it only checks signature + expiry.

### 0.6 Tenant is hardcoded to "1" everywhere
- `internal/middleware/api_auth.go:112` (session-cookie branch, lines 107-113): `TenantID: "1"`.
- `internal/middleware/middleware.go:145` (`AuthRequired`): `shared.ContextWithTenantID(ctx, "1")`.
- `internal/handlers/auth.go`, `internal/handlers/users.go` never set tenant — it is always "1".
The DB models carry `tenant_id` columns elsewhere, but auth never derives or scopes it.

### 0.7 DRIVER role exists in DB but not in domain/RBAC
- `db/migrations/00027_add_driver_role.sql:3`: `INSERT ... VALUES (5, 'driver', ...)`.
- `internal/domain/user/entity.go:45-50`: only `RoleAdmin/RoleDispatcher/RoleAccountant/RoleViewer`.
  There is **no `RoleDriver`** and `DefaultRoleID` (`entity.go:53-66`) has no `driver` case.
- `db/migrations/00002_drivers.sql:2-19`: the `drivers` table has **no `user_id` column**,
  so a driver account cannot be linked to a login user, and there is no mobile-login path.

### 0.8 RAG embedder is hardcoded dim 1536 + random hash
- `internal/rag/embedder.go:51`: `dimension: 1536` hardcoded for OpenAI.
- `internal/rag/embedder.go:108-116` `NewHashEmbedder` + `hashEmbed` (`embedder.go:140-158`)
  uses `rand.New(rand.NewSource(42))` -> **deterministic random** vectors; fine for dev but
  must be clearly flagged as non-semantic.
- No provider registry: only `OpenAIEmbedder` and `HashEmbedder` exist; HuggingFace/Ollama
  are absent.

### 0.9 RAG vector store is a full table scan with no tenant
- `internal/rag/vector_store.go:45-58` `chunks` table: `id, content, source, line_from,
  line_to, chunk_idx, embedding TEXT, created_at`. **No `tenant_id`, no `document_id`.**
- `Search` (`vector_store.go:119-168`) does `SELECT ... FROM chunks` with **no WHERE**, loads
  **every** row into memory, decodes JSON, computes cosine in Go -> O(N) full scan, no ANN.
- `embedding` stored as JSON text (float array) — bloated and slow.

### 0.10 RAG config surface
`internal/config/config.go:127-135` `RAGConfig`:
`RAG_ENABLED, RAG_EMBEDDING_API_KEY, RAG_EMBEDDING_BASE_URL, RAG_EMBEDDING_MODEL,
RAG_CHUNK_SIZE, RAG_CHUNK_OVERLAP, RAG_VECTOR_DB_PATH`. No provider selector, no dimension
setting, no tenant handling. (Note: there is **no** `internal/rag/config.go`; RAG config lives
in `internal/config/config.go`.)

---

### 0.11 Verification Log

| Claim | Verdict | Correction / Evidence |
|---|---|---|
| 0.1 RAG routes unauthenticated (`handler.go:29-36`; `main.go:492` `ragHandler.RegisterRoutes(r)`) | TRUE | `main.go:492` mounts on top-level `r` with no `RequireAPIAuth`; `handler.go:30-35` mount 6 routes. `handleIndex`/`handleReindex` take arbitrary `directory` (`handler.go:106`/`161`). |
| 0.2 Sessions sign-only (`session.go:54` `securecookie.New(secret, nil)`) | TRUE | blockKey nil ⇒ HMAC only, JSON readable. `SessionData` fields at `session.go:70-76`. |
| 0.3 Non-revocable fallback (`auth.go:100-104`; `session.go:87-89,136`) | TRUE | else-branch calls `CreateSession` (Token:"") ⇒ `ValidateSession` skips server-side (`session.go:136` `data.Token != ""`). |
| 0.4 Casbin SavePolicy no-op, no tenant (`casbin.go:40-104,12-27`) | TRUE | `SavePolicy:90-92` `return nil`; `AddPolicy`/`RemovePolicy`/`RemoveFilteredPolicy:94-104` no-ops; model `g=_,_`; `AddRoleForUser` mutates memory only (`151-154`). |
| 0.5 API token base64 plaintext (`apitoken.go:43,14-20`) | TRUE | `base64.RawURLEncoding.EncodeToString(payload)`; no `api_tokens` table in `db/migrations` (grep: none); `ParseAPIToken:49` never hits DB. |
| 0.6 Tenant hardcoded `"1"` (`api_auth.go:112`; `middleware.go:145`) | TRUE (was cited as 113) | cookie branch returns `TenantID:"1"` at `api_auth.go:112`; `AuthRequired` sets "1" at `middleware.go:145`. Corrected line ref. |
| 0.7 Driver role in DB not domain (`00027:3`; `entity.go:45-50,53-66`; `00002:2-19`) | TRUE | role id 5 seeded; `entity.go` lacks `RoleDriver`; `drivers` has no `user_id`. |
| 0.8 Embedder dim 1536 + seeded hash (`embedder.go:51,108-116,140-158`) | TRUE | `OpenAIEmbedder` `dimension:1536`; `HashEmbedder` `rand.New(rand.NewSource(42))` deterministic. |
| 0.9 Vector store full scan (`vector_store.go:45-58,119-168`) | TRUE | `chunks` has no `tenant_id`/`document_id`; `Search` `SELECT ... FROM chunks` no WHERE, cosine in Go. |
| 0.10 RAG config (`config.go:127-135,139-144`; no `internal/rag/config.go`) | TRUE | `RAGConfig` fields confirmed; `ls internal/rag` shows no `config.go`. |
| 00056 / 00062 reserved (`00-index:35,41`) | TRUE | `00-migration-ownership-index.md` lines 35 & 41; repo head is `00039`, no `00056`/`00062` files exist. |
| **§3 `REFERENCES tenants(id)` FK** | **WRONG** | **No `tenants` table exists (grep `db/`+`internal/` → only `experiments_test.go`). FK would fail at `goose up`. Removed from §3.1/§3.2 (tenant_id is a free-form '1' string, per 00016-00019).** |
| §11 `hkdf` dependency | RESOLVED | `golang.org/x/crypto v0.54.0` present in `go.mod` — no `go get` needed. |

**Severity / Effort (applied corrections):**
- `tenants` FK removal (§3.1/§3.2): **Severity: High** — would break `goose up`; **Effort: Low** — delete 3 FK clauses + add comments.

### 0.12 Decisions & Tradeoffs

**Session encryption** — *Decision:* derive an AES-GCM block key from the cookie secret via
HKDF-SHA256 and pass it as the 2nd arg to `securecookie.New` (§5.1); do **not** pull a separate
encryption library. *Tradeoff:* reuses the existing secret and `golang.org/x/crypto` (already in
`go.mod`), zero wire-format change, no KMS. *Cost:* key is deterministic per secret, so rotating
`COOKIE_SECRET` invalidates all sessions (documented, acceptable); only `SESSION_BLOCK_KEY` gives
an independently rotated key.

**Casbin DB-backed RBAC** — *Decision:* back the enforcer with a writable `casbin_rules` table +
per-tenant enforcer cache; keep `SavePolicy` a no-op (§5.4). *Tradeoff:* fast in-memory enforce
with durable writes; kills the silent permission-drift bug. *Cost:* one-time backfill of
`role_permissions`/`user_roles` → `casbin_rules`; per-request tenant resolution; more to test.

**RAG provider pluggability + ANN** — *Decision:* provider registry
(`openai`/`huggingface`/`ollama`/`hash`) selected by `RAG_PROVIDER` + `RAG_EMBEDDING_DIM`; ANN via
sqlite-vec when the extension loads, else brute-force cosine, fused with an FTS5 lexical leg by RRF
(§5.7/§5.9). *Tradeoff:* `hash` stays zero-config dev default; semantic providers opt-in. *Cost:*
sqlite-vec/FTS5 are runtime-detected (graceful degrade); the dimension guard forces a reindex on
provider swap; HuggingFace/Ollama embedders are net-new code.

---

## 1. Overview / goal

**Goal.** Make the auth layer confidential (encrypted sessions), revocable (API tokens +
server-side sessions), tenant-correct (tenant derived from identity, scoped in Casbin), and
complete the DRIVER role. Make RAG safe (authenticated + tenant-isolated), pluggable
(provider registry), and fast (ANN / hybrid search) with a dimension guard so a provider swap
cannot silently corrupt the index.

**Non-goals.**
- Not changing the public `APITokenClaims` wire format (still `b64payload.sig`) — we add a DB
  revocation table and encrypt-at-rest by hashing the token, not by changing the token string.
- Not implementing full OIDC/SAML in this spec (future, see 8).
- Not migrating existing `chunks` data automatically; 00062 is additive and ships empty.

---

## 2. API contract

All RAG routes move behind `middleware.RequireAPIAuth(store, secret)`. Browser/Datastar
clients already carry the session cookie, which `RequireAPIAuth` also accepts
(`api_auth.go:101-113`); API clients send `Authorization: Bearer <api_token>`.

### 2.1 Auth / RBAC (new + changed)

| Method | Path | Auth | Permission | Body / Notes |
|---|---|---|---|---|
| POST | `/api/auth/api-tokens` | session | `apikeys:create` | create revocable token (see 2.1.1) |
| GET | `/api/auth/api-tokens` | session | `apikeys:read` | list tokens (token value hidden) |
| DELETE | `/api/auth/api-tokens/{id}` | session | `apikeys:revoke` | revoke token |
| POST | `/api/auth/session/revoke` | session | — | revoke current session (logout all) |
| POST | `/api/auth/login` | none | — | unchanged; now always server-side session (no fallback) |
| POST | `/api/auth/register` | none | — | unchanged; **hard fail** if session creation fails |
| POST | `/api/auth/driver/login` | none | — | mobile driver login (phone+OTP/password) -> session bound to `drivers.user_id` |

#### 2.1.1 Create API token — `POST /api/auth/api-tokens`
Request:
```json
{ "name": "mobile-backend", "expires_in_hours": 720, "scopes": ["bookings:read","trips:read"] }
```
Response `201`:
```json
{
  "id": "tok_8f3a...",
  "name": "mobile-backend",
  "token": "b64payload.sig",
  "expires_at": "2026-09-18T12:00:00Z"
}
```
Errors: `400` missing name; `403` no permission; `401` unauthenticated.

Notes:
- The returned `token` is the **only** time the raw value is shown. We store
  `token_hash = HashToken(raw)` in `api_tokens`. `ParseAPIToken` (`apitoken.go:49`) still
  validates signature; `middleware/api_auth.go:76-99` then looks up `api_tokens` for
  revocation/`expires_at`/scope enforcement.
- `tenant_id` is **derived from the issuing user's tenant** (see 5.3), never from the request.

### 2.2 RAG (all behind `RequireAPIAuth` + `RequirePermission(ragSrv,"rag","<action>")`)

| Method | Path | Permission | Body |
|---|---|---|---|
| POST | `/api/rag/search` | `rag:search` | `{ "query":"...", "top_k":5, "source":"", "hybrid":true }` |
| POST | `/api/rag/index` | `rag:write` | `{ "directory":"<allowlisted>" }`  (see 5.8) |
| POST | `/api/rag/teach` | `rag:write` | `{ "name":"topic", "content":"..." }` |
| POST | `/api/rag/upload` | `rag:write` | multipart `file` (txt/md/pdf) |
| POST | `/api/rag/reindex` | `rag:admin` | `{ "directory":"<allowlisted>" }` |
| GET | `/api/rag/stats` | `rag:read` | — |
| GET | `/api/rag/documents` | `rag:read` | list `rag_documents` for tenant |
| DELETE | `/api/rag/documents/{id}` | `rag:admin` | delete document + its chunks |
| GET | `/api/rag/providers` | `rag:read` | list configured/available providers + dimension |

#### 2.2.1 Search response
```json
{
  "query": "cancellation policy",
  "total": 5,
  "chunks": [
    { "id":"doc_12#3", "content":"...", "source":"teach/cancellation.md",
      "document_id":"doc_12", "score":0.81, "rerank_score":0.74 }
  ],
  "scores": [0.81]
}
```
All rows are filtered to the caller's `tenant_id` (see 5.3).

---

## 3. DB contract

### 3.1 Migration `00056` — auth hardening (Up)

```sql
-- +goose Up
-- 1. Revocable API tokens, tenant-scoped and hashed (raw token never stored)
CREATE TABLE api_tokens (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL,
    tenant_id     TEXT NOT NULL DEFAULT '1',
    name          TEXT NOT NULL,
    token_hash    TEXT NOT NULL UNIQUE,          -- HashToken(raw)
    role_at_issue TEXT NOT NULL,                -- role snapshot for audit
    scopes        TEXT NOT NULL DEFAULT '{}',    -- JSON array of "resource:action"
    expires_at    DATETIME NOT NULL,
    last_used_at  DATETIME,
    revoked_at    DATETIME,
    created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id)   REFERENCES users(id)      ON DELETE CASCADE
    -- NOTE: there is no `tenants` table in this repo (single-tenant; `tenant_id` is a
    -- free-form '1' string, consistent with migrations 00016-00019). Do NOT add a FK to a
    -- non-existent `tenants` table or `goose up` will fail.
);
CREATE INDEX idx_api_tokens_user    ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_tenant  ON api_tokens(tenant_id);
CREATE INDEX idx_api_tokens_hash    ON api_tokens(token_hash);

-- 2. Sessions: add tenant + device + encryption metadata + explicit revocation
ALTER TABLE sessions ADD COLUMN tenant_id   TEXT NOT NULL DEFAULT '1';
ALTER TABLE sessions ADD COLUMN device_id   TEXT;                       -- stable per-device id
ALTER TABLE sessions ADD COLUMN enc_blob     TEXT;                      -- encrypted session state (optional)
ALTER TABLE sessions ADD COLUMN revoked_at  DATETIME;
ALTER TABLE sessions ADD COLUMN updated_at  DATETIME NOT NULL DEFAULT (datetime('now'));
CREATE INDEX idx_sessions_tenant   ON sessions(tenant_id);
CREATE INDEX idx_sessions_revoked  ON sessions(revoked_at);

-- 3. Tenant-scoped Casbin rules (replaces ad-hoc role_permissions/user_roles reads)
--    Casbin model gains a domain; this table is the single source of truth.
CREATE TABLE casbin_rules (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id  TEXT NOT NULL DEFAULT '1',
    ptype      TEXT NOT NULL,                 -- 'p' (policy) or 'g' (grouping)
    v0         TEXT NOT NULL,                 -- sub / user
    v1         TEXT,                          -- obj / role
    v2         TEXT,                          -- act
    v3         TEXT,
    v4         TEXT,
    v5         TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(tenant_id, ptype, v0, v1, v2, v3, v4, v5)
);
CREATE INDEX idx_casbin_tenant_ptype ON casbin_rules(tenant_id, ptype);

-- 4. Finish DRIVER role: link drivers to users + seed driver permission subset
ALTER TABLE drivers ADD COLUMN user_id TEXT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX idx_drivers_user ON drivers(user_id);

-- Seed a minimal driver permission set (read own trips/vehicles, update own status)
INSERT OR IGNORE INTO permissions (name, description) VALUES
    ('driver:read_own',  'Driver reads own assignments'),
    ('driver:update_own','Driver updates own trip status'),
    ('rag:search',       'Search the knowledge base');

-- Assign those permissions to role id 5 (driver) if not already present
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT 5, p.id FROM permissions p
WHERE p.name IN ('driver:read_own','driver:update_own','rag:search')
  AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp WHERE rp.role_id=5 AND rp.permission_id=p.id
  );

-- 5. Ensure api-token permission resources exist for RBAC screen
INSERT OR IGNORE INTO permissions (name, description) VALUES
    ('apikeys:create', 'Create API tokens'),
    ('apikeys:read',   'List API tokens'),
    ('apikeys:revoke', 'Revoke API tokens'),
    ('rag:write',      'Teach/index/upload RAG'),
    ('rag:admin',      'Reindex/delete RAG documents');
```

#### 3.1.1 Migration `00056` — Down
```sql
-- +goose Down
DROP TABLE IF EXISTS api_tokens;
ALTER TABLE sessions DROP COLUMN enc_blob;
ALTER TABLE sessions DROP COLUMN device_id;
ALTER TABLE sessions DROP COLUMN tenant_id;
ALTER TABLE sessions DROP COLUMN revoked_at;
ALTER TABLE sessions DROP COLUMN updated_at;
DROP TABLE IF EXISTS casbin_rules;
ALTER TABLE drivers DROP COLUMN user_id;
DELETE FROM role_permissions WHERE role_id = 5 AND permission_id IN (
    SELECT id FROM permissions WHERE name IN ('driver:read_own','driver:update_own','rag:search')
);
DELETE FROM permissions WHERE name IN
    ('driver:read_own','driver:update_own','rag:search',
     'apikeys:create','apikeys:read','apikeys:revoke','rag:write','rag:admin');
```

### 3.2 Migration `00062` — RAG multi-tenant vectors + providers (Up)

```sql
-- +goose Up
-- A document is the upload/teach unit; chunks belong to it and to a tenant.
CREATE TABLE rag_documents (
    id             TEXT PRIMARY KEY,
    tenant_id      TEXT NOT NULL DEFAULT '1',
    title          TEXT NOT NULL,
    source         TEXT NOT NULL,                -- original filename / teach topic
    doc_type       TEXT NOT NULL DEFAULT 'text', -- text|pdf|md|code
    chunk_count    INTEGER NOT NULL DEFAULT 0,
    status         TEXT NOT NULL DEFAULT 'indexed'
                       CHECK (status IN ('indexing','indexed','failed','deleted')),
    embedding_dim  INTEGER NOT NULL,             -- dimension of stored vectors (guard)
    embedding_model TEXT NOT NULL DEFAULT '',    -- provider+model that produced vectors
    created_by     TEXT,
    created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    deleted_at     DATETIME,
    -- NOTE: no `tenants` table exists in this repo; `tenant_id` is a free-form '1' string.
    -- No FK to a non-existent table (would break `goose up`).
);
CREATE INDEX idx_rag_documents_tenant  ON rag_documents(tenant_id);
CREATE INDEX idx_rag_documents_deleted ON rag_documents(deleted_at);

CREATE TABLE rag_chunks (
    id             TEXT PRIMARY KEY,
    tenant_id      TEXT NOT NULL DEFAULT '1',
    document_id    TEXT NOT NULL,
    content        TEXT NOT NULL,
    source         TEXT NOT NULL,
    chunk_idx      INTEGER NOT NULL DEFAULT 0,
    embedding      BLOB NOT NULL,                -- raw float32 little-endian vector
    embedding_dim  INTEGER NOT NULL,
    token_count    INTEGER NOT NULL DEFAULT 0,
    created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (document_id) REFERENCES rag_documents(id) ON DELETE CASCADE
    -- NOTE: no `tenants` table exists; `tenant_id` is a free-form '1' string. No FK.
);
CREATE INDEX idx_rag_chunks_tenant_doc ON rag_chunks(tenant_id, document_id);
CREATE INDEX idx_rag_chunks_tenant     ON rag_chunks(tenant_id);
CREATE INDEX idx_rag_chunks_source     ON rag_chunks(tenant_id, source);

-- Optional ANN acceleration (sqlite-vec). Created only if the extension is present;
-- the app detects availability at startup and falls back to brute-force cosine.
-- Guard DDL (run conditionally from Go, not inline here):
--   SELECT load_extension('vec0');
--   CREATE VIRTUAL TABLE IF NOT EXISTS rag_vec_index USING vec0(
--       tenant_id TEXT, document_id TEXT, embedding float[<DIM>] distance_metric=cosine);
```

#### 3.2.1 Migration `00062` — Down
```sql
-- +goose Down
DROP TABLE IF EXISTS rag_chunks;
DROP TABLE IF EXISTS rag_documents;
```

> **Index update:** append to `00-migration-ownership-index.md`:
> `| 00062 | RAG multi-tenant vectors + provider registry | 10 |`

---

## 4. UI

| Page / partial | File | RBAC resource/action | Notes |
|---|---|---|---|
| API token list | `internal/static/templates/api_tokens_list.html` (new) | `apikeys:read` | table of tokens (name, created, expires, last used, revoked) + "Create" button |
| API token create modal | `api_tokens_form.html` (new) | `apikeys:create` | name, expires_in_hours, scopes (checkbox list) |
| Revoke button | fragment in list | `apikeys:revoke` | `DELETE /api/auth/api-tokens/{id}` via Datastar |
| RAG documents page | `rag_documents.html` (new) | `rag:read` | lists `rag_documents` for tenant; delete + reindex actions |
| RAG upload widget | fragment | `rag:write` | drag/drop txt/md/pdf |
| Driver<->user link | `user_edit.html` / `driver_edit.html` | `users:update` / `drivers:update` | dropdown to bind `drivers.user_id` to a `driver`-role user |

Wire routes in the existing router setup (where `RequireAPIAuth`/`RequirePermission` already
exist) — see 12 file list. Keep the session-cookie path working for browsers; only the
`/api/rag/*` and `/api/auth/api-tokens` routes need the JSON API middleware. The agent
`knowledge_search` tool (`subagents.go:108`) continues to work but now the HTTP surface it
mirrors is authenticated.

---

## 5. Business logic

### 5.1 Encrypted signed sessions (fix 0.2)
Derive an **authenticated-encryption block key** from the cookie secret via HKDF-SHA256 and
pass it as the second arg to `securecookie.New`. This upgrades sign-only -> encrypt-then-MAC
with zero wire-format change.

Add to `internal/auth/session.go` (imports: `crypto/sha256`, `io`, `golang.org/x/crypto/hkdf`):

```go
// DeriveBlockKey derives a 32-byte AES key from the cookie secret using HKDF-SHA256.
// Same secret in => same key out (stable across restarts); secret rotation requires re-login.
func DeriveBlockKey(secret []byte) []byte {
    if len(secret) == 0 {
        secret = []byte("transport-app-fallback-token-secret-change-me")
    }
    r := hkdf.New(sha256.New, secret, nil, []byte("session-enc-v1"))
    key := make([]byte, 32)
    if _, err := io.ReadFull(r, key); err != nil {
        copy(key, secret) // deterministic fallback (dev only)
    }
    return key
}
```

Change `NewSessionStore` (`session.go:50-57`):
```go
func NewSessionStore(cookieSecret string, secure bool) *SessionStore {
    SetTokenSecret([]byte(cookieSecret))
    blockKey := DeriveBlockKey([]byte(cookieSecret))
    return &SessionStore{
        cookieName: "session",
        signer:     securecookie.New([]byte(cookieSecret), blockKey), // <- now ENCRYPTS
        secure:     secure,
    }
}
```
No other code changes — `mustEncode`/`ValidateSession` now transparently encrypt/decrypt.
`SessionData` stays the plaintext struct; `securecookie` handles confidentiality.

### 5.2 Kill the non-revocable fallback (fix 0.3)
`internal/handlers/auth.go:100-104` — remove the `else` branch; if server-side session
creation fails, return 500 (do NOT log the user in):
```go
sessResult, err := h.Services.Auth.CreateSessionForUser(r.Context(), user.ID)
if err != nil || sessResult == nil {
    http.Error(w, "session creation failed; please retry login", http.StatusInternalServerError)
    return
}
role := string(domain.RoleViewer)
if user.Role.Name != "" { role = string(user.Role.Name) }
h.AuthStore.CreateSessionWithToken(w, user.ID.String(), role, user.Name, sessResult.SessionToken)
```
`Login` (`handlers/auth.go:186`) already uses `CreateSessionWithToken`; just ensure
`result.SessionToken` is always present (it is, from `auth_service.go:73-76`).

### 5.3 Tenant derived from identity (fix 0.6)
- In `middleware/api_auth.go:109-113` replace `TenantID: "1"` with the user's tenant from DB.
  Add a `TenantResolver func(ctx context.Context, userID string) (string, error)` parameter to
  `RequireAPIAuth`, or extend `SessionValidator` with
  `TenantForUser(ctx, userID) (string, error)` and call it in both Bearer and cookie branches.
- In `middleware/middleware.go:145` replace `shared.ContextWithTenantID(ctx, "1")` with
  `shared.ContextWithTenantID(ctx, tenant)` resolved from the session's `UserID`.
- Store `tenant_id` on every `sessions` insert (`auth_service.go:59-64`) read from the user row.
- New helper `middleware.TenantFromContext(r)` returns `shared.TenantID` for RAG handlers.

### 5.4 Tenant-scoped, writable Casbin (fix 0.4)
Add a domain to the model (`casbin.go:12-27`): change `g = _, _` to `g = _, _, _` and add the
tenant domain to the matcher (`r.dom == p.dom`). Then back the enforcer with `casbin_rules`:

1. `LoadPolicy` (`casbin.go:40`) reads from `casbin_rules WHERE tenant_id = ?`, mapping
   `ptype`/`v0..v5` to policies/groups (honor Casbin's `persist` adapter contract).
2. Implement `AddPolicy`/`RemovePolicy`/`RemoveFilteredPolicy` to **write to `casbin_rules`**
   (not no-op) and keep memory in sync. Now `handlers/users.go:105,154-155` actually persist.
3. Maintain a per-tenant enforcer cache keyed by `tenant_id` (model is domain-scoped per tenant).
   `Can(userID, resource, action)` resolves the enforcer for the caller's tenant
   (from `auth.ContextUser`/context tenant) and enforces with the tenant as the domain.
4. `SavePolicy` stays a no-op (Casbin pulls/pushes via the fine-grained methods).

Provide a one-time backfill: on migrate, copy existing `role_permissions` + `user_roles` rows
into `casbin_rules` for tenant `'1'`. Add `migrateCasbinToRules(ctx, db)` called once from the
migration runner, guarded by `SELECT COUNT(*) FROM casbin_rules`.

### 5.5 Revocable API tokens (fix 0.5)
- `IssueAPIToken` (`apitoken.go:30`) unchanged (still `b64payload.sig`).
- New storage in `internal/service/auth_service.go`:
  - `CreateAPIToken(ctx, userID, tenantID, name, scopes, expiresAt) (rawToken, id, err)`:
    generate raw via `IssueAPIToken`, store `token_hash = HashToken(raw)`, return raw once.
  - `ListAPITokens(ctx, userID)` -> rows with token omitted.
  - `RevokeAPIToken(ctx, userID, id)` -> set `revoked_at`.
- In `middleware/api_auth.go:76-99` after `ParseAPIToken` succeeds:
  - look up `api_tokens` by `token_hash = HashToken(token)`;
  - if not found OR `revoked_at IS NOT NULL` OR `expires_at < now` -> `ErrTokenRevoked`;
  - enforce `scopes` (deny if requested resource:action not in token scopes, when scopes != `{}`);
  - refresh `last_used_at`.
- Keep `ValidateAPITokenUser` (`auth_service.go:144`) for live role/active check.

### 5.6 Finish DRIVER role (fix 0.7)
- Add to `internal/domain/user/entity.go:45-50`:
  ```go
  RoleDriver RoleName = "driver"
  ```
  and to `DefaultRoleID` (`entity.go:53-66`): `case RoleDriver: return 5`; add `RoleDriver`
  mapping in `RoleNameForID` (`session.go:350-363`).
- Mobile login: `POST /api/auth/driver/login` authenticates by phone+OTP/password, then issues
  a server-side session bound to `drivers.user_id`. The linking UI in 4 sets `drivers.user_id`;
  a driver without a linked user cannot log in.
- Grant `driver:read_own` + `driver:update_own` + `rag:search` (seeded in 3.1).

### 5.7 RAG provider registry (fix 0.8)
Add `internal/rag/providers.go` with a registry and factory:

```go
package rag

import "fmt"

type ProviderName string

const (
    ProviderOpenAI      ProviderName = "openai"
    ProviderHuggingFace ProviderName = "huggingface"
    ProviderOllama      ProviderName = "ollama"
    ProviderHash        ProviderName = "hash"
)

type ProviderConfig struct {
    APIKey    string
    BaseURL   string
    Model     string
    Dimension int
}

type EmbedderFactory func(cfg ProviderConfig) (Embedder, error)

var Registry = map[ProviderName]EmbedderFactory{
    ProviderOpenAI:      newOpenAIEmbedder,
    ProviderHuggingFace: newHuggingFaceEmbedder,
    ProviderOllama:      newOllamaEmbedder,
    ProviderHash: func(c ProviderConfig) (Embedder, error) {
        return NewHashEmbedder(c.Dimension), nil
    },
}

func NewEmbedder(provider ProviderName, cfg ProviderConfig) (Embedder, error) {
    f, ok := Registry[provider]
    if !ok {
        return nil, fmt.Errorf("unknown rag provider %q", provider)
    }
    return f(cfg)
}
```

- `OpenAIEmbedder` (`embedder.go:18-101`): make `dimension` a constructor parameter instead of
  hardcoded `1536` (`embedder.go:51`). The factory reads `RAG_EMBEDDING_DIM` (new env, see 6).
- `HuggingFaceEmbedder` / `OllamaEmbedder`: implement the same `Embed`/`EmbedBatch`/`Dimension`
  interface (`embedder.go:12-16`) hitting their REST endpoints. Both return real semantic
  vectors; default `Dimension` 384/768 depending on model.
- `HashEmbedder` remains the zero-config default when `RAG_PROVIDER=hash` (no network needed).

### 5.8 RAG index path allow-list + lifecycle (fix 0.9, 0.1)
- `handleIndex`/`handleReindex` (`handler.go:106,161`) must reject any `directory` not present in
  an allow-list `RAG_INDEX_DIRS` (already parsed in `config.go:139-144`). Return `403` otherwise.
  This closes the arbitrary-file-read hole from 0.1.
- `Service.IndexDirectory`/`Teach`/`UploadFile` insert a `rag_documents` row first (status
  `indexing`), then write `rag_chunks` (tenant_id + BLOB embedding + dim), then flip to
  `indexed` and set `chunk_count`. Deletion cascades via FK.
- Embedding stored as `[]byte` float32 LE: add helpers
  `floatsToBlob([]float64) []byte` / `blobToFloats([]byte) []float64` in `vector_store.go`.

### 5.9 RAG ANN / hybrid search + reranking + dimension guard (fix 0.9)
New `internal/rag/retriever.go`:
- `HybridSearch(tenantID, query, topK)`: run vector search (sqlite-vec if available else
  brute cosine over `rag_chunks WHERE tenant_id=?`) AND a keyword pass (FTS5 virtual table over
  `rag_chunks(content)` scoped by `tenant_id`), then fuse with Reciprocal Rank Fusion (RRF):
  `score = sum_k 1/(k + rank_k)` with `k=60`. Return merged top-K.
- `Rerank(query, hits)`: optional cross-encoder/LLM rerank step; if disabled, pass-through.
- **Dimension guard**: before storing, assert `len(embedding) == cfg.Dimension`; before search,
  assert `document.embedding_dim == cfg.Dimension`. On mismatch return HTTP `409` with message
  "embedding dimension changed (have X, want Y) — run reindex" (the reindex guard).

### 5.10 RAG route auth wiring (fix 0.1)
In the router setup (where `h.AuthSrv` is available), wrap the RAG handler:
```go
ragRoutes := chi.NewRouter()
ragRoutes.Use(middleware.RequireAPIAuth(app.AuthStore, []byte(cfg.APITokenSecret)))
ragRoutes.Use(middleware.RequirePermission(app.AuthSrv, "rag", "search")) // default; per-route overrides below
ragRoutes.Post("/search", ragH.handleSearch)        // rag:search
ragRoutes.Post("/teach",  ragH.handleTeach)         // rag:write
ragRoutes.Post("/index",  ragH.handleIndex)         // rag:write
ragRoutes.Post("/upload", ragH.handleUpload)        // rag:write
ragRoutes.Post("/reindex",ragH.handleReindex)       // rag:admin (extra middleware.RequirePermission)
ragRoutes.Get ("/stats",  ragH.handleStats)         // rag:read
ragRoutes.Get ("/documents", ragH.handleListDocs)   // rag:read
ragRoutes.Delete("/documents/{id}", ragH.handleDeleteDoc) // rag:admin
r.Mount("/api/rag", ragRoutes)
```
`handleSearch` etc. read `tenant_id` from `middleware.TenantFromContext(r)`.

---

## 6. Config / env

| Var | Default | Purpose | Package reading |
|---|---|---|---|
| `RAG_PROVIDER` | `hash` | Embedder provider: `openai|huggingface|ollama|hash` | `internal/config` -> `rag.NewEmbedder` |
| `RAG_EMBEDDING_DIM` | `1536` (openai) / `384` (hash) | Vector dimension; drives BLOB size + guard | `internal/config`, `embedder.go` |
| `RAG_EMBEDDING_API_KEY` | "" | Provider API key | `config.go:129` |
| `RAG_EMBEDDING_BASE_URL` | `https://api.openai.com/v1` | Provider base URL (HF/Ollama override) | `config.go:130` |
| `RAG_EMBEDDING_MODEL` | `text-embedding-3-small` | Model name | `config.go:131` |
| `RAG_CHUNK_SIZE` | `512` | Chunk size | `config.go:132` |
| `RAG_CHUNK_OVERLAP` | `50` | Chunk overlap | `config.go:133` |
| `RAG_VECTOR_DB_PATH` | `./rag_vectors.db` | Vector store DB path | `config.go:134` |
| `RAG_INDEX_DIRS` | "" | Comma list of allow-listed index roots (see 5.8) | `config.go:139` |
| `RAG_HYBRID` | `true` | Enable FTS5+vector RRF fusion | `internal/rag` |
| `RAG_RERANK` | `false` | Enable reranking pass | `internal/rag` |
| `SQLITE_VEC_ENABLED` | `false` | Use sqlite-vec ANN when present | `internal/rag` |
| `SESSION_BLOCK_KEY` | "" | Optional override for HKDF-derived enc key (see 5.1) | `internal/auth` |

Add the new fields to `RAGConfig` in `internal/config/config.go:127`. Fall back gracefully:
unknown provider -> error at startup; missing API key with openai -> error at startup.

---

## 7. Tests

Put unit tests next to the code; HTTP/integration tests under `internal/auth` + `internal/rag`
with a throwaway SQLite DB (goose `00056`+`00062` applied) and a test tenant `'t1'`.

### 7.1 Auth encrypt round-trip
`internal/auth/session_test.go`:
- `TestSessionEncryptRoundTrip`: `NewSessionStore(secret,false)`; `CreateSessionWithToken`
  writes a cookie; assert the raw `Cookie.Value` is NOT valid JSON (i.e. encrypted) and that
  `ValidateSession` recovers `UserID/Role/Token`. Decoding the cookie value as JSON must fail.
- `TestSessionNotForgeable`: tamper one byte of the cookie -> `ValidateSession` returns false.
- `TestHKDFStable`: `DeriveBlockKey([]byte("x"))` equals `DeriveBlockKey([]byte("x"))` (stable).

### 7.2 Tenant isolation
- `internal/rag/retriever_test.go::TestTenantIsolation`: insert chunk A for tenant `t1` and chunk
  B for tenant `t2` with identical content; `HybridSearch("t1", ...)` returns only A.
- `internal/auth/casbin_test.go::TestTenantCasbin`: grant `rag:write` to a `t1` user; assert
  `Can(t1user,"rag","write")` true and `Can(t2user,"rag","write")` false.

### 7.3 RAG dimension guard
- `internal/rag/retriever_test.go::TestDimensionGuard`: store embeddings with dim 384, then
  set `cfg.Dimension=1536`; `Search`/`Index` must return `409`/error "dimension changed".

### 7.4 Unauth 401
- `internal/rag/handler_test.go::TestRagRequiresAuth`: `POST /api/rag/search` with no
  `Authorization` and no session cookie -> `401`. With a valid session cookie -> `200`.
- `internal/rag/handler_test.go::TestRagIndexAllowList`: `POST /api/rag/index {"directory":"/etc"}`
  with auth but dir not in `RAG_INDEX_DIRS` -> `403`.

### 7.5 API token revocation
- `internal/auth/apitoken_test.go`: create token via service, search with it -> 200; revoke ->
  same token -> `401`/`403`. Expired token -> rejected. Scope mismatch -> `403`.

### 7.6 Coverage gate / pass-before-merge checklist
- `go test ./internal/auth/... ./internal/rag/... ./internal/middleware/...` all green.
- `go vet ./...` clean.
- Manual: log in -> create API token -> call `/api/rag/search` with token -> 200; delete token
  -> 401. Register a new user -> session row exists in `sessions` (revocable). Assign driver
  role + link `drivers.user_id` -> driver mobile login works.

---

## 8. Future / GPS-provider

- **Device auth (mTLS / SIM-ICCID):** extend `sessions`/`api_tokens` with a `device_id` bound to
  a client cert thumbprint (mTLS at the ingress) or to a SIM `ICCID` for vehicle tablets. On
  login, verify the device binding; revoke `device_id` to disable a lost tablet fleet-wide.
  Define a `DeviceAuthenticator` interface (same shape as `SessionValidator`) so mTLS and
  ICCID adapters are pluggable behind a config flag (per AGENTS.md integration rule: MOCK by
  default, real check only when enabled).
- **GPS doc ingestion:** add a `TelematicsProvider`-style adapter that ingests AIS-140/VLT
  device logs and GPS tracks as RAG `rag_documents` of `doc_type='gps'`, chunked by trip/time
  window, so the agent `knowledge_search` can answer "where was vehicle X at time T". Reuse the
  provider interface pattern from AGENTS.md (own MQTT/IMEI hardware primary; LocoNav/WheelsEye/
  MapMyIndia/TelaBit/OBD as pluggable adapters).
- **Scalability:** when `rag_chunks` exceeds ~1M rows, promote the sqlite-vec virtual table to
  the primary ANN index and keep FTS5 for the lexical leg; shard by `tenant_id`.

---

## 9. Edge cases

- Cookie secret rotation: HKDF is deterministic per secret, so rotating `COOKIE_SECRET` invalidates
  all existing sessions (users re-login) — acceptable, documented.
- Two tenants with identical doc content: hybrid search must never cross `tenant_id` (test 7.2).
- Provider returns fewer dims than configured: dimension guard rejects before store (test 7.3).
- `api_tokens` row deleted but token still signed: lookup miss -> `ErrTokenRevoked` (5.5).
- `handleReindex` clears a tenant's docs only, not other tenants (scope by `tenant_id`).
- Driver with `drivers.user_id` pointing at an inactive user: login denied (status check).
- Casbin `Reload()` after a `SavePolicy` no-op: now safe because writes go to `casbin_rules`.
- FTS5 not available in the SQLite build: disable lexical leg, vector-only, log a warning.

---

## 10. Phased rollout (build order)

1. **00056 migration** + `DeriveBlockKey`/encrypted sessions (5.1) + kill fallback (5.2).
2. **Tenant derivation** (5.3) in both middleware paths; backfill `sessions.tenant_id`.
3. **api_tokens** table + service + middleware check (5.5); UI tokens page (4).
4. **Casbin tenant scoping** (5.4): model domain, `casbin_rules`, writable adapter, backfill.
5. **DRIVER role** (5.6): domain const, `drivers.user_id`, seeds, mobile login.
6. **00062 migration** + RAG provider registry (5.7) + lifecycle (5.8).
7. **RAG retriever** hybrid/ANN/rerank/dimension guard (5.9).
8. **RAG route auth** wiring (5.10) + index allow-list (5.8) — closes the CRITICAL gap.
9. Tests (7) + manual checklist; flip `RAG_ENABLED=true` in staging only after 8.

---

## 11. Open items / VERIFY (resolve before coding)

- **`hkdf` dependency:** RESOLVED — `golang.org/x/crypto v0.54.0` is already in `go.mod`
  (no `go get` needed). `DeriveBlockKey` (5.1) compiles as written.
- **`tenants` table:** RESOLVED as ABSENT — there is no `tenants` table in the repo; `tenant_id`
  is a free-form `'1'` string (see migrations 00016-00019). The `REFERENCES tenants(id)` FKs in
  §3.1/§3.2 were removed; do NOT re-add them or `goose up` fails. Single-tenant for now; a future
  multi-tenant spec should create `tenants` and backfill before adding FKs.
- **Casbin domain model semantics:** confirm the chosen RBAC-with-domains matcher matches the
  existing `role_permissions`/`user_roles` seed shape before backfilling `casbin_rules`.
- **sqlite-vec availability:** detect at runtime; never `load_extension` if the binary lacks it.
- **FTS5 build tag:** confirm the SQLite driver (`modernc.org/sqlite`) is compiled with FTS5; if
  not, the lexical leg degrades to `LIKE` over `content`.
- **`device_id` source:** decide generation strategy (UUID per browser stored in localStorage vs
  mTLS thumbprint) before implementing 8.

---

## 12. File list (create / modify)

Create:
- `db/migrations/00056_auth_hardening.sql`            (Up/Down from 3.1)
- `db/migrations/00062_rag_vectors.sql`              (Up/Down from 3.2)
- `internal/rag/providers.go`                        (5.7 registry)
- `internal/rag/retriever.go`                        (5.9 hybrid/ANN/rerank/guard)
- `internal/rag/fts.go`                              (FTS5 lexical leg)
- `internal/auth/session_test.go`, `internal/auth/apitoken_test.go`, `internal/auth/casbin_test.go`
- `internal/rag/retriever_test.go`, `internal/rag/handler_test.go`
- `internal/static/templates/api_tokens_list.html`, `api_tokens_form.html`
- `internal/static/templates/rag_documents.html`

Modify:
- `internal/auth/session.go`                         (5.1 DeriveBlockKey + encrypted signer)
- `internal/handlers/auth.go`                        (5.2 kill fallback; 5.6 driver login)
- `internal/middleware/api_auth.go`                  (5.3 tenant; 5.5 token revocation/scope)
- `internal/middleware/middleware.go`               (5.3 tenant in AuthRequired)
- `internal/auth/casbin.go`                          (5.4 domain model + writable adapter + tenant)
- `internal/service/auth_service.go`                (5.3 tenant on session; 5.5 token CRUD)
- `internal/domain/user/entity.go`                  (5.6 RoleDriver + DefaultRoleID)
- `internal/rag/embedder.go`                         (5.7 dimension as param; HuggingFace/Ollama)
- `internal/rag/vector_store.go`                     (5.8 tenant+BLOB; replace full scan)
- `internal/rag/rag_service.go`                     (5.8 document lifecycle)
- `internal/rag/handler.go`                          (5.8 allow-list; 5.10 per-route perms; tenant)
- `internal/config/config.go`                        (6 new RAG fields)
- `internal/agent/subagents.go`                      (no change; knowledge_search now authenticated)
- `docs/tech-specs/00-migration-ownership-index.md` (add 00062 row)
