# Mobile Driver App (React Native / Expo) — Full Working Design v2

Status: ready (v2 replaces v1; v1's Phases 1–3 + 5 are implemented and verified)
Depends-on: 08 (`drivers.user_id`, `db/migrations/00056_*`), 01 (telemetry ingestion), 02 (geofence), 05 (e-POD via `00032_kharcha_and_epod.sql`)
Migration owner: none — NO new backend DB tables. All driver identity flows through existing APIs.

> v2 ground truth verified 2026-08-22 against live code. Every claim below is
> `path:line` checked. Sections: [1 current state] [2 architecture] [3 screens]
> [4 API contract] [5 offline/sync] [6 GPS/MQTT] [7 auth/security] [8 i18n/analytics]
> [9 gaps ranked] [10 env] [11 tests/CI] [12 phases to "full working"] [13 open items]

---

## 1. Verified current state (what works TODAY)

| Capability | Status | Evidence |
|---|---|---|
| Login vs real backend, flat parse | DONE | `LoginScreen.tsx:32,49-61` parses `data.token && data.user_id`; role `'driver'` client concept |
| Driver profile | DONE | `GET /api/v1/drivers/me` exists (`cmd/server/main.go:678`, handler `internal/handlers/drivers.go:305`); mobile fetches at `LoginScreen.tsx:65-74`, stores `driverId` in zustand |
| Driver trip list (no mocks) | DONE | `App.tsx:328` `GET /api/v1/trips?driver_id=me&page=1&limit=50`; cache fallback `DB.getTrips()` (`App.tsx:342`) |
| e-POD submit (real) | DONE | `DeliveryVerificationScreen.tsx:151-159` multipart POST w/ Bearer; backend under RequireAPIAuth group (`main.go:679`); photo compressed 0.7 JPEG (`:59-69`) |
| Offline queue | DONE | `offlineQueue.ts` SQLite `queued_pods`/`queued_gps`/`offline_expenses`, POD dedupe by trip_id, 7-day expiry purge |
| Sync triggers | DONE | NetInfo reconnect watcher (`syncEngine.ts:16-34`), 15s timer (`:53-62`), opportunistic flush (`:95-96`), manual button (`App.tsx:481`) |
| Telemetry sync endpoint | DONE | `POST /api/v1/telemetry/sync` `{driver_id, logs:[{latitude,longitude,timestamp}]}` → `{synced_ids}` (`syncEngine.ts:101,117-144`) |
| Navigation stack + auth gate | DONE | `App.tsx:192` `isAuthenticated ? <DriverNavigator/> : <AuthNavigator/>`; AuthStack Splash→…→Login→Register→ForgotPassword; DriverStack Main→FirstTimeSetup→ActiveNavigation→DeliveryVerification |
| Demo driver `DRV-9042` / mock trips removed | DONE | zero hits in `mobile/src` (fixtures only in tests) |
| MQTT client (WS, raw-JWT password) | CODE DONE | `mqtt.ts:15-22` clientId `driver_{id}_{rand}`, username=driverId, password=raw JWT; pub `avandab/telemetry/drivers/{id}/gps`, sub `avandab/trips/drivers/{id}/updates` |
| Expense capture (kharcha) | BUILT BUT UNMOUNTED | `ExpenseScreen.tsx:125` real POST `/api/v1/kharcha/expense`; never registered in any navigator |
| i18n (7 locales en/hi/ta/te/kn/mr/gu) | BUILT BUT UNWIRED | `src/i18n.ts` + `src/locales/*.json`; no screen imports it |

### 1.1 Known broken / mock remnants (verified)

| # | Issue | Evidence | Sev |
|---|---|---|---|
| G1 | `ExpenseScreen` unreachable — kharcha flow dead in app | not imported by `App.tsx` | M |
| G2 | `ForgotPasswordScreen` POSTs `${API}/forgot-password` (web-form route, urlencoded, renders HTML) and shows success on ANY error incl. network fail | `ForgotPasswordScreen.tsx:26,35-39`; route is page+form (`main.go:904-905`, `auth.go:469-470`) | M |
| G3 | `RegisterScreen` reads nested `data.user?.id`, sets uppercase `'DRIVER'`, silently sends hardcoded plate `MH-12-AB-9942` when empty | `RegisterScreen.tsx:42,62,64` | M |
| G4 | Silent GPS fake relocated: App.tsx falls back `loc.latitude \|\| DEFAULT_LATITUDE` | `App.tsx:263-264` | M |
| G5 | ActiveNavigation nav HUD 100% static mock (turn card, speed, stops, ETA, REF#) | `ActiveNavigationScreen.tsx:70-106` | M |
| G6 | LiveDriverTrackingMap marker labels hardcoded ("Mumbai Port Terminal 2", "#TRK-9942", "Pune Logistics Hub B"); straight-line polyline, no route geometry | `LiveDriverTrackingMap.tsx:24-33,63-77` | L |
| G7 | Logout does not call `MQTT.disconnect()` nor stop location tracking | `App.tsx:358` only `logout()` | M |
| G8 | Signature pad missing-native-dep fallback injects `"MOCK SIGN"` base64 | `DeliveryVerificationScreen.tsx:23-30,255-260` | L |
| G9 | syncEngine DB-path logs omit `accuracy_m` (storage schema lacks column); offlineQueue path sends it | `syncEngine.ts:117-120` vs `offlineQueue.ts:363` | L |
| G10 | No `mobile/.env.example` committed | file absent | L |
| G11 | Backend broker TCP-only (`tcp://localhost:1883`, `main.go:550-561`); no WS listener on 9001 → MQTT always fails → HTTP-only telemetry in practice | `network.ts:11-13` expects `ws://127.0.0.1:9001` | M |
| G12 | DriverStack params fall back `tripId \|\| '1'` — soft mock default | `App.tsx:145,147,156` | L |

---

## 2. Architecture

```
mobile/
├── App.tsx                    # root: navigators, auth gate, MainScreen (trip list/tabs)
├── src/
│   ├── components/            # screens (see §3)
│   ├── stores/authStore.ts    # zustand + expo-secure-store (token, user, driverId)
│   ├── services/
│   │   ├── mqtt.ts            # WS MQTT pub/sub, raw-JWT password, never throws
│   │   ├── telemetry.ts       # permissions, getCurrentPosition, watchPositionAsync 10s/20m
│   │   ├── syncEngine.ts      # batched GPS flush (20/chunk), auto-sync timer, queue flush
│   │   ├── offlineQueue.ts    # SQLite queues: pods, gps, expenses; flush(); dedupe; TTL
│   │   ├── storage.ts         # AsyncStorage trips cache + SQLite avandab_offline.db (trips, gps_logs, expenses)
│   │   └── analytics.ts       # PostHog (gated EXPO_PUBLIC_POSTHOG_API_KEY)
│   ├── utils/tripMapper.ts    # backend snake_case → mobile Trip union
│   ├── types/api.ts           # Driver, Trip, Vehicle
│   ├── constants/network.ts   # API/MQTT base URLs from EXPO_PUBLIC_*
│   └── i18n.ts                # 7 locales, AsyncStorage persistence (unwired)
└── __tests__/                 # jest: auth, network, offlineQueue, tripMapping, DeliveryVerification, smoke
```

Data flow (one direction of truth):

```
UI screens ──fetch──▶ REST API (Bearer) ──on failure──▶ SQLite cache/offline queue
     ▲                                                      │
     └──────────── react-query / zustand rehydrate ◀── flush() on reconnect/timer
GPS: expo-location ──▶ SQLite log ──▶ SyncEngine ──▶ POST /telemetry/sync (+ MQTT best-effort)
```

Principles:
1. REST is source of truth. GraphQL is dead for this app (spec 14 recommends removal).
2. MQTT is best-effort push channel; **never block UI on MQTT** — HTTP telemetry is the reliable path.
3. Every mutating action (POD, expense) survives offline via SQLite queue with dedupe.
4. Driver identity ALWAYS from authStore (`user.driverId || user.id`), never constants.

---

## 3. Screens (full spec)

### AuthStack
| Screen | Purpose | Backend | State |
|---|---|---|---|
| Splash | 2s animated brand → auto-nav GetStarted; version tag | none | done |
| GetStarted | marketing hero | none (static stats acceptable for shell) | done |
| OnboardingOverview / BookingSchedule / EarningsOverview | static info shells during onboarding tour | none | done (permitted shells) |
| Login | email+password → token | `POST /api/v1/auth/token` then `GET /api/v1/drivers/me` | done |
| Register | self-signup | `POST /api/v1/auth/register` | G3 fixes needed |
| ForgotPassword | reset request | needs JSON API (see G2) | BROKEN |

### DriverStack
| Screen | Purpose | Backend | State |
|---|---|---|---|
| Main | tabs: trip list (react-query, cached), manual sync, GPS permission flow, camera/barcode panel, SQLite log viewer | `GET /api/v1/trips?driver_id=me` | done (remove G4 fake coords) |
| FirstTimeSetup | checklist | none (local state) | shell; persist later |
| ActiveNavigation | live map + turn HUD for active trip | coords live via telemetry; nav data needs trip destination + routing (G5) | PARTIAL |
| DeliveryVerification | ePOD: photo, signature, consignee, qty short/damage/refusal, GPS | `POST /api/v1/trips/{id}/deliver-pod` multipart | done (kill MOCK SIGN G8) |
| ExpenseScreen (kharcha) | fuel/toll/rto/tyre + receipt photo + GPS | `POST /api/v1/kharcha/expense` multipart | UNMOUNTED (G1) |

---

## 4. API contract (verified against backend)

All calls: `Authorization: Bearer <token>`; base = `getApiBaseURL()`.
Token lifetime per `POST /api/v1/auth/token` response `expires_at`. On any `401`: clear session → AuthStack.

### 4.1 POST /api/v1/auth/token
```json
// req  { "email": "...", "password": "..." }
// res  { "token", "expires_at", "user_id", "role" }   // FLAT — never data.user.id
// err  400 invalid body · 401 invalid credentials
```

### 4.2 GET /api/v1/drivers/me  (main.go:678)
```json
{ "driver_id": "drv_12", "user_id": "u_8f3a2", "name": "...", "phone": "...",
  "status": "on_trip", "vehicle_plate": "...", "current_location": { "latitude", "longitude" } }
// 404 → "Contact dispatcher to link your driver profile"; block trip list
```

### 4.3 GET /api/v1/trips?driver_id=me&page=1&limit=50
```json
{ "trips": [ { "id", "trip_number", "driver_name", "vehicle_plate", "origin",
  "destination", "status", "departure_time" } ], "total" }
// status enum snake_case: pending assigned started reached_pickup in_transit delivered completed cancelled
// mapped by utils/tripMapper.ts → PENDING | IN_TRANSIT | COMPLETED | CANCELLED
```
Cache: success → `DB.saveTrips`; failure → serve `DB.getTrips()` stale-with-banner.

### 4.4 POST /api/v1/trips/{id}/deliver-pod  (multipart, main.go:679)
Fields: `consignee_name`*, `consignee_phone`, `notes`, `pod_photo` (jpeg/png ≤10MB),
`pod_signature_data` + `signature_dataurl`, `quantity_short`, `damage_qty`,
`refusal_reason`, `latitude`, `longitude`.
```json
// 200 { "trip_number", "status": "delivered", "pod_url" }
// 401 bearer · 403 not your trip (do NOT queue) · 404 · 400 bad form (JSON errors)
```
Client validation: require photo OR consignee_name OR signature before submit.

### 4.5 POST /api/v1/kharcha/expense  (multipart)
Fields: `trip_id`, `type` (= `expense_type`: fuel|toll|rto|tyre), `amount`, `notes`,
`receipt_photo`, `lat`, `lng`. Failure → `OfflineQueue.enqueueExpense`.

### 4.6 POST /api/v1/telemetry/sync
```json
// req { "driver_id", "logs": [ { "latitude", "longitude", "timestamp", "accuracy_m"? } ] }
// res { "success": true, "synced_ids": [1,2,3] }
```
Batch 20/chunk, continue-on-failure, mark synced only ids in response.

### 4.7 MQTT (best-effort)
- Broker: dev `ws://127.0.0.1:9001`, prod `wss://api.avandab.com:8883` (needs ops WS listener — G11).
- Auth: username = driverId, password = **raw JWT** (no `Bearer ` prefix).
- Publish QoS1: `avandab/telemetry/drivers/{driverId}/gps` `{driver_id,latitude,longitude,timestamp}`.
- Subscribe: `avandab/trips/drivers/{driverId}/updates` (dispatch push — payload consumed when backend publishes).
- Disconnect/error → silent; HTTP sync continues.

---

## 5. Offline & sync design

Local SQLite (device-only, never a backend migration):
- `storage.ts` → `avandab_offline.db` (WAL): `trips` (list cache), `offline_gps_logs(id, driver_id, latitude, longitude, timestamp, synced)` — unsynced read `LIMIT 50`, `offline_expenses`.
- `offlineQueue.ts` → `offline_queue.db`: `queued_pods` (dedupe UNIQUE trip_id, full field set incl. signature + lat/lng), `queued_gps` (carries `accuracy_m`), `offline_expenses`. Column-upgrade path for schema drift. 7-day expiry purge on init.

Flush triggers (all four): NetInfo reconnect watcher · 15s auto-sync timer · opportunistic flush inside `syncPendingLogs` · manual SYNC BACKEND button.
Flush order: PODs → expenses → GPS. Clear-on-success only; 4xx (esp. 403) drops the item (never retryable), 5xx/network keeps it.

State machine (POD):
```
capture → validate(photo∨name∨signature) → POST
  200        → clearPOD(tripId) → mark COMPLETED → onComplete()
  403/404    → Alert error, drop (not queued)
  net-fail   → enqueuePOD(dedupe tripId) → "Saved offline" → onComplete()
reconnect  → flush() retries until acked or TTL expiry (7d)
```

---

## 6. GPS & background design (current + target)

Today: foreground `watchPositionAsync` (10s / 20m) in `telemetry.ts`; every fix → SQLite; SyncEngine batches to `/telemetry/sync`; MQTT publish attempted per fix when connected. GPS-denied → explicit error state.

Target (Phase P4):
1. Background: `expo-task-manager` + `Location.startLocationUpdatesAsync` (deferred updates ~15s) so stream survives backgrounding; iOS `showsBackgroundLocationIndicator`.
2. Adaptive rate: 10s while `in_transit`, 60s idle/parked (battery guard).
3. Dedup/idempotency: offline log id becomes `provider_msg_id` so server `(imei/provider_msg_id)` unique index dedupes retries (aligns spec 01 §12).
4. Device identity: driver phones have no IMEI → send `driver_id` as identity field (decision D2 in §13).

---

## 7. Auth & security

- Token in `expo-secure-store` (`auth_token`), user JSON `auth_user`. Never AsyncStorage for token.
- Role is a CLIENT concept (`'driver'` lowercase) derived from successful `/drivers/me`; backend roles remain admin/dispatcher/accountant/viewer. Register must stop sending `'DRIVER'` (G3).
- Logout hygiene: `logout()` must also `MQTT.disconnect()` + `Telemetry.stopLiveLocationTracking()` + clear SQLite unsynced flag choice prompt (keep queued PODs! they belong to completed work) (G7).
- 401-anywhere → wipe session → AuthStack (single interceptor point in a shared `authFetch` helper — recommended refactor).
- Photo/signature uploads ride the same 10MB multipart cap; compress 0.7 JPEG first (done).

---

## 8. i18n & analytics

- i18n: `i18n.ts` ready with en/hi/ta/te/kn/mr/gu; locale persisted `@avandab_language`. Wire-in = wrap app in provider + replace literal strings screen-by-screen starting with Login + DeliveryVerification (highest driver value). Dead until wired.
- Analytics: PostHog gated on `EXPO_PUBLIC_POSTHOG_API_KEY`; console fallback. Track: login, trip_open, pod_submit, pod_queued, sync_flush, expense_submit.

---

## 9. Gap closure list (ranked, each = shippable PR)

| PR | Fix | Files |
|---|---|---|
| 1 | Mount `ExpenseScreen` in DriverNavigator + entry point (button/tab on Main) | `App.tsx` |
| 2 | ForgotPassword: add JSON API `POST /api/v1/auth/forgot-password` (email → 200 `{ok:true}`, generic response regardless of account existence); mobile posts JSON there; remove fake-success catch | `internal/handlers/auth.go`, `cmd/server/main.go`, `ForgotPasswordScreen.tsx` |
| 3 | RegisterScreen: flat parse `data.user_id`, role `'driver'`, drop hardcoded plate fallback (require field) | `RegisterScreen.tsx` |
| 4 | Remove GPS fake in App.tsx → error state + "enable location" CTA | `App.tsx:263-264` |
| 5 | Logout hygiene: disconnect MQTT, stop tracking | `App.tsx`, `mqtt.ts` already has `disconnect()` |
| 6 | Remove MOCK SIGN path; ship `react-native-signature-canvas` properly (config plugin/prebuild) | `DeliveryVerificationScreen.tsx:23-30,255-260` |
| 7 | Add `accuracy_m` column to storage.ts gps table + emit in syncEngine | `storage.ts`, `syncEngine.ts` |
| 8 | Commit `mobile/.env.example` | new file |
| 9 | Ops: mosquitto `listener 9001` + `protocol websockets` (or accept HTTP-only telemetry permanently — decision D1) | deploy config |
| 10 | ActiveNavigation real data: trip origin/destination coords from `/trips`, real ETA via routing provider (OSRM public or Mapbox), stop list from booking legs; kill static HUD strings | `ActiveNavigationScreen.tsx`, maybe backend trip coords |
| 11 | Wire i18n into Login + DeliveryVerification first | `i18n.ts`, 2 screens |
| 12 | Shared `authFetch` wrapper: single 401 interceptor, JSON error parsing | new `src/services/http.ts` |

---

## 10. Config / env

| Var | Dev default | Prod | Read by |
|---|---|---|---|
| `EXPO_PUBLIC_API_SCHEME` | `http` | `https` | network.ts |
| `EXPO_PUBLIC_BACKEND_HOST` | `127.0.0.1` | `api.avandab.com` | network.ts |
| `EXPO_PUBLIC_API_PORT` | `8080` | `443` | network.ts |
| `EXPO_PUBLIC_MQTT_SCHEME` | `ws` | `wss` | network.ts |
| `EXPO_PUBLIC_MQTT_BROKER_PORT` | `9001` | `8883` | network.ts |
| `EXPO_PUBLIC_POSTHOG_API_KEY` | unset (console fallback) | set | analytics.ts |

Create `mobile/.env.example` with the above (G10). Backend: `MQTT_URL` stays `tcp://localhost:1883` for server-side producers; WS listener is pure broker config (D1).

---

## 11. Tests & CI

Existing: `__tests__/` authStore, network, offlineQueue, tripMapping, DeliveryVerification(.tsx), smoke. Scripts: `typecheck/lint/test/coverage/mutation` (Stryker configured).

Rules:
- CI gates: `cd mobile && npm run typecheck && npm run lint && npm test` (`.github/workflows/mobile.yml`).
- New code ships with tests: each gap-fix PR includes one happy-path + one offline/failure-path test.
- Coverage floor ≥70% on `stores/`, `services/`, `LoginScreen`, `DeliveryVerificationScreen`.
- Mock trips (`TRP-8492`) allowed ONLY inside test fixtures.

---

## 12. Phases to "full working"

- **P0 (done):** build toolchain, auth, drivers/me, trip list, POD, offline queue, sync engine, navigation.
- **P1 — reachability:** PRs 1–5 above (expense mount, forgot-password API, register fixes, GPS fake removal, logout hygiene). Exit: every registered screen hits real backend or is an intentional shell; no fake success paths.
- **P2 — trust:** PRs 6–8 (signature dep, accuracy_m, .env.example) + JSON error alignment audit. Exit: zero mock strings reachable from UI.
- **P3 — transport decision:** D1 broker WS vs HTTP-only; if WS: PR 9 + verify MQTT pub/sub E2E. Exit: telemetry streaming confirmed either way.
- **P4 — background + nav:** background location task (§6), adaptive rate, PRs 10–12. Exit: POD possible after force-quit + reopen offline; nav HUD shows real trip data.
- **P5 — polish:** i18n rollout all screens, FirstTimeSetup persistence, push dispatch via `avandab/trips/drivers/{id}/updates` + `expo-notifications`.

Definition of "full working app":
1. Fresh install → login → sees own trips (online or stale cache).
2. Go offline mid-trip → POD + expense captured → kill app → reopen → still queued → network returns → auto-flush → server shows delivered + expense.
3. GPS streams (foreground or background) within 15s of movement, deduped server-side.
4. Logout leaves no live listeners; relaunch restores session from SecureStore.
5. `typecheck && lint && test` green in CI.

---

## 13. Open items / decisions

- **D1 — MQTT WS broker:** ops runs `listener 9001 protocol websockets`, else mobile permanently HTTP-telemetry-only and we delete `mqtt.ts`. Either is fine; decide before P3.
- **D2 — device identity:** no IMEI on phones → carry `driver_id` as identity in telemetry frames (recommended; aligns spec 01 VERIFY item). Confirm with spec 01 owner.
- **D3 — forgot-password API shape:** tokenized email link (needs mailer) vs OTP-vs-phone flow. PR 2 blocks on this choice; generic-response anti-enumeration mandatory.
- **D4 — web e-POD caller:** deliver-pod now mounted under BOTH groups (main.go:679 API + :1074 cookie) — confirm web fragment flow still used; if not, delete :1074 mount.
- **Routing provider** for turn-by-turn (OSRM free vs Mapbox key) — cost decision before P4.
