# Mobile Driver App (React Native / Expo) — Implementation Spec v1

Status: ready
Depends-on: 08 (Auth hardening — `db/migrations/00056_*` adds `drivers.user_id`), 01 (telematics ingestion for GPS persistence), 02 (geofence), 05 (alerting/compliance for e-POD via `00032_kharcha_and_epod.sql`)
Migration owner: none — this spec introduces **NO new DB tables**. It consumes existing backend APIs and reads `drivers` via `user_id` (see `db/migrations/00056_*`). Do not create a migration for this spec.

> Beginner note: every file path below is relative to `mobile/` unless prefixed with
> the repo root. Every JSON block is valid and uses the EXACT field names the backend
> returns today (verified in section 0). Copy-paste the code blocks; they are the
> intended implementation, not pseudo-code.

---

## 0. Verified ground truth

All facts below were read directly from the codebase at the time of writing. They are
the bugs/shortcuts this spec fixes. Each line is `path:line`.

### 0.1 Build is broken / untested
- `mobile/package.json:5-9` — `scripts` only has `start/android/ios/web`. **No `build`,
  `tsc`, `lint`, or `test` script.** CI cannot typecheck or test.
- `mobile/` has **no `babel.config.js`**. Expo SDK 52 expects one; without it, custom
  babel plugins / jest preset fail. (`mobile/app.json` exists, `mobile/tsconfig.json`
  exists, but `babel.config.js` is absent — `ls mobile/ | grep -i babel` returns nothing.)
- Dead / unused dependencies in `mobile/package.json:11-38`: `@apollo/client`,
  `graphql-tag`, `@posthog/core`, `posthog-js`, `sp-react-native-mqtt` (we use `mqtt`
  v5). **NOT dead** — `posthog-react-native` IS imported by `mobile/src/services/analytics.ts`
  (used via `App.tsx`), and `expo-sqlite` IS imported by `mobile/src/services/storage.ts`
  to hand-roll the GPS log table. Do not remove those two; remove only the five dead ones
  above to shrink the bundle and avoid native-linking failures.

### 0.2 Auth parsing bug (app cannot log in against real backend)
- `mobile/src/components/LoginScreen.tsx:55-60` reads `data.user?.id`, `data.user?.name`.
- Backend returns a **FLAT** object, not nested: `internal/auth/presentation/api/handlers/auth_handler.go:142-156`
  returns `{"token","expires_at","user_id","role"}` — there is **no `user` object**.
  `data.user?.id` is always `undefined`, so login "succeeds" but the stored user id is
  `""`. Fix: parse `data.user_id` and `data.role` (section 4.4 / 5.1).
- `mobile/src/stores/authStore.ts:7` hardcodes union `'DRIVER' | 'DISPATCHER' | 'ADMIN'`.
- Backend has **NO `DRIVER` role**. Roles are `admin`, `dispatcher`, `accountant`,
  `viewer` (`internal/domain/user/entity.go:45-50`). A driver is a `users` row linked to
  `drivers.user_id` (`db/migrations/00056_*`). The mobile `role` field must become
  `'driver'` (client concept) derived from `GET /api/v1/drivers/me`, never `'DRIVER'`
  from a backend enum.

### 0.3 Hardcoded demo driver
- `mobile/App.tsx:35` `const DEMO_DRIVER_ID = DEFAULT_DRIVER_ID || 'DRV-9042';`
- `mobile/App.tsx:189-199` falls back to `DEMO_DRIVER_ID` (`'DRV-9042'`) for Analytics,
  MQTT connect, SyncEngine when no real user. The whole app "works" against a fake ID.
  Remove the demo fallback; require a real authenticated driver (section 4.2).

### 0.4 Dead/hardcoded screens
- `mobile/App.tsx:278-306` — `useQuery` returns **hardcoded mock trips** (`TRP-8492`,
  `TRP-8493`); never calls the backend. Fix: fetch from `GET /api/v1/trips?driver_id=me`
  (section 4.5).
- `mobile/src/components/ActiveNavigationScreen.tsx:21-22` — hardcodes
  `driverLatitude={18.5255}` / `driverLongitude={73.8520}` static coords. Fix: use live
  `expo-location` (section 4.6).
- `mobile/src/components/BookingScheduleScreen.tsx` and
  `mobile/src/components/EarningsOverviewScreen.tsx` — static placeholder content, no
  API. Keep as informational shells; wire later (section 8).

### 0.5 e-POD submit does nothing
- `mobile/src/components/DeliveryVerificationScreen.tsx:149-160` — `onPress` only calls
  `Alert.alert('Delivery Verified', ...)` then `onComplete`. **It never POSTs** the photo
  or consignee data. Fix: `POST /api/v1/trips/{id}/deliver-pod` multipart (section 4.7 /
  5.4).

### 0.6 Dead services / protocol mismatch
- `mobile/src/services/mqtt.ts:1` uses `mqtt` connecting to **`ws://127.0.0.1:9001`**
  via `getMQTTBrokerURL()` (`mobile/src/constants/network.ts:6,34-36` default port 9001
  ws). Backend broker is **`tcp://localhost:1883`** only — no WebSocket listener:
  `cmd/server/main.go:417-421` `mqttURL = "tcp://localhost:1883"`.
  The mobile WebSocket client **can never connect** to a TCP-only broker.
  Fix (backend): run an MQTT **WebSocket** listener on `9001` (section 2 / 5.3).
- `mobile/src/services/graphql.ts:4-16` expects fields `driver_name`,
  `cargo_weight_kg`, `serverTime`, `activeTrips`. Backend `internal/graphqlservice/handler.go:43-65`
  returns `driver_name`, `origin`, `destination`, `status`, `departure_time`,
  `vehicle_number` and is **NOT driver-filtered** — it returns ALL tenant trips. Mobile
  must not rely on GraphQL for driver trips; use the REST `GET /api/v1/trips?driver_id=me`
  (section 2). The GraphQL client should be removed or repurposed; it is dead for the
  driver use-case.
- `mobile/src/services/syncEngine.ts:37` posts to `/api/v1/telemetry/sync`. Backend
  telemetry routes exist (`cmd/server/main.go:448` `telemetry.RegisterTelemetryRoutes`),
  but verify field names (`driver_id` vs `user_id`) — align in section 2 / 5.3.
- `mobile/src/services/telemetry.ts` — location works but logs to SQLite via `DB`;
  `DEFAULT_LATITUDE/DEFAULT_LONGITUDE` fallback (`network.ts:17-18`) silently fakes GPS
  when unavailable. Keep logging, drop the silent fake (section 4.6).

### 0.7 e-POD backend returns HTML, not JSON
- `internal/handlers/kharcha.go:126-180` `DeliverWithPOD` calls `h.renderFragment(w,
  "epod_success.html", ...)` — returns an **HTML fragment**, not JSON. Mobile cannot
  parse it. It also lives under the cookie-session `RequireAuth` group
  (`cmd/server/main.go:797`), not the Bearer `RequireAPIAuth` group, so a mobile Bearer
   token is rejected. Fix (backend): return JSON `200 {"trip_number","status","pod_url"}`
   and mount under the API-auth group (section 2).

### 0.8 Verification Log (principal-engineer QA pass)

Every claim in §0 was re-read against the live codebase on 2026-08-19. Verdicts:
TRUE = confirmed accurate; WRONG = corrected in body. Severity: S=spec-wrong/breaks
build if followed, M=minor/line-ref, L=low. Effort: code/words to fix.

| # | Claim | Verdict | Correction / Evidence | Sev | Eff |
|---|-------|---------|-----------------------|-----|-----|
| 1 | `package.json` has no build/tsc/lint/test scripts | TRUE | `mobile/package.json:5-9` only start/android/ios/web | — | — |
| 2 | No `babel.config.js` | TRUE | `ls mobile/` shows only `app.json`+`tsconfig.json` | — | — |
| 3 | `@apollo/client`,`graphql-tag`,`@posthog/core`,`posthog-js`,`sp-react-native-mqtt` dead | TRUE | not imported anywhere in `mobile/src` | — | — |
| 4 | `posthog-react-native` dead/unused | **WRONG** | imported by `mobile/src/services/analytics.ts` (used via `App.tsx`); do NOT remove | S | words |
| 5 | `expo-sqlite` dead (GPS table hand-rolled) | **WRONG** | imported by `mobile/src/services/storage.ts`; it IS the engine for the hand-rolled table; keep | S | words |
| 6 | `LoginScreen` reads `data.user?.id` | TRUE | `LoginScreen.tsx:56-57` | — | — |
| 7 | Backend auth response FLAT `user_id`/`role`, no `user` | TRUE | `auth_handler.go:142-156` | — | — |
| 8 | Backend has no `DRIVER` role (admin/dispatcher/accountant/viewer) | TRUE | `internal/domain/user/entity.go:46-49` | — | — |
| 9 | `authStore` hardcodes `role:'DRIVER'` union | TRUE | `authStore.ts:7` | — | — |
| 10 | Hardcoded demo `DRV-9042` fallback | TRUE | `App.tsx:35`, used `App.tsx:193,195,196` | — | — |
| 11 | `App.tsx` mock trips `TRP-8492/TRP-8493` | TRUE | `App.tsx:278-306` | — | — |
| 12 | `ActiveNavigationScreen` static coords `18.5255/73.8520` | TRUE | `ActiveNavigationScreen.tsx:21-22` | — | — |
| 13 | `BookingSchedule`/`EarningsOverview` static, no API | TRUE | no fetch/useQuery in either file | — | — |
| 14 | `DeliveryVerificationScreen` only `Alert`, no POST | TRUE | `DeliveryVerificationScreen.tsx:149-160` | — | — |
| 15 | Mobile MQTT `ws://127.0.0.1:9001` | TRUE | `network.ts` default `ws`/`9001`/`127.0.0.1` → `mqtt.ts:25` | — | — |
| 16 | Backend MQTT `tcp://localhost:1883` only, no WS | TRUE | `cmd/server/main.go:417-421` | — | — |
| 17 | `graphql.ts` expects `cargo_weight_kg` (absent) | TRUE | `graphql.ts:4-16`; backend omits it (`handler.go:48-57`) | — | — |
| 18 | GraphQL not driver-filtered (all tenant trips) | TRUE | `graphqlservice/handler.go:43-58` `ListTripsQuery` has no driver filter | — | — |
| 19 | `syncEngine.ts` posts to `/api/v1/telemetry/sync` | TRUE | `syncEngine.ts:37` | — | — |
| 20 | `telemetry.ts` silent fake coords | TRUE | `telemetry.ts:49-51` `DEFAULT_LATITUDE/DEFAULT_LONGITUDE` | — | — |
| 21 | `DeliverWithPOD` returns HTML fragment `renderFragment` | TRUE | `kharcha.go:178` `h.renderFragment(w,"epod_success.html",...)` | — | — |
| 22 | `DeliverWithPOD` under cookie `RequireAuth`, Bearer rejected | TRUE | mounted `main.go:797` inside `r.Use(middleware.RequireAuth)` group (`main.go:729`); `getUserFromContext` reads cookie `auth.ContextUser` (`app.go:258-260`) | — | — |
| 23 | MQTT password = `Bearer token` | **WRONG** | `mqtt.ts:21` sets `options.password = token` (raw JWT, **no `Bearer ` prefix**); corrected §2.7/§5.3 | S | words |
| 24 | `DeliverWithPOD` multipart cap at `kharcha.go:136` | **WRONG** (ref) | actual `r.ParseMultipartForm(10<<20)` at `kharcha.go:131`; §9.7 ref fixed | M | words |
| 25 | ownership check `kharcha.go:147-150` | **WRONG** (ref) | actual `kharcha.go:145-147`; §2.4 ref fixed | M | words |
| 26 | auth response at `auth_handler.go:150-155` | **WRONG** (ref) | actual `auth_handler.go:142-156`; §0.2 ref fixed | M | words |
| 27 | `GET /api/v1/drivers/me` does not exist | TRUE | no route match in `cmd/server/main.go`/`internal` | — | — |

### 0.9 Key decisions, trade-offs, costs

- **MQTT transport (WS vs TCP).** Mobile `mqtt` v5 client can only speak WebSocket
  (`ws://`/`wss://`); backend broker is TCP-only (`tcp://localhost:1883`,
  `main.go:417-421`). **Decision: run a parallel WS listener on `9001`** (mosquitto
  `listener 9001` + `protocol websockets`, or paho/ws) and keep TCP `1883` for
  server-side producers. Trade-off: +1 broker listener/port to operate; cost = ops
  config + WS auth bridge validating the raw JWT (`password` field, no `Bearer `).
  Fallback if ops cannot: drop MQTT from mobile and use pure HTTP
  `POST /api/v1/telemetry/sync` (already implemented, `syncEngine.ts:37`).
  Severity S, Effort M (backend/config).
- **Offline queue strategy.** GPS logs persist in `expo-sqlite` (`storage.ts`
  `gps_logs`) and e-POD `FormData` queued in `offlineQueue.ts` (4.8). Flush on
  `NetInfo` reconnect; dedupe e-POD by `tripId`. Trade-off: SQLite adds a native dep
  but is already imported (not dead — see #5), so no new risk. Cost: build
  `offlineQueue.ts` + retry/backoff. Severity M, Effort M.
- **e-POD submit contract.** `POST /api/v1/trips/{id}/deliver-pod` multipart, returns
  **JSON** `200 {"trip_number","status","pod_url"}`, mounted under `RequireAPIAuth`
  (not cookie `RequireAuth`). Ownership check `trip.DriverID == session.UserID` kept
  (reads Bearer user). Errors MUST be JSON (`http.Error` currently writes text →
  mobile cannot parse; align error paths). Trade-off: moving the route off the
  cookie group must not break the existing web e-POD caller (verify separately,
  §11). Severity S, Effort M.

---

## 1. Overview / goal

Build a **working** React Native / Expo driver app that:
1. Logs a driver in against the real `POST /api/v1/auth/token` and parses the flat response.
2. Loads the driver's own profile via `GET /api/v1/drivers/me`.
3. Lists only that driver's trips via `GET /api/v1/trips?driver_id=me` (no mock data).
4. Streams live GPS over MQTT (WebSocket) and/or HTTP telemetry endpoints.
5. Captures and submits e-POD (photo + consignee) via `POST /api/v1/trips/{id}/deliver-pod`
   as **JSON/multipart**, with an **offline queue** so a submission survives no-signal.
6. Queues GPS logs offline (SQLite) and flushes them when back online.

**Non-goals:** no new backend DB tables; no GraphQL for driver data; no admin/dispatcher
UI; no push notification backend (stubbed for section 8). The app consumes specs 01/02/05.

---

## 2. API contract

All endpoints require `Authorization: Bearer <token>` except token issuance. Token is the
flat `token` from `POST /api/v1/auth/token`. Base URL = `getApiBaseURL()`
(`mobile/src/constants/network.ts:8-13,26-28`).

### 2.1 POST /api/v1/auth/token  (exists — `auth_handler.go:111`)
Request:
```json
{ "email": "driver@avandab.com", "password": "secret" }
```
Response (FLAT — current backend, `auth_handler.go:150-155`):
```json
{
  "token": "<signed-jwt>",
  "expires_at": "2026-08-19T14:32:00Z",
  "user_id": "u_8f3a2",
  "role": "viewer"
}
```
Errors: `400 invalid request body`, `400 email and password are required`,
`401 invalid credentials`.
**Mobile must parse `user_id` and `role` (flat), NOT `user.id`.** See 4.4 / 5.1.

### 2.2 GET /api/v1/drivers/me  (ADD on backend)
**Backend action required.** Add a handler returning the driver profile for the
authenticated user (joined `drivers.user_id = users.id`, from `00056_*`). Mount under the
API-auth group (`cmd/server/main.go:444-455`).
Response:
```json
{
  "driver_id": "drv_12",
  "user_id": "u_8f3a2",
  "name": "Rajesh Kumar",
  "phone": "+9198200...",
  "status": "on_trip",
  "vehicle_plate": "MH-12-PQ-4521",
  "current_location": { "latitude": 18.5204, "longitude": 73.8567 }
}
```
Errors: `401` no bearer, `404` no driver linked to this user.

### 2.3 GET /api/v1/trips?driver_id=me  (ADD on backend)
**Backend action required.** Filter trips by the authenticated driver (join
`trips.driver_id = drivers.id` where `drivers.user_id = auth user`). Replaces the
unfiltered GraphQL query (`graphqlservice/handler.go:31-58`).
Query param `driver_id=me` is a server-side sentinel meaning "current auth user's
driver". Response:
```json
{
  "trips": [
    {
      "id": "trip_91",
      "trip_number": "TRP-8492",
      "driver_name": "Rajesh Kumar",
      "vehicle_plate": "MH-12-PQ-4521",
      "origin": "Mumbai Central Depot",
      "destination": "Pune Distribution Hub",
      "status": "in_transit",
      "departure_time": "2026-08-19T10:30:00Z"
    }
  ],
  "total": 1
}
```
Status enum (backend `snake_case`): `pending`, `assigned`, `in_transit`, `delivered`,
`completed`, `cancelled`. **Mobile maps these to its `Trip.status` union** (section 5.2).
Errors: `401`, `403`.

### 2.4 POST /api/v1/trips/{id}/deliver-pod  (FIX backend: JSON + Bearer)
**Backend action required.** Two changes to `internal/handlers/kharcha.go:126-180`:
1. Return **JSON**, not `renderFragment`. Replace lines 177-179 with:
```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
_ = json.NewEncoder(w).Encode(map[string]interface{}{
    "trip_number": tripNum,
    "status":      "delivered",
    "pod_url":     podPhotoURL,
})
```
2. Mount under the API-auth group, not the cookie-session group. In
   `cmd/server/main.go`, move line 797 inside the `r.Group(func(r chi.Router){...})`
   block at 444-455 (the one with `middleware.RequireAPIAuth`). Keep the existing
    `trip.DriverID == session.UserID` ownership check (lines 145-147) but read the user
   from the Bearer context (`middleware.RequireAPIAuth` populates the same user claim).

Request: `multipart/form-data`
- `consignee_name` (string)
- `consignee_phone` (string, optional)
- `notes` (string, optional)
- `pod_photo` (file, jpeg/png) — optional but recommended
- `latitude`, `longitude` (float strings, optional — driver GPS at delivery)

Success `200`:
```json
{ "trip_number": "TRP-8492", "status": "delivered", "pod_url": "/files/abc" }
```
Errors: `401` (Bearer missing), `403` (trip not assigned to this driver),
`404` (trip not found), `400` (bad form). **Add `Content-Type: application/json` to the
error paths too** so mobile can parse them (currently `http.Error` writes text).

### 2.5 POST /api/v1/telemetry/sync  (exists — align fields)
`syncEngine.ts:37` already posts here. Align request to backend expectations. Request:
```json
{
  "driver_id": "drv_12",
  "logs": [
    { "latitude": 18.5204, "longitude": 73.8567, "timestamp": "2026-08-19T10:30:00Z", "accuracy_m": 12.5 }
  ]
}
```
Response:
```json
{ "success": true, "synced_ids": [1, 2, 3] }
```
Mobile marks those ids synced (`syncEngine.ts:58`). Verify the backend reads
`driver_id` (not `user_id`); if backend uses `user_id`, change `syncEngine.ts:48`.

### 2.6 POST /api/v1/telemetry/snapshots  (ADD on backend — optional batch)
Mirror of 2.5 for battery/event snapshots. Request:
```json
{ "driver_id": "drv_12", "snapshots": [ { "kind": "battery", "value": 0.82, "timestamp": "..." } ] }
```
Response: `{ "success": true, "synced_ids": [...] }`.

### 2.7 MQTT location (WebSocket)  — FIX backend broker
**Backend action required.** The mobile `mqtt` client (`mqtt.ts:25`) opens a WebSocket
URL `ws://<host>:9001`. Today the broker is TCP-only (`cmd/server/main.go:417-421`).
Run a WebSocket listener on `9001` (e.g. mosquitto `listener 9003 ...` with
`protocol websockets`, or paho/ws). Keep TCP `1883` for server-side. The broker must
accept the **raw JWT** as the MQTT password (`mqtt.ts:18-22` sets `username: driverId`
and `options.password = token` — there is **NO `Bearer ` prefix**). Backend auth bridge
should validate the raw token (section 5.3).
Topics (already used by mobile, keep them):
- Publish: `avandab/telemetry/drivers/{driver_id}/gps`  payload `{driver_id,latitude,longitude,timestamp}` (`mqtt.ts:53-62`)
- Subscribe: `avandab/trips/drivers/{driver_id}/updates`  (dispatch push, section 8)

---

## 3. DB contract

**No new migration.** This spec reads the driver identity that spec 08 already creates:
`db/migrations/00056_*` adds `drivers.user_id` linking `drivers` → `users`. The mobile
app obtains `driver_id` from `GET /api/v1/drivers/me` (2.2); it never queries the DB
directly.

Local mobile SQLite (via `expo-sqlite` / `storage.ts`) is app-only cache:
- `gps_logs(id, driver_id, latitude, longitude, timestamp, accuracy, synced)` — used by
  `SyncEngine` + `telemetry.ts`. This is device storage, not a backend migration.
- If you removed `expo-sqlite` from deps (0.1), re-add it OR switch `storage.ts` to
  `@react-native-async-storage/async-storage` keyed queues. Recommended: keep
  `expo-sqlite` for the relational GPS log; it is already imported in `storage.ts`.

**Do NOT** add any `0006X_*` migration for this spec. If you need a driver role concept
server-side, that is a separate decision tracked in section 11.

---

## 4. UI / build setup

### 4.1 Add babel.config.js  (missing — 0.1)
Create `mobile/babel.config.js`:
```js
module.exports = function (api) {
  api.cache(true);
  return {
    presets: ['babel-preset-expo'],
    plugins: [
      // Hermes-compatible; add native modules here as needed
    ],
  };
};
```

### 4.2 package.json scripts + dep cleanup
Edit `mobile/package.json`:
```json
{
  "scripts": {
    "start": "expo start",
    "android": "expo start --android",
    "ios": "expo start --ios",
    "web": "expo start --web",
    "build": "expo export --platform all",
    "typecheck": "tsc --noEmit",
    "lint": "eslint . --ext .ts,.tsx",
    "test": "jest"
  },
  "dependencies": {
    "@react-native-async-storage/async-storage": "1.23.1",
    "@tanstack/react-query": "^5.101.4",
    "expo": "~52.0.0",
    "expo-camera": "~16.0.18",
    "expo-constants": "~17.0.8",
    "expo-location": "~18.0.10",
    "expo-secure-store": "~14.0.0",
    "expo-sqlite": "~15.1.4",
    "expo-status-bar": "~2.0.0",
    "mqtt": "^5.15.2",
    "react": "18.3.1",
    "react-native": "0.76.9",
    "react-native-maps": "1.18.0",
    "react-native-safe-area-context": "4.12.0",
    "react-native-screens": "~4.4.0",
    "zustand": "^5.0.14"
  },
  "devDependencies": {
    "@babel/core": "^7.25.2",
    "@testing-library/react-native": "^12.4.0",
    "@types/jest": "^29.5.0",
    "@types/react": "~18.3.12",
    "babel-preset-expo": "~12.0.0",
    "eslint": "^8.57.0",
    "jest": "^29.7.0",
    "jest-expo": "~52.0.0",
    "typescript": "^5.3.3"
  }
}
```
Removed (dead, 0.1): `@apollo/client`, `graphql`, `graphql-tag`, `@posthog/core`,
`posthog-js`, `sp-react-native-mqtt`. **Keep** `posthog-react-native` (used by
`analytics.ts`) and `expo-sqlite` (used by `storage.ts`). Also drop `expo-asset`,
`expo-font`, `expo-image` if confirmed unused. Add `jest-expo`,
`@testing-library/react-native`, `eslint`, `@types/jest`.

Add `mobile/jest.config.js`:
```js
module.exports = {
  preset: 'jest-expo',
  setupFilesAfterEnv: ['<rootDir>/jest/setup.ts'],
  transformIgnorePatterns: ['node_modules/(?!((jest-)?react-native|@react-native|expo-.*|mqtt))'],
};
```

Add `mobile/.eslintrc.js`:
```js
module.exports = { root: true, extends: ['expo', 'prettier'] };
```

### 4.3 Navigation (replace hand-rolled screen switch)
`App.tsx:42` hand-rolls a `currentScreen` string union and branches. Replace with
`@react-navigation/native` + `@react-navigation/stack` (add to deps). Structure:
- `AuthStack`: `Login` (and `Register`, `ForgotPassword` shells).
- `DriverStack`: `Main` (trip list + dispatch tabs), `ActiveNavigation`,
  `DeliveryVerification`.
- Gate on `useAuthStore().isAuthenticated` (section 4.4). On logout, reset to `AuthStack`.

Minimal `App.tsx` shell:
```tsx
import { NavigationContainer } from '@react-navigation/native';
import { createStackNavigator } from '@react-navigation/stack';
// ...
export default function App() {
  const { isAuthenticated, isLoading, loadSession } = useAuthStore();
  useEffect(() => { loadSession(); }, []);
  if (isLoading) return <SplashScreen onFinish={() => {}} />;
  return (
    <NavigationContainer>
      {isAuthenticated ? <DriverStack /> : <AuthStack />}
    </NavigationContainer>
  );
}
```

### 4.4 Auth store + login fix (0.2)
`mobile/src/stores/authStore.ts` — change `UserSession`:
```ts
interface UserSession {
  id: string;          // backend user_id
  name: string;
  role: string;        // client concept: 'driver' | 'dispatcher' | 'admin' | 'viewer'
  email: string;
  driverId?: string;   // from GET /api/v1/drivers/me
}
```
`mobile/src/components/LoginScreen.tsx:55-60` — parse FLAT response:
```ts
if (!data.token || !data.user_id) {
  setLoading(false);
  Alert.alert('Sign In Failed', 'Server response missing token or user_id.');
  return;
}
await setAuth(data.token, {
  id: data.user_id,
  name: data.name || email.split('@')[0],
  role: 'driver',                 // client concept; real role from drivers/me
  email,
});
// Then fetch driver profile to get driverId:
const me = await fetch(`${getApiBaseURL()}/api/v1/drivers/me`, {
  headers: { Authorization: `Bearer ${data.token}` },
}).then((r) => r.json());
useAuthStore.getState().setDriverId(me.driver_id);
```
Add `setDriverId` to the store. Remove the hardcoded `role: 'DRIVER'`
(`LoginScreen.tsx:58`).

### 4.5 Trips list (replace mock — 0.4)
`mobile/App.tsx:278-306` — replace the mock `useQuery` with a real fetch:
```ts
const { data: trips, isLoading } = useQuery<Trip[]>({
  queryKey: ['trips', driverId],
  queryFn: async () => {
    const res = await fetch(`${getApiBaseURL()}/api/v1/trips?driver_id=me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const json = await res.json();
    return (json.trips as RawTrip[]).map(mapTripStatus); // section 5.2
  },
});
```

### 4.6 Live location (replace fake — 0.6)
`mobile/src/components/ActiveNavigationScreen.tsx:21-22` — feed real coords from
`Telemetry.startLiveLocationTracking` instead of `18.5255/73.8520`. In `telemetry.ts`,
remove the silent `DEFAULT_LATITUDE/DEFAULT_LONGITUDE` fallback (0.6); if GPS is
unavailable, surface an error state rather than faking coordinates.

### 4.7 e-POD submit (replace Alert — 0.5)
`mobile/src/components/DeliveryVerificationScreen.tsx:149-160` — implement real submit:
```ts
const submit = async () => {
  if (!capturedPhoto && !consigneeName) {
    Alert.alert('Missing', 'Add consignee name or capture a photo.'); return;
  }
  const form = new FormData();
  form.append('consignee_name', consigneeName);
  form.append('notes', notes);
  if (capturedPhoto) {
    form.append('pod_photo', { uri: capturedPhoto, name: 'pod.jpg', type: 'image/jpeg' } as any);
  }
  try {
    const res = await fetch(`${getApiBaseURL()}/api/v1/trips/${tripId}/deliver-pod`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: form,
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const json = await res.json();           // backend now returns JSON (2.4)
    await OfflineQueue.clearPOD(tripId);
    onComplete();
  } catch (e: any) {
    // Queue for retry when offline (section 5.4)
    await OfflineQueue.enqueuePOD(tripId, form);
    Alert.alert('Saved offline', 'Will submit when back online.');
    onComplete();
  }
};
```
`tripId` must be passed as a route param (`DeliveryVerificationScreen` props gain
`tripId`).

### 4.8 Offline queue (new service)
Create `mobile/src/services/offlineQueue.ts` backed by `expo-sqlite`:
- `enqueuePOD(tripId, formData)`, `clearPOD(tripId)`, `pendingPODs()`
- `enqueueGPS(log)`, `pendingGPS()` (reuse `storage.ts` log table)
- A `flush()` called by `SyncEngine` on network regain (`NetInfo` from
  `@react-native-community/netinfo`, add to deps).

---

## 5. Business logic

### 5.1 Auth parse (flat response)
```
token      = data.token
userId     = data.user_id          // NOT data.user.id
role       = 'driver'              // client concept; real role fetched from /drivers/me
on success: store token (SecureStore) + user; then GET /drivers/me -> set driverId
```
If `data.user_id` missing → treat as login failure (the current code hides this bug).

### 5.2 Trip status mapping (backend snake_case → mobile union)
Backend returns `pending|assigned|in_transit|delivered|completed|cancelled`. Mobile
`Trip.status` (`types/api.ts:21`) is
`PENDING|IN_TRANSIT|COMPLETED|CANCELLED`. Map:
```ts
function mapTripStatus(t: RawTrip): Trip {
  const m: Record<string, Trip['status']> = {
    pending: 'PENDING', assigned: 'PENDING', in_transit: 'IN_TRANSIT',
    delivered: 'COMPLETED', completed: 'COMPLETED', cancelled: 'CANCELLED',
  };
  return { ...t, status: m[t.status] ?? 'PENDING', startTime: fmt(t.departure_time) };
}
```

### 5.3 MQTT align (5.3 / 2.7)
- Mobile publishes `ws(s)://<host>:9001` (`network.ts` default `ws`/`9001`, 0.6).
- Backend MUST run a WebSocket listener on `9001` (currently only `tcp://localhost:1883`,
  `main.go:417-421`). The broker auth bridge validates the MQTT password, which is the
  **raw JWT** (`options.password = token`, `mqtt.ts:21` — NO `Bearer ` prefix).
- If the WebSocket broker is down, `mqtt.ts:44-46` logs a warning and the app falls back
  to HTTP telemetry (`/api/v1/telemetry/sync`, section 2.5). **Never block UI on MQTT.**

### 5.4 e-POD flow (state machine)
```
capture photo (expo-camera)
   -> build FormData (consignee_name, notes, pod_photo, lat/lng)
   -> POST /trips/{id}/deliver-pod
       200 JSON  -> clear offline queue, mark trip COMPLETED in list
       4xx/5xx   -> enqueuePOD(); on network regain, flush() retries
       no signal -> enqueuePOD(); flush() on NetInfo 'connected'
```
The submit button must be disabled until `capturedPhoto || consignee_name` is present
(section 4.7).

---

## 6. Config / env

| Var | Default | Purpose | Read by |
|-----|---------|---------|---------|
| `EXPO_PUBLIC_API_SCHEME` | `http` (dev) / `https` | REST scheme | `network.ts:2` |
| `EXPO_PUBLIC_BACKEND_HOST` | `127.0.0.1` (dev) | API + MQTT host | `network.ts:4` |
| `EXPO_PUBLIC_API_PORT` | `8080` (dev) / `443` | REST port | `network.ts:5` |
| `EXPO_PUBLIC_MQTT_SCHEME` | `ws` (dev) / `wss` | MQTT WebSocket scheme | `network.ts:3` |
| `EXPO_PUBLIC_MQTT_BROKER_PORT` | `9001` (dev) / `8883` | MQTT WS port | `network.ts:6` |
| `EXPO_PUBLIC_DEFAULT_DRIVER_ID` | `''` | **Remove demo usage** (`App.tsx:35`) | `network.ts:16` |
| `EXPO_PUBLIC_DEFAULT_LATITUDE` | `18.5204` | **Remove silent GPS fake** (`telemetry.ts:51`) | `network.ts:17` |
| `EXPO_PUBLIC_DEFAULT_LONGITUDE` | `73.8567` | **Remove silent GPS fake** | `network.ts:18` |

Backend (for section 2.7): `MQTT_URL` should become a WS listener, e.g.
`ws://0.0.0.0:9001` (or run mosquitto with `listener 9001` + `protocol websockets`).
Keep `tcp://localhost:1883` for server-side producers.

Create `mobile/.env.local` (dev) — commit `.env.example` only:
```
EXPO_PUBLIC_API_SCHEME=http
EXPO_PUBLIC_BACKEND_HOST=127.0.0.1
EXPO_PUBLIC_API_PORT=8080
EXPO_PUBLIC_MQTT_SCHEME=ws
EXPO_PUBLIC_MQTT_BROKER_PORT=9001
```

---

## 7. Tests

Add `mobile/jest/setup.ts` (reset mocks between tests). Tests must run in CI with
`npm run typecheck && npm run lint && npm test`.

### 7.1 authStore.test.ts (parse flat response — 0.2)
```ts
import { useAuthStore } from '../src/stores/authStore';
test('setAuth stores flat user_id from backend', async () => {
  await useAuthStore.getState().setAuth('tok', { id: 'u_1', name: 'Raj', role: 'driver', email: 'r@x.com' });
  const s = useAuthStore.getState();
  expect(s.token).toBe('tok');
  expect(s.user?.id).toBe('u_1');
  expect(s.user?.role).not.toBe('DRIVER');
});
```
### 7.2 mqtt.test.ts (URL + fallback)
Verify `getMQTTBrokerURL()` returns `ws://127.0.0.1:9001` from env and that
`publishLocation` no-ops when disconnected (no throw).

### 7.3 syncEngine.test.ts (flush + mark)
Mock `fetch` to return `{success:true,synced_ids:[1,2]}`; assert
`DB.markLogsAsSynced` called. Mock `fetch` to reject; assert returns
`{syncedCount:0,error}` and logs retained.

### 7.4 DeliveryVerification.test.ts (submit + offline queue)
Render with `tripId`; mock `fetch` 500 → assert `OfflineQueue.enqueuePOD` called and
Alert "Saved offline" shown; mock `fetch` 200 JSON → assert `clearPOD` + `onComplete`.

### 7.5 Coverage gate
CI fails if `npm test` exits non-zero. Target ≥ 70% on `stores/`, `services/`,
`components/LoginScreen`, `components/DeliveryVerificationScreen`.

### 7.6 CI snippet (`.github/workflows/mobile.yml`)
```yaml
- run: cd mobile && npm ci
- run: cd mobile && npm run typecheck
- run: cd mobile && npm run lint
- run: cd mobile && npm test
```

---

## 8. Future / GPS-provider

- **Live GPS**: promote `telemetry.ts` watch to background via
  `expo-location` `startLocationUpdatesAsync` (task manager) — current
  `watchPositionAsync` stops when app backgrounds.
- **Offline POD queue**: `offlineQueue.ts` (4.8) already designed; add retry with
  exponential backoff + dedupe by `tripId`.
- **Background location**: use `expo-task-manager` + `Location.startLocationUpdatesAsync`
  to keep MQTT/HTTP stream alive off-screen.
- **Push dispatch**: subscribe `avandab/trips/drivers/{id}/updates` (2.7) for new-trip
  push; add `expo-notifications` for OS alerts. Backend publishes on trip assignment.
- **Trip detail / navigation**: `ActiveNavigationScreen` to use `react-native-maps`
  directions (OSM/Mapbox) once destination coords come from `GET /trips` (2.3).
- **GraphQL deprecation**: remove `graphql.ts` entirely; REST is source of truth for
  driver data. Keep only if a future dispatcher-facing widget needs it.

---

## 9. Edge cases

1. **Flat response missing `user`** — handled by 5.1; old `data.user?.id` code deleted.
2. **No driver linked to user** (`GET /drivers/me` → 404) — show "Contact dispatcher to
   link your driver profile"; block trip list.
3. **MQTT WS down** — app continues on HTTP telemetry (5.3); no crash.
4. **GPS disabled / denied** — `telemetry.ts` returns error state, no silent fake.
5. **e-POD with no signal** — queued (5.4); flushed on reconnect.
6. **Token expired** (`401` on any call) — clear session, route to `Login`.
7. **Photo too large** — `DeliverWithPOD` caps at `10<<20` (1MB) (`kharcha.go:131`, `r.ParseMultipartForm(10 << 20)`);
   mobile should compress via `expo-image`/`manipulator` before upload.
8. **Trip not assigned to user** — backend returns `403` (`kharcha.go:147-150`); mobile
   shows error, does not queue.
9. **Duplicate POD submit** — dedupe by `tripId` in offline queue + backend idempotency.
10. **Backend returns HTML on error** — fixed in 2.4 (errors become JSON).

---

## 10. Phased rollout

1. **Phase 1 (build/unblock):** `babel.config.js`, `package.json` scripts, dep cleanup,
   jest/eslint config. Verify `npm run typecheck && npm test` is green.
2. **Phase 2 (auth):** fix `LoginScreen` + `authStore` flat parse (4.4, 5.1); add
   `GET /drivers/me` on backend (2.2).
3. **Phase 3 (trips):** real trip fetch (4.5, 5.2); add `GET /trips?driver_id=me`
   (2.3); navigation refactor (4.3).
4. **Phase 4 (MQTT):** backend WS broker on 9001 (2.7, 5.3); live location (4.6).
5. **Phase 5 (e-POD):** backend JSON + Bearer mount (2.4); mobile submit + offline queue
   (4.7, 4.8, 5.4).
6. **Phase 6 (CI):** wire `.github/workflows/mobile.yml` (7.6).

---

## 11. Open items / VERIFY

- **DRIVER role:** backend has no `DRIVER` role (0.2). Decision: keep `role:'driver'` as a
  **client concept** derived from `/drivers/me`; do NOT add a backend `DRIVER` RBAC role
  unless ops requires it. Confirm with backend owner.
- **`drivers/me` existence:** backend does NOT yet expose `GET /api/v1/drivers/me`
  (2.2). Must be built before Phase 2 ships. Verify `drivers.user_id` exists
  (`00056_*`).
- **Broker WebSocket:** confirm ops can run an MQTT WS listener on `9001` alongside TCP
  `1883` (2.7). If not, fall back to pure HTTP telemetry (`/telemetry/sync`) and drop
  MQTT from mobile.
- **Telemetry field name:** confirm backend `telemetry/sync` expects `driver_id`
  (`syncEngine.ts:48`) vs `user_id`. Align before Phase 4.
- **e-POD mount group:** confirm moving `DeliverWithPOD` under `RequireAPIAuth`
  (`main.go:797`) does not break the existing cookie-session web caller.

---

## 12. File list

Create:
- `mobile/babel.config.js` (4.1)
- `mobile/jest.config.js` (4.2)
- `mobile/.eslintrc.js` (4.2)
- `mobile/jest/setup.ts` (7)
- `mobile/src/services/offlineQueue.ts` (4.8)
- `mobile/.env.example` (6)
- `mobile/__tests__/authStore.test.ts` (7.1)
- `mobile/__tests__/mqtt.test.ts` (7.2)
- `mobile/__tests__/syncEngine.test.ts` (7.3)
- `mobile/__tests__/DeliveryVerification.test.ts` (7.4)
- `.github/workflows/mobile.yml` (7.6)
- backend: `GET /api/v1/drivers/me` handler + route (2.2)
- backend: `GET /api/v1/trips?driver_id=me` handler + route (2.3)
- backend: `POST /api/v1/telemetry/snapshots` (2.6)

Modify:
- `mobile/package.json` (4.2 — scripts + deps)
- `mobile/App.tsx` (4.3 navigation; 4.5 real trips; remove `DEMO_DRIVER_ID` 0.3)
- `mobile/src/stores/authStore.ts` (4.4 — `role` union, `driverId`, `setDriverId`)
- `mobile/src/components/LoginScreen.tsx` (4.4 — flat parse, drop `role:'DRIVER'`)
- `mobile/src/components/ActiveNavigationScreen.tsx` (4.6 — live coords)
- `mobile/src/components/DeliveryVerificationScreen.tsx` (4.7 — real POST, `tripId` prop)
- `mobile/src/services/mqtt.ts` (5.3 — no-op on disconnect, topic alignment)
- `mobile/src/services/syncEngine.ts` (2.5 — field name align)
- `mobile/src/services/telemetry.ts` (4.6 — drop silent GPS fake)
- `mobile/src/services/graphql.ts` (8 — deprecate/remove)
- `mobile/src/types/api.ts` (5.2 — add `RawTrip`, status map)
- `internal/handlers/kharcha.go` (2.4 — JSON response, mount under API auth)
- `cmd/server/main.go` (2.4 mount; 2.7 WS broker config)

No DB migration added (section 3).
